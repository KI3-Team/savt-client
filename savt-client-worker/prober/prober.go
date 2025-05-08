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

package prober

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

type ValidationResult struct {
	ClientAddress         string `json:"clientAddress"`         //IPv4 address, if filled in, display as ""
	Asn                   int    `json:"asn"`                   //IPv4 AS, if filled in, display as ""
	OutboundPrivate       string `json:"outboundPrivate"`       //IPv4 outbound private measurement result, if filled in, display as ""
	OutboundRoutable      string `json:"outboundRoutable"`      //IPv4 outbound routable measurement result, if filled in, display as ""
	SpoofablePrefixLength int    `json:"spoofablePrefixLength"` //IPv4 spoofable prefix length, if filled in, display as ""
	InboundPrivate        string `json:"inboundPrivate"`        //IPv4 inbound private, if filled in, display as ""
	InboundInternal       string `json:"inboundInternal"`       //IPv4 inbound internal, if filled in, display as ""
}

type Task struct {
	Step   int `json:"step"`   //Currently in which step: 1-IPv4 outbound test, 2-IPv4 inbound test, 3-IPv6 outbound test, 4-IPv6 inbound test, 5-complete
	Status int `json:"status"` //Current step status: 0-Not started, 1-In progress, 2-Success, 4-Failure
}

type Status struct {
	StartTime   string           `json:"starTime"` //Measurement start time
	Token       string           `json:"token"`    //Token
	Tasks       []Task           `json:"step"`     //Currently in which step: 1-IPv4 outbound test, 2-IPv4 inbound test, 3-IPv6 outbound test, 4-IPv6 inbound test, 5-complete
	ProgressBar int              `json:"progressBar"`
	Ipv4        ValidationResult `json:"ipv4"`
	Ipv6        ValidationResult `json:"ipv6"`
}

type StatusCallback func(status Status)

type LogCallback func(message string)

type DoneCallback func()

// TestRunner is responsible for managing the execution of test functions
type Prober struct {
	mu             sync.Mutex
	running        bool
	stopChan       chan struct{}
	doneChan       chan struct{}
	logChan        chan string
	statusChan     chan Status
	wg             sync.WaitGroup
	statusCallback StatusCallback
	logCallback    LogCallback
	doneCallback   DoneCallback
}

func NewProber(statusCallback StatusCallback, logCallback LogCallback, doneCallback DoneCallback) *Prober {
	return &Prober{
		logChan:        make(chan string, 100),
		statusChan:     make(chan Status, 10),
		statusCallback: statusCallback,
		logCallback:    logCallback,
		doneCallback:   doneCallback,
	}
}

func (tr *Prober) Start(port string, isPublic bool) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if tr.running {
		return fmt.Errorf("Prober is already running")
	}

	tr.running = true
	tr.stopChan = make(chan struct{})
	tr.doneChan = make(chan struct{})
	tr.wg.Add(1)

	go tr.run(port, isPublic)

	return nil
}

func (tr *Prober) Stop() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if !tr.running {
		return fmt.Errorf("Prober is not running")
	}

	// Send stop signal
	close(tr.stopChan)

	tr.wg.Wait()
	tr.running = false
	tr.stopChan = nil

	return nil
}

func (tr *Prober) IsRunning() bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.running
}

func (tr *Prober) run(port string, isPublic bool) {
	defer tr.wg.Done()

	done := make(chan struct{})

	go func() {
		tr.validate(port, isPublic, "", "", "")
		done <- struct{}{}
	}()

	for {
		select {
		case <-tr.stopChan:
			return
		case logMsg := <-tr.logChan:
			tr.logCallback(logMsg)
		case status := <-tr.statusChan:
			tr.statusCallback(status)
		case <-done:
			tr.mu.Lock()
			tr.running = false
			close(tr.doneChan)
			tr.mu.Unlock()
			tr.doneCallback()
			return
		}
	}
}

func (tr *Prober) Wait() {
	if tr.doneChan == nil {
		// If nil, means it has not been started, return directly
		return
	}
	<-tr.doneChan
}

func (tr *Prober) validate(port string, isPublic bool, token string, spoofIPv4 string, spoofIPv6 string) {
	var status Status
	status.ProgressBar = 0
	status.StartTime = time.Now().Format(time.RFC3339)
	tr.statusChan <- status

	//fmt.Printf("SAV-T Version: %s\n", Version)
	tr.logChan <- fmt.Sprintf("SAV-T Version: %s\n", Version)
	//fmt.Printf("Contact\n")
	tr.logChan <- "Contact\n"
	//fmt.Printf("    Email: ki3contact@163.com\n")
	tr.logChan <- "    Email: ki3contact@163.com\n"
	//fmt.Printf("    Website: https://ki3.org.cn\n")
	tr.logChan <- "    Website: https://ki3.org.cn\n"

	subs := make([]Validation, 0)

	ips4, err := net.LookupIP("v4.sav-t.ki3.org.cn")
	test4cnt := 0
	if err == nil && len(ips4) != 0 {
		for _, ip := range ips4 {
			addr4 := ip.String() + ":" + port
			//addr4 = "202.112.237.201:41452"
			httpAddr4, err := net.ResolveTCPAddr("tcp", addr4)
			if err != nil {
				continue
			}
			if httpAddr4.IP.To4() != nil {
				test4cnt++
				//fmt.Printf("\n------------------------------------------------\n")
				tr.logChan <- "\n------------------------------------------------\n"
				//fmt.Printf("IPv4 SAV Test #%d\n", test4cnt)
				tr.logChan <- fmt.Sprintf("IPv4 SAV Test #%d\n", test4cnt)
				//fmt.Printf("------------------------------------------------\n")
				tr.logChan <- "------------------------------------------------\n"
				sub, err := tr.validateIPv4(addr4, token, spoofIPv4, &status)
				if err != nil {
					//fmt.Println(err)
					tr.logChan <- fmt.Sprintf("Error: %v", err)
					continue
				}
				subs = append(subs, *sub)
			}
		}
	}

	ips6, err := net.LookupIP("v6.sav-t.ki3.org.cn")
	test6cnt := 0
	if err == nil && len(ips6) != 0 {
		for _, ip := range ips6 {
			addr6 := "[" + ip.String() + "]:" + port
			//addr6 = "[2001:da8:24c::7:2]:41452"
			httpAddr6, err := net.ResolveTCPAddr("tcp", addr6)
			if err != nil {
				continue
			}
			if httpAddr6.IP.To16() != nil {
				test6cnt++
				//fmt.Printf("\n------------------------------------------------\n")
				tr.logChan <- "\n------------------------------------------------\n"
				//fmt.Printf("IPv6 SAV Test #%d\n", test6cnt)
				tr.logChan <- fmt.Sprintf("IPv6 SAV Test #%d\n", test6cnt)
				//fmt.Printf("------------------------------------------------\n")
				tr.logChan <- "------------------------------------------------\n"
				sub, err := tr.validateIPv6(addr6, token, spoofIPv6, &status)
				if err != nil {
					//fmt.Println(err)
					tr.logChan <- fmt.Sprintf("Error: %v", err)
					continue
				}
				subs = append(subs, *sub)
			}
		}
	}

	if len(subs) != 0 {
		//fmt.Printf("Token: %s\n\n", hex.EncodeToString(subs[0].Token))
		//fmt.Printf("Sending data...\n")
		tr.logChan <- "Sending data...\n"
		client := resty.New()
		_, err := client.R().
			SetQueryParam("token", token).
			SetBody(&FinalWrite{
				Version:  Version,
				IsPublic: isPublic,
				Sub:      subs,
			}).
			Post(fmt.Sprintf("http://v4.sav-t.ki3.org.cn:%s/api/final", port))
		if err != nil {
			log.Printf("Error sending final result: %v", err)
		}
		//Post("http://202.112.237.201:41452/api/final")

		tr.logChan <- fmt.Sprintf("Token: %s\n\n", hex.EncodeToString(subs[0].Token))
		status.Token = hex.EncodeToString(subs[0].Token)
		status.Tasks = append(status.Tasks, Task{Step: 9, Status: 2})
		status.ProgressBar = 100
		tr.statusChan <- status
		tr.logChan <- "Run successfully!\n"
	}
	time.Sleep(500 * time.Microsecond)
}
