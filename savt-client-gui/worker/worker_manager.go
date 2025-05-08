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

package worker

import (
	"fmt"
	"os/exec"
	"runtime"
	"savt-client/savt-client-api/savt"
	"sync"
)

type WorkerManager struct {
	serviceName string
	proxy       GRPCProxy
	mu          sync.Mutex
}

func NewWorkerManager(proxy GRPCProxy) *WorkerManager {
	var serviceName string
	switch runtime.GOOS {
	case "linux":
		serviceName = "savt-client.savt-client-worker.service"
	case "darwin":
		serviceName = "savt-client.savt-client-worker"
	case "windows":
		serviceName = "savt-client.savt-client-worker"
	default:
		serviceName = "savt-client.savt-client-worker"
	}
	return &WorkerManager{
		serviceName: serviceName,
		proxy:       proxy,
	}
}

func (wm *WorkerManager) IsRunning() bool {
	echoReq := &savt.EchoRequest{Message: "ping"}
	resp, err := wm.proxy.Echo(echoReq)
	if err != nil {
		return false
	}
	return resp.GetMessage() == "ping"
}

func (wm *WorkerManager) Start() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wm.IsRunning() {
		return fmt.Errorf("worker service is already running")
	}

	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("sudo", "systemctl", "start", wm.serviceName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start service: %v", err)
		}
	case "darwin":
		appleScript := fmt.Sprintf(`
            do shell script "launchctl start %s" with administrator privileges
        `, wm.serviceName)
		cmd := exec.Command("osascript", "-e", appleScript)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start service: %v", err)
		}

	case "windows":
		cmd := exec.Command("powershell", "-Command", "Start-Service", wm.serviceName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start service: %v", err)
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return nil
}

func (wm *WorkerManager) Stop() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !wm.IsRunning() {
		return fmt.Errorf("worker service is not running")
	}

	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("sudo", "systemctl", "stop", wm.serviceName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}
	case "darwin":
		appleScript := fmt.Sprintf(`
            do shell script "launchctl stop %s" with administrator privileges
        `, wm.serviceName)

		cmd := exec.Command("osascript", "-e", appleScript)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}
	case "windows":
		cmd := exec.Command("powershell", "-Command", "Stop-Service", wm.serviceName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to stop service: %v", err)
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return nil
}
