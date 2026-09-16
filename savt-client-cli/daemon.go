// daemon.go: 守护模式——定时调度器 + 配置持久化 + 历史记录。
// 从老 worker(savt-client 2.x scheduler.go)移植,行为对齐:
//   1秒tick 的五组倒计时状态机: Complete(周期)/Incomplete(失败重试)/
//   CheckNetwork(网络侦测)/WaitAfterChange(网络稳定等待)/RetryLimit(重试上限)。
// 差异: 测量核心换成 runOnce()(新协议),网络侦测用 /api/validation/check(兼容层端点)。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"savt-client/savt-client-cli/common"
)

// ---- 配置(与老 worker config.json 字段名一致,GUI 可无缝读写) ----

type daemonConfig struct {
	Enable          bool `json:"Enable"`
	Complete        int  `json:"Complete"`        // 周期(秒),默认7天
	Incomplete      int  `json:"Incomplete"`      // 失败重试间隔(秒)
	CheckNetwork    int  `json:"CheckNetwork"`    // 网络侦测间隔(秒)
	WaitAfterChange int  `json:"WaitAfterChange"` // 网络变化后稳定等待(秒)
	RetryLimit      int  `json:"RetryLimit"`
	IsPublic        bool `json:"IsPublic"`
}

func defaultConfig() daemonConfig {
	return daemonConfig{
		Enable:          true,
		Complete:        604800,
		Incomplete:      600,
		CheckNetwork:    1800,
		WaitAfterChange: 60,
		RetryLimit:      3,
		IsPublic:        true,
	}
}

// daemonDirs 配置/历史目录(与老 worker 的 GetSysConfig 布局一致)
func daemonDirs() (configDir, historyDir string) {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("PROGRAMDATA")
		configDir = filepath.Join(base, "savt-client")
	case "darwin":
		configDir = "/Library/Application Support/savt-client"
	default:
		configDir = "/var/lib/savt-client"
	}
	return configDir, filepath.Join(configDir, "history")
}

func loadDaemonConfig() daemonConfig {
	dir, _ := daemonDirs()
	cfg := defaultConfig()
	raw, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return defaultConfig()
	}
	return cfg
}

func saveDaemonConfig(cfg daemonConfig) {
	dir, _ := daemonDirs()
	os.MkdirAll(dir, 0755)
	raw, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(dir, "config.json"), raw, 0644)
}

// ---- 历史记录(与老 worker history/ 布局一致,上限10份) ----

const maxHistoryFiles = 10

func writeHistory(jobID string, rec map[string]any) {
	_, histDir := daemonDirs()
	os.MkdirAll(histDir, 0755)
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(histDir, jobID+".json"), raw, 0644)
	enforceHistoryLimit(histDir, maxHistoryFiles)
}

func enforceHistoryLimit(dir string, maxFiles int) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var infos []os.FileInfo
	for _, f := range files {
		if !f.IsDir() {
			if info, err := f.Info(); err == nil {
				infos = append(infos, info)
			}
		}
	}
	if len(infos) <= maxFiles {
		return
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].ModTime().Before(infos[j].ModTime()) })
	for _, info := range infos[:len(infos)-maxFiles] {
		os.Remove(filepath.Join(dir, info.Name()))
	}
}

func jobIDNow() string {
	t := time.Now().Format(time.RFC1123)
	t = strings.ReplaceAll(t, ",", "")
	t = strings.ReplaceAll(t, " ", "_")
	return strings.ReplaceAll(t, ":", "-")
}

// ---- 调度器(移植自老 worker runScheduler: 1秒tick倒计时状态机) ----

type timeCounter struct {
	complete        int
	incomplete      int
	checkNetwork    int
	waitAfterChange int
	retryLimit      int
}

type scheduler struct {
	mu  sync.Mutex
	cfg daemonConfig

	// 测量互斥: 手动(Start)与定时共用一个执行锁
	runningMu sync.Mutex
	running   bool

	stopCh chan struct{}
}

var sched = &scheduler{}

// runMeasurement 执行一次测量(手动与定时共用),返回是否成功。
func (s *scheduler) runMeasurement(scheduled bool) (bool, *common.Result) {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return false, nil
	}
	s.running = true
	s.runningMu.Unlock()
	defer func() {
		s.runningMu.Lock()
		s.running = false
		s.runningMu.Unlock()
	}()

	jobID := jobIDNow()
	start := time.Now()
	log.Printf("[daemon] %s 测量开始 (%s)", jobID, map[bool]string{true: "定时", false: "手动"}[scheduled])

	res := runOnce(false)

	ok := res != nil
	rec := map[string]any{
		"job_id":           jobID,
		"status":           map[bool]int{true: 2, false: 4}[ok], // 与老pb状态对齐: 2=SUCCEEDED 4=FAILED
		"start_time":       start.Format(time.RFC3339),
		"duration_seconds": int(time.Since(start).Seconds()),
		"scheduled":        scheduled,
	}
	if res != nil {
		rec["result"] = res
	}
	writeHistory(jobID, rec)
	log.Printf("[daemon] %s 测量结束 ok=%v 耗时%ds", jobID, ok, int(time.Since(start).Seconds()))
	return ok, res
}

// checkNetworkChange 网络变化侦测: 对比当前公网IP与缓存(经兼容层check端点)
func checkNetworkChange(oldV4, oldV6 string) (changed bool, newV4, newV6 string) {
	for _, cand := range resolveServerCandidates() {
		resp, err := http.Post(cand+"/api/validation/check", "application/json", nil)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		ip := strings.TrimSpace(string(body))
		if ip == "" {
			continue
		}
		if strings.Contains(ip, ":") {
			newV6 = ip
		} else {
			newV4 = ip
		}
	}
	changed = (oldV4 != "" && newV4 != "" && newV4 != oldV4) || (oldV6 != "" && newV6 != "" && newV6 != oldV6)
	return changed, newV4, newV6
}

// runSchedulerLoop 主循环(移植自老 worker runScheduler,语义一致)
func (s *scheduler) runSchedulerLoop() {
	s.mu.Lock()
	tc := &timeCounter{
		complete:        s.cfg.Complete,
		incomplete:      -1,
		checkNetwork:    0,
		waitAfterChange: -1,
		retryLimit:      -1,
	}
	cfg := s.cfg
	s.mu.Unlock()

	clientV4, clientV6 := "", ""

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			cfg = s.cfg
			s.mu.Unlock()

			// 网络侦测到点
			if tc.checkNetwork == 0 {
				changed, v4, v6 := checkNetworkChange(clientV4, clientV6)
				clientV4, clientV6 = v4, v6
				if changed {
					tc.waitAfterChange = cfg.WaitAfterChange
					log.Printf("[daemon] 网络变化: v4=%s v6=%s, %ds后重测", v4, v6, cfg.WaitAfterChange)
				}
				tc.checkNetwork = cfg.CheckNetwork
			}
			// 网络变化稳定等待结束 → 立即进入测量
			if tc.waitAfterChange == 0 {
				tc.waitAfterChange = -1
				tc.complete = 0
			}
			// 周期到点 → 测量
			if tc.complete == 0 && cfg.Enable {
				if ok, _ := s.runMeasurement(true); ok {
					tc.complete = cfg.Complete
					tc.incomplete = -1
					tc.retryLimit = -1
					tc.waitAfterChange = -1
				} else {
					tc.complete = -1
					tc.incomplete = cfg.Incomplete
					tc.retryLimit = cfg.RetryLimit
				}
			}
			// 失败重试到点
			if tc.incomplete == 0 && cfg.Enable {
				if tc.retryLimit == 0 {
					tc.complete = cfg.Complete // 放弃重试,回到周期
					tc.incomplete = -1
					tc.retryLimit = -1
				} else if ok, _ := s.runMeasurement(true); ok {
					tc.complete = cfg.Complete
					tc.incomplete = -1
					tc.retryLimit = -1
					tc.waitAfterChange = -1
				} else {
					tc.incomplete = cfg.Incomplete
					tc.retryLimit--
				}
			}

			// 递减(-1=未激活)
			tc.checkNetwork--
			if tc.complete != -1 {
				tc.complete--
			}
			if tc.incomplete != -1 {
				tc.incomplete--
			}
			if tc.waitAfterChange != -1 {
				tc.waitAfterChange--
			}
		case <-s.stopCh:
			return
		}
	}
}

// ---- 守护模式入口 ----

func runDaemon() {
	// 权限: 守护模式必须root(要写系统目录+测量)
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		log.Fatal("守护模式需要 root (系统配置目录 + pcap测量)")
	}

	sched.cfg = loadDaemonConfig()
	sched.stopCh = make(chan struct{})

	dir, _ := daemonDirs()
	log.Printf("[daemon] 配置目录 %s | 调度: %s", dir, daemonConfigSummary(sched.cfg))

	// 先起 gRPC 服务面(老GUI可遥控),再起调度器
	startGRPCServer(40001)
	go sched.runSchedulerLoop()

	log.Printf("[daemon] 运行中 (gRPC :40001, 定时%s)", fmt.Sprintf("Complete=%ds", sched.cfg.Complete))
	select {} // 常驻
}

func daemonConfigSummary(c daemonConfig) string {
	return fmt.Sprintf("Enable=%v Complete=%ds Incomplete=%ds CheckNetwork=%ds Retry=%d",
		c.Enable, c.Complete, c.Incomplete, c.CheckNetwork, c.RetryLimit)
}

// 防止"未使用导入"的小工具
var _ = net.JoinHostPort
