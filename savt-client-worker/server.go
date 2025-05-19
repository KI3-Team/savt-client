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

/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "savt-client/savt-client-api/savt"
	prober "savt-client/savt-client-worker/prober"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedSavtIpcServer
	pmu              sync.Mutex
	prober           *prober.Prober
	job              pb.Job
	jobMu            sync.Mutex
	logMu            sync.Mutex
	logs             []string
	logUpdateCh      chan string
	historyMu        sync.Mutex
	config           *pb.Config
	confMu           sync.Mutex
	scheduledProber  *prober.Prober
	scheMu           sync.Mutex
	shJob            pb.Job
	shJobMu          sync.Mutex
	schedulerStopCh  chan struct{} // Channel to control runScheduler
	schedulerRunning bool          // Flag to check if the scheduler is running
}

func NewServer() *server {
	s := &server{
		logUpdateCh: make(chan string, 200),
	}
	s.prober = prober.NewProber(s.handleStatus, s.handleLog, s.handleDone)
	s.scheduledProber = prober.NewProber(s.handleShStatus, s.handleShLog, s.handleShDone)
	s.job = pb.Job{
		JobId:       "",
		Status:      pb.Status_INITIAL,
		StartTime:   nil,
		Token:       "",
		Tasks:       []*pb.Task{},
		ProgressBar: 0,
		Ipv4:        &pb.MeasurementResult{},
		Ipv6:        &pb.MeasurementResult{},
		JobType:     pb.Jobtype_MANUAL,
	}
	s.shJob = pb.Job{
		JobId:       "",
		Status:      pb.Status_INITIAL,
		StartTime:   nil,
		Token:       "",
		Tasks:       []*pb.Task{},
		ProgressBar: 0,
		Ipv4:        &pb.MeasurementResult{},
		Ipv6:        &pb.MeasurementResult{},
		JobType:     pb.Jobtype_SCHEDULED,
	}
	if err := s.initConfig(); err != nil {
		log.Fatalf("Failed to initialize configuration file: %v", err)
	}
	s.confMu.Lock()
	defer s.confMu.Unlock()
	if s.config.EnableScheduler {
		s.schedulerStopCh = make(chan struct{})
		go s.runScheduler()
		s.schedulerRunning = true
	} else {
		s.schedulerRunning = false
	}
	log.Printf("Initialize configuration successfully: %+v\n", s.config)
	return s
}

// SayHello implements helloworld.GreeterServer
func (s *server) SetConfig(_ context.Context, in *pb.Config) (*emptypb.Empty, error) {
	s.confMu.Lock()
	defer s.confMu.Unlock()
	log.Printf("Received SetConfig request: %+v", in)
	// Convert ConfigRequest to JSON
	configData := map[string]interface{}{
		"Enable":          in.EnableScheduler,
		"Complete":        in.CompleteInterval,
		"Incomplete":      in.IncompleteRetryInterval,
		"CheckNetwork":    in.CheckNetworkInterval,
		"WaitAfterChange": in.WaitAfterChangeInterval,
		"RetryLimit":      in.RetryLimit,
		"IsPublic":        in.IsPublic,
	}
	// Create configuration file path
	if err := os.MkdirAll(filepath.Dir(configFilePath), 0755); err != nil {
		return &emptypb.Empty{}, err
	}
	// Open or create configuration file
	file, err := os.OpenFile(configFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return &emptypb.Empty{}, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing config file: %v", err)
		}
	}()
	// Write configuration to file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Format JSON file
	if err := encoder.Encode(configData); err != nil {
		return &emptypb.Empty{}, err
	}
	log.Printf("Set configuration successfully: %+v", in)
	s.config, err = s.getConfig()
	if err != nil {
		log.Printf("Error loading configuration: %v", err)
	}
	// Determine whether to start or stop runScheduler
	if s.config.EnableScheduler && !s.schedulerRunning {
		s.schedulerStopCh = make(chan struct{})
		go s.runScheduler()
		s.schedulerRunning = true
	} else if !s.config.EnableScheduler && s.schedulerRunning {
		close(s.schedulerStopCh)
		s.schedulerRunning = false
	}

	return &emptypb.Empty{}, nil
}

func (s *server) GetConfig(_ context.Context, _ *emptypb.Empty) (*pb.Config, error) {
	s.confMu.Lock()
	defer s.confMu.Unlock()
	log.Printf("Received GetConfig request")
	return s.config, nil
}

func (s *server) Start(_ context.Context, _ *emptypb.Empty) (*pb.Job, error) {
	log.Printf("Received Start request")
	s.pmu.Lock()
	defer s.pmu.Unlock()

	if s.prober.IsRunning() {
		log.Printf("Error starting prober: Prober is already running")
		return nil, fmt.Errorf("prober is already running")
	}

	s.logMu.Lock()
	defer s.logMu.Unlock()
	s.logs = []string{}

	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	currentTime := time.Now().Format(time.RFC1123)
	sanitizedTime := strings.ReplaceAll(currentTime, ",", "")
	sanitizedTime = strings.ReplaceAll(sanitizedTime, " ", "_")
	sanitizedTime = strings.ReplaceAll(sanitizedTime, ":", "-")
	s.job = pb.Job{
		JobId:       sanitizedTime,
		Status:      pb.Status_RUNNING,
		StartTime:   timestamppb.Now(),
		Token:       "",
		Tasks:       []*pb.Task{},
		ProgressBar: 0,
		Ipv4:        &pb.MeasurementResult{},
		Ipv6:        &pb.MeasurementResult{},
		JobType:     pb.Jobtype_MANUAL,
	}

	s.confMu.Lock()
	defer s.confMu.Unlock()
	if err := s.prober.Start(remotePort, s.config.IsPublic); err != nil {
		log.Printf("Error starting prober: %v", err)
		newJob := deepCopyJob(&s.job)
		return newJob, nil
	}
	newJob := deepCopyJob(&s.job)
	return newJob, nil
}

func (s *server) Kill(_ context.Context, jobId *pb.JobIdRequest) (*pb.Job, error) {
	log.Printf("Received Stop request")
	s.pmu.Lock()
	defer s.pmu.Unlock()

	if !s.prober.IsRunning() {
		log.Printf("Error stopping prober: Prober is not running")
		return deepCopyJob(&s.job), fmt.Errorf("prober is not running")
	}

	if err := s.prober.Stop(); err != nil {
		log.Printf("Error stopping prober: %v", err)
		return deepCopyJob(&s.job), err
	}
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	s.job.Status = pb.Status_CANCELLED
	if len(s.job.Tasks) > 0 && s.job.Tasks[len(s.job.Tasks)-1].Status == pb.Status_RUNNING {
		s.job.Tasks[len(s.job.Tasks)-1].Status = pb.Status_CANCELLED
	}

	return deepCopyJob(&s.job), nil
}

func (s *server) GetStatus(_ context.Context, jobId *pb.JobIdRequest) (*pb.Job, error) {
	log.Printf("Received GetStatus request")
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if s.job.Status == pb.Status_RUNNING {
		nowTime := timestamppb.Now()
		s.job.DurationSeconds = int32(nowTime.AsTime().Sub(s.job.StartTime.AsTime()).Seconds())
	}
	return deepCopyJob(&s.job), nil
}

func (s *server) Echo(_ context.Context, in *pb.EchoRequest) (*pb.EchoRequest, error) {
	log.Printf("Received Echo request: %s", in.Message)
	return &pb.EchoRequest{Message: in.Message}, nil
}

func (s *server) ReadLog(req *pb.JobIdRequest, stream pb.SavtIpc_ReadLogServer) error {
	// First send existing logs in batch
	s.logMu.Lock()
	existingLogs := append([]string(nil), s.logs...)
	s.logMu.Unlock()
	for _, line := range existingLogs {
		if err := stream.Send(&pb.LogEntry{Line: []string{line}}); err != nil {
			return err
		}
	}
	// Continuously monitor for new logs and client disconnection
	for {
		select {
		case <-stream.Context().Done():
			// Client disconnected or cancelled
			return nil
		case newLine := <-s.logUpdateCh:
			// Received new log, send to client
			if err := stream.Send(&pb.LogEntry{Line: []string{newLine}}); err != nil {
				return err
			}
		}
	}
}

func (s *server) GetHistory(ctx context.Context, req *emptypb.Empty) (*pb.GetHistoryResponse, error) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()
	historyDirPath := historyDirPath

	if err := ensureDir(historyDirPath); err != nil {
		log.Printf("Error ensuring history directory: %v", err)
		return nil, fmt.Errorf("failed to ensure history directory: %v", err)
	}

	entries, err := os.ReadDir(historyDirPath)
	if err != nil {
		log.Printf("Error reading history directory: %v", err)
		return nil, fmt.Errorf("failed to read history directory: %v", err)
	}

	var jobs []*pb.Job

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Only process .json files
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		filePath := filepath.Join(historyDirPath, entry.Name())
		// Read file content
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading file: %v", err)
			continue // Skip problematic files
		}
		// Deserialize JSON data to Job object
		var job pb.Job
		if err := json.Unmarshal(data, &job); err != nil {
			log.Printf("Error unmarshaling JSON from file: %v", err)
			continue // Skip problematic files
		}
		jobs = append(jobs, &job)
	}
	// Sort in reverse order
	sort.Slice(jobs, func(i, j int) bool {
		tsI := jobs[i].StartTime
		tsJ := jobs[j].StartTime
		if tsI == nil && tsJ == nil {
			return false
		}
		if tsI == nil {
			return false // Or return true as needed
		}
		if tsJ == nil {
			return true // Or return false as needed
		}
		return tsI.AsTime().After(tsJ.AsTime())
	})
	// Apply file count limit if needed
	if len(jobs) > MAX_FILES {
		jobs = jobs[:MAX_FILES]
	}
	return &pb.GetHistoryResponse{
		Jobs: jobs,
	}, nil
}

func (s *server) getConfig() (*pb.Config, error) {
	// Open configuration file
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing config file: %v", err)
		}
	}()

	// Read configuration file
	var configData map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&configData); err != nil {
		return nil, err
	}

	// Convert JSON to ConfigEntry
	config := &pb.Config{
		EnableScheduler:         configData["Enable"].(bool),
		CompleteInterval:        int32(configData["Complete"].(float64)),
		IncompleteRetryInterval: int32(configData["Incomplete"].(float64)),
		CheckNetworkInterval:    int32(configData["CheckNetwork"].(float64)),
		WaitAfterChangeInterval: int32(configData["WaitAfterChange"].(float64)),
		RetryLimit:              int32(configData["RetryLimit"].(float64)),
		IsPublic:                configData["IsPublic"].(bool),
	}

	return config, nil
}

func (s *server) initConfig() error {
	s.confMu.Lock()
	defer s.confMu.Unlock()
	if _, err := os.Stat(configFilePath); err == nil {
		s.config, err = s.getConfig()
		if err != nil {
			return err
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	configDir := filepath.Dir(configFilePath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(configFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing config file: %v", err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(initConfig); err != nil {
		return err
	}

	s.config, err = s.getConfig()
	if err != nil {
		return err
	}
	return nil
}

func deepCopyJob(src *pb.Job) *pb.Job {
	if src == nil {
		return nil
	}

	// Use proto.Clone for deep copying, avoid copying locks
	dst := proto.Clone(src).(*pb.Job)

	// Ensure all Tasks are individually cloned
	if src.Tasks != nil {
		dst.Tasks = make([]*pb.Task, len(src.Tasks))
		for i, t := range src.Tasks {
			if t != nil {
				dst.Tasks[i] = proto.Clone(t).(*pb.Task)
			}
		}
	}

	return dst
}
