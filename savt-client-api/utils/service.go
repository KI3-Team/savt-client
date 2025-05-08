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

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Env string

const (
	Test       Env = "test"
	Production Env = "production"
)

type ServiceManager struct {
	version string
	env     Env
}

var (
	instance     *ServiceManager
	version_once sync.Once
)

type ServiceConfig struct {
	Version string `json:"version"`
	Env     Env    `json:"env"`
}

var defaultConfigPath = "service.json"

// GetServiceManagerIns returns the singleton instance of ServiceManager
func GetServiceManagerIns() (*ServiceManager, error) {
	var err error
	version_once.Do(func() {
		instance = &ServiceManager{}
		err = instance.loadVersionAndEnv()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ServiceManager: %w", err)
	}
	return instance, nil
}

func (sm *ServiceManager) loadVersionAndEnv() error {
	if sm.version != "" && sm.env != "" {
		return nil
	}

	if err := sm.readServiceConfigFile(filepath.Join("..", defaultConfigPath)); err == nil {
		return nil
	}

	if err := sm.readServiceConfigFile(defaultConfigPath); err == nil {
		return nil
	}

	sysConfig, err := GetSysConfig()
	if err != nil {
		return fmt.Errorf("failed to get system configuration: %w", err)
	}
	configServiceFile := filepath.Join(sysConfig.ConfigDir, defaultConfigPath)
	if err := sm.readServiceConfigFile(configServiceFile); err == nil {
		return nil
	}

	return fmt.Errorf("failed to read service.json: not found in root, current directory or '%s'", configServiceFile)
}

func (sm *ServiceManager) readServiceConfigFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read '%s': %w", filePath, err)
	}

	var config ServiceConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse '%s': %w", filePath, err)
	}

	sm.version = config.Version
	sm.env = config.Env
	return nil
}

// GetVersion returns the current version number
func (sm *ServiceManager) GetVersion() string {
	return sm.version
}

// GetEnv returns the current environment type
func (sm *ServiceManager) GetEnv() Env {
	return sm.env
}

// CheckForUpdate checks if there is a new version available
func (sm *ServiceManager) CheckForUpdate() (bool, string, error) {
	const versionURL = "https://ki3.org.cn/public/sav/version.txt"

	resp, err := http.Get(versionURL)
	if err != nil {
		return false, "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", fmt.Errorf("failed to read response body: %w", err)
	}
	latestVersion := strings.TrimSpace(string(respBody))

	isNewer := latestVersion != sm.GetVersion()
	return isNewer, latestVersion, nil
}
