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

package device

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"savt-client/savt-client-upgrader/internal/paths"
)

// Info holds host identity and telemetry used in heartbeats.
type Info struct {
	ID                   string
	MAC                  string
	IP                   string
	OS                   string
	OSVersion            string
	Arch                 string
	CPUCount             int
	NetworkBandwidthMbps float64
}

const deviceIDHexLen = 32

// Collect gathers local device information.
// Device ID is SHA-256(mac + local timestamp + rand) truncated to 32 hex chars,
// generated once and persisted outside the install tree so uninstall/reinstall keeps it.
func Collect() (Info, error) {
	mac, ip, err := primaryMACAndIP()
	if err != nil {
		return Info{}, err
	}
	id, err := loadOrCreateDeviceID(mac)
	if err != nil {
		return Info{}, err
	}
	bw := linkSpeedMbps()
	return Info{
		ID:                   id,
		MAC:                  mac,
		IP:                   ip,
		OS:                   runtime.GOOS,
		OSVersion:            osVersion(),
		Arch:                 runtime.GOARCH,
		CPUCount:             runtime.NumCPU(),
		NetworkBandwidthMbps: bw,
	}, nil
}

// loadOrCreateDeviceID returns a stable host id.
// Priority: durable file → legacy install-root file (migrate) → new SHA-256 id.
func loadOrCreateDeviceID(mac string) (string, error) {
	durable := paths.DeviceIDFile()
	if id := readDeviceIDFile(durable); id != "" {
		return id, nil
	}

	legacy := paths.LegacyDeviceIDFile()
	if id := readDeviceIDFile(legacy); id != "" {
		_ = writeDeviceIDFile(durable, id)
		return id, nil
	}

	id, err := generateDeviceID(mac)
	if err != nil {
		return "", err
	}
	if err := writeDeviceIDFile(durable, id); err != nil {
		return "", err
	}
	return id, nil
}

// generateDeviceID builds SHA-256(mac + unix_nano + rand) and returns the first 32 hex chars.
func generateDeviceID(mac string) (string, error) {
	var randBuf [16]byte
	if _, err := rand.Read(randBuf[:]); err != nil {
		return "", fmt.Errorf("device id rand: %w", err)
	}
	material := strings.ToLower(strings.TrimSpace(mac)) +
		strconv.FormatInt(time.Now().UnixNano(), 10) +
		hex.EncodeToString(randBuf[:])
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])[:deviceIDHexLen], nil
}

func readDeviceIDFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeDeviceIDFile(path, id string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create device id dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0644); err != nil {
		return fmt.Errorf("persist device id: %w", err)
	}
	return nil
}

func primaryMACAndIP() (string, string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", fmt.Errorf("list interfaces: %w", err)
	}
	var fallbackMAC, fallbackIP string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		mac := iface.HardwareAddr.String()
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip4 := ip.To4()
			if ip4 == nil {
				if fallbackMAC == "" {
					fallbackMAC = mac
					fallbackIP = ip.String()
				}
				continue
			}
			return mac, ip4.String(), nil
		}
		if fallbackMAC == "" {
			fallbackMAC = mac
		}
	}
	if fallbackMAC != "" {
		if fallbackIP == "" {
			fallbackIP = "0.0.0.0"
		}
		return fallbackMAC, fallbackIP, nil
	}
	return "", "", fmt.Errorf("no usable network interface found")
}

var windowsVerNumber = regexp.MustCompile(`\d+\.\d+(?:\.\d+){0,2}`)

func osVersion() string {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	case "linux":
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				}
			}
		}
	case "windows":
		// `ver` prints localized text (e.g. "版本") in the OEM/ANSI code page
		// (often GBK on Chinese Windows). Interpreting that as UTF-8 garbles it.
		// Keep only the ASCII version number.
		out, err := exec.Command("cmd", "/C", "ver").Output()
		if err == nil {
			if m := windowsVerNumber.Find(out); len(m) > 0 {
				return "Microsoft Windows " + string(m)
			}
			return strings.TrimSpace(string(out))
		}
	}
	return runtime.GOOS
}

func linkSpeedMbps() float64 {
	switch runtime.GOOS {
	case "linux":
		entries, err := os.ReadDir("/sys/class/net")
		if err != nil {
			return 0
		}
		var max float64
		for _, e := range entries {
			name := e.Name()
			if name == "lo" {
				continue
			}
			b, err := os.ReadFile("/sys/class/net/" + name + "/speed")
			if err != nil {
				continue
			}
			v, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
			if err != nil || v <= 0 {
				continue
			}
			if v > max {
				max = v
			}
		}
		return max
	default:
		return 0
	}
}
