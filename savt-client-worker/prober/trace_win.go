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

//go:build windows
// +build windows

package prober

import (
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/ipv4"

	inet "savt-client/savt-client-worker/prober/net"
)

func (d UDPv4) SendIPv4Trace() error {
	handle, err := pcap.OpenLive(d.Device.Pcap.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()

	for p := range d.NewIPv4TracePackets() {
		for i := 0; i < 3; i++ {
			if err := handle.WritePacketData(p.Data()); err != nil {
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
			if err := handle.WritePacketData(p.Data()); err != nil {
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

	numPackets := int(d.NumPaths) * int(d.MaxTTL-d.MinTTL)

	recvErrors := make(chan error)
	recvChan := make(chan []*ProbeResponseUDPv4, 1)
	go func(errch chan error, rc chan []*ProbeResponseUDPv4) {
		howLong := d.Delay*time.Duration(numPackets) + d.Timeout
		received, err := d.listenForPCAPv4(d.Device.Pcap.Name, localUDPAddr.IP, howLong)
		errch <- err
		rc <- received
	}(recvErrors, recvChan)

	handle, err := pcap.OpenLive(d.Device.Pcap.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return nil, nil, err
	}
	defer handle.Close()

	sent := make([]*ProbeUDPv4, 0, numPackets)
	for p := range d.NewIPv4TracePackets(localUDPAddr.IP, d.Target) {
		for i := 0; i < 3; i++ {
			if err := handle.WritePacketData(p.Gopkt.Data()); err != nil {
				return nil, nil, fmt.Errorf("failed to send IPv4 packet (pcap): %w", err)
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

func (d trUDPv4) listenForPCAPv4(iface string, localIP net.IP, howLong time.Duration) ([]*ProbeResponseUDPv4, error) {
	packets := make([]*ProbeResponseUDPv4, 0)
	deadline := time.Now().Add(howLong)

	handle, err := pcap.OpenLive(iface, SnapLen, true, time.Second)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	if err := handle.SetBPFFilter(fmt.Sprintf("icmp and dst host %s", localIP.String())); err != nil {
		return nil, err
	}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for time.Now().Before(deadline) {
		packet, err := packetSource.NextPacket()
		if err != nil {
			continue
		}
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
		if ipLayer == nil || icmpLayer == nil {
			continue
		}
		ip4 := ipLayer.(*layers.IPv4)
		icmp := icmpLayer.(*layers.ICMPv4)
		data := append(icmp.LayerContents(), icmp.LayerPayload()...)
		ipHeader := &ipv4.Header{
			Version:  int(ip4.Version),
			Len:      int(ip4.IHL) * 4,
			Protocol: int(ip4.Protocol),
			TTL:      int(ip4.TTL),
			Src:      ip4.SrcIP,
			Dst:      ip4.DstIP,
			TotalLen: int(ip4.Length),
			ID:       int(ip4.Id),
		}
		packets = append(packets, &ProbeResponseUDPv4{
			Header:    ipHeader,
			Addr:      ip4.SrcIP,
			Payload:   data,
			Timestamp: time.Now(),
		})
	}
	return packets, nil
}

func (d trUDPv6) SendReceiveIPv6Trace() ([]*ProbeUDPv6, []*ProbeResponseUDPv6, error) {
	localAddr, err := inet.GetLocalAddr("udp6", d.Target)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get local address for target %s with network type 'udp6': %w", d.Target, err)
	}
	localUDPAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		return nil, nil, fmt.Errorf("invalid address type for %s: want %T, got %T", localAddr, localUDPAddr, localAddr)
	}

	numPackets := int(d.NumPaths) * int(d.MaxHopLimit-d.MinHopLimit)

	recvErrors := make(chan error)
	recvChan := make(chan []*ProbeResponseUDPv6, 1)
	go func(errch chan error, rc chan []*ProbeResponseUDPv6) {
		howLong := d.Delay*time.Duration(numPackets) + d.Timeout
		received, err := d.listenForPCAPv6(d.Device.Pcap.Name, localUDPAddr.IP, howLong)
		errch <- err
		rc <- received
	}(recvErrors, recvChan)

	handle, err := pcap.OpenLive(d.Device.Pcap.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return nil, nil, err
	}
	defer handle.Close()

	sent := make([]*ProbeUDPv6, 0, numPackets)
	for p := range d.NewIPv6TracePackets(localUDPAddr.IP, d.Target) {
		for i := 0; i < 3; i++ {
			if err := handle.WritePacketData(p.Gopkt.Data()); err != nil {
				return nil, nil, fmt.Errorf("failed to send IPv6 packet (pcap): %w", err)
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

func (d trUDPv6) listenForPCAPv6(iface string, localIP net.IP, howLong time.Duration) ([]*ProbeResponseUDPv6, error) {
	packets := make([]*ProbeResponseUDPv6, 0)
	deadline := time.Now().Add(howLong)

	handle, err := pcap.OpenLive(iface, SnapLen, true, time.Second)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	if err := handle.SetBPFFilter(fmt.Sprintf("icmp6 and dst host %s", localIP.String())); err != nil {
		return nil, err
	}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for time.Now().Before(deadline) {
		packet, err := packetSource.NextPacket()
		if err != nil {
			continue
		}
		ipLayer := packet.Layer(layers.LayerTypeIPv6)
		icmpLayer := packet.Layer(layers.LayerTypeICMPv6)
		if ipLayer == nil || icmpLayer == nil {
			continue
		}
		ip6 := ipLayer.(*layers.IPv6)
		icmp := icmpLayer.(*layers.ICMPv6)
		data := append(icmp.LayerContents(), icmp.LayerPayload()...)

		packets = append(packets, &ProbeResponseUDPv6{
			Data:      data,
			Addr:      ip6.SrcIP,
			Timestamp: time.Now(),
		})
	}
	return packets, nil
}
