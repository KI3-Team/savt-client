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

package config

import (
	"flag"
	"time"
)

// Config holds runtime flags for the upgrader daemon.
type Config struct {
	WSBaseURL         string
	PackageBaseURL    string
	HeartbeatInterval time.Duration
	ReconnectDelay    time.Duration
	DownloadTimeout   time.Duration
	MaxRetries        int
	DownloadDir       string
	AllowHTTP         bool
}

// Load parses command-line flags into Config.
func Load() Config {
	cfg := Config{}
	flag.StringVar(&cfg.WSBaseURL, "ws-base", "wss://ki3.org.cn/app-ws/savt", "WebSocket base URL; device id is appended as /{id}")
	flag.StringVar(&cfg.PackageBaseURL, "package-base", "https://ki3.org.cn/public/sav/upgrader", "base URL for version zip packages")
	flag.DurationVar(&cfg.HeartbeatInterval, "heartbeat", time.Minute, "heartbeat interval")
	flag.DurationVar(&cfg.ReconnectDelay, "reconnect", 5*time.Second, "delay before websocket reconnect")
	flag.DurationVar(&cfg.DownloadTimeout, "download-timeout", 10*time.Minute, "package download timeout")
	flag.IntVar(&cfg.MaxRetries, "max-retries", 3, "max update retries on failure")
	flag.StringVar(&cfg.DownloadDir, "download-dir", "", "staging directory for downloads (default: {ConfigDir}/upgrader/downloads)")
	flag.BoolVar(&cfg.AllowHTTP, "allow-http", false, "allow non-HTTPS download URLs (disabled by default)")
	flag.Parse()
	return cfg
}
