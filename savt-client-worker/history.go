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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// enforceFileLimit checks the number of files in the specified directory, and if it exceeds the limit, it deletes the oldest files
func enforceFileLimit(dir string, maxFiles int) error {
	if maxFiles <= 0 {
		return fmt.Errorf("maxFiles should be greater than 0")
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	// Filter out directories and sort files by modification time
	var fileInfos []os.FileInfo
	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info() // Get FileInfo using Info() method
			if err != nil {
				log.Println("Error getting file info:", err)
				continue
			}
			fileInfos = append(fileInfos, info) // Now we can append FileInfo to the slice
		}
	}

	// If the number of files exceeds maxFiles, delete the oldest files
	if len(fileInfos) > maxFiles {
		// Sort by modification time in ascending order, oldest files first
		sort.Slice(fileInfos, func(i, j int) bool {
			return fileInfos[i].ModTime().Before(fileInfos[j].ModTime())
		})

		// Delete excess files
		filesToDelete := fileInfos[:len(fileInfos)-maxFiles]
		for _, file := range filesToDelete {
			filePath := filepath.Join(dir, file.Name()) // Ensure path is joined
			err = os.Remove(filePath)
			if err != nil {
				log.Printf("Failed to delete file %s: %v\n", filePath, err)
				continue // Log error and continue deleting other files
			}
		}
	}

	return nil
}

func ensureDir(dirPath string) error {
	// Use MkdirAll to create directory if it doesn't exist
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	return nil
}
