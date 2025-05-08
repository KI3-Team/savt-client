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
	"crypto/rand"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type UDPv4 struct {
	ValidationId uint32
	Source       net.IP
	Target       net.IP
	SrcPort      uint16
	DstPort      uint16
	MinTTL       uint8
	MaxTTL       uint8
	Delay        time.Duration
	Timeout      time.Duration
	Device       *Device
}

func tracefilterValidation4(vid int64, schedule *Schedule, dev *Device) error {
	for _, probe := range schedule.Probes {
		dt := &UDPv4{
			ValidationId: uint32(vid),
			Source:       probe.SrcAddr,
			Target:       probe.DstAddr,
			SrcPort:      uint16(probe.SrcPort),
			DstPort:      uint16(probe.DstPort),
			MinTTL:       1,
			MaxTTL:       30,
			Delay:        time.Duration(50) * time.Millisecond,
			Timeout:      10 * time.Second,
			Device:       dev,
		}
		err := dt.SendIPv4Trace()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d UDPv4) NewIPv4TracePacket(ttl uint8) (gopacket.Packet, error) {
	payloadLength := ttl
	payload := make([]byte, payloadLength)
	// Generate random bytes
	_, err := rand.Read(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random payload: %w", err)
	}

	high16 := uint16(d.ValidationId >> 16)
	low16 := uint16(d.ValidationId & 0xFFFF)

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ipv4 := &layers.IPv4{
		Version:  4,
		Id:       high16,
		TTL:      ttl,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    d.Source.To4(),
		DstIP:    d.Target.To4(),
		Flags:    layers.IPv4DontFragment,
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(d.SrcPort),
		DstPort: layers.UDPPort(low16),
	}
	err = udp.SetNetworkLayerForChecksum(ipv4)
	if err != nil {
		return nil, err
	}
	err = gopacket.SerializeLayers(buf, opts,
		gopacket.Payload(d.Device.LinkLayerContents),
		ipv4,
		udp,
		gopacket.Payload(payload),
	)
	if err != nil {
		return nil, err
	}
	return gopacket.NewPacket(buf.Bytes(), d.Device.LinkLayerType, gopacket.Default), nil
}

func (d UDPv4) NewIPv4TracePackets() <-chan gopacket.Packet {
	numPackets := int(d.MaxTTL - d.MinTTL)
	ret := make(chan gopacket.Packet, numPackets)

	go func() {
		for ttl := d.MinTTL; ttl <= d.MaxTTL; ttl++ {
			pkt, err := d.NewIPv4TracePacket(ttl)
			if err == nil {
				ret <- pkt
			}
		}
		close(ret)
	}()
	return ret
}

type UDPv6 struct {
	ValidationId uint32
	Source       net.IP
	Target       net.IP
	SrcPort      uint16
	DstPort      uint16
	MinHopLimit  uint8
	MaxHopLimit  uint8
	Delay        time.Duration
	Timeout      time.Duration
	Device       *Device
}

func tracefilterValidation6(vid int64, schedule *Schedule, dev *Device) error {
	for _, probe := range schedule.Probes {
		dt := &UDPv6{
			ValidationId: uint32(vid),
			Source:       probe.SrcAddr,
			Target:       probe.DstAddr,
			SrcPort:      uint16(probe.SrcPort),
			DstPort:      uint16(probe.DstPort),
			MinHopLimit:  1,
			MaxHopLimit:  30,
			Delay:        time.Duration(50) * time.Millisecond,
			Timeout:      10 * time.Second,
			Device:       dev,
		}
		err := dt.SendIPv6Trace()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d UDPv6) NewIPv6TracePacket(ttl uint8) (gopacket.Packet, error) {
	payloadLength := ttl
	payload := make([]byte, payloadLength)
	// Generate random bytes
	_, err := rand.Read(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random payload: %w", err)
	}

	high16 := uint16(d.ValidationId >> 16)
	low16 := uint16(d.ValidationId & 0xFFFF)

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ipv6 := &layers.IPv6{
		Version:    6,
		NextHeader: layers.IPProtocolUDP,
		HopLimit:   ttl,
		SrcIP:      d.Source.To16(),
		DstIP:      d.Target.To16(),
		FlowLabel:  uint32(high16),
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(d.SrcPort),
		DstPort: layers.UDPPort(low16),
	}
	err = udp.SetNetworkLayerForChecksum(ipv6)
	if err != nil {
		return nil, err
	}
	err = gopacket.SerializeLayers(buf, opts,
		gopacket.Payload(d.Device.LinkLayerContents),
		ipv6,
		udp,
		gopacket.Payload(payload),
	)
	if err != nil {
		return nil, err
	}
	return gopacket.NewPacket(buf.Bytes(), d.Device.LinkLayerType, gopacket.Default), nil
}

func (d UDPv6) NewIPv6TracePackets() <-chan gopacket.Packet {
	numPackets := int(d.MaxHopLimit - d.MinHopLimit)
	ret := make(chan gopacket.Packet, numPackets)

	go func() {
		for hl := d.MinHopLimit; hl <= d.MaxHopLimit; hl++ {
			pkt, err := d.NewIPv6TracePacket(hl)
			if err == nil {
				ret <- pkt
			}
		}
		close(ret)
	}()
	return ret
}
