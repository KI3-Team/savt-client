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

package update

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"savt-client/savt-client-upgrader/internal/idle"
	"savt-client/savt-client-upgrader/internal/paths"
	"savt-client/savt-client-upgrader/internal/protocol"
	"savt-client/savt-client-upgrader/internal/svcjson"
)

const (
	gatePollInterval     = 30 * time.Second
	defaultIdleSeconds   = 300
	progressReportMinGap = 500 * time.Millisecond
)

// StatusSender sends update progress / failure frames to the server.
type StatusSender func(status protocol.UpdateStatus) error

// Manager orchestrates zip download, extract, apply, and restart.
type Manager struct {
	Downloader *Downloader
	Applier    *Applier
	MaxRetries int
	SendStatus StatusSender

	mu       sync.Mutex
	running  bool
	updateID string
	cancelFn context.CancelFunc
}

// Cancel aborts the in-flight update matching updateID (or any if updateID is empty).
// Returns true if a cancel signal was delivered.
func (m *Manager) Cancel(updateID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running || m.cancelFn == nil {
		return false
	}
	if updateID != "" && m.updateID != "" && updateID != m.updateID {
		return false
	}
	m.cancelFn()
	return true
}

// HandleUpdate runs the update pipeline asynchronously.
func (m *Manager) HandleUpdate(cmd protocol.UpdateCommand) {
	cmd.Version = strings.TrimSpace(cmd.Version)
	if cmd.Version == "" {
		_ = m.report(cmd, protocol.StatusFailed, "invalid", "version is required", 0)
		return
	}
	if cmd.UpdateID == "" {
		cmd.UpdateID = "upd-" + cmd.Version + "-" + fmt.Sprintf("%d", time.Now().Unix())
	}

	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		cancel()
		_ = m.report(cmd, protocol.StatusFailed, "busy", "another update is in progress", 0)
		return
	}
	m.running = true
	m.updateID = cmd.UpdateID
	m.cancelFn = cancel
	m.mu.Unlock()

	go func() {
		defer func() {
			cancel()
			m.mu.Lock()
			m.running = false
			m.updateID = ""
			m.cancelFn = nil
			m.mu.Unlock()
		}()
		retries := cmd.MaxRetries
		if retries <= 0 {
			retries = m.MaxRetries
		}
		var lastErr error
		for attempt := 0; attempt <= retries; attempt++ {
			if ctx.Err() != nil {
				_ = m.report(cmd, protocol.StatusCancelled, "cancelled", "update cancelled", attempt)
				return
			}
			if attempt > 0 {
				backoff := time.Duration(attempt*attempt) * time.Second
				log.Printf("retrying update %s in %s (attempt %d/%d)", cmd.UpdateID, backoff, attempt, retries)
				select {
				case <-ctx.Done():
					_ = m.report(cmd, protocol.StatusCancelled, "cancelled", "update cancelled", attempt)
					return
				case <-time.After(backoff):
				}
			}
			lastErr = m.runOnce(ctx, cmd, attempt)
			if lastErr == nil {
				return
			}
			if errors.Is(lastErr, context.Canceled) || errors.Is(lastErr, context.DeadlineExceeded) {
				_ = m.report(cmd, protocol.StatusCancelled, "cancelled", "update cancelled", attempt)
				return
			}
			_ = m.report(cmd, protocol.StatusFailed, "retry", lastErr.Error(), attempt)
		}
		_ = m.report(cmd, protocol.StatusFailed, "exhausted", lastErr.Error(), retries)
	}()
}

func (m *Manager) runOnce(ctx context.Context, cmd protocol.UpdateCommand, attempt int) error {
	_ = m.report(cmd, protocol.StatusAccepted, "accepted", "", attempt)

	if err := m.waitPolicy(ctx, cmd, attempt); err != nil {
		return err
	}

	workDir := filepath.Join(m.Downloader.Staging, "work-"+cmd.Version)
	extractDir := filepath.Join(workDir, "extract")
	defer func() { _ = os.RemoveAll(workDir) }()

	_ = m.report(cmd, protocol.StatusDownloading, "download", "", attempt)
	var lastProgress time.Time
	onProgress := func(downloaded, total int64) {
		now := time.Now()
		if !lastProgress.IsZero() && now.Sub(lastProgress) < progressReportMinGap && (total <= 0 || downloaded < total) {
			return
		}
		lastProgress = now
		_ = m.reportProgress(cmd, downloaded, total, attempt)
	}
	zipPath, _, err := m.Downloader.FetchZip(ctx, cmd.Version, cmd.URL, cmd.SHA256, onProgress)
	if err != nil {
		return fmt.Errorf("download package: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	_ = m.report(cmd, protocol.StatusExtracting, "extract", "", attempt)
	if err := ExtractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extract package: %w", err)
	}

	manifest, manifestPath, err := LoadManifest(extractDir)
	if err != nil {
		return err
	}
	packageRoot := ManifestRoot(manifestPath)
	log.Printf("loaded package manifest %s version=%s", manifestPath, manifest.Version)

	if manifest.Version == "" {
		manifest.Version = cmd.Version
	}
	if manifest.Version != cmd.Version {
		log.Printf("warning: manifest version %s differs from commanded version %s", manifest.Version, cmd.Version)
	}

	_ = m.report(cmd, protocol.StatusVerifying, "verify", "", attempt)
	allEntries := append(append([]protocol.FileEntry{}, manifest.Add...), manifest.Replace...)
	if err := VerifyPackageEntries(packageRoot, allEntries); err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	stopList := manifest.StopProcesses
	if NeedsWorkerStop(manifest, stopList) {
		var lastWaitReport time.Time
		if err := WaitWorkerIdle(ctx, func(snap WorkerWaitSnapshot) {
			now := time.Now()
			// Always report start/finish; throttle intermediate polls to ~5s.
			if snap.Waiting && !lastWaitReport.IsZero() && now.Sub(lastWaitReport) < 5*time.Second {
				return
			}
			lastWaitReport = now
			msg := ""
			if snap.Waiting {
				msg = fmt.Sprintf("waiting for worker measurement to finish: job=%s type=%s progress=%d%% duration=%ds",
					snap.JobID, snap.JobType, snap.Progress, snap.Duration)
			} else {
				msg = fmt.Sprintf("worker measurement finished: job=%s progress=%d%% duration=%ds; continuing update",
					snap.JobID, snap.Progress, snap.Duration)
			}
			_ = m.reportMsg(cmd, protocol.StatusWaitingMeasurement, "wait_measurement", msg, attempt, float64(snap.Progress))
		}); err != nil {
			return fmt.Errorf("wait worker measurement: %w", err)
		}
	}

	_ = m.report(cmd, protocol.StatusStopping, "stop", "", attempt)
	if err := StopProcesses(stopList); err != nil {
		log.Printf("stop processes warning: %v", err)
	}

	selfUpdate := false
	for _, entry := range allEntries {
		ok, err := IsSelfTarget(paths.ResolvePath(entry.Path))
		if err == nil && ok {
			selfUpdate = true
		}
	}

	_ = m.report(cmd, protocol.StatusApplying, "apply", "", attempt)
	if err := m.Applier.DeleteFiles(manifest.Delete); err != nil {
		return err
	}
	if err := m.Applier.ApplyEntries(manifest.Replace, packageRoot); err != nil {
		return err
	}
	if err := m.Applier.ApplyEntries(manifest.Add, packageRoot); err != nil {
		return err
	}

	versionToPersist := manifest.Version
	if versionToPersist == "" {
		versionToPersist = cmd.Version
	}
	if err := persistVersion(versionToPersist); err != nil {
		log.Printf("persist version warning: %v", err)
	}

	_ = m.report(cmd, protocol.StatusRestarting, "restart", "", attempt)
	if err := RunRestartSteps(manifest.Restart); err != nil {
		log.Printf("restart warning: %v", err)
	}

	// Remove downloaded zip (and resume .part) after a successful update.
	if zipPath != "" {
		if err := os.Remove(zipPath); err != nil && !os.IsNotExist(err) {
			log.Printf("remove zip warning: %v", err)
		}
		if err := os.Remove(zipPath + ".part"); err != nil && !os.IsNotExist(err) {
			log.Printf("remove zip part warning: %v", err)
		}
	}

	_ = m.report(cmd, protocol.StatusSuccess, "done", "", attempt)
	if selfUpdate {
		log.Printf("self-update applied; exiting so the service manager can relaunch")
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}
	return nil
}

func (m *Manager) waitPolicy(ctx context.Context, cmd protocol.UpdateCommand, attempt int) error {
	policy := cmd.Policy
	if policy == nil {
		return nil
	}
	// WifiOnly 门已移除: 以太网设备(RackNerd等)会永久等待,静默阻塞更新;
	// 服务器下发的 wifi_only 选项一律忽略,立即放行。
	_ = policy.WifiOnly
	if policy.IdleOnly {
		need := policy.IdleSeconds
		if need <= 0 {
			need = defaultIdleSeconds
		}
		if err := m.waitIdle(ctx, cmd, attempt, need); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) waitIdle(ctx context.Context, cmd protocol.UpdateCommand, attempt int, needSeconds int) error {
	for {
		secs, err := idle.IdleSeconds()
		if err == nil && secs >= int64(needSeconds) {
			return nil
		}
		_ = m.report(cmd, protocol.StatusWaitingIdle, "idle", "", attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(gatePollInterval):
		}
	}
}

func persistVersion(version string) error {
	return svcjson.WriteVersion(version)
}

func (m *Manager) report(cmd protocol.UpdateCommand, status, stage, errMsg string, retries int) error {
	return m.reportFull(cmd, status, stage, "", errMsg, retries, 0, 0, 0)
}

func (m *Manager) reportMsg(cmd protocol.UpdateCommand, status, stage, message string, retries int, progress float64) error {
	return m.reportFull(cmd, status, stage, message, "", retries, 0, 0, progress)
}

func (m *Manager) reportProgress(cmd protocol.UpdateCommand, downloaded, total int64, retries int) error {
	progress := 0.0
	if total > 0 {
		progress = float64(downloaded) / float64(total) * 100
		if progress > 100 {
			progress = 100
		}
	}
	return m.reportFull(cmd, protocol.StatusDownloading, "download", "", "", retries, downloaded, total, progress)
}

func (m *Manager) reportFull(cmd protocol.UpdateCommand, status, stage, message, errMsg string, retries int, downloaded, total int64, progress float64) error {
	if m.SendStatus == nil {
		return nil
	}
	return m.SendStatus(protocol.UpdateStatus{
		Type:            protocol.TypeUpdateStatus,
		UpdateID:        cmd.UpdateID,
		Status:          status,
		Stage:           stage,
		Message:         message,
		Error:           errMsg,
		Retries:         retries,
		Version:         cmd.Version,
		BytesDownloaded: downloaded,
		BytesTotal:      total,
		Progress:        progress,
		TS:              time.Now().Unix(),
	})
}
