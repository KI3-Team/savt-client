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

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"

	"github.com/natefinch/lumberjack"

	"savt-client/savt-client-api/utils"
	"savt-client/savt-client-upgrader/internal/config"
	"savt-client/savt-client-upgrader/internal/device"
	"savt-client/savt-client-upgrader/internal/svcjson"
	"savt-client/savt-client-upgrader/internal/update"
	"savt-client/savt-client-upgrader/internal/ws"
)

func main() {
	cfg := config.Load()

	sysConfig, err := utils.GetSysConfig()
	if err != nil {
		log.Fatalf("get sys config: %v", err)
	}
	logPath := filepath.Join(sysConfig.LogDir, "savt-client-upgrader.log")
	setupLogging(logPath)

	sm, err := utils.GetServiceManagerIns()
	if err != nil {
		log.Fatalf("service manager: %v", err)
	}
	appVersion := svcjson.ReadVersion()
	if appVersion == "" {
		appVersion = sm.GetVersion()
	}

	info, err := device.Collect()
	if err != nil {
		log.Fatalf("collect device info: %v", err)
	}

	downloadDir := cfg.DownloadDir
	if downloadDir == "" {
		downloadDir = filepath.Join(sysConfig.ConfigDir, "upgrader", "downloads")
	}
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		log.Fatalf("create download dir: %v", err)
	}

	wsURL := fmt.Sprintf("%s/%s", trimRightSlash(cfg.WSBaseURL), info.ID)
	client := &ws.Client{
		URL:               wsURL,
		Device:            info,
		AppVersion:        appVersion,
		UpgraderVersion:   appVersion,
		HeartbeatInterval: cfg.HeartbeatInterval,
		ReconnectDelay:    cfg.ReconnectDelay,
	}
	mgr := &update.Manager{
		Downloader: update.NewDownloader(downloadDir, cfg.PackageBaseURL, cfg.DownloadTimeout, cfg.AllowHTTP),
		Applier:    &update.Applier{},
		MaxRetries: cfg.MaxRetries,
		SendStatus: client.SendStatus,
	}
	client.Manager = mgr

	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			log.Printf("crash: %v\n%s", r, stack)
			_ = client.SendCrash("panic", fmt.Sprint(r), stack)
		}
	}()

	stop := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down", sig)
		close(stop)
	}()

	log.Printf("savt-client-upgrader starting id=%s version=%s url=%s", info.ID, appVersion, wsURL)
	client.Run(stop)
	log.Printf("savt-client-upgrader stopped")
}

func setupLogging(logPath string) {
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
	multi := io.MultiWriter(os.Stdout, lumberjackLogger)
	log.SetOutput(multi)
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[savt-client-upgrader] ")
}

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
