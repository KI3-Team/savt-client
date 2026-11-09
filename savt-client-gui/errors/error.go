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

package errors

import (
	"fmt"
)

// ErrorCode represents a type of error with its explanation
type ErrorCode struct {
	Explanation string
}

// Predefined error types
var (
	ConnectionError    = ErrorCode{Explanation: "Unable to connect to remote service"}
	TimeoutError       = ErrorCode{Explanation: "Request timeout"}
	ServiceUnavailable = ErrorCode{Explanation: "Service unavailable"}
	InvalidRequest     = ErrorCode{Explanation: "Invalid request"}
	NotFoundError      = ErrorCode{Explanation: "Resource not found"}
)

// ErrorMessage represents a detailed error with code and additional information
type ErrorMessage struct {
	Code    ErrorCode
	Details string
}

// NewErrorMessage creates a new error message with the given code and details
func NewErrorMessage(code ErrorCode, details string) *ErrorMessage {
	return &ErrorMessage{
		Code:    code,
		Details: details,
	}
}

// Error implements the error interface
func (e *ErrorMessage) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code.Explanation, e.Code.Explanation, e.Details)
}
