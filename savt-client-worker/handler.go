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

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/types/known/timestamppb"

	pb "savt-client/savt-client-api/savt"
	prober "savt-client/savt-client-worker/prober"
)

type JobWithoutLock struct {
	JobId           string                 `json:"job_id"`
	Status          pb.Status              `json:"status"`
	StartTime       *timestamppb.Timestamp `json:"start_time"`
	DurationSeconds int32                  `json:"duration_seconds"`
	Tasks           []*pb.Task             `json:"tasks"`
	ProgressBar     int32                  `json:"progress_bar"`
	Ipv4            *pb.MeasurementResult  `json:"ipv4"`
	Ipv6            *pb.MeasurementResult  `json:"ipv6"`
	Token           string                 `json:"token"`
	JobType         pb.Jobtype             `json:"job_type"`
}

func (s *server) handleStatus(status prober.Status) {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	var tasks []*pb.Task
	for _, task := range status.Tasks {
		if task.Step == 9 {
			continue
		}
		tasks = append(tasks, &pb.Task{
			Step:       int32(task.Step),
			Status:     statusMap[task.Status],
			StatusDesc: taskDescMap[task.Step],
		})
	}
	nowTime := timestamppb.Now()
	s.job.DurationSeconds = int32(nowTime.AsTime().Sub(s.job.StartTime.AsTime()).Seconds())
	s.job.Tasks = tasks
	s.job.Token = status.Token
	s.job.ProgressBar = int32(status.ProgressBar)
	s.job.Ipv4.ClientAddress = status.Ipv4.ClientAddress
	s.job.Ipv4.Asn = int32(status.Ipv4.Asn)
	s.job.Ipv4.OutboundPrivate = measurementResultMap[status.Ipv4.OutboundPrivate]
	s.job.Ipv4.OutboundRoutable = measurementResultMap[status.Ipv4.OutboundRoutable]
	s.job.Ipv4.SpoofablePrefixLength = int32(status.Ipv4.SpoofablePrefixLength)
	s.job.Ipv4.InboundPrivate = measurementResultMap[status.Ipv4.InboundPrivate]
	s.job.Ipv4.InboundInternal = measurementResultMap[status.Ipv4.InboundInternal]
	s.job.Ipv6.ClientAddress = status.Ipv6.ClientAddress
	s.job.Ipv6.Asn = int32(status.Ipv6.Asn)
	s.job.Ipv6.OutboundPrivate = measurementResultMap[status.Ipv6.OutboundPrivate]
	s.job.Ipv6.OutboundRoutable = measurementResultMap[status.Ipv6.OutboundRoutable]
	s.job.Ipv6.SpoofablePrefixLength = int32(status.Ipv6.SpoofablePrefixLength)
	s.job.Ipv6.InboundPrivate = measurementResultMap[status.Ipv6.InboundPrivate]
	s.job.Ipv6.InboundInternal = measurementResultMap[status.Ipv6.InboundInternal]
	log.Printf("Status Update: %+v", status)
}

func (s *server) handleLog(message string) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	s.logs = append(s.logs, message)

	select {
	case s.logUpdateCh <- message:
	default:
	}
}

func (s *server) handleDone() {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	nowTime := timestamppb.Now()
	s.job.DurationSeconds = int32(nowTime.AsTime().Sub(s.job.StartTime.AsTime()).Seconds())
	if s.job.ProgressBar == 100 {
		statusAllSucceeded := true
		statusAllFailed := true
		if len(s.job.Tasks) == 0 {
			s.job.Status = pb.Status_FAILED
		} else {
			for _, t := range s.job.Tasks {
				if t.Status != pb.Status_SUCCEEDED {
					statusAllSucceeded = false
				}
				if t.Status != pb.Status_FAILED {
					statusAllFailed = false
				}
			}
			switch {
			case statusAllSucceeded:
				s.job.Status = pb.Status_SUCCEEDED
			case statusAllFailed:
				s.job.Status = pb.Status_FAILED
			default:
				s.job.Status = pb.Status_SEMI_SUCCEEDED
			}
		}
	} else {
		s.job.Status = pb.Status_FAILED
	}
	if s.job.Status == pb.Status_SUCCEEDED || s.job.Status == pb.Status_SEMI_SUCCEEDED {
		s.historyMu.Lock()
		defer s.historyMu.Unlock()
		historyDirPath := historyDirPath
		// Check and create directory
		err := ensureDir(historyDirPath)
		if err != nil {
			log.Println("Error:", err)
		}
		jobWithoutLock := JobWithoutLock{
			JobId:           s.job.JobId,
			Status:          s.job.Status,
			StartTime:       s.job.StartTime,
			DurationSeconds: s.job.DurationSeconds,
			Tasks:           s.job.Tasks,
			ProgressBar:     s.job.ProgressBar,
			Ipv4:            s.job.Ipv4,
			Ipv6:            s.job.Ipv6,
			Token:           s.job.Token,
			JobType:         s.job.JobType,
		}
		// Serialize to JSON
		jsonData, err := json.Marshal(jobWithoutLock)
		if err != nil {
			log.Println("Error serializing to JSON:", err)
			return
		}
		// Create filename with timestamp to avoid conflicts
		fileName := fmt.Sprintf("%s.json", s.job.JobId)
		filePath := filepath.Join(historyDirPath, fileName)
		// Write to file
		err = os.WriteFile(filePath, jsonData, 0644)
		if err != nil {
			log.Println("Error writing file:", err)
			return
		}
		// Check and limit the number of files
		err = enforceFileLimit(historyDirPath, MAX_FILES)
		if err != nil {
			log.Println("Error enforcing file limit:", err)
		}
	}
	log.Printf("Test Done")
}

func (s *server) handleShStatus(status prober.Status) {
	s.shJobMu.Lock()
	defer s.shJobMu.Unlock()
	var tasks []*pb.Task
	for _, task := range status.Tasks {
		if task.Step == 9 {
			continue
		}
		tasks = append(tasks, &pb.Task{
			Step:       int32(task.Step),
			Status:     statusMap[task.Status],
			StatusDesc: taskDescMap[task.Step],
		})
	}
	nowTime := timestamppb.Now()
	s.shJob.DurationSeconds = int32(nowTime.AsTime().Sub(s.shJob.StartTime.AsTime()).Seconds())
	s.shJob.Tasks = tasks
	s.shJob.Token = status.Token
	s.shJob.ProgressBar = int32(status.ProgressBar)
	s.shJob.Ipv4.ClientAddress = status.Ipv4.ClientAddress
	s.shJob.Ipv4.Asn = int32(status.Ipv4.Asn)
	s.shJob.Ipv4.OutboundPrivate = measurementResultMap[status.Ipv4.OutboundPrivate]
	s.shJob.Ipv4.OutboundRoutable = measurementResultMap[status.Ipv4.OutboundRoutable]
	s.shJob.Ipv4.SpoofablePrefixLength = int32(status.Ipv4.SpoofablePrefixLength)
	s.shJob.Ipv4.InboundPrivate = measurementResultMap[status.Ipv4.InboundPrivate]
	s.shJob.Ipv4.InboundInternal = measurementResultMap[status.Ipv4.InboundInternal]
	s.shJob.Ipv6.ClientAddress = status.Ipv6.ClientAddress
	s.shJob.Ipv6.Asn = int32(status.Ipv6.Asn)
	s.shJob.Ipv6.OutboundPrivate = measurementResultMap[status.Ipv6.OutboundPrivate]
	s.shJob.Ipv6.OutboundRoutable = measurementResultMap[status.Ipv6.OutboundRoutable]
	s.shJob.Ipv6.SpoofablePrefixLength = int32(status.Ipv6.SpoofablePrefixLength)
	s.shJob.Ipv6.InboundPrivate = measurementResultMap[status.Ipv6.InboundPrivate]
	s.shJob.Ipv6.InboundInternal = measurementResultMap[status.Ipv6.InboundInternal]
	log.Printf("Scheduler Status Update: %+v", status)
}

func (s *server) handleShLog(message string) {
}

func (s *server) handleShDone() {
	s.shJobMu.Lock()
	defer s.shJobMu.Unlock()
	nowTime := timestamppb.Now()
	s.shJob.DurationSeconds = int32(nowTime.AsTime().Sub(s.shJob.StartTime.AsTime()).Seconds())
	if s.shJob.ProgressBar == 100 {
		statusAllSucceeded := true
		statusAllFailed := true
		if len(s.shJob.Tasks) == 0 {
			s.shJob.Status = pb.Status_FAILED
		} else {
			for _, t := range s.shJob.Tasks {
				if t.Status != pb.Status_SUCCEEDED {
					statusAllSucceeded = false
				}
				if t.Status != pb.Status_FAILED {
					statusAllFailed = false
				}
			}
			switch {
			case statusAllSucceeded:
				s.shJob.Status = pb.Status_SUCCEEDED
			case statusAllFailed:
				s.shJob.Status = pb.Status_FAILED
			default:
				s.shJob.Status = pb.Status_SEMI_SUCCEEDED
			}
		}
	} else {
		s.shJob.Status = pb.Status_FAILED
	}
	if s.shJob.Status == pb.Status_SUCCEEDED || s.shJob.Status == pb.Status_SEMI_SUCCEEDED {
		s.historyMu.Lock()
		defer s.historyMu.Unlock()
		historyDirPath := historyDirPath
		// Check and create directory
		err := ensureDir(historyDirPath)
		if err != nil {
			log.Println("Error:", err)
		}
		jobWithoutLock := JobWithoutLock{
			JobId:           s.shJob.JobId,
			Status:          s.shJob.Status,
			StartTime:       s.shJob.StartTime,
			DurationSeconds: s.shJob.DurationSeconds,
			Tasks:           s.shJob.Tasks,
			ProgressBar:     s.shJob.ProgressBar,
			Ipv4:            s.shJob.Ipv4,
			Ipv6:            s.shJob.Ipv6,
			Token:           s.shJob.Token,
			JobType:         s.shJob.JobType,
		}
		// Serialize to JSON
		jsonData, err := json.Marshal(jobWithoutLock)
		if err != nil {
			log.Println("Error serializing to JSON:", err)
			return
		}
		// Create filename with timestamp to avoid conflicts
		fileName := fmt.Sprintf("%s.json", s.shJob.JobId)
		filePath := filepath.Join(historyDirPath, fileName)
		// Write to file
		err = os.WriteFile(filePath, jsonData, 0644)
		if err != nil {
			log.Println("Error writing file:", err)
			return
		}
		// Check and limit the number of files
		err = enforceFileLimit(historyDirPath, MAX_FILES)
		if err != nil {
			log.Println("Error enforcing file limit:", err)
		}
	}
	log.Printf("Scheduler Test Done")
}
