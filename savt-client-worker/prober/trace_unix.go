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

//go:build darwin || unix
// +build darwin unix

package prober

import (
	"fmt"
	"log"
	"net"
	"time"

	inet "savt-client/savt-client-worker/prober/net"

	"github.com/google/gopacket/pcap"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (d UDPv4) SendIPv4Trace() error {
	handle, err := pcap.OpenLive(d.Device.Pcap.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()

	for p := range d.NewIPv4TracePackets() {
		for i := 0; i < 3; i++ {
			err = handle.WritePacketData(p.
				Data())
			if err != nil {
				return fmt.Errorf("failed to send packet: %w", err)
			}
			time.Sleep(d.Delay)
		}
	}
	return nil
}

func (d UDPv6) SendIPv6Trace() error {
	handle, err := pcap.OpenLive(d.Device.Pcap.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()

	for p := range d.NewIPv6TracePackets() {
		for i := 0; i < 3; i++ {
			err = handle.WritePacketData(p.
				Data())
			if err != nil {
				return fmt.Errorf("failed to send packet: %w", err)
			}
			time.Sleep(d.Delay)
		}
	}
	return nil
}

func (d trUDPv4) SendReceiveIPv4Trace() ([]*ProbeUDPv4, []*ProbeResponseUDPv4, error) {
	localAddr, err := inet.GetLocalAddr("udp4", d.Target)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get local address for target %s with network type 'udp4': %w", d.Target, err)
	}
	localUDPAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		return nil, nil, fmt.Errorf("invalid address type for %s: want %T, got %T", localAddr, localUDPAddr, localAddr)
	}
	conn, err := net.ListenPacket("ip4:icmp", localUDPAddr.IP.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ICMPv4 packet listener: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing ICMP connection: %v", err)
		}
	}()
	rconn, err := ipv4.NewRawConn(conn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new RawConn: %w", err)
	}

	numPackets := int(d.NumPaths) * int(d.MaxTTL-d.MinTTL)

	recvErrors := make(chan error)
	recvChan := make(chan []*ProbeResponseUDPv4, 1)
	go func(errch chan error, rc chan []*ProbeResponseUDPv4) {
		howLong := d.Delay*time.Duration(numPackets) + d.Timeout
		received, err := d.ListenFor(rconn, howLong)
		errch <- err
		rc <- received
	}(recvErrors, recvChan)

	sent := make([]*ProbeUDPv4, 0, numPackets)
	for p := range d.NewIPv4TracePackets(localUDPAddr.IP, d.Target) {
		for i := 0; i < 3; i++ {
			if err := rconn.WriteTo(p.Header, p.Payload, nil); err != nil {
				return nil, nil, fmt.Errorf("failed to send IPv4 packet: %w", err)
			}
			time.Sleep(d.Delay)
		}
		ts := time.Now()
		data, err := p.Header.Marshal()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal IPv4 header: %w", err)
		}
		data = append(data, p.Payload...)
		sent = append(sent, &ProbeUDPv4{Data: data, LocalAddr: localUDPAddr.IP, Timestamp: ts})
	}

	if err = <-recvErrors; err != nil {
		return nil, nil, err
	}
	received := <-recvChan
	return sent, received, nil
}

func (d trUDPv4) ListenFor(rconn *ipv4.RawConn, howLong time.Duration) ([]*ProbeResponseUDPv4, error) {
	packets := make([]*ProbeResponseUDPv4, 0)
	deadline := time.Now().Add(howLong)
	for time.Until(deadline) > 0 {
		data := make([]byte, 1024)
		now := time.Now()
		if err := rconn.SetReadDeadline(now.Add(time.Millisecond * 100)); err != nil {
			return nil, fmt.Errorf("failed to set read deadline: %w", err)
		}
		hdr, payload, _, err := rconn.ReadFrom(data)
		receivedAt := time.Now()
		if err != nil {
			if nerr, ok := err.(*net.OpError); ok {
				if nerr.Timeout() {
					continue
				}
				return nil, err
			}
		}
		packets = append(packets, &ProbeResponseUDPv4{
			Header:    hdr,
			Payload:   payload,
			Timestamp: receivedAt,
		})
	}
	return packets, nil
}

func (d trUDPv6) SendReceiveIPv6Trace() ([]*ProbeUDPv6, []*ProbeResponseUDPv6, error) {
	localAddr, err := inet.GetLocalAddr("udp6", d.Target)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get local address for target %s with network type 'udp4': %w", d.Target, err)
	}
	localUDPAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		return nil, nil, fmt.Errorf("invalid address type for %s: want %T, got %T", localAddr, localUDPAddr, localAddr)
	}
	conn, err := net.ListenPacket("udp6", net.JoinHostPort(localUDPAddr.IP.String(), "0"))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create UDPv6 packet listener: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing UDP connection: %v", err)
		}
	}()
	pconn := ipv6.NewPacketConn(conn)

	iconn, err := net.ListenPacket("ip6:ipv6-icmp", localUDPAddr.IP.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ICMPv6 packet listener: %w", err)
	}
	defer func() {
		if err := iconn.Close(); err != nil {
			log.Printf("Error closing ICMP connection: %v", err)
		}
	}()

	numPackets := int(d.NumPaths) * int(d.MaxHopLimit-d.MinHopLimit)

	recvErrors := make(chan error)
	recvChan := make(chan []*ProbeResponseUDPv6, 1)
	go func(errch chan error, rc chan []*ProbeResponseUDPv6) {
		howLong := d.Delay*time.Duration(numPackets) + d.Timeout
		received, err := d.ListenFor(ipv6.NewPacketConn(iconn), howLong)
		errch <- err
		rc <- received
	}(recvErrors, recvChan)

	sent := make([]*ProbeUDPv6, 0, numPackets)
	for p := range d.NewIPv6TracePackets(localUDPAddr.IP, d.Target) {
		cm := ipv6.ControlMessage{
			HopLimit: p.HopLimit,
			Src:      localUDPAddr.IP,
		}
		for i := 0; i < 3; i++ {
			if _, err := pconn.WriteTo(p.Payload, &cm, &net.UDPAddr{IP: d.Target, Port: p.DstPort}); err != nil {
				return nil, nil, fmt.Errorf("WriteTo failed: %w", err)
			}
			time.Sleep(d.Delay)
		}
		ts := time.Now()
		probe := ProbeUDPv6{
			Payload:    append(p.UDPHeader, p.Payload...),
			HopLimit:   p.HopLimit,
			LocalAddr:  p.Src,
			RemoteAddr: d.Target,
			Timestamp:  ts,
		}
		if err := probe.Validate(); err != nil {
			return nil, nil, err
		}
		sent = append(sent, &probe)
	}
	if err = <-recvErrors; err != nil {
		return nil, nil, err
	}
	received := <-recvChan
	return sent, received, nil
}

func (d trUDPv6) ListenFor(conn *ipv6.PacketConn, howLong time.Duration) ([]*ProbeResponseUDPv6, error) {
	packets := make([]*ProbeResponseUDPv6, 0)
	deadline := time.Now().Add(howLong)
	for time.Until(deadline) > 0 {
		data := make([]byte, 4096)
		now := time.Now()
		if err := conn.SetReadDeadline(now.Add(time.Millisecond * 100)); err != nil {
			return nil, fmt.Errorf("failed to set read deadline: %w", err)
		}
		n, _, addr, err := conn.ReadFrom(data)
		receivedAt := time.Now()
		if err != nil {
			if nerr, ok := err.(*net.OpError); ok {
				if nerr.Timeout() {
					continue
				}
				return nil, err
			}
		}
		packets = append(packets, &ProbeResponseUDPv6{
			Data:      data[:n],
			Addr:      (*(addr).(*net.IPAddr)).IP,
			Timestamp: receivedAt,
		})
	}
	return packets, nil
}
