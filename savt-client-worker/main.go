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

/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/natefinch/lumberjack"

	pb "savt-client/savt-client-api/savt"
	utils "savt-client/savt-client-api/utils"

	"google.golang.org/grpc"
)

var (
	sm, _      = utils.GetServiceManagerIns()
	env        = sm.GetEnv()
	rport      = utils.GetRemotePort(env)
	remotePort = *flag.String("remotePort", rport, "The SAVT server port")

	port = flag.Int("port", utils.LocalPort, "The server port")

	initConfig = map[string]interface{}{
		"Complete":        608700,
		"Incomplete":      600,
		"CheckNetwork":    1800,
		"WaitAfterChange": 60,
		"RetryLimit":      3,
		"IsPublic":        true,
		"Enable":          true,
	}

	measurementResultMap = map[string]pb.MeasurementStatus{
		"blocked":    pb.MeasurementStatus_BLOCKED,
		"received":   pb.MeasurementStatus_RECEIVED,
		"rewritten":  pb.MeasurementStatus_REWRITTEN,
		"not tested": pb.MeasurementStatus_NOT_TESTED,
	}

	statusMap = map[int]pb.Status{
		0: pb.Status_INITIAL,
		1: pb.Status_RUNNING,
		2: pb.Status_SUCCEEDED,
		3: pb.Status_SEMI_SUCCEEDED,
		4: pb.Status_FAILED,
		5: pb.Status_CANCELLED,
	}

	taskDescMap = map[int]string{
		1: "IPv4 outbound test",
		2: "IPv4 inbound test",
		3: "IPv4 tracefilter test",
		4: "IPv4 traceroute test",
		5: "IPv6 outbound test",
		6: "IPv6 inbound test",
		7: "IPv6 tracefilter test",
		8: "IPv6 traceroute test",
	}

	sysConfig, _   = utils.GetSysConfig()
	logFilePath    = sysConfig.LogFile
	historyDirPath = sysConfig.DataDir
	configFilePath = sysConfig.ConfigFile
)

const MAX_FILES = 10
const APP_NAME = "savt-client"

func main() {
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10,   // Unit: MB (maximum 10 MB per file)
		MaxBackups: 3,    // Keep at most 3 old files
		MaxAge:     30,   // Keep for maximum 30 days
		Compress:   true, // Whether to compress old log files (gz)
	}
	// Combined output: output to both file (rotation) and console
	multiWriter := io.MultiWriter(os.Stdout, lumberjackLogger)
	log.SetOutput(multiWriter)

	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterSavtIpcServer(s, NewServer())
	log.Printf("server listening at %v\n", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
