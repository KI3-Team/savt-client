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
	"fmt"
	"log"
	"strings"
	"time"

	pb "savt-client/savt-client-api/savt"
	"savt-client/savt-client-api/utils"
	"savt-client/savt-client-upgrader/internal/protocol"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const workerPollInterval = 2 * time.Second

// WorkerWaitSnapshot is reported while waiting for an in-flight measurement.
type WorkerWaitSnapshot struct {
	JobID    string
	Progress int32
	Duration int32
	JobType  string
	Waiting  bool // true while measurement still running
}

// NeedsWorkerStop reports whether this update will stop or replace the worker.
func NeedsWorkerStop(manifest *protocol.PackageManifest, stopProcesses []string) bool {
	if len(stopProcesses) == 0 {
		return true // default stop list includes worker
	}
	for _, name := range stopProcesses {
		if strings.Contains(strings.ToLower(name), "worker") {
			return true
		}
	}
	if manifest == nil {
		return false
	}
	for _, e := range append(append([]protocol.FileEntry{}, manifest.Add...), manifest.Replace...) {
		if strings.Contains(e.Path, "savt-client-worker") {
			return true
		}
	}
	return false
}

// WaitWorkerIdle blocks until the local worker is not running a measurement,
// or until ctx is cancelled. If the worker gRPC endpoint is unreachable, it
// returns nil so the update can proceed (worker already down / not installed).
// onWaiting is called on each poll while a measurement is still running, and
// once more with Waiting=false when the measurement has finished.
func WaitWorkerIdle(ctx context.Context, onWaiting func(WorkerWaitSnapshot)) error {
	addr := fmt.Sprintf("127.0.0.1:%d", utils.LocalPort)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("worker idle check: dial failed, proceed: %v", err)
		return nil
	}
	defer conn.Close()

	client := pb.NewSavtIpcClient(conn)
	ticker := time.NewTicker(workerPollInterval)
	defer ticker.Stop()

	wasWaiting := false
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		job, err := client.GetStatus(cctx, &pb.JobIdRequest{})
		cancel()
		if err != nil {
			log.Printf("worker idle check: GetStatus failed, proceed: %v", err)
			return nil
		}
		snap := WorkerWaitSnapshot{
			JobID:    job.GetJobId(),
			Progress: job.GetProgressBar(),
			Duration: job.GetDurationSeconds(),
			JobType:  job.GetJobType().String(),
			Waiting:  job.GetStatus() == pb.Status_RUNNING,
		}
		if !snap.Waiting {
			if wasWaiting {
				log.Printf("worker measurement finished (status=%s job=%s), continuing update", job.GetStatus().String(), snap.JobID)
				if onWaiting != nil {
					onWaiting(snap)
				}
			}
			return nil
		}
		if !wasWaiting {
			log.Printf("worker is measuring (job=%s type=%s); waiting before stop/update", snap.JobID, snap.JobType)
			wasWaiting = true
		}
		if onWaiting != nil {
			onWaiting(snap)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
