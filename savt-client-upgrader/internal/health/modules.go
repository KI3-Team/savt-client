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

package health

import (
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"savt-client/savt-client-upgrader/internal/paths"
)

// Modules returns a map of component name -> health status string.
func Modules(wsConnected bool) map[string]string {
	status := map[string]string{
		"upgrader": "running",
		"ws":       "disconnected",
	}
	if wsConnected {
		status["ws"] = "connected"
	}
	status["worker"] = processStatus("savt-client-worker")
	status["gui"] = processStatus("savt-client-gui")
	status["cli"] = processStatus("savt-client-cli")
	status["worker_grpc"] = grpcStatus()
	return status
}

func processStatus(name string) string {
	if isProcessRunning(name) {
		return "running"
	}
	return "stopped"
}

func isProcessRunning(name string) bool {
	switch runtime.GOOS {
	case "windows":
		// Match savt-client-gui.exe and legacy versioned names
		// (e.g. savt-client-gui-2.2.3-windows-amd64.exe) during upgrades.
		out, err := exec.Command("tasklist").Output()
		if err != nil {
			return false
		}
		lower := strings.ToLower(string(out))
		needle := strings.ToLower(name)
		if strings.Contains(lower, needle+".exe") {
			return true
		}
		// Also match versioned binaries discovered next to the upgrader.
		matches, _ := filepath.Glob(filepath.Join(paths.InstallRoot(), name+"*.exe"))
		for _, m := range matches {
			if strings.Contains(lower, strings.ToLower(filepath.Base(m))) {
				return true
			}
		}
		// Prefix match for versioned names not present under InstallRoot.
		return strings.Contains(lower, needle+"-")
	default:
		cmd := exec.Command("pgrep", "-f", name)
		if err := cmd.Run(); err == nil {
			return true
		}
		// Fallback: also match versioned binary names under install root.
		cmd = exec.Command("pgrep", "-f", paths.InstallRoot()+"/"+name)
		return cmd.Run() == nil
	}
}

func grpcStatus() string {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:40001", 800*time.Millisecond)
	if err != nil {
		return "unreachable"
	}
	_ = conn.Close()
	return "ok"
}
