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

//go:build linux

package netinfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func isWifi() (bool, bool, error) {
	iface, err := defaultRouteIface()
	if err != nil {
		// Fail-open when we cannot determine the default route interface.
		return true, true, nil
	}
	if iface == "" || iface == "lo" {
		return true, true, nil
	}
	wirelessPath := filepath.Join("/sys/class/net", iface, "wireless")
	if st, err := os.Stat(wirelessPath); err == nil && st.IsDir() {
		return true, false, nil
	}
	// Interface exists but has no wireless sysfs node — treat as non-Wi-Fi.
	if _, err := os.Stat(filepath.Join("/sys/class/net", iface)); err != nil {
		return true, true, nil
	}
	return false, false, nil
}

// defaultRouteIface returns the interface name for the IPv4 default route (dest 00000000).
func defaultRouteIface() (string, error) {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return "", fmt.Errorf("empty /proc/net/route")
	}
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		iface := fields[0]
		dest := fields[1]
		if dest == "00000000" {
			return iface, nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no default route")
}
