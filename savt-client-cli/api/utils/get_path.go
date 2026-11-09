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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"savt-client/savt-client-api/constants"
)

// SysConfig contains system-level paths for log files, log directories, data directories, config files and config directories
type SysConfig struct {
	LogFile    string // Path to log file
	LogDir     string // Log directory
	DataDir    string // Data directory
	ConfigFile string // Path to config file
	ConfigDir  string // Config directory
}

// UserConfig contains user-level paths for log files, log directories, data directories, config files and config directories
type UserConfig struct {
	LogFile    string // Path to user log file
	LogDir     string // User log directory
	DataDir    string // User data directory
	ConfigFile string // Path to user config file
	ConfigDir  string // User config directory
}

// Helper function to create directories
func createDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create dir %s: %w", dir, err)
		}
	}
	return nil
}

// GetSysConfig returns system-level configuration
func GetSysConfig() (SysConfig, error) {
	var config SysConfig

	switch runtime.GOOS {
	case "windows":
		// Windows
		// 1) Log file   => %LOCALAPPDATA%\MyApp\Logs\app.log
		// 2) Data dir   => %LOCALAPPDATA%\MyApp\History
		// 3) Config file => %APPDATA%\MyApp\config.json
		// 1) Log file   => %PROGRAMDATA%\MyApp\Logs\app.log
		// 2) Data dir   => %PROGRAMDATA%\MyApp\History
		// 3) Config file => %PROGRAMDATA%\MyApp\config.json
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			return config, errors.New("PROGRAMDATA not set on Windows")
		}
		config.LogDir = filepath.Join(programData, constants.APP_NAME, "Logs")
		config.LogFile = filepath.Join(config.LogDir, "app.log")
		config.DataDir = filepath.Join(programData, constants.APP_NAME, "History")
		config.ConfigDir = filepath.Join(programData, constants.APP_NAME)
		config.ConfigFile = filepath.Join(config.ConfigDir, "config.json")

		// Create related directories
		err := createDirs(config.LogDir, config.DataDir, config.ConfigDir)
		if err != nil {
			return config, err
		}

	case "darwin":
		// macOS
		// 1) Log file   => ~/Library/Logs/MyApp/app.log
		// 2) Data dir   => ~/Library/Application Support/MyApp/history
		// 3) Config file => ~/Library/Application Support/MyApp/config.json
		// 1) Log file   => /var/log/MyApp/app.log
		// 2) Data dir   => /Library/Application Support/MyApp/history
		// 3) Config file => /Library/Preferences/MyApp-config.json
		config.LogDir = filepath.Join("/var/log", constants.APP_NAME)
		config.LogFile = filepath.Join(config.LogDir, "app.log")
		config.DataDir = filepath.Join("/Library/Application Support", constants.APP_NAME, "history")
		config.ConfigDir = filepath.Join("/Library/Application Support", constants.APP_NAME)
		config.ConfigFile = filepath.Join(config.ConfigDir, "config.json")

		// Create related directories
		err := createDirs(config.LogDir, config.DataDir, config.ConfigDir)
		if err != nil {
			return config, err
		}

	case "linux":
		// Linux follows XDG specification:
		// 1) Log file   => $XDG_STATE_HOME/MyApp/app.log  (default: ~/.local/state/MyApp/app.log)
		// 2) Data dir   => $XDG_DATA_HOME/MyApp          (default: ~/.local/share/MyApp/history)
		// 3) Config file => $XDG_CONFIG_HOME/MyApp/config.json (default: ~/.config/MyApp/config.json)
		// Linux
		// 1) Log file   => /var/log/MyApp/app.log
		// 2) Data dir   => /var/lib/MyApp/history
		// 3) Config file => /etc/MyApp/config.json
		config.LogDir = filepath.Join("/var/log", constants.APP_NAME)
		config.LogFile = filepath.Join(config.LogDir, "app.log")
		config.DataDir = filepath.Join("/var/lib", constants.APP_NAME, "history")
		config.ConfigDir = filepath.Join("/var/lib", constants.APP_NAME)
		config.ConfigFile = filepath.Join(config.ConfigDir, "config.json")

		// Create related directories
		err := createDirs(config.LogDir, config.DataDir, config.ConfigDir)
		if err != nil {
			return config, err
		}

	default:
		config.LogDir = filepath.Join("/var/log", constants.APP_NAME)
		config.LogFile = filepath.Join(config.LogDir, "app.log")
		config.DataDir = filepath.Join("/var/lib", constants.APP_NAME, "history")
		config.ConfigDir = filepath.Join("/var/lib", constants.APP_NAME)
		config.ConfigFile = filepath.Join(config.ConfigDir, "config.json")

		// Create related directories
		err := createDirs(config.LogDir, config.DataDir, config.ConfigDir)
		if err != nil {
			return config, err
		}
	}

	return config, nil
}

// GetUserConfig returns user-level configuration
func GetUserConfig() (UserConfig, error) {
	var config UserConfig

	switch runtime.GOOS {
	case "windows":
		userProfile := os.Getenv("USERPROFILE")
		if userProfile == "" {
			return config, errors.New("USERPROFILE not set on Windows")
		}
		config.LogDir = filepath.Join(userProfile, constants.APP_NAME, "Logs")
		config.LogFile = filepath.Join(config.LogDir, "user_app.log")
		config.DataDir = filepath.Join(userProfile, constants.APP_NAME, "History")
		config.ConfigDir = filepath.Join(userProfile, constants.APP_NAME)
		config.ConfigFile = filepath.Join(config.ConfigDir, "config.json")

		// Create related directories
		err := createDirs(config.LogDir, config.DataDir, config.ConfigDir)
		if err != nil {
			return config, err
		}

	case "darwin":
		userHome := os.Getenv("HOME")
		if userHome == "" {
			return config, errors.New("HOME not set on macOS")
		}
		config.LogDir = filepath.Join(userHome, "Library", "Logs", constants.APP_NAME)
		config.LogFile = filepath.Join(config.LogDir, "user_app.log")
		config.DataDir = filepath.Join(userHome, "Library", "Application Support", constants.APP_NAME, "history")
		config.ConfigDir = filepath.Join(userHome, "Library", "Application Support", constants.APP_NAME)
		config.ConfigFile = filepath.Join(config.ConfigDir, "config.json")

		// Create related directories
		err := createDirs(config.LogDir, config.DataDir, config.ConfigDir)
		if err != nil {
			return config, err
		}

	case "linux":
		userHome := os.Getenv("HOME")
		if userHome == "" {
			return config, errors.New("HOME not set on Linux")
		}
		config.LogDir = filepath.Join(userHome, ".local", "share", constants.APP_NAME, "logs")
		config.LogFile = filepath.Join(config.LogDir, "user_app.log")
		config.DataDir = filepath.Join(userHome, ".local", "share", constants.APP_NAME, "history")
		config.ConfigDir = filepath.Join(userHome, ".local", "share", constants.APP_NAME)
		config.ConfigFile = filepath.Join(config.ConfigDir, "config.json")

		// Create related directories
		err := createDirs(config.LogDir, config.DataDir, config.ConfigDir)
		if err != nil {
			return config, err
		}

	default:
		config.LogDir = filepath.Join("/var", "log", constants.APP_NAME)
		config.LogFile = filepath.Join(config.LogDir, "user_app.log")
		config.DataDir = filepath.Join("/var", "lib", constants.APP_NAME, "history")
		config.ConfigDir = filepath.Join("/var", "lib", constants.APP_NAME)
		config.ConfigFile = filepath.Join(config.ConfigDir, "config.json")

		// Create related directories
		err := createDirs(config.LogDir, config.DataDir, config.ConfigDir)
		if err != nil {
			return config, err
		}
	}

	return config, nil
}
