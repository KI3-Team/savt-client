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

//go:build windows

package update

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
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
	_ = exec.Command("net", "stop", paths.WorkerServiceName()).Run()

	var errs []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || strings.EqualFold(name, "savt-client-upgrader") {
			continue
		}
		image := name
		if !strings.HasSuffix(strings.ToLower(image), ".exe") {
			image += ".exe"
		}
		// Prefer killing by image name; also try versioned names under install root.
		if out, err := exec.Command("taskkill", "/F", "/IM", image).CombinedOutput(); err != nil {
			msg := strings.ToLower(string(out))
			if !strings.Contains(msg, "not found") && !strings.Contains(msg, "not running") {
				errs = append(errs, fmt.Sprintf("%s: %v (%s)", image, err, strings.TrimSpace(string(out))))
			}
		}
		matches, _ := filepath.Glob(filepath.Join(paths.InstallRoot(), name+"*.exe"))
		for _, m := range matches {
			_ = exec.Command("taskkill", "/F", "/IM", filepath.Base(m)).Run()
		}
	}
	time.Sleep(500 * time.Millisecond)
	if len(errs) > 0 {
		log.Printf("stop processes warnings: %s", strings.Join(errs, "; "))
	}
	return nil
}

// RunRestartSteps brings modules back according to the update plan.
func RunRestartSteps(steps []protocol.RestartStep) error {
	if len(steps) == 0 {
		steps = []protocol.RestartStep{{Type: "nssm", Name: paths.WorkerServiceName()}}
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

func runRestartStep(step protocol.RestartStep) error {
	switch strings.ToLower(step.Type) {
	case "nssm", "service":
		name := step.Name
		_ = exec.Command("net", "stop", name).Run()
		if out, err := exec.Command("net", "start", name).CombinedOutput(); err != nil {
			return fmt.Errorf("net start %s: %w (%s)", name, err, strings.TrimSpace(string(out)))
		}
		return nil
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
