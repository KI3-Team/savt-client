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

const (
	// OS constants
	OSLinux   = "linux"
	OSDarwin  = "darwin"
	OSWindows = "windows"

	// Service name constants
	ServiceNameLinux   = "savt-client.savt-client-worker.service"
	ServiceNameDefault = "savt-client.savt-client-worker"

	// Command constants
	CmdStartLinux   = "start"
	CmdStopLinux    = "stop"
	CmdStartWindows = "Start-Service"
	CmdStopWindows  = "Stop-Service"
)

type WorkerManager struct {
	serviceName string
	proxy       GRPCProxy
	mu          sync.Mutex
}

func NewWorkerManager(proxy GRPCProxy) *WorkerManager {
	var serviceName string
	switch runtime.GOOS {
	case OSLinux:
		serviceName = ServiceNameLinux
	case OSDarwin, OSWindows:
		serviceName = ServiceNameDefault
	default:
		serviceName = ServiceNameDefault
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

func (wm *WorkerManager) executeServiceCommand(action string) error {
	switch runtime.GOOS {
	case OSLinux:
		cmd := exec.Command("sudo", "systemctl", action, wm.serviceName) // #nosec G204
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to %s service: %v", action, err)
		}
	case OSDarwin:
		appleScript := fmt.Sprintf(`
            do shell script "launchctl %s %s" with administrator privileges
        `, action, wm.serviceName)
		cmd := exec.Command("osascript", "-e", appleScript) // #nosec G204
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to %s service: %v", action, err)
		}
	case OSWindows:
		cmdAction := CmdStartWindows
		if action == CmdStopLinux {
			cmdAction = CmdStopWindows
		}
		cmd := exec.Command("powershell", "-Command", cmdAction, wm.serviceName) // #nosec G204
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to %s service: %v", action, err)
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return nil
}

func (wm *WorkerManager) Start() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wm.IsRunning() {
		return fmt.Errorf("worker service is already running")
	}

	return wm.executeServiceCommand(CmdStartLinux)
}

func (wm *WorkerManager) Stop() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !wm.IsRunning() {
		return fmt.Errorf("worker service is not running")
	}

	return wm.executeServiceCommand(CmdStopLinux)
}
