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

//go:build !windows

package update

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"savt-client/savt-client-upgrader/internal/paths"
	"savt-client/savt-client-upgrader/internal/protocol"
)

// StopProcesses terminates listed processes / related services before install.
func StopProcesses(names []string) error {
	if len(names) == 0 {
		names = paths.DefaultStopProcesses()
	}
	var errs []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || name == "savt-client-upgrader" {
			continue
		}
		if runtime.GOOS == "darwin" && strings.Contains(name, "worker") {
			_ = exec.Command("launchctl", "stop", paths.WorkerServiceName()).Run()
		}
		if runtime.GOOS == "linux" && strings.Contains(name, "worker") {
			_ = exec.Command("systemctl", "stop", paths.WorkerServiceName()).Run()
		}
		cmd := exec.Command("pkill", "-f", name)
		if out, err := cmd.CombinedOutput(); err != nil {
			// pkill returns 1 when no process matched — treat as success.
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				continue
			}
			errs = append(errs, fmt.Sprintf("%s: %v (%s)", name, err, strings.TrimSpace(string(out))))
		}
	}
	time.Sleep(500 * time.Millisecond)
	if len(errs) > 0 {
		return fmt.Errorf("stop processes: %s", strings.Join(errs, "; "))
	}
	return nil
}

// RunRestartSteps brings modules back according to the update plan.
func RunRestartSteps(steps []protocol.RestartStep) error {
	if len(steps) == 0 {
		steps = defaultRestartSteps()
	}
	var errs []string
	for _, step := range steps {
		if err := runRestartStep(step); err != nil {
			errs = append(errs, err.Error())
			log.Printf("restart step failed: %v", err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("restart: %s", strings.Join(errs, "; "))
	}
	return nil
}

func defaultRestartSteps() []protocol.RestartStep {
	switch runtime.GOOS {
	case "darwin":
		return []protocol.RestartStep{{Type: "launchctl", Name: paths.WorkerServiceName()}}
	case "linux":
		return []protocol.RestartStep{{Type: "systemctl", Name: paths.WorkerServiceName()}}
	default:
		return nil
	}
}

func runRestartStep(step protocol.RestartStep) error {
	switch strings.ToLower(step.Type) {
	case "launchctl":
		name := step.Name
		_ = exec.Command("launchctl", "stop", name).Run()
		if out, err := exec.Command("launchctl", "start", name).CombinedOutput(); err != nil {
			return fmt.Errorf("launchctl start %s: %w (%s)", name, err, strings.TrimSpace(string(out)))
		}
		return nil
	case "systemctl":
		name := step.Name
		if out, err := exec.Command("systemctl", "restart", name).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl restart %s: %w (%s)", name, err, strings.TrimSpace(string(out)))
		}
		return nil
	case "service":
		if runtime.GOOS == "darwin" {
			return runRestartStep(protocol.RestartStep{Type: "launchctl", Name: step.Name})
		}
		return runRestartStep(protocol.RestartStep{Type: "systemctl", Name: step.Name})
	case "exec":
		cmd := exec.Command(step.Command, step.Args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("exec %s: %w (%s)", step.Command, err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("unknown restart type %q", step.Type)
	}
}
