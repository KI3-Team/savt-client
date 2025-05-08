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
)

// saveJobId saves the job ID to a file
func saveJobId(jobId string) error {
	userConfig, err := utils.GetUserConfig()
	if err != nil {
		log.Fatalf("Failed to get system configuration: %v", err)
	}
	jobIdFile := filepath.Join(userConfig.ConfigDir, "job_id.json")
	dir := filepath.Dir(jobIdFile) // Get parent directory path
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create parent directory: %v", err)
	}
	file, err := os.Create(jobIdFile)
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

// loadJobId loads the job ID from a file
func loadJobId() (string, error) {
	userConfig, err := utils.GetUserConfig()
	if err != nil {
		log.Fatalf("Failed to get system configuration: %v", err)
	}
	jobIdFile := filepath.Join(userConfig.ConfigDir, "job_id.json")
	file, err := os.Open(jobIdFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty string if file doesn't exist
			return "", nil
		}
		return "", fmt.Errorf("failed to open jobId file: %v", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing file: %v", closeErr)
		}
	}()

	// Read jobId
	var jobId string
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&jobId)
	if err != nil {
		return "", fmt.Errorf("failed to read jobId from file: %v", err)
	}
	return jobId, nil
}
