// grpc.go: gRPC 服务面——让老 GUI(以及任何 gRPC 客户端)遥控新 CLI 核心。
// 接口与老 worker(server.go)对齐: Echo/GetStatus/Start/Kill/ReadLog/GetConfig/SetConfig/GetHistory。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "savt-client/savt-client-api/savt"

	"savt-client/savt-client-cli/common"
)

// ---- 任务/日志状态(GUI 可见的全部状态) ----

type daemonState struct {
	mu      sync.Mutex
	job     *pb.Job
	logs    []string
	started time.Time
	// 手动测量完成信号
	doneCh chan struct{}
}

var dstate = &daemonState{}

func (d *daemonState) appendLog(msg string) {
	d.mu.Lock()
	d.logs = append(d.logs, msg)
	if len(d.logs) > 500 {
		d.logs = d.logs[len(d.logs)-500:]
	}
	d.mu.Unlock()
}

// logCapture: 把 runOnce 期间的 log 输出同步收进 GUI 日志流
type logCapture struct{ lines chan string }

func (l *logCapture) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if line != "" {
			l.lines <- line
		}
	}
	return len(p), nil
}

// ---- gRPC 服务实现 ----

type grpcServer struct {
	pb.UnimplementedSavtIpcServer
}

func (g *grpcServer) Echo(ctx context.Context, in *pb.EchoRequest) (*pb.EchoRequest, error) {
	log.Printf("[grpc] Echo")
	return in, nil
}

func (g *grpcServer) Start(ctx context.Context, _ *emptypb.Empty) (*pb.Job, error) {
	log.Printf("[grpc] Start")
	sched.runningMu.Lock()
	if sched.running {
		sched.runningMu.Unlock()
		return nil, fmt.Errorf("prober is already running")
	}
	sched.runningMu.Unlock()

	jobID := jobIDNow()
	dstate.mu.Lock()
	dstate.job = &pb.Job{
		JobId:     jobID,
		Status:    pb.Status_RUNNING,
		StartTime: timestamppb.Now(),
		Token:     "",
		JobType:   pb.Jobtype_MANUAL,
		Tasks:     []*pb.Task{},
	}
	dstate.logs = nil
	dstate.started = time.Now()
	done := make(chan struct{})
	dstate.doneCh = done
	dstate.mu.Unlock()

	// 轮次进度钩子: 测量过程实时点亮任务步骤与进度条(新协议轮次,GUI每5s拉GetStatus)
	roundDesc := map[string]string{
		"natmap":      "NAT mapping test",
		"natfilter":   "NAT filtering test",
		"inbound":     "Inbound SAV test",
		"outbound":    "Outbound SAV test",
		"tracefilter": "Trace filter test",
		"traceroute":  "Traceroute test",
	}
	roundProgress = func(family, name string, round, total int, doneRound bool) {
		dstate.mu.Lock()
		defer dstate.mu.Unlock()
		if dstate.job == nil || dstate.job.Status != pb.Status_RUNNING {
			return
		}
		if total <= 0 {
			total = 6
		}
		// 步骤序号 = 栈偏移(IPv6排在IPv4后) + 栈内轮序; 每栈最多6步
		idx := round
		if family == "IPv6" {
			idx += 6
		}
		for len(dstate.job.Tasks) < idx+1 {
			i := len(dstate.job.Tasks)
			dstate.job.Tasks = append(dstate.job.Tasks, &pb.Task{
				Step:       int32(i + 1),
				Status:     pb.Status_INITIAL,
				StatusDesc: "Measurement step " + fmt.Sprint(i+1),
			})
		}
		if t := dstate.job.Tasks[idx]; t != nil {
			if name != "" {
				t.StatusDesc = family + ": " + roundDesc[name]
			} else if t.StatusDesc == "" {
				t.StatusDesc = family + " measurement"
			}
			if doneRound {
				t.Status = pb.Status_SUCCEEDED
			} else {
				t.Status = pb.Status_RUNNING
			}
		}
		// 进度: 已完成步骤数 / 步骤总数(双栈最多12步), 上限95%(100%留给终态)
		steps := total
		if *serverFlag == "" {
			steps = total * 2 // 双栈
		}
		finished := 0
		for _, t := range dstate.job.Tasks {
			if t.Status == pb.Status_SUCCEEDED {
				finished++
			}
		}
		dstate.job.ProgressBar = int32(5 + 90*finished/steps)
	}

	go func() {
		// 捕获测量日志进 GUI 流
		lines := make(chan string, 256)
		lc := &logCapture{lines: lines}
		logOutput(lc)
		go func() {
			for l := range lines {
				dstate.appendLog(l)
			}
		}()

		ok, res4, res6 := sched.runMeasurement(false)
		roundProgress = nil // 测量结束摘钩子(防定时测量误写GUI状态)

		logOutputRestore()
		close(lines)

		dstate.mu.Lock()
		st := pb.Status_FAILED
		if ok {
			st = pb.Status_SUCCEEDED
		}
		dstate.job.Status = st
		dstate.job.DurationSeconds = int32(time.Since(dstate.started).Seconds())
		for _, t := range dstate.job.Tasks {
			if ok {
				t.Status = pb.Status_SUCCEEDED
			} else if t.Status == pb.Status_RUNNING {
				t.Status = pb.Status_FAILED
			}
		}
		if ok {
			dstate.job.ProgressBar = 100
		}
		if res4 != nil {
			dstate.job.Ipv4 = resultToMeasurementResult(res4)
		}
		if res6 != nil {
			dstate.job.Ipv6 = resultToMeasurementResult(res6)
		}
		dstate.mu.Unlock()
		close(done)
	}()

	dstate.mu.Lock()
	defer dstate.mu.Unlock()
	return dstate.job, nil
}

func (g *grpcServer) Kill(ctx context.Context, in *pb.JobIdRequest) (*pb.Job, error) {
	log.Printf("[grpc] Kill")
	// 新核心的单次测量不长短可控(轮次由服务端下发);此实现返回当前状态。
	dstate.mu.Lock()
	defer dstate.mu.Unlock()
	if dstate.job != nil {
		return dstate.job, nil
	}
	return &pb.Job{JobId: in.JobId, Status: pb.Status_CANCELLED}, nil
}

func (g *grpcServer) GetStatus(ctx context.Context, in *pb.JobIdRequest) (*pb.Job, error) {
	log.Printf("[grpc] GetStatus")
	dstate.mu.Lock()
	defer dstate.mu.Unlock()
	if dstate.job != nil {
		return dstate.job, nil
	}
	return &pb.Job{JobId: in.JobId, Status: pb.Status_INITIAL}, nil
}

func (g *grpcServer) ReadLog(in *pb.JobIdRequest, stream pb.SavtIpc_ReadLogServer) error {
	log.Printf("[grpc] ReadLog")
	// 从当前日志快照开始推流(老GUI语义: 读历史日志)
	dstate.mu.Lock()
	logs := append([]string{}, dstate.logs...)
	dstate.mu.Unlock()
	if len(logs) == 0 {
		logs = []string{"(暂无日志: 等待测量或测量无输出)"}
	}
	return stream.Send(&pb.LogEntry{Line: logs})
}

func (g *grpcServer) GetConfig(ctx context.Context, _ *emptypb.Empty) (*pb.Config, error) {
	log.Printf("[grpc] GetConfig")
	sched.mu.Lock()
	defer sched.mu.Unlock()
	c := sched.cfg
	return &pb.Config{
		EnableScheduler:         c.Enable,
		CompleteInterval:        int32(c.Complete),
		IncompleteRetryInterval: int32(c.Incomplete),
		CheckNetworkInterval:    int32(c.CheckNetwork),
		WaitAfterChangeInterval: int32(c.WaitAfterChange),
		RetryLimit:              int32(c.RetryLimit),
		IsPublic:                c.IsPublic,
	}, nil
}

func (g *grpcServer) SetConfig(ctx context.Context, in *pb.Config) (*emptypb.Empty, error) {
	log.Printf("[grpc] SetConfig")
	sched.mu.Lock()
	sched.cfg = daemonConfig{
		Enable:          in.EnableScheduler,
		Complete:        int(in.CompleteInterval),
		Incomplete:      int(in.IncompleteRetryInterval),
		CheckNetwork:    int(in.CheckNetworkInterval),
		WaitAfterChange: int(in.WaitAfterChangeInterval),
		RetryLimit:      int(in.RetryLimit),
		IsPublic:        in.IsPublic,
	}
	cfg := sched.cfg
	sched.mu.Unlock()
	saveDaemonConfig(cfg)
	return &emptypb.Empty{}, nil
}

func (g *grpcServer) GetHistory(ctx context.Context, _ *emptypb.Empty) (*pb.GetHistoryResponse, error) {
	log.Printf("[grpc] GetHistory")
	_, histDir := daemonDirs()
	jobs := []*pb.Job{}
	entries, err := readDirSortedDesc(histDir)
	if err == nil {
		for i, e := range entries {
			if i >= 10 {
				break
			}
			jobs = append(jobs, historyFileToJob(histDir, e))
		}
	}
	return &pb.GetHistoryResponse{Jobs: jobs}, nil
}

// ---- 启动 ----

func startGRPCServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("[daemon] gRPC监听失败: %v", err)
	}
	srv := grpc.NewServer()
	pb.RegisterSavtIpcServer(srv, &grpcServer{})
	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Printf("[daemon] gRPC服务退出: %v", err)
		}
	}()
	log.Printf("[daemon] gRPC服务面就绪 :%d (老GUI可遥控)", port)
}

// ---- 辅助 ----

// resultToMeasurementResult: 把服务端判定结果转成老GUI的MeasurementResult(单栈会话,v4/v6按clientIP判断)
func resultToMeasurementResult(res *common.Result) *pb.MeasurementResult {
	m := &pb.MeasurementResult{
		ClientAddress:    res.ClientIP,
		OutboundRoutable: statusToEnum(res.OutboundPublic),
		OutboundPrivate:  statusToEnum(res.OutboundPrivate),
		InboundInternal:  statusToEnum(res.InboundPublic),
		InboundPrivate:   statusToEnum(res.InboundPrivate),
	}
	return m
}

func statusToEnum(s string) pb.MeasurementStatus {
	switch s {
	case "received":
		return pb.MeasurementStatus_RECEIVED
	case "rewritten":
		return pb.MeasurementStatus_REWRITTEN
	case "blocked":
		return pb.MeasurementStatus_BLOCKED
	}
	return pb.MeasurementStatus_NOT_TESTED
}

func readDirSortedDesc(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	// 按名字倒序(文件名含时间戳,新在前)
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}
	return files, nil
}

func historyFileToJob(dir, name string) *pb.Job {
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return &pb.Job{JobId: name, Status: pb.Status_API_UNKNOWN}
	}
	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil {
		return &pb.Job{JobId: name, Status: pb.Status_API_UNKNOWN}
	}
	job := &pb.Job{JobId: strings.TrimSuffix(name, ".json")}
	if st, ok := rec["status"].(float64); ok {
		job.Status = pb.Status(int32(st))
	}
	if jt, ok := rec["scheduled"].(bool); ok && jt {
		job.JobType = pb.Jobtype_SCHEDULED
	} else {
		job.JobType = pb.Jobtype_MANUAL
	}
	toResult := func(key string) *common.Result {
		raw, ok := rec[key]
		if !ok {
			return nil
		}
		b, err := json.Marshal(raw)
		if err != nil {
			return nil
		}
		var r common.Result
		if json.Unmarshal(b, &r) != nil {
			return nil
		}
		return &r
	}
	if r := toResult("result"); r != nil {
		job.Ipv4 = resultToMeasurementResult(r)
	}
	if r := toResult("result6"); r != nil {
		job.Ipv6 = resultToMeasurementResult(r)
	}
	return job
}

// logOutput/logOutputRestore: 重定向标准log到捕获器(测完后恢复)
var logMu sync.Mutex

func logOutput(w io.Writer) {
	logMu.Lock()
	log.SetOutput(io.MultiWriter(os.Stderr, w))
	logMu.Unlock()
}
func logOutputRestore() {
	logMu.Lock()
	log.SetOutput(os.Stderr)
	logMu.Unlock()
}
