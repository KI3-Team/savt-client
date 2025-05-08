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
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "savt-client/savt-client-api/savt"
)

type Client struct {
	Client    *resty.Client
	Exception *APIException
	Addr      string
	Base      string
	Token     string
}

type TimeCounter struct {
	Complete        int
	Incomplete      int
	CheckNetwork    int
	WaitAfterChange int
	RetryLimit      int
}

func NewClient(addr string, token string) *Client {
	base := "http://" + addr
	client := resty.New()
	// client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	return &Client{
		Client:    client,
		Exception: &APIException{},
		Addr:      addr,
		Base:      base,
		Token:     token,
	}
}

func (c *Client) GetClientIP() (string, error) {
	var clientIP string
	resp, err := c.Client.R().
		SetResult(&clientIP).
		SetError(c.Exception).
		SetQueryParam("token", c.Token).
		Post(c.Base + "/api/validation/check")
	if err != nil {
		return "", err
	}
	if resp.IsError() {
		return "", fmt.Errorf(c.Exception.Message)
	}
	return clientIP, nil
}

func checkNetwork(token string, oldIPv4 string, oldIPv6 string) (bool, string, string) {
	var clientIPv4, clientIPv6 string
	ips4, err := net.LookupIP("v4.sav-t.ki3.org.cn")
	if err == nil && len(ips4) != 0 {
		for _, ip := range ips4 {
			addr4 := ip.String() + ":" + remotePort
			httpAddr4, err := net.ResolveTCPAddr("tcp", addr4)
			if err != nil {
				continue
			}
			if httpAddr4.IP.To4() != nil {
				client := NewClient(addr4, token)
				clientIPv4, err = client.GetClientIP()
				if err != nil {
					continue
				} else {
					break
				}
			}
		}
	}
	ips6, err := net.LookupIP("v6.sav-t.ki3.org.cn")
	if err == nil && len(ips6) != 0 {
		for _, ip := range ips6 {
			addr6 := "[" + ip.String() + "]:" + remotePort
			httpAddr6, err := net.ResolveTCPAddr("tcp", addr6)
			if err != nil {
				continue
			}
			if httpAddr6.IP.To16() != nil {
				client := NewClient(addr6, token)
				clientIPv6, err = client.GetClientIP()
				if err != nil {
					continue
				} else {
					break
				}
			}
		}
	}
	flag := (oldIPv4 != "" && clientIPv4 != oldIPv4) || (oldIPv6 != "" && clientIPv6 != oldIPv6)
	return flag, clientIPv4, clientIPv6
}

func (s *server) runScheduler() {
	timeCounter := &TimeCounter{
		Complete:        int(s.config.CompleteInterval),
		Incomplete:      -1,
		CheckNetwork:    0,
		WaitAfterChange: -1,
		RetryLimit:      -1,
	}

	clientIPv4 := ""
	clientIPv6 := ""
	var isLastTestComplete bool
	isChange := false
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.confMu.Lock()
			cf := pb.Config{
				CompleteInterval:        int32(s.config.CompleteInterval),
				IncompleteRetryInterval: int32(s.config.IncompleteRetryInterval),
				CheckNetworkInterval:    int32(s.config.CheckNetworkInterval),
				WaitAfterChangeInterval: int32(s.config.WaitAfterChangeInterval),
				RetryLimit:              int32(s.config.RetryLimit),
				IsPublic:                s.config.IsPublic,
				EnableScheduler:         s.config.EnableScheduler,
			}
			s.confMu.Unlock()
			if timeCounter.CheckNetwork == 0 {
				isChange, clientIPv4, clientIPv6 = checkNetwork("", clientIPv4, clientIPv6)
				if isChange {
					timeCounter.WaitAfterChange = int(cf.WaitAfterChangeInterval)
				}
				timeCounter.CheckNetwork = int(cf.CheckNetworkInterval)
			}
			//wait
			if timeCounter.WaitAfterChange == 0 {
				timeCounter.WaitAfterChange = -1
				timeCounter.Complete = 0
			}
			//complete
			if timeCounter.Complete == 0 {
				isLastTestComplete, _ = s.RunScheduledProbe()
				if isLastTestComplete {
					timeCounter.Complete = int(cf.CompleteInterval)
					timeCounter.Incomplete = -1
					timeCounter.RetryLimit = -1
					timeCounter.WaitAfterChange = -1
				} else {
					timeCounter.Complete = -1
					timeCounter.Incomplete = int(cf.IncompleteRetryInterval)
					timeCounter.RetryLimit = int(cf.RetryLimit)
				}
			}
			//incomplete
			if timeCounter.Incomplete == 0 {
				if timeCounter.RetryLimit == 0 {
					timeCounter.Complete = int(cf.CompleteInterval)
					timeCounter.Incomplete = -1
					timeCounter.RetryLimit = -1
				} else {
					isLastTestComplete, _ = s.RunScheduledProbe()
					if isLastTestComplete {
						timeCounter.Complete = int(cf.CompleteInterval)
						timeCounter.Incomplete = -1
						timeCounter.RetryLimit = -1
						timeCounter.WaitAfterChange = -1
					} else {
						timeCounter.Complete = -1
						timeCounter.Incomplete = int(cf.IncompleteRetryInterval)
						timeCounter.RetryLimit--
					}
				}
			}
			timeCounter.CheckNetwork--
			if timeCounter.Complete != -1 {
				timeCounter.Complete--
			}
			if timeCounter.Incomplete != -1 {
				timeCounter.Incomplete--
			}
			if timeCounter.WaitAfterChange != -1 {
				timeCounter.WaitAfterChange--
			}
		case <-s.schedulerStopCh:
			return
		}
	}
}

func (s *server) RunScheduledProbe() (bool, error) {
	s.scheMu.Lock()
	defer s.scheMu.Unlock()

	/*
		for s.scheduledProber.IsRunning() {
			time.Sleep(1000 * time.Millisecond)
		}
	*/

	s.shJobMu.Lock()
	currentTime := time.Now().Format(time.RFC1123)
	sanitizedTime := strings.ReplaceAll(currentTime, ",", "")
	sanitizedTime = strings.ReplaceAll(sanitizedTime, " ", "_")
	sanitizedTime = strings.ReplaceAll(sanitizedTime, ":", "-")
	s.shJob = pb.Job{
		JobId:       sanitizedTime,
		Status:      pb.Status_RUNNING,
		StartTime:   timestamppb.Now(),
		Token:       "",
		Tasks:       []*pb.Task{},
		ProgressBar: 0,
		Ipv4:        &pb.MeasurementResult{},
		Ipv6:        &pb.MeasurementResult{},
		JobType:     pb.Jobtype_SCHEDULED,
	}
	s.shJobMu.Unlock()

	s.confMu.Lock()
	if err := s.scheduledProber.Start(remotePort, s.config.IsPublic); err != nil {
		s.confMu.Unlock()
		return false, err
	}
	s.confMu.Unlock()
	s.scheduledProber.Wait()

	return true, nil
}
