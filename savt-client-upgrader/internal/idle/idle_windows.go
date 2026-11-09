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

//go:build windows

package idle

import (
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procGetLastInput = user32.NewProc("GetLastInputInfo")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGetTickCount = kernel32.NewProc("GetTickCount")
)

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

func idleSeconds() (int64, error) {
	var info lastInputInfo
	info.cbSize = uint32(unsafe.Sizeof(info))
	r1, _, err := procGetLastInput.Call(uintptr(unsafe.Pointer(&info)))
	if r1 == 0 {
		return largeIdle, err
	}
	tick, _, _ := procGetTickCount.Call()
	idleMs := uint32(tick) - info.dwTime
	return int64(idleMs / 1000), nil
}
