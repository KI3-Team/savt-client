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

package protocol

import (
	"encoding/json"
	"fmt"
)

// Message type constants exchanged over the upgrader websocket.
const (
	TypeHeartbeat    = "heartbeat"
	TypeUpdate       = "update"
	TypeUpdateStatus = "update_status"
	TypeUpdateCancel = "update_cancel"
	TypeCrashReport  = "crash_report"
	TypePing         = "ping"
	TypePong         = "pong"
)

// Update status values reported back to the server.
const (
	StatusAccepted           = "accepted"
	StatusWaitingWifi        = "waiting_wifi"
	StatusWaitingIdle        = "waiting_idle"
	StatusWaitingMeasurement = "waiting_measurement"
	StatusDownloading        = "downloading"
	StatusExtracting         = "extracting"
	StatusVerifying          = "verifying"
	StatusStopping           = "stopping"
	StatusApplying           = "applying"
	StatusRestarting         = "restarting"
	StatusSuccess            = "success"
	StatusFailed             = "failed"
	StatusCancelled          = "cancelled"
)

// Envelope is the common websocket JSON envelope.
type Envelope struct {
	Type string `json:"type"`
}

// Heartbeat is the periodic device telemetry payload.
type Heartbeat struct {
	Type          string            `json:"type"`
	ID            string            `json:"id"`
	MAC           string            `json:"mac"`
	IP            string            `json:"ip"`
	AppVersion    string            `json:"app_version"`
	OS            string            `json:"os"`
	OSVersion     string            `json:"os_version"`
	Arch          string            `json:"arch"`
	ModulesStatus map[string]string `json:"modules_status"`
	Timestamp     int64             `json:"timestamp"`
}

// UpdatePolicy controls when an update may download / install.
type UpdatePolicy struct {
	WifiOnly     bool `json:"wifi_only,omitempty"`
	IdleOnly     bool `json:"idle_only,omitempty"`
	IdleSeconds  int  `json:"idle_seconds,omitempty"` // default 300 if idle_only and <=0
	AutoDownload bool `json:"auto_download,omitempty"`
	AutoInstall  bool `json:"auto_install,omitempty"`
}

// UpdateCommand is the server-pushed update notice.
// Client downloads https://.../savt-client-{version}.zip then follows upgrade.json inside.
type UpdateCommand struct {
	Type       string        `json:"type"`
	UpdateID   string        `json:"update_id,omitempty"`
	Version    string        `json:"version"`
	SHA256     string        `json:"sha256,omitempty"` // optional zip checksum
	URL        string        `json:"url,omitempty"`    // optional full zip URL override
	MaxRetries int           `json:"max_retries,omitempty"`
	Policy     *UpdatePolicy `json:"policy,omitempty"`
}

// FileEntry describes a file to add or replace from the extracted package.
type FileEntry struct {
	Path   string `json:"path"`             // install destination relative to install root
	Source string `json:"source,omitempty"` // path inside zip; defaults to Path
	Mode   string `json:"mode,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

// SourcePath returns the path of this entry inside the extracted zip.
func (e FileEntry) SourcePath() string {
	if e.Source != "" {
		return e.Source
	}
	return e.Path
}

// FileEntryList accepts either ["a","b"] or [{"path":"a"},...].
type FileEntryList []FileEntry

// UnmarshalJSON supports string array or object array forms.
func (l *FileEntryList) UnmarshalJSON(data []byte) error {
	var objs []FileEntry
	if err := json.Unmarshal(data, &objs); err == nil {
		*l = objs
		return nil
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return fmt.Errorf("file list must be string[] or object[]: %w", err)
	}
	out := make([]FileEntry, 0, len(names))
	for _, n := range names {
		out = append(out, FileEntry{Path: n})
	}
	*l = out
	return nil
}

// RestartStep describes how to bring modules back after an update.
// Type is one of: "service", "exec", "launchctl", "systemctl", "nssm".
type RestartStep struct {
	Type    string   `json:"type"`
	Name    string   `json:"name,omitempty"`
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

// PackageManifest is the upgrade config shipped inside the version zip (upgrade.json).
type PackageManifest struct {
	Version       string        `json:"version"`
	Delete        []string      `json:"delete"`
	Add           FileEntryList `json:"add"`
	Replace       FileEntryList `json:"replace"`
	StopProcesses []string      `json:"stop_processes"`
	Restart       []RestartStep `json:"restart"`
	PublicKeyPEM  string        `json:"public_key_pem,omitempty"`
}

// UpdateStatus reports progress or failure for an update job.
type UpdateStatus struct {
	Type            string  `json:"type"`
	UpdateID        string  `json:"update_id"`
	Status          string  `json:"status"`
	Stage           string  `json:"stage,omitempty"`
	Message         string  `json:"message,omitempty"` // non-error progress detail
	Error           string  `json:"error,omitempty"`
	Retries         int     `json:"retries,omitempty"`
	Version         string  `json:"version,omitempty"`
	BytesDownloaded int64   `json:"bytes_downloaded,omitempty"`
	BytesTotal      int64   `json:"bytes_total,omitempty"`
	Progress        float64 `json:"progress,omitempty"` // 0-100
	TS              int64   `json:"ts"`
}

// UpdateCancel requests cancellation of an in-flight update.
type UpdateCancel struct {
	Type     string `json:"type"`
	UpdateID string `json:"update_id"`
}

// CrashReport carries panic / unexpected failure details.
type CrashReport struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Stage   string `json:"stage,omitempty"`
	Error   string `json:"error"`
	Stack   string `json:"stack,omitempty"`
	Version string `json:"version,omitempty"`
	TS      int64  `json:"ts"`
}
