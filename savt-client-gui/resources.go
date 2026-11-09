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

// RegisterResources registers all resources to ensure they are not marked as unused
func RegisterResources() {
	_ = resourceIconAbout1024Png
	_ = resourceIconAbout256Png
	_ = resourceIconBlocked256Png
	_ = resourceIconCheck256Png
	_ = resourceIconEmpty256Png
	_ = resourceIconError256Png
	_ = resourceIconHistory1024Png
	_ = resourceIconHome1024Png
	_ = resourceIconHome256Png
	_ = resourceIconReceived256Png
	_ = resourceIconReport1024Png
	_ = resourceIconRewritten256Png
	_ = resourceIconRunning256Png
	_ = resourceIconSetting1024Png
	_ = resourceIconSetting256Png
	_ = resourceIconTime256Png
	_ = resourceImgPng
}
