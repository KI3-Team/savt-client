// Copyright 2025 KI3
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// savt-client-cli: worker(savt-client-worker)的 gRPC 命令行客户端。
// 用法:
//   savt-client-cli run       触发一次测量并等待完成(轮询状态+日志)
//   savt-client-cli start     触发测量(立即返回)
//   savt-client-cli status    查询当前/最近任务状态
//   savt-client-cli log       读取最近任务日志
//   savt-client-cli history   读取历史任务(最近10条)
//   savt-client-cli config    读取调度配置
// 选项:
//   -addr localhost:40001    worker 的 gRPC 地址
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "savt-client/savt-client-api/savt"
	utils "savt-client/savt-client-api/utils"
)

var addr = flag.String("addr", "localhost:"+strconv.Itoa(utils.LocalPort), "worker gRPC address")

func client() (pb.SavtIpcClient, *grpc.ClientConn) {
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect %s: %v", *addr, err)
	}
	return pb.NewSavtIpcClient(conn), conn
}

func statusString(s pb.Status) string { return s.String() }

func printJob(j *pb.Job) {
	fmt.Printf("job %s  status=%s  progress=%d%%\n", j.JobId, statusString(j.Status), j.ProgressBar)
	for _, t := range j.Tasks {
		fmt.Printf("  step %d: [%s] %s\n", t.Step, statusString(t.Status), t.StatusDesc)
	}
	if j.Ipv4 != nil {
		fmt.Printf("  IPv4: client=%s outboundPublic=%s inbound=%s/%s\n", j.Ipv4.ClientAddress,
			j.Ipv4.OutboundRoutable, j.Ipv4.InboundInternal, j.Ipv4.InboundPrivate)
	}
	if j.Ipv6 != nil {
		fmt.Printf("  IPv6: client=%s outboundPublic=%s inbound=%s/%s\n", j.Ipv6.ClientAddress,
			j.Ipv6.OutboundRoutable, j.Ipv6.InboundInternal, j.Ipv6.InboundPrivate)
	}
}

func cmdRun() {
	c, conn := client()
	defer conn.Close()
	ctx := context.Background()

	job, err := c.Start(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("start: %v", err)
	}
	fmt.Printf("measurement started: %s\n", job.JobId)

	deadline := time.Now().Add(6 * time.Minute)
	var lastLog int
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)
		j, err := c.GetStatus(ctx, &pb.JobIdRequest{JobId: job.JobId})
		if err != nil {
			log.Fatalf("get status: %v", err)
		}
		// 增量打日志尾部
		stream, err := c.ReadLog(ctx, &pb.JobIdRequest{JobId: job.JobId})
		if err == nil {
			var lines []string
			for {
				e, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}
				lines = append(lines, e.Line...)
			}
			for ; lastLog < len(lines); lastLog++ {
				fmt.Println(lines[lastLog])
			}
		}
		if j.Status != pb.Status_RUNNING && j.Status != pb.Status_INITIAL {
			printJob(j)
			os.Exit(int(4 - j.Status)) // SUCCEEDED(2)->2, FAILED(4)->0
		}
	}
	fmt.Println("timeout waiting for measurement to finish")
	os.Exit(1)
}

func cmdStatus() {
	c, conn := client()
	defer conn.Close()
	j, err := c.GetStatus(context.Background(), &pb.JobIdRequest{})
	if err != nil {
		log.Fatalf("get status: %v", err)
	}
	printJob(j)
}

func cmdLog() {
	c, conn := client()
	defer conn.Close()
	stream, err := c.ReadLog(context.Background(), &pb.JobIdRequest{})
	if err != nil {
		log.Fatalf("read log: %v", err)
	}
	for {
		e, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("read log: %v", err)
		}
		fmt.Println(strings.Join(e.Line, "\n"))
	}
}

func cmdHistory() {
	c, conn := client()
	defer conn.Close()
	resp, err := c.GetHistory(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("get history: %v", err)
	}
	for _, j := range resp.Jobs {
		printJob(j)
		fmt.Println()
	}
}

func cmdConfig() {
	c, conn := client()
	defer conn.Close()
	cfg, err := c.GetConfig(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("get config: %v", err)
	}
	fmt.Printf("scheduler enable=%v complete=%ds incomplete=%ds checkNetwork=%ds waitAfterChange=%ds retry=%d isPublic=%v\n",
		cfg.EnableScheduler, cfg.CompleteInterval, cfg.IncompleteRetryInterval,
		cfg.CheckNetworkInterval, cfg.WaitAfterChangeInterval, cfg.RetryLimit, cfg.IsPublic)
}

func main() {
	flag.Parse()
	cmd := "run"
	if flag.NArg() > 0 {
		cmd = flag.Arg(0)
	}
	switch cmd {
	case "run":
		cmdRun()
	case "start":
		c, conn := client()
		defer conn.Close()
		job, err := c.Start(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.Fatalf("start: %v", err)
		}
		fmt.Printf("measurement started: %s\n", job.JobId)
	case "status":
		cmdStatus()
	case "log":
		cmdLog()
	case "history":
		cmdHistory()
	case "config":
		cmdConfig()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (run|start|status|log|history|config)\n", cmd)
		os.Exit(2)
	}
}
