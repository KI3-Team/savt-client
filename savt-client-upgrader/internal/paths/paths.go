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

package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// DeviceDataRoot is a durable host-local directory kept across uninstall/reinstall.
// Uninstall scripts must not remove DeviceIDFile under this root.
func DeviceDataRoot() string {
	switch runtime.GOOS {
	case "darwin":
		return "/var/db/savt-client"
	case "windows":
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "savt-client")
	default:
		return "/var/lib/savt-client"
	}
}

// DeviceIDFile is the persisted upgrader device id (survives app uninstall).
func DeviceIDFile() string {
	return filepath.Join(DeviceDataRoot(), "device.id")
}

// LegacyDeviceIDFile is the old install-root location used before durable storage.
func LegacyDeviceIDFile() string {
	return filepath.Join(InstallRoot(), "savt-client.device.id")
}

// InstallRoot returns the directory that holds installed savt-client binaries.
// It is derived from the running upgrader executable so Windows installs under
// Program Files vs Program Files (x86) (and non-default paths) resolve correctly.
func InstallRoot() string {
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if dir := filepath.Dir(exe); dir != "" && dir != "." {
			return dir
		}
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Applications/savt-client.app/Contents/MacOS"
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		if programFiles == "" {
			programFiles = `C:\Program Files`
		}
		return filepath.Join(programFiles, "savt-client")
	default:
		return "/usr/local/bin"
	}
}

// ResolvePath expands a relative path against InstallRoot; absolute paths are kept.
func ResolvePath(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(InstallRoot(), filepath.Clean(p))
}

// DefaultStopProcesses are the processes stopped before applying an update.
func DefaultStopProcesses() []string {
	names := []string{
		"savt-client-gui",
		"savt-client-cli",
		"savt-client-worker",
	}
	if runtime.GOOS == "windows" {
		return []string{
			"savt-client-gui",
			"savt-client-cli",
			"savt-client-worker",
		}
	}
	return names
}

// WorkerServiceName is the OS service / launchd label used for the worker daemon.
func WorkerServiceName() string {
	switch runtime.GOOS {
	case "darwin":
		return "savt-client.savt-client-worker"
	case "windows":
		return "savt-client.savt-client-worker"
	default:
		return "savt-client-worker"
	}
}

// UpgraderServiceName is the OS service / launchd label for this upgrader.
func UpgraderServiceName() string {
	switch runtime.GOOS {
	case "darwin":
		return "savt-client.savt-client-upgrader"
	case "windows":
		return "savt-client.savt-client-upgrader"
	default:
		return "savt-client-upgrader"
	}
}
