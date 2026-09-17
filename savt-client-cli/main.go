// nextclient: 原语解释器。循环: POST /next(带上一轮报告) -> 执行 send/listen/wait -> 汇报。
// send.srcMode: spoof=pcap照抄探测源(伪造) / auto=本机真实源(内核socket,打洞/真实源探测)
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"sav-next/common"
)

var (
	serverFlag  = flag.String("server", "", "server url (缺省: 依据 service.json 的 env 自动解析)")
	server6Flag = flag.String("server6", "", "IPv6 server url (守护模式双栈的v6栈; 绕过DNS劫持环境如Surge fake-ip)")
	daemonFlag  = flag.Bool("daemon", false, "守护模式: gRPC服务面(老GUI可遥控) + 定时调度器")
)

// serverURL 解析后的服务器地址(启动时填入)
var serverURL string

var (
	token   []byte
	vid     int64
	dev     *common.Device
	sockets = map[int]*net.UDPConn{} // port -> 内核socket(auto发送与udp监听共用)

	repMu   sync.Mutex // 并发监听goroutine写报告map

	sentMu sync.Mutex
	sent   = map[uint16]sentInfo{} // 内层IP ID -> 发送记录(供ICMP匹配)
)

type sentInfo struct {
	seq int64
	ttl int64
}

func main() {
	flag.Parse()
	log.SetFlags(log.Ltime)

	// 权限提示(测量需要 root/管理员)
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		log.Printf("警告: 未以 root 运行, 测量(pcap注入/原始socket)将失败, 请使用 sudo")
	}

	if *daemonFlag {
		runDaemon() // daemon.go: gRPC服务面 + 定时调度器
		return
	}

	if runOnce(true, "") == nil {
		os.Exit(1)
	}
}

// measuredFamily 由服务器地址推断本次测量的栈族
func measuredFamily(serverURL string) string {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "IPv4"
	}
	host := u.Hostname()
	if strings.Contains(host, ":") {
		return "IPv6"
	}
	return "IPv4"
}

// runOnce 完成一次完整测量(建会话→六轮→finish)。
// verbose=true 时打印结果JSON(一次性模式); 守护模式传false(结果走gRPC/历史记录)。
// 返回服务端判定结果(失败时返回nil)。
func runOnce(verbose bool, family string) *common.Result {
	// 0. 重置会话级状态(守护模式多次测量之间不串扰; serverURL残留会让失败遍历误用上一栈地址)
	token = nil
	vid = 0
	allProbes = nil
	serverURL = ""
	if family != "" && *serverFlag == "" && len(resolveServerCandidatesFamily(family)) == 0 {
		log.Printf("[%s] 无可用服务器候选(解析失败), 跳过该栈", family)
		return nil
	}

	// 1. 服务器地址解析 + 建会话(family="4"/"6"指定栈, ""任意; --server 最优先)
	candidates := resolveServerCandidatesFamily(family)
	var info common.SessionInfo
	var err error
	for _, cand := range candidates {
		if err = postJSONintoErr(cand+"/api/session", nil, &info); err == nil {
			serverURL = cand
			break
		}
		log.Printf("server %s 不可达: %v", cand, err)
	}
	if serverURL == "" {
		if family == "6" {
			log.Printf("IPv6 服务器不可达(本机可能无 v6 路由), 跳过 v6 测量")
		} else {
			log.Printf("所有候选服务器均不可达")
		}
		return nil
	}
	token, _ = hex.DecodeString(info.TokenHex)
	vid = info.ID
	log.Printf("server %s", serverURL)
	log.Printf("session %d created, rounds: %v", vid, info.Rounds)

	// 2. 出口设备发现(伪造发送需要二层帧头)
	srvUDP, err := urlToUDP(serverURL)
	if err != nil {
		log.Printf("server addr: %v", err)
		return nil
	}
	if d, err := common.DiscoverDevice(srvUDP); err != nil {
		log.Printf("device discovery: %v", err)
		return nil
	} else {
		dev = d
	}
	log.Printf("egress %s dev %s", dev.LocalIP, dev.PcapName)

	// 3. 轮次循环
	total := len(info.Rounds)
	var report *common.ClientReport
	for round := 0; ; round++ {
		body := common.ClientReport{Round: int64(round)}
		if report != nil {
			body = *report
		}
		var resp common.NextResponse
		if err := postJSONintoErr(fmt.Sprintf("%s/api/session/%d/next", serverURL, vid), body, &resp); err != nil {
			log.Printf("round %d: %v", round, err)
			return nil
		}
		if resp.Op == "finish" {
			// 不再调用roundProgress: 此处round已自增到最后一轮+1,会产生幽灵步骤;
			// 最后一轮的完成态已在上一次executeRound之后标记。
			raw, _ := json.MarshalIndent(resp.Result, "", " ")
			if verbose {
				fmt.Printf("\n===== 测量结果 =====\n%s\n", raw)
			}
			return resp.Result
		}
		if roundProgress != nil {
			roundProgress(measuredFamily(serverURL), resp.Name, round, total, false)
		}
		report = executeRound(round, resp.Name, resp.Actions)
		if roundProgress != nil {
			roundProgress(measuredFamily(serverURL), resp.Name, round, total, true)
		}
	}
}

// roundProgress 轮次进度钩子(守护模式注册用于GUI实时显示; 一次性模式为nil)。
// family="IPv4"/"IPv6" name=轮名 round=轮序(栈内) total=栈内总轮数 done=该轮是否完成
var roundProgress func(family, name string, round, total int, done bool)

// executeRound 执行一轮原语动作,产出该轮报告。
// 所有监听(icmp+udp)先于其它动作并发启动(icmp的差错在发送期间返回;多端口udp监听需同时开窗)。
func executeRound(round int, name string, actions []common.Action) *common.ClientReport {
	log.Printf("-- round %d [%s]: %d actions", round, name, len(actions))
	rep := &common.ClientReport{Round: int64(round), UDP: map[int64]string{}, Hops: map[string]map[int64]string{}}
	var waiters []chan struct{}
	for _, a := range actions {
		if a.Op != "listen" {
			continue
		}
		isICMP := a.Proto == "icmp"
		// 多个udp监听也并发(不同端口各自开窗);单udp监听若与打洞同socket(同端口)则仍串行,
		// 但并发启动socket由getSocket复用,无冲突。
		if !isICMP && countListens(actions, "udp") <= 1 {
			continue // 单udp监听保持原有串行位置(打洞send先行,随后监听)
		}
		ch := make(chan struct{})
		waiters = append(waiters, ch)
		if isICMP {
			go func(a common.Action) { defer close(ch); doListenICMP(a, rep) }(a)
		} else {
			go func(a common.Action) { defer close(ch); doListenUDP(a, rep) }(a)
		}
	}
	for _, a := range actions {
		if a.Op == "listen" {
			isICMP := a.Proto == "icmp"
			if isICMP || countListens(actions, "udp") > 1 {
				continue // 已并发启动
			}
		}
		switch a.Op {
		case "send":
			doSend(a, rep)
		case "listen":
			doListenUDP(a, rep)
		case "wait":
			log.Printf("   wait %dms", a.DurationMs)
			time.Sleep(time.Duration(a.DurationMs) * time.Millisecond)
		default:
			log.Printf("   unknown op %q skipped", a.Op)
		}
	}
	for _, ch := range waiters {
		<-ch
	}
	return rep
}

func countListens(actions []common.Action, proto string) int {
	n := 0
	for _, a := range actions {
		if a.Op == "listen" && a.Proto == proto {
			n++
		}
	}
	return n
}

// ---- send 原语 ----

func doSend(a common.Action, rep *common.ClientReport) {
	if a.SrcMode == "auto" && len(a.Probes) == 0 {
		// 打洞: 内核socket真实源,单包(目的缺省为服务端)
		sk := getSocket(a.SrcPort)
		if sk == nil {
			return
		}
		dstHost := a.DstAddr
		if dstHost == "" {
			u, _ := url.Parse(serverURL)
			dstHost = u.Hostname()
		}
		dst, err := net.ResolveUDPAddr("udp", net.JoinHostPort(dstHost, fmt.Sprintf("%d", a.DstPort)))
		if err != nil {
			log.Printf("   punch resolve: %v", err)
			return
		}
		for i := 0; i < max(1, a.Count); i++ {
			sk.WriteToUDP(common.EncodePunch(vid, token), dst)
		}
		log.Printf("   send(auto/punch) -> %s from :%d", dst, a.SrcPort)
		return
	}
	if len(a.Probes) == 0 {
		return
	}
	interval := time.Duration(a.IntervalMs) * time.Millisecond
	rememberProbes(a.Probes)
	ttls := []int{0}
	if a.TTLMax > a.TTLMin {
		ttls = nil
		for t := a.TTLMin; t <= a.TTLMax; t++ {
			ttls = append(ttls, t)
		}
	}
	var packets [][]byte
	var useDev *common.Device
	for _, p := range a.Probes {
		src := net.ParseIP(p.SrcAddr)
		if a.SrcMode == "auto" || src == nil {
			src = dev.LocalIP // AUTO: 本机真实源(在客户端就地解析)
		}
		dst := net.ParseIP(p.DstAddr)
		for _, ttl := range ttls {
			payload := common.EncodePayload(vid, p.Seq, token)
			var pkt []byte
			var ipID uint16
			var err error
			if src.To4() == nil || dst.To4() == nil {
				pkt, ipID, err = common.BuildUDPPacket6(dev, src, dst, p.SrcPort, p.DstPort, ttl, payload)
			} else {
				pkt, ipID, err = common.BuildUDPPacket(dev, src, dst, p.SrcPort, p.DstPort, ttl, payload)
			}
			if err != nil {
				log.Printf("   build: %v", err)
				continue
			}
			useDev = dev
			packets = append(packets, pkt)
			sentMu.Lock()
			sent[ipID] = sentInfo{seq: p.Seq, ttl: int64(ttl)}
			sentMu.Unlock()
		}
	}
	if len(packets) == 0 || useDev == nil {
		return
	}
	n := 0
	for i := 0; i < max(1, a.Count); i++ {
		if err := common.SendPackets(dev.PcapName, packets, interval); err != nil {
			log.Printf("   send: %v", err)
			return
		}
		n += len(packets)
	}
	log.Printf("   send(%s) %d probes x ttl%d x %d = %d pkts", a.SrcMode, len(a.Probes), len(ttls), a.Count, n)
}

// ---- listen 原语 ----

func doListenUDP(a common.Action, rep *common.ClientReport) {
	sk := getSocket(a.Port)
	if sk == nil {
		time.Sleep(time.Duration(a.DurationMs) * time.Millisecond) // bind失败: 保持窗口时长后空手返回
		return
	}
	deadline := time.Now().Add(time.Duration(a.DurationMs) * time.Millisecond)
	buf := make([]byte, 1500)
	got := 0
	for time.Now().Before(deadline) {
		sk.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, addr, err := sk.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		p := buf[:n]
		v, seq, ok := common.Peek(p)
		if !ok || v != vid || !common.Verify(p, token) || seq == common.PunchSeq {
			continue
		}
		repMu.Lock()
		rep.UDP[seq] = addr.IP.String()
		repMu.Unlock()
		got++
	}
	log.Printf("   listen(udp :%d %dms) received %d probes", a.Port, a.DurationMs, got)
}

func doListenICMP(a common.Action, rep *common.ClientReport) {
	network, v6 := "ip4:icmp", dev.LocalIP.To4() == nil
	if v6 {
		network = "ip6:ipv6-icmp"
	}
	conn, err := net.ListenPacket(network, dev.LocalIP.String())
	if err != nil {
		log.Printf("   listen(icmp): %v", err)
		return
	}
	defer conn.Close()
	deadline := time.Now().Add(time.Duration(a.DurationMs) * time.Millisecond)
	buf := make([]byte, 4096)
	got, raw := 0, 0
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}
		raw++
		var seq, hopKey int64
		var ok bool
		if v6 {
			seq, hopKey, ok = matchInner6(buf[8:n])
		} else {
			seq, hopKey, ok = matchInner(buf[8:n])
		}
		if !ok {
			continue
		}
		tag := fmt.Sprintf("seq-%d", seq)
		if p := findProbe(seq); p != nil {
			tag = p.Tag
		}
		repMu.Lock()
		if rep.Hops[tag] == nil {
			rep.Hops[tag] = map[int64]string{}
		}
		rep.Hops[tag][hopKey] = addr.String()
		repMu.Unlock()
		got++
	}
	log.Printf("   listen(icmp %dms) raw=%d matched=%d", a.DurationMs, raw, got)
}

// matchInner6: ICMPv6头8B + 内层IPv6头40B定长 + UDP 8B + 载荷;RFC4443引用至多1280B,载荷匹配可靠
func matchInner6(inner []byte) (seq int64, hopKey int64, ok bool) {
	if len(inner) < 40+8+common.PayloadLen || inner[0]>>4 != 6 || inner[6] != 17 {
		return 0, 0, false
	}
	payload := inner[48:]
	v, s, good := common.Peek(payload)
	if !good || v != vid || !common.Verify(payload, token) {
		return 0, 0, false
	}
	return s, int64(inner[7]), true // hop limit
}

// matchInner 优先载荷匹配(Linux路由器引用较长);不足时按内层IP ID对照本地发送记录,
// 并用发送时记录的TTL作跳键(RFC792只引用内层IP头+8字节,且各栈引用TTL语义不一)。
func matchInner(inner []byte) (seq int64, hopKey int64, ok bool) {
	ihl := int(inner[0]&0x0f) * 4
	if len(inner) >= ihl+8+common.PayloadLen {
		if payload := inner[ihl+8:]; len(payload) == common.PayloadLen {
			if v, s, good := common.Peek(payload); good && v == vid && common.Verify(payload, token) {
				return s, int64(inner[8]), true
			}
		}
	}
	sentMu.Lock()
	defer sentMu.Unlock()
	info, found := sent[binary.BigEndian.Uint16(inner[4:6])]
	if !found {
		return 0, 0, false
	}
	return info.seq, info.ttl, true
}

// ---- 辅助 ----

var allProbes []common.Probe

func findProbe(seq int64) *common.Probe {
	for i := range allProbes {
		if allProbes[i].Seq == seq {
			return &allProbes[i]
		}
	}
	return nil
}

var sockMu sync.Mutex

func getSocket(port int) *net.UDPConn {
	sockMu.Lock()
	defer sockMu.Unlock()
	if sk, ok := sockets[port]; ok {
		return sk
	}
	sk, err := net.ListenUDP("udp", &net.UDPAddr{IP: dev.LocalIP, Port: port})
	if err != nil {
		log.Printf("bind udp :%d: %v (端口被占? 本轮监听/打洞将无效)", port, err)
		return nil
	}
	sockets[port] = sk
	return sk
}

// ---- 服务器地址解析(机制与老架构 savt-client-api 一致) ----

type serviceConfig struct {
	Version string `json:"version"`
	Env     string `json:"env"`
}

// systemConfigDir 系统配置目录(与老架构 GetSysConfig 相同)
func systemConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("PROGRAMDATA"), "savt-client")
	case "darwin":
		return "/Library/Application Support/savt-client"
	default:
		return "/var/lib/savt-client"
	}
}

// loadServiceConfig 按老架构的搜索顺序读 service.json: ../ → ./ → 系统配置目录
func loadServiceConfig() (serviceConfig, bool) {
	for _, p := range []string{
		filepath.Join("..", "service.json"),
		"service.json",
		filepath.Join(systemConfigDir(), "service.json"),
	} {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var c serviceConfig
		if err := json.Unmarshal(data, &c); err != nil || c.Env == "" {
			continue
		}
		log.Printf("config %s: env=%s version=%s", p, c.Env, c.Version)
		return c, true
	}
	return serviceConfig{}, false
}

// resolveServerCandidates 解析候选服务器: --server 优先; 否则 env→端口 + 域名DNS(与老架构一致)
func resolveServerCandidates() []string {
	return resolveServerCandidatesFamily("")
}

// resolveServerCandidatesFamily 按栈族过滤候选(""/"4"/"6"); --server 指定时原样返回。
func resolveServerCandidatesFamily(family string) []string {
	if family == "6" && *server6Flag != "" {
		return []string{*server6Flag}
	}
	if *serverFlag != "" && *server6Flag == "" {
		return []string{*serverFlag}
	}
	if *serverFlag != "" && family == "" {
		return []string{*serverFlag}
	}
	cfg, ok := loadServiceConfig()
	if !ok {
		log.Printf("未指定 --server, 且未找到 service.json(搜索: ./、../、%s)", systemConfigDir())
		return nil
	}
	port := "31452" // 非 test 环境
	if cfg.Env == "test" {
		port = "41452"
	}
	var cands []string
	for _, dom := range []string{"v4.sav-t.ki3.org.cn", "v6.sav-t.ki3.org.cn"} {
		ips, err := net.LookupIP(dom)
		if err != nil {
			time.Sleep(500 * time.Millisecond) // 瞬时抖动重试一次(守护模式不容失败)
			ips, err = net.LookupIP(dom)
		}
		if err != nil {
			log.Printf("DNS %s: %v", dom, err)
			continue
		}
		for _, ip := range ips {
			if v4 := ip.To4(); v4 != nil {
				if family == "6" {
					continue
				}
				cands = append(cands, fmt.Sprintf("http://%s:%s", v4, port))
			} else {
				if family == "4" {
					continue
				}
				cands = append(cands, fmt.Sprintf("http://[%s]:%s", ip, port))
			}
		}
	}
	if len(cands) == 0 {
		log.Printf("无法解析服务器域名(v4/v6.sav-t.ki3.org.cn), 跳过本次[fam=%s]", family)
	}
	return cands
}

// postJSONintoErr 同 postJSONinto 但返回错误(供多候选探测)
func postJSONintoErr(u string, body any, out any) error {
	raw, _ := json.Marshal(body)
	resp, err := http.Post(u, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// 记录见过的探测(供ICMP匹配tag)
func rememberProbes(ps []common.Probe) { allProbes = append(allProbes, ps...) }

func urlToUDP(s string) (*net.UDPAddr, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	return net.ResolveUDPAddr("udp", u.Host)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
