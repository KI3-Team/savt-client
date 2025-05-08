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
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"log"
	"net"
	"time"
)

func inboundValidation(schedule *Schedule, dev *Device, client *Client) error {
	done := make(chan bool)
	if net.ParseIP(client.Validation.ClientPublicIP).To4() != nil {
		go func() {
			validateIPv4Inbound(dev.PrivateIP, schedule.Probes[0].DstPort, client.Validation, schedule, done)
		}()
	} else {
		go func() {
			validateIPv6Inbound(dev.PrivateIP, schedule.Probes[0].DstPort, client.Validation, schedule, done)
		}()
	}
	err := client.StartInboundValidationByID()
	if err != nil {
		return err
	}
	<-done
	return nil
}

func validateIPv4Inbound(privateIPv4 net.IP, port int, validation *Validation, schedule *Schedule, done chan bool) {
	dstAddr := &net.UDPAddr{IP: privateIPv4, Port: port}
	udp, err := net.ListenUDP("udp4", dstAddr)
	if err != nil {
		done <- true
		return
	}
	defer func() {
		if err := udp.Close(); err != nil {
			log.Printf("Error closing UDP connection: %v", err)
		}
	}()
	timeout := time.Now().Add(15 * time.Second)
	buf := make([]byte, 1500)
	for time.Now().Before(timeout) {
		err = udp.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err != nil {
			continue
		}
		n, remoteAddr, err := udp.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if n != 24+sha1.Size {
			continue
		}
		HandleUDPPacket(buf, remoteAddr, validation, schedule)
	}
	done <- true
}

func validateIPv6Inbound(privateIPv6 net.IP, port int, validation *Validation, schedule *Schedule, done chan bool) {
	dstAddr := &net.UDPAddr{IP: privateIPv6, Port: port}
	udp, err := net.ListenUDP("udp6", dstAddr)
	if err != nil {
		done <- true
		return
	}
	defer func() {
		if err := udp.Close(); err != nil {
			log.Printf("Error closing UDP connection: %v", err)
		}
	}()
	timeout := time.Now().Add(15 * time.Second)
	buf := make([]byte, 1500)
	for time.Now().Before(timeout) {
		err = udp.SetReadDeadline(time.Now().Add(1 * time.Second))
		if err != nil {
			continue
		}
		n, remoteAddr, err := udp.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if n != 24+sha1.Size {
			continue
		}
		HandleUDPPacket(buf, remoteAddr, validation, schedule)
	}
	done <- true
}

func HandleUDPPacket(data []byte, srcAddr *net.UDPAddr, validation *Validation, schedule *Schedule) {
	validationId := int64(binary.BigEndian.Uint64(data))
	if validationId != validation.ID {
		return
	}
	scheduleId := int64(binary.BigEndian.Uint64(data[8:16]))
	if scheduleId < 0 || scheduleId >= int64(len(validation.Schedules)) {
		return
	}
	seq := int64(binary.BigEndian.Uint64(data[16:24]))
	if seq < 0 || seq >= int64(len(schedule.Probes)) {
		return
	}
	if !validMAC(data[0:24], data[24:24+sha1.Size], validation.Token) {
		return
	}
	validation.InboundReceivedProbes[seq] = srcAddr.IP.String()
}

func validMAC(message, messageMAC, key []byte) bool {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	valid := hmac.Equal(messageMAC, expectedMAC)
	return valid
}
