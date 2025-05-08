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
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func TestFileLogger_Output(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "test-log-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	defaultWorkerLoggerConfig.LogFile = tempFile.Name()
	defaultWorkerLoggerConfig.Console = false

	logger := LoggerFactoryInstance.GetWorkerLogger()

	logger.Info("This is a test log message")
	logger.Warn("This is a warning message")

	tempFile.Close()

	content, err := ioutil.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp log file: %v", err)
	}

	if !bytes.Contains(content, []byte("This is a test log message")) {
		t.Error("Log file does not contain expected log message")
	}
	if !bytes.Contains(content, []byte("This is a warning message")) {
		t.Error("Log file does not contain expected warning message")
	}
}
