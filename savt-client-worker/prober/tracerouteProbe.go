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
	"errors"
	"fmt"
	"net"
	"time"

	inet "savt-client/savt-client-worker/prober/net"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type ProbeUDPv4 struct {
	Data      []byte
	ip        *ipv4.Header
	udp       *inet.UDP
	payload   []byte
	Timestamp time.Time
	LocalAddr net.IP
}

func (p *ProbeUDPv4) Validate() error {
	if p.ip == nil {
		hdr, err := ipv4.ParseHeader(p.Data)
		if err != nil {
			return err
		}
		p.ip = hdr
	}
	if p.ip.Protocol != int(inet.ProtoUDP) {
		return fmt.Errorf("IP payload is not UDP, expected type %d, got %d", inet.ProtoUDP, p.ip.Protocol)
	}
	p.payload = p.Data[p.ip.Len:]
	if len(p.payload) == 0 {
		return errors.New("IP layer has no payload")
	}
	udp, err := inet.NewUDP(p.Data[p.ip.Len:])
	if err != nil {
		return fmt.Errorf("failed to parse UDP header: %w", err)
	}
	p.udp = udp
	p.payload = p.Data[p.ip.Len+inet.UDPHeaderLen:]
	return nil
}

func (p ProbeUDPv4) IP() *ipv4.Header {
	return p.ip
}

func (p ProbeUDPv4) UDP() *inet.UDP {
	return p.udp
}

type ProbeResponseUDPv4 struct {
	Header    *ipv4.Header
	Payload   []byte
	Addr      net.IP
	Timestamp time.Time

	icmp         *inet.ICMP
	innerIP      *ipv4.Header
	innerUDP     *inet.UDP
	innerPayload []byte
}

func (pr *ProbeResponseUDPv4) Validate() error {
	if pr.icmp == nil {
		icmp, err := inet.NewICMP(pr.Payload)
		if err != nil {
			return err
		}
		pr.icmp = icmp
	}
	if len(pr.icmp.Payload) == 0 {
		return errors.New("ICMP layer has no payload")
	}
	ip, err := ipv4.ParseHeader(pr.icmp.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse inner IPv4 header: %w", err)
	}
	pr.innerIP = ip
	payload := pr.icmp.Payload[ip.Len:]
	if len(payload) == 0 {
		return errors.New("inner IP layer has no payload")
	}
	if ip.Protocol != int(inet.ProtoUDP) {
		return fmt.Errorf("inner IP payload is not UDP, want protocol %d, got %d", inet.ProtoUDP, ip.Protocol)
	}
	udp, err := inet.NewUDP(payload)
	if err != nil {
		return fmt.Errorf("failed to decode inner UDP header: %w", err)
	}
	pr.innerUDP = udp
	pr.innerPayload = payload[inet.UDPHeaderLen:]
	return nil
}

func (pr *ProbeResponseUDPv4) ICMP() *inet.ICMP {
	return pr.icmp
}

func (pr *ProbeResponseUDPv4) InnerIP() *ipv4.Header {
	return pr.innerIP
}

func (pr *ProbeResponseUDPv4) InnerUDP() *inet.UDP {
	return pr.innerUDP
}

func (pr *ProbeResponseUDPv4) Matches(p *ProbeUDPv4) bool {
	if p == nil {
		return false
	}
	if pr.icmp.Type != inet.ICMPTimeExceeded && pr.icmp.Type != inet.ICMPDestUnreachable ||
		(pr.icmp.Type == inet.ICMPDestUnreachable && pr.icmp.Code != 3) {
		return false
	}
	if !pr.InnerIP().Dst.To4().Equal(p.ip.Dst.To4()) {
		return false
	}
	if p.UDP().Src != pr.InnerUDP().Src || p.UDP().Dst != pr.InnerUDP().Dst {
		// source and destination ports do not match
		return false
	}
	if pr.InnerIP().ID != p.ip.ID {
		// the two packets do not belong to the same flow
		return false
	}
	return true
}

type ProbeUDPv6 struct {
	Payload               []byte
	HopLimit              int
	Timestamp             time.Time
	LocalAddr, RemoteAddr net.IP
	udp                   *inet.UDP
}

func (p *ProbeUDPv6) Validate() error {
	if p.udp == nil {
		udp, err := inet.NewUDP(p.Payload)
		if err != nil {
			return err
		}
		p.udp = udp
	}
	return nil
}

func (p ProbeUDPv6) UDP() *inet.UDP {
	return p.udp
}

type ProbeResponseUDPv6 struct {
	Data      []byte
	Timestamp time.Time
	Addr      net.IP
	icmp      *inet.ICMPv6
	innerIPv6 *ipv6.Header
	innerUDP  *inet.UDP
	payload   []byte
}

func (pr *ProbeResponseUDPv6) Validate() error {
	if pr.icmp != nil && pr.innerIPv6 != nil && pr.innerUDP != nil {
		return nil
	}
	icmp, err := inet.NewICMPv6(pr.Data)
	if err != nil {
		return fmt.Errorf("failed to decode ICMPv6: %w", err)
	}
	pr.icmp = icmp
	ip, err := ipv6.ParseHeader(pr.Data[inet.ICMPv6HeaderLen:])
	if err != nil {
		return fmt.Errorf("failed to decode inner IPv6: %w", err)
	}
	pr.innerIPv6 = ip
	udp, err := inet.NewUDP(pr.Data[inet.ICMPv6HeaderLen+inet.IPv6HeaderLen:])
	if err != nil {
		return fmt.Errorf("failed to decode inner UDP: %w", err)
	}
	pr.innerUDP = udp
	pr.payload = pr.Data[inet.ICMPv6HeaderLen+inet.IPv6HeaderLen+inet.UDPHeaderLen:]
	return nil
}

func (pr ProbeResponseUDPv6) Matches(p *ProbeUDPv6) bool {
	if p == nil {
		return false
	}
	icmp := pr.ICMPv6()
	if icmp.Type != inet.ICMPv6TypeTimeExceeded &&
		(icmp.Type != inet.ICMPv6TypeDestUnreachable || icmp.Code != inet.ICMPv6CodePortUnreachable) {
		// we want time-exceeded or port-unreachable
		return false
	}
	// TODO check that To16() is the right thing to call here
	if !pr.InnerIPv6().Dst.To16().Equal(p.RemoteAddr.To16()) {
		// this is not a response to any of our probes, discard it
		return false
	}
	innerUDP := pr.InnerUDP()
	if p.UDP().Dst != innerUDP.Dst {
		// this is not our packet
		return false
	}
	if pr.InnerIPv6().PayloadLen != len(p.Payload) {
		// different payload length, not our packet
		// NOTE: here I am using pr.InnerIPv6().PayloadLen instead of len(pr.payload)
		// because the responding hop might use an RFC4884 multi-part ICMPv6 message,
		// which has extra data at the end of time-exceeded and destination-unreachable
		// messages
		return false
	}
	return true
}

func (pr *ProbeResponseUDPv6) ICMPv6() *inet.ICMPv6 {
	return pr.icmp
}

func (pr *ProbeResponseUDPv6) InnerIPv6() *ipv6.Header {
	return pr.innerIPv6
}

func (pr *ProbeResponseUDPv6) InnerUDP() *inet.UDP {
	return pr.innerUDP
}

func (pr *ProbeResponseUDPv6) InnerPayload() []byte {
	return pr.payload
}
