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

package svcjson

import (
	"encoding/json"
	"os"
	"path/filepath"

	"savt-client/savt-client-api/utils"
	"savt-client/savt-client-upgrader/internal/paths"
)

// File returns the service.json path under the install directory.
func File() string {
	return filepath.Join(paths.InstallRoot(), "service.json")
}

type config struct {
	Version string `json:"version"`
	Env     string `json:"env"`
}

// ReadVersion returns the version from install-dir service.json.
func ReadVersion() string {
	data, err := os.ReadFile(File())
	if err != nil {
		return ""
	}
	var c config
	if err := json.Unmarshal(data, &c); err != nil {
		return ""
	}
	return c.Version
}

// WriteVersion updates version in install-dir service.json (and ConfigDir copy).
// Existing env is preserved when the file already exists.
func WriteVersion(version string) error {
	if version == "" {
		return nil
	}
	if err := writeAt(File(), version); err != nil {
		return err
	}
	if sys, err := utils.GetSysConfig(); err == nil {
		_ = writeAt(filepath.Join(sys.ConfigDir, "service.json"), version)
	}
	return nil
}

func writeAt(path, version string) error {
	cfg := map[string]any{"version": version, "env": "production"}
	if data, err := os.ReadFile(path); err == nil {
		var existing map[string]any
		if json.Unmarshal(data, &existing) == nil {
			for k, v := range existing {
				cfg[k] = v
			}
			cfg["version"] = version
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}
