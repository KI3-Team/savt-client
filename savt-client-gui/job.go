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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"savt-client/savt-client-api/utils"
	"strings"
)

const (
	// File permissions
	DirPerm  = 0750 // Directory permissions
	FilePerm = 0640 // File permissions
)

// sanitizeFilePath ensures the file path is safe to use
func sanitizeFilePath(path string) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %v", err)
	}

	// Get user config directory
	userConfig, err := utils.GetUserConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get user config: %v", err)
	}
	configDir, err := filepath.Abs(userConfig.ConfigDir)
	if err != nil {
		return "", fmt.Errorf("invalid config directory: %v", err)
	}

	// Check if the file path is within the config directory
	if !strings.HasPrefix(absPath, configDir) {
		return "", fmt.Errorf("file path must be within config directory")
	}

	// Clean the path to remove any .. or . components
	cleanPath := filepath.Clean(absPath)
	if !strings.HasPrefix(cleanPath, configDir) {
		return "", fmt.Errorf("invalid file path after cleaning")
	}

	return cleanPath, nil
}

// saveJobId saves the job ID to a file
func saveJobId(jobId string) error {
	userConfig, err := utils.GetUserConfig()
	if err != nil {
		log.Fatalf("Failed to get system configuration: %v", err)
	}

	// Sanitize the job ID
	if !isValidJobID(jobId) {
		return fmt.Errorf("invalid job ID format")
	}

	jobIdFile := filepath.Join(userConfig.ConfigDir, "job_id.json")

	// Sanitize and validate file path
	sanitizedPath, err := sanitizeFilePath(jobIdFile)
	if err != nil {
		return fmt.Errorf("invalid file path: %v", err)
	}

	dir := filepath.Dir(sanitizedPath)

	// Create directory with restricted permissions
	err = os.MkdirAll(dir, DirPerm)
	if err != nil {
		return fmt.Errorf("failed to create parent directory: %v", err)
	}

	// Create file with restricted permissions
	file, err := os.OpenFile(sanitizedPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, FilePerm) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to create jobId file: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing file: %v", closeErr)
		}
	}()

	// Write jobId to file
	encoder := json.NewEncoder(file)
	err = encoder.Encode(jobId)
	if err != nil {
		return fmt.Errorf("failed to write jobId to file: %v", err)
	}
	return nil
}

// isValidJobID checks if the job ID is valid
func isValidJobID(jobId string) bool {
	// Add your job ID validation logic here
	// For example, check length, format, etc.
	return len(jobId) > 0 && len(jobId) <= 100
}

// loadJobId loads the job ID from a file
func loadJobId() (string, error) {
	userConfig, err := utils.GetUserConfig()
	if err != nil {
		log.Fatalf("Failed to get system configuration: %v", err)
	}

	jobIdFile := filepath.Join(userConfig.ConfigDir, "job_id.json")

	// Sanitize and validate file path
	sanitizedPath, err := sanitizeFilePath(jobIdFile)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %v", err)
	}

	// Open file with read-only permissions
	file, err := os.OpenFile(sanitizedPath, os.O_RDONLY, FilePerm) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to open jobId file: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing file: %v", closeErr)
		}
	}()

	var jobId string
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&jobId)
	if err != nil {
		return "", fmt.Errorf("failed to read jobId from file: %v", err)
	}

	// Validate the loaded job ID
	if !isValidJobID(jobId) {
		return "", fmt.Errorf("invalid job ID in file")
	}

	return jobId, nil
}
