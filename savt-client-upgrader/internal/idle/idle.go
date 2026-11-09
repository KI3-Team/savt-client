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

// Package idle reports user idle time for update gating.
package idle

// IdleSeconds returns how long the interactive session has been idle, in seconds.
// Implementations fail-open with a large value when idle time cannot be determined.
func IdleSeconds() (int64, error) {
	return idleSeconds()
}

// largeIdle is used when there is no interactive session or idle time is unknown.
const largeIdle int64 = 24 * 3600
