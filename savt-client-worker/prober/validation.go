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
	"fmt"
	"net"
	"time"
)

type FinalWrite struct {
	Version  string       `json:"version"`
	IsPublic bool         `json:"isPublic"`
	Sub      []Validation `json:"sub"`
}

type Adapter struct {
	AdapterName string `json:"adapterName"`
	AdapterDesc string `json:"adapterDesc"`
	AdapterID   string `json:"adapterID"`
	AdapterIP   string `json:"adapterIP"`
	AdapterMAC  string `json:"adapterMAC"`
}

type Validation struct {
	ID                        int64                       `json:"id"`
	CreateTime                time.Time                   `json:"createTime"`
	Token                     []byte                      `json:"token"`
	ControllerEndpoint        string                      `json:"controllerEndpoint"`
	ClientPublicIP            string                      `json:"clientPublicIP"`
	ClientIPASN               int                         `json:"clientIPASN"`
	IsNAT                     string                      `json:"isNAT"`
	Adapter                   Adapter                     `json:"adapter"`
	Schedules                 []Schedule                  `json:"schedule"`
	OutbonundReceivedProbes   map[int64]string            `json:"outboundReceivedProbes"`
	InboundReceivedProbes     map[int64]string            `json:"inboundReceivedProbes"`
	PublicOutbound            string                      `json:"outboundPublicStatus"`
	PrivateOutbound           string                      `json:"outboundPrivateStatus"`
	PublicInbound             string                      `json:"inboundPublicStatus"`
	PrivateInbound            string                      `json:"inboundPrivateStatus"`
	Minprefix                 int                         `json:"spoofPrefixLen"`
	CustomizedOutbound        string                      `json:"customizedOutboundStatus"`
	NATtest                   bool                        `json:"nATtest"`
	TracefilterReceivedProbes map[string]map[int64]string `json:"tracefilterReceivedProbes"`
	TracerouteReceivedProbes  map[string]map[int64]string `json:"tracerouteReceivedProbes"`
}

type Probe struct {
	SrcAddr     net.IP `json:"srcAddr"`
	DstAddr     net.IP `json:"dstAddr"`
	SrcPort     int    `json:"srcPort"`
	DstPort     int    `json:"dstPort"`
	Seq         int64  `json:"seq"`
	SpoofIPType string `json:"spoofIpType"`
	SpoofASN    int    `json:"spoofASN"`
	Status      string `json:"status"`
	RevAddr     string `json:"revAddr"`
}

type Schedule struct {
	ID        int64     `json:"id"`
	Class     string    `json:"class"`
	StartTime time.Time `json:"startTime"`
	Probes    []Probe   `json:"probes"`
}

func (tr *Prober) result(validation *Validation, spoofIP string, status *Status) {
	//fmt.Printf("\n*Results*:\n")
	tr.logChan <- "\n*Results*:\n"

	//fmt.Printf("    Outbound:\n")
	tr.logChan <- "    Outbound:\n"
	//fmt.Printf("        Routable: %s. ", validation.PublicOutbound)
	tr.logChan <- fmt.Sprintf("        Routable: %s. ", validation.PublicOutbound)
	minPrefix := validation.Minprefix
	if validation.PublicOutbound == "received" {
		//fmt.Printf("(Largest spoofable neighbor prefix length: %d)\n", validation.Minprefix)
		tr.logChan <- fmt.Sprintf("(Largest spoofable neighbor prefix length: %d)\n", validation.Minprefix)
	} else {
		//fmt.Printf("\n")
		minPrefix = -1
		tr.logChan <- "\n"
	}
	//fmt.Printf("        Private: %s.\n", validation.PrivateOutbound)
	tr.logChan <- fmt.Sprintf("        Private: %s.\n", validation.PrivateOutbound)
	if spoofIP != "" {
		//fmt.Printf("        Customized: %s.\n", validation.CustomizedOutbound)
		tr.logChan <- fmt.Sprintf("        Customized: %s.\n", validation.CustomizedOutbound)
	}
	//fmt.Printf("    Inbound:\n")
	tr.logChan <- "    Inbound:\n"
	//fmt.Printf("        Internal: %s.\n", validation.PublicInbound)
	tr.logChan <- fmt.Sprintf("        Internal: %s.\n", validation.PublicInbound)
	//fmt.Printf("        Private: %s.\n\n", validation.PrivateInbound)
	tr.logChan <- fmt.Sprintf("        Private: %s.\n\n", validation.PrivateInbound)
	if net.ParseIP(validation.ClientPublicIP).To4() != nil {
		status.Ipv4 = ValidationResult{
			ClientAddress:         validation.ClientPublicIP,
			Asn:                   validation.ClientIPASN,
			OutboundPrivate:       validation.PrivateOutbound,
			OutboundRoutable:      validation.PublicOutbound,
			SpoofablePrefixLength: minPrefix,
			InboundPrivate:        validation.PrivateInbound,
			InboundInternal:       validation.PublicInbound,
		}
	} else {
		status.Ipv6 = ValidationResult{
			ClientAddress:         validation.ClientPublicIP,
			Asn:                   validation.ClientIPASN,
			OutboundPrivate:       validation.PrivateOutbound,
			OutboundRoutable:      validation.PublicOutbound,
			SpoofablePrefixLength: minPrefix,
			InboundPrivate:        validation.PrivateInbound,
			InboundInternal:       validation.PublicInbound,
		}
	}
	tr.statusChan <- *status
}

func (tr *Prober) validateIPv4(addr string, token string, spoofIP string, status *Status) (*Validation, error) {
	client := NewClient(addr, token)
	err := client.PostValidation()
	if err != nil {
		tr.logChan <- "You have no public IPv4 network available.\n"
		return nil, fmt.Errorf("you have no public IPv4 network available")
	}
	ControllerEndpoint, err := net.ResolveUDPAddr("udp4", client.Validation.ControllerEndpoint)
	if err != nil {
		return nil, err
	}
	if ControllerEndpoint.IP.To4() == nil {
		tr.logChan <- "Controller's endpoint IP is not a valid IPv4 address.\n"
		return nil, fmt.Errorf("controller's endpoint IP is not a valid IPv4 address")
	}
	dev, err := newIPv4Dev(ControllerEndpoint, client.Validation)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("Interface:\n")
	tr.logChan <- "Interface:\n"
	//fmt.Printf("    Interface Name: %s\n", dev.Net.Name)
	tr.logChan <- fmt.Sprintf("    Interface Name: %s\n", dev.Net.Name)
	//fmt.Printf("    Interface ID: %s\n", dev.Pcap.Name)
	tr.logChan <- fmt.Sprintf("    Interface ID: %s\n", dev.Pcap.Name)
	//fmt.Printf("    Interface Index: %d\n", dev.Net.Index)
	tr.logChan <- fmt.Sprintf("    Interface Index: %d\n", dev.Net.Index)
	//fmt.Printf("    Interface Description: %s\n", dev.Pcap.Description)
	tr.logChan <- fmt.Sprintf("    Interface Description: %s\n", dev.Pcap.Description)
	//fmt.Printf("    Hardware Address: %s\n", dev.Net.HardwareAddr)
	tr.logChan <- fmt.Sprintf("    Hardware Address: %s\n", dev.Net.HardwareAddr)
	//fmt.Printf("    Local IP Address: %s\n", dev.PrivateIP.String())
	tr.logChan <- fmt.Sprintf("    Local IP Address: %s\n", dev.PrivateIP.String())
	publicIP := net.ParseIP(client.Validation.ClientPublicIP)
	if publicIP == nil {
		tr.logChan <- "Cannot get client IPv4 address\n"
		return nil, fmt.Errorf("cannot get client IPv4 address")
	}
	dev.PublicIP = publicIP.To4()
	if dev.PublicIP == nil {
		tr.logChan <- "Public IP is not a valid IPv4 address\n"
		return nil, fmt.Errorf("public IP is not a valid IPv4 address")
	}
	//fmt.Printf("    Public IP Address: %s\n", dev.PublicIP.String())
	tr.logChan <- fmt.Sprintf("    Public IP Address: %s\n", dev.PublicIP.String())
	if client.Validation.ClientIPASN != -1 {
		//fmt.Printf("    ASN of Public IP Address: %d\n", client.Validation.ClientIPASN)
		tr.logChan <- fmt.Sprintf("    ASN of Public IP Address: %d\n", client.Validation.ClientIPASN)
	} else {
		//fmt.Printf("    ASN of Public IP Address: unknown\n")
		tr.logChan <- "    ASN of Public IP Address: unknown\n"
	}

	if dev.PublicIP.String() != dev.PrivateIP.String() {
		client.Validation.IsNAT = "true"
	} else {
		client.Validation.IsNAT = "false"
	}

	status.Ipv4.ClientAddress = client.Validation.ClientPublicIP
	status.Ipv4.Asn = client.Validation.ClientIPASN
	tr.statusChan <- *status

	err = client.GetScheduleByID(spoofIP)
	if err != nil {
		tr.logChan <- "Can't get schedule.\n"
		return nil, fmt.Errorf("can't get schedule")
	}

	fmt.Printf("\n")
	tr.logChan <- "\n"
	for index, schedule := range client.Validation.Schedules {
		//fmt.Printf("#  Schedule[%d]:\n", index)
		tr.logChan <- fmt.Sprintf("#  Schedule[%d]:\n", index)
		//fmt.Printf("#   %s schedule (targets: %d)\n", schedule.Class, len(schedule.Probes))
		tr.logChan <- fmt.Sprintf("#   %s schedule (targets: %d)\n", schedule.Class, len(schedule.Probes))
		//fmt.Printf("#   Earlist start time: %s\n", schedule.StartTime.Format(time.RFC1123))
		tr.logChan <- fmt.Sprintf("#   Earlist start time: %s\n", schedule.StartTime.Format(time.RFC3339))
		for inx, probe := range schedule.Probes {
			var spoofIPType string
			if probe.SpoofIPType == "private" {
				spoofIPType = "private (RFC1918)"
			} else {
				spoofIPType = probe.SpoofIPType
			}
			//fmt.Printf("#    [%d] %s -> %s   spoofIPType: %s\n", inx, probe.SrcAddr.String(), probe.DstAddr.String(), spoofIPType)
			tr.logChan <- fmt.Sprintf("#    [%d] %s -> %s   spoofIPType: %s\n", inx, probe.SrcAddr.String(), probe.DstAddr.String(), spoofIPType)
		}
	}
	//fmt.Printf("\n")
	tr.logChan <- "\n"
	for _, schedule := range client.Validation.Schedules {
		currtentTime := time.Now()
		for currtentTime.Before(schedule.StartTime) {
			time.Sleep(1 * time.Second)
			currtentTime = time.Now()
		}
		if schedule.Class == "SpoofOutbound" {
			status.Tasks = append(status.Tasks, Task{Step: 1, Status: 1})
			status.ProgressBar = 10
			tr.statusChan <- *status
			//fmt.Printf("Testing Outbound IP Spoofing...\n")
			tr.logChan <- "Testing Outbound IP Spoofing...\n"
			err = outboundValidation(client.Validation, &schedule, dev)
			if err != nil {
				status.Tasks[len(status.Tasks)-1].Status = 4
				status.ProgressBar = 15
				tr.statusChan <- *status
				tr.logChan <- fmt.Sprintf("Error: %v", err)
				return nil, err
			} else {
				status.Tasks[len(status.Tasks)-1].Status = 2
				status.ProgressBar = 15
				tr.statusChan <- *status
			}
		}
		if schedule.Class == "SpoofInbound" {
			status.Tasks = append(status.Tasks, Task{Step: 2, Status: 1})
			status.ProgressBar = 20
			tr.statusChan <- *status
			//fmt.Printf("Testing Inbound IP Spoofing...\n")
			tr.logChan <- "Testing Inbound IP Spoofing...\n"
			err = inboundValidation(&schedule, dev, client)
			if err != nil {
				status.Tasks[len(status.Tasks)-1].Status = 4
				status.ProgressBar = 25
				tr.statusChan <- *status
				tr.logChan <- fmt.Sprintf("Error: %v", err)
				return nil, err
			} else {
				status.Tasks[len(status.Tasks)-1].Status = 2
				status.ProgressBar = 25
				tr.statusChan <- *status
			}
		}
		if schedule.Class == "Tracefilter" {
			status.Tasks = append(status.Tasks, Task{Step: 3, Status: 1})
			status.ProgressBar = 30
			tr.statusChan <- *status
			//fmt.Printf("Testing Tracefilter...\n")
			tr.logChan <- "Testing Tracefilter...\n"
			err = tracefilterValidation4(client.Validation.ID, &schedule, dev)
			if err != nil {
				status.Tasks[len(status.Tasks)-1].Status = 4
				status.ProgressBar = 35
				tr.statusChan <- *status
				tr.logChan <- fmt.Sprintf("Error: %v", err)
			} else {
				status.Tasks[len(status.Tasks)-1].Status = 2
				status.ProgressBar = 35
				tr.statusChan <- *status
			}
			time.Sleep(5 * time.Second)
		}
		if schedule.Class == "Traceroute" {
			status.Tasks = append(status.Tasks, Task{Step: 4, Status: 1})
			status.ProgressBar = 40
			tr.statusChan <- *status
			//fmt.Printf("Testing Tracetoute...\n")
			tr.logChan <- "Testing Traceroute...\n"
			err = tracerouteValidation4(&schedule, dev, client.Validation)
			if err != nil {
				status.Tasks[len(status.Tasks)-1].Status = 4
				status.ProgressBar = 45
				tr.statusChan <- *status
				tr.logChan <- fmt.Sprintf("Error: %v", err)
			} else {
				status.Tasks[len(status.Tasks)-1].Status = 2
				status.ProgressBar = 45
				tr.statusChan <- *status
			}
		}
	}

	adapter := Adapter{AdapterName: dev.Net.Name,
		AdapterDesc: dev.Pcap.Description,
		AdapterID:   dev.Pcap.Name,
		AdapterIP:   dev.PrivateIP.String(),
		AdapterMAC:  dev.Net.HardwareAddr.String()}
	client.Validation.Adapter = adapter
	err = client.DeleteValidationByID()
	if err != nil {
		return nil, err
	}

	tr.result(client.Validation, spoofIP, status)

	status.ProgressBar = 50
	tr.statusChan <- *status

	return client.Validation, nil
}

func (tr *Prober) validateIPv6(addr string, token string, spoofIP string, status *Status) (*Validation, error) {
	client := NewClient(addr, token)
	err := client.PostValidation()
	if err != nil {
		tr.logChan <- "You have no public IPv6 network available.\n"
		return nil, fmt.Errorf("you have no public IPv6 network available")
	}

	ControllerEndpoint, err := net.ResolveUDPAddr("udp6", client.Validation.ControllerEndpoint)
	if err != nil {
		return nil, err
	}
	if ControllerEndpoint.IP.To4() != nil {
		tr.logChan <- "Controller's endpoint IP is not a valid IPv6 address.\n"
		return nil, fmt.Errorf("controller's endpoint IP is not a valid IPv6 address")
	}
	dev, err := newIPv6Dev(ControllerEndpoint, client.Validation)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("Interface:\n")
	tr.logChan <- "Interface:\n"
	//fmt.Printf("    Interface Name: %s\n", dev.Net.Name)
	tr.logChan <- fmt.Sprintf("    Interface Name: %s\n", dev.Net.Name)
	//fmt.Printf("    Interface ID: %s\n", dev.Pcap.Name)
	tr.logChan <- fmt.Sprintf("    Interface ID: %s\n", dev.Pcap.Name)
	//fmt.Printf("    Interface Index: %d\n", dev.Net.Index)
	tr.logChan <- fmt.Sprintf("    Interface Index: %d\n", dev.Net.Index)
	//fmt.Printf("    Interface Description: %s\n", dev.Pcap.Description)
	tr.logChan <- fmt.Sprintf("    Interface Description: %s\n", dev.Pcap.Description)
	//fmt.Printf("    Hardware Address: %s\n", dev.Net.HardwareAddr)
	tr.logChan <- fmt.Sprintf("    Hardware Address: %s\n", dev.Net.HardwareAddr)
	//fmt.Printf("    Local IP Address: %s\n", dev.PrivateIP.String())
	tr.logChan <- fmt.Sprintf("    Local IP Address: %s\n", dev.PrivateIP.String())
	publicIP := net.ParseIP(client.Validation.ClientPublicIP)
	if publicIP == nil {
		tr.logChan <- "Cannot get client IPv6 address.\n"
		return nil, fmt.Errorf("cannot get client IPv6 address")
	}
	dev.PublicIP = publicIP.To16()
	if dev.PublicIP == nil {
		tr.logChan <- "Public IP is not a valid IPv6 address.\n"
		return nil, fmt.Errorf("public IP is not a valid IPv6 address")
	}
	//fmt.Printf("    Public IP Address: %s\n", dev.PublicIP.String())
	tr.logChan <- fmt.Sprintf("    Public IP Address: %s\n", dev.PublicIP.String())
	if client.Validation.ClientIPASN != -1 {
		//fmt.Printf("    ASN of Public IP Address: %d\n", client.Validation.ClientIPASN)
		tr.logChan <- fmt.Sprintf("    ASN of Public IP Address: %d\n", client.Validation.ClientIPASN)
	} else {
		//fmt.Printf("    ASN of Public IP Address: unknown.\n")
		tr.logChan <- "    ASN of Public IP Address: unknown.\n"
	}

	if dev.PublicIP.String() != dev.PrivateIP.String() {
		client.Validation.IsNAT = "true"
	} else {
		client.Validation.IsNAT = "false"
	}

	status.Ipv6.ClientAddress = client.Validation.ClientPublicIP
	status.Ipv6.Asn = client.Validation.ClientIPASN
	tr.statusChan <- *status

	err = client.GetScheduleByID(spoofIP)
	if err != nil {
		tr.logChan <- "Can't get schedule.\n"
		return nil, fmt.Errorf("can't get schedule")
	}
	//fmt.Printf("\n")
	tr.logChan <- "\n"

	for index, schedule := range client.Validation.Schedules {
		//fmt.Printf("#  Schedule[%d]:\n", index)
		tr.logChan <- fmt.Sprintf("#  Schedule[%d]:\n", index)
		//fmt.Printf("#   %s schedule (targets: %d)\n", schedule.Class, len(schedule.Probes))
		tr.logChan <- fmt.Sprintf("#   %s schedule (targets: %d)\n", schedule.Class, len(schedule.Probes))
		//fmt.Printf("#   Earlist start time: %s\n", schedule.StartTime.Format(time.RFC1123))
		tr.logChan <- fmt.Sprintf("#   Earlist start time: %s\n", schedule.StartTime.Format(time.RFC3339))
		for inx, probe := range schedule.Probes {
			var spoofIPType string
			if probe.SpoofIPType == "private" {
				spoofIPType = "private (RFC4193)"
			} else {
				spoofIPType = probe.SpoofIPType
			}
			//fmt.Printf("#    [%d] %s -> %s   spoofIPType: %s\n", inx, probe.SrcAddr.String(), probe.DstAddr.String(), spoofIPType)
			tr.logChan <- fmt.Sprintf("#    [%d] %s -> %s   spoofIPType: %s\n", inx, probe.SrcAddr.String(), probe.DstAddr.String(), spoofIPType)
		}
	}
	//fmt.Printf("\n")
	tr.logChan <- "\n"
	for _, schedule := range client.Validation.Schedules {
		currtentTime := time.Now()
		for currtentTime.Before(schedule.StartTime) {
			time.Sleep(1 * time.Second)
			currtentTime = time.Now()
		}
		if schedule.Class == "SpoofOutbound" {
			status.Tasks = append(status.Tasks, Task{Step: 5, Status: 1})
			status.ProgressBar = 60
			tr.statusChan <- *status
			//fmt.Printf("Testing Outbound IP Spoofing...\n")
			tr.logChan <- "Testing Outbound IP Spoofing...\n"
			err = outboundValidation(client.Validation, &schedule, dev)
			if err != nil {
				status.Tasks[len(status.Tasks)-1].Status = 4
				status.ProgressBar = 65
				tr.statusChan <- *status
				tr.logChan <- fmt.Sprintf("Error: %v", err)
				return nil, err
			} else {
				status.Tasks[len(status.Tasks)-1].Status = 2
				status.ProgressBar = 65
				tr.statusChan <- *status
			}
		}
		if schedule.Class == "SpoofInbound" {
			status.Tasks = append(status.Tasks, Task{Step: 6, Status: 1})
			status.ProgressBar = 70
			tr.statusChan <- *status
			//fmt.Printf("Testing Inbound IP Spoofing...\n")
			tr.logChan <- "Testing Inbound IP Spoofing...\n"
			err = inboundValidation(&schedule, dev, client)
			if err != nil {
				status.Tasks[len(status.Tasks)-1].Status = 4
				status.ProgressBar = 75
				tr.statusChan <- *status
				tr.logChan <- fmt.Sprintf("Error: %v", err)
				return nil, err
			} else {
				status.Tasks[len(status.Tasks)-1].Status = 2
				status.ProgressBar = 75
				tr.statusChan <- *status
			}
		}
		if schedule.Class == "Tracefilter" {
			status.Tasks = append(status.Tasks, Task{Step: 7, Status: 1})
			status.ProgressBar = 80
			tr.statusChan <- *status
			//fmt.Printf("Testing Tracefilter...\n")
			tr.logChan <- "Testing Tracefilter...\n"
			err = tracefilterValidation6(client.Validation.ID, &schedule, dev)
			if err != nil {
				status.Tasks[len(status.Tasks)-1].Status = 4
				status.ProgressBar = 85
				tr.statusChan <- *status
				tr.logChan <- fmt.Sprintf("Error: %v", err)
			} else {
				status.Tasks[len(status.Tasks)-1].Status = 2
				status.ProgressBar = 85
				tr.statusChan <- *status
			}
			time.Sleep(5 * time.Second)
		}
		if schedule.Class == "Traceroute" {
			status.Tasks = append(status.Tasks, Task{Step: 8, Status: 1})
			status.ProgressBar = 90
			tr.statusChan <- *status
			//fmt.Printf("Testing Tracetoute...\n")
			tr.logChan <- "Testing Traceroute...\n"
			err = tracerouteValidation6(&schedule, dev, client.Validation)
			if err != nil {
				status.Tasks[len(status.Tasks)-1].Status = 4
				status.ProgressBar = 95
				tr.statusChan <- *status
				tr.logChan <- fmt.Sprintf("Error: %v", err)
			} else {
				status.Tasks[len(status.Tasks)-1].Status = 2
				status.ProgressBar = 95
				tr.statusChan <- *status
			}
		}
	}

	adapter := Adapter{AdapterName: dev.Net.Name,
		AdapterDesc: dev.Pcap.Description,
		AdapterID:   dev.Pcap.Name,
		AdapterIP:   dev.PrivateIP.String(),
		AdapterMAC:  dev.Net.HardwareAddr.String()}
	client.Validation.Adapter = adapter
	err = client.DeleteValidationByID()
	if err != nil {
		return nil, err
	}

	tr.result(client.Validation, spoofIP, status)

	status.ProgressBar = 99
	tr.statusChan <- *status

	return client.Validation, nil
}
