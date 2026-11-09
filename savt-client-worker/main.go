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

// savt-client-worker: 测量引擎服务(新协议六轮, 含 NAT 映射/过滤)。
// gRPC 服务面 :40001(老 GUI 零改动遥控) + 定时调度器。
// 测量逻辑在共享包 savt-client-engine; 本程序是引擎的常驻宿主。
package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"

	utils "savt-client/savt-client-api/utils"
	engine "savt-client/savt-client-engine"
)

var dataDir = flag.String("data-dir", "", "override client config, history, and logs directory")

var port = flag.Int("port", utils.LocalPort, "The IPC server port")

func main() {
	flag.StringVar(&engine.Server, "server", "", "IPv4 controller URL (single stack unless --server6 is also set)")
	flag.StringVar(&engine.Server6, "server6", "", "IPv6 controller URL")
	flag.Parse()

	sysConfig, err := utils.GetSysConfig()
	if err != nil {
		log.Fatalf("failed to get system config: %v", err)
	}
	logFilePath := sysConfig.LogFile
	if *dataDir != "" {
		logFilePath = filepath.Join(*dataDir, "Logs", "app.log")
	}
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    10, // MB
		MaxBackups: 3,
		MaxAge:     30, // days
		Compress:   true,
	}
	log.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))

	runDaemon()
}
