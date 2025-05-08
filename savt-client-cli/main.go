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

// client/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "savt-client/savt-client-api/savt" // Update the import path as necessary
	utils "savt-client/savt-client-api/utils"
)

// ServerAddress defines the address of the gRPC server.
// You can modify this if your server is running elsewhere.
var ServerAddress = "localhost:" + strconv.Itoa(utils.LocalPort)

// TerminalStatuses defines the job statuses that will terminate the prober.
var (
	vm, err          = utils.GetServiceManagerIns()
	Version          = vm.GetVersion()
	TerminalStatuses = map[pb.Status]bool{
		pb.Status_SUCCEEDED:      true,
		pb.Status_FAILED:         true,
		pb.Status_SEMI_SUCCEEDED: true,
	}
)

// BoolFlag is a custom flag type to track if it was set.
type BoolFlag struct {
	value bool
	set   bool
}

func (b *BoolFlag) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	b.value = v
	b.set = true
	return nil
}

func (b *BoolFlag) String() string {
	return fmt.Sprintf("%v", b.value)
}

func (b *BoolFlag) IsBoolFlag() bool {
	return true
}

// IntFlag is a custom flag type to track if it was set.
type IntFlag struct {
	value int
	set   bool
}

func (i *IntFlag) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	i.value = v
	i.set = true
	return nil
}

func (i *IntFlag) String() string {
	return fmt.Sprintf("%d", i.value)
}

func main() {
	// Check if --version flag is passed
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Printf("savt-client version %s\n", Version)
			os.Exit(0)
		}
	}

	if len(os.Args) < 2 {
		fmt.Print("Usage: client [config|prober] [options]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "config":
		handleConfig(os.Args[2:])
	case "prober":
		handleProber(os.Args[2:])
	default:
		fmt.Print("Unknown command. Use 'config' or 'prober'.")
		os.Exit(1)
	}
}

// handleConfig processes the 'config' command to update server configuration.
func handleConfig(args []string) {
	// Define custom flags to track if they were set.
	enableScheduler := &BoolFlag{}
	completeInterval := &IntFlag{}
	incompleteRetryInterval := &IntFlag{}
	checkNetworkInterval := &IntFlag{}
	waitAfterChangeInterval := &IntFlag{}
	retryLimit := &IntFlag{}
	isPublic := &BoolFlag{}

	// Define flags using custom flag types with appropriate descriptions.
	configFlags := flag.NewFlagSet("config", flag.ExitOnError)
	configFlags.Var(enableScheduler, "enable", "Enable Scheduler (default=true)")
	configFlags.Var(completeInterval, "complete", "Complete interval after measurement in seconds (default=608700, must be >=0)")
	configFlags.Var(incompleteRetryInterval, "incomplete", "Wait time before retrying incomplete measurements in seconds (default=600, must be >=0)")
	configFlags.Var(checkNetworkInterval, "checknetwork", "Network check interval in seconds (default=120, must be >=60)")
	configFlags.Var(waitAfterChangeInterval, "waitafterchange", "Wait time after network change before measurement in seconds (default=60, must be >=0)")
	configFlags.Var(retryLimit, "retrylimit", "Retry limit (default=3, must be >=1)")
	configFlags.Var(isPublic, "ispublic", "Display results on KI3 (default=true)")

	// Parse the flags.
	if err := configFlags.Parse(args); err != nil {
		fmt.Printf("Failed to parse flags: %v", err)
		os.Exit(1)
	}

	// Establish a connection to the server.
	conn, err := grpc.Dial(ServerAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Failed to connect to server at %s: %v", ServerAddress, err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewSavtIpcClient(conn)

	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retrieve the existing config.
	existingConfig, err := client.GetConfig(ctx, &emptypb.Empty{})
	if err != nil {
		fmt.Printf("Failed to get existing config: %v", err)
		os.Exit(1)
	}

	// Create a new config based on existing config.
	newConfig := &pb.Config{
		EnableScheduler:         existingConfig.EnableScheduler,
		CompleteInterval:        existingConfig.CompleteInterval,
		IncompleteRetryInterval: existingConfig.IncompleteRetryInterval,
		CheckNetworkInterval:    existingConfig.CheckNetworkInterval,
		WaitAfterChangeInterval: existingConfig.WaitAfterChangeInterval,
		RetryLimit:              existingConfig.RetryLimit,
		IsPublic:                existingConfig.IsPublic,
	}

	// Update the config fields if the flags were set.
	if enableScheduler.set {
		newConfig.EnableScheduler = enableScheduler.value
	}
	if completeInterval.set {
		newConfig.CompleteInterval = int32(completeInterval.value)
	}
	if incompleteRetryInterval.set {
		newConfig.IncompleteRetryInterval = int32(incompleteRetryInterval.value)
	}
	if checkNetworkInterval.set {
		newConfig.CheckNetworkInterval = int32(checkNetworkInterval.value)
	}
	if waitAfterChangeInterval.set {
		newConfig.WaitAfterChangeInterval = int32(waitAfterChangeInterval.value)
	}
	if retryLimit.set {
		newConfig.RetryLimit = int32(retryLimit.value)
	}
	if isPublic.set {
		newConfig.IsPublic = isPublic.value
	}

	// Send the updated config.
	_, err = client.SetConfig(ctx, newConfig)
	if err != nil {
		fmt.Printf("SetConfig failed: %v", err)
		os.Exit(1)
	}

	fmt.Print("Configuration updated successfully.")
}

// handleProber processes the 'prober' command to start the prober and monitor its status.
func handleProber(args []string) {
	// Define any additional flags if needed in the future.
	proberFlags := flag.NewFlagSet("prober", flag.ExitOnError)
	// Currently, no specific flags are defined for 'prober'.
	// You can add flags here if needed.

	// Parse the flags.
	if err := proberFlags.Parse(args); err != nil {
		fmt.Printf("Failed to parse flags: %v", err)
		os.Exit(1)
	}

	// Establish a connection to the server.
	conn, err := grpc.Dial(ServerAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Failed to connect to server at %s: %v", ServerAddress, err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewSavtIpcClient(conn)

	// Create a context that can be canceled.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the prober.
	var jobId string
	job, err := client.Start(ctx, &emptypb.Empty{})
	if err != nil {
		fmt.Printf("Prober is already running. Switching to the last run.\n\n")
		jobId = "0"
		//os.Exit(1)
	} else {
		jobId = job.JobId
	}

	//fmt.Printf("Prober started with Job ID: %s\n\n", job.JobId)

	// WaitGroup to synchronize goroutines.
	var wg sync.WaitGroup
	wg.Add(2) // One for log reader, one for status monitor.

	// Channel to notify when the job reaches a terminal status.
	doneCh := make(chan struct{})

	// Start reading logs.
	go func() {
		defer wg.Done()
		readLogs(ctx, client, jobId)
	}()

	// Start monitoring job status.
	go func() {
		defer wg.Done()
		monitorStatus(ctx, client, jobId, doneCh)
	}()

	// Wait for the job to complete.
	<-doneCh
	//fmt.Print("Prober job has reached a terminal status.")

	// Cancel the context to stop log reading.
	cancel()

	// Wait for goroutines to finish.
	wg.Wait()

	//fmt.Print("Prober completed.")
}

// readLogs streams logs from the server and prints them to the console.
func readLogs(ctx context.Context, client pb.SavtIpcClient, jobId string) {
	// Create a JobIdRequest.
	req := &pb.JobIdRequest{JobId: jobId}

	// Initiate the ReadLog stream.
	stream, err := client.ReadLog(ctx, req)
	if err != nil {
		fmt.Printf("Failed to initiate ReadLog stream: %v", err)
		return
	}

	for {
		// Receive log entries.
		logEntry, err := stream.Recv()
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				//fmt.Print("Log streaming canceled.")
			} else {
				fmt.Printf("Error receiving log entry: %v", err)
			}
			return
		}

		// Print each line in the log entry.
		for _, line := range logEntry.Line {
			fmt.Print(line)
		}
	}
}

// monitorStatus periodically checks the job status until it reaches a terminal state.
func monitorStatus(ctx context.Context, client pb.SavtIpcClient, jobId string, doneCh chan<- struct{}) {
	ticker := time.NewTicker(5 * time.Second) // Adjust the interval as needed.
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			//fmt.Print("Status monitoring canceled.")
			return
		case <-ticker.C:
			statusJob, err := client.GetStatus(ctx, &pb.JobIdRequest{JobId: jobId})
			if err != nil {
				fmt.Printf("GetStatus failed: %v", err)
				continue
			}

			if TerminalStatuses[statusJob.Status] {
				fmt.Printf("Prober run result: %s\n", statusJob.Status.String())
				doneCh <- struct{}{}
				return
			}
		}
	}
}
