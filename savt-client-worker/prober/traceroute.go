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
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"

	inet "savt-client/savt-client-worker/prober/net"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"golang.org/x/net/ipv4"
)

type trUDPv4 struct {
	Target     net.IP
	SrcPort    uint16
	DstPort    uint16
	UseSrcPort bool
	NumPaths   uint16
	MinTTL     uint8
	MaxTTL     uint8
	Delay      time.Duration
	Timeout    time.Duration
	BrokenNAT  bool
	Device     *Device
	Validation *Validation
}

func tracerouteValidation4(schedule *Schedule, dev *Device, validation *Validation) error {
	for _, probe := range schedule.Probes {
		dt := &trUDPv4{
			Target:     probe.DstAddr,
			SrcPort:    uint16(probe.SrcPort),
			DstPort:    uint16(probe.DstPort),
			UseSrcPort: false,
			NumPaths:   1,
			MinTTL:     1,
			MaxTTL:     30,
			Delay:      time.Duration(50) * time.Millisecond,
			Timeout:    10 * time.Second,
			BrokenNAT:  false,
			Device:     dev,
			Validation: validation,
		}
		if err := dt.Validate(); err != nil {
			return err
		}
		sent, received, err := dt.SendReceiveIPv4Trace()
		if err != nil {
			return err
		}
		dt.Match(sent, received)
	}
	return nil
}

func (d *trUDPv4) Validate() error {
	if d.Target.To4() == nil {
		return errors.New("invalid IPv4 address")
	}
	if d.NumPaths == 0 {
		return errors.New("number of paths must be a positive integer")
	}
	if d.UseSrcPort {
		if int(d.SrcPort)+int(d.NumPaths) > 0xffff {
			return errors.New("source port plus number of paths cannot exceed 65535")
		}
	} else {
		if int(d.DstPort)+int(d.NumPaths) > 0xffff {
			return errors.New("destination port plus number of paths cannot exceed 65535")
		}
	}
	if d.MinTTL == 0 {
		return errors.New("minimum TTL must be a positive integer")
	}
	if d.MaxTTL < d.MinTTL {
		return errors.New("invalid maximum TTL, must be greater or equal than minimum TTL")
	}
	if d.Delay < 1 {
		return errors.New("invalid delay, must be positive")
	}
	return nil
}

func (d trUDPv4) NewIPv4TracePacket(ttl uint8, src, dst net.IP, srcport, dstport uint16) (*ipv4.Header, []byte, gopacket.Packet, error) {
	payload := []byte("NSMNC\x00\x00\x00")
	id := uint16(ttl)
	if d.UseSrcPort {
		id += d.SrcPort
	} else {
		id += d.DstPort
	}
	payload[6] = byte((id >> 8) & 0xff)
	payload[7] = byte(id & 0xff)

	iph := ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		Flags:    ipv4.DontFragment,
		TTL:      int(ttl),
		Protocol: int(inet.ProtoUDP),
		Src:      src,
		Dst:      dst,
	}
	pseudoheader, err := inet.IPv4HeaderToPseudoHeader(&iph, inet.UDPHeaderLen+len(payload))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to compute IPv4 pseudoheader: %w", err)
	}
	udp := inet.UDP{
		Src:          srcport,
		Dst:          dstport,
		Len:          uint16(inet.UDPHeaderLen + len(payload)),
		Payload:      payload,
		PseudoHeader: pseudoheader,
	}
	udpBytes, err := udp.MarshalBinary()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal UDP header: %w", err)
	}
	iph.ID = int(udp.Csum)
	iph.TotalLen = ipv4.HeaderLen + len(udpBytes)

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ipv4pkt := &layers.IPv4{
		Version:  4,
		Id:       uint16(iph.ID),
		TTL:      ttl,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    src,
		DstIP:    dst,
		Flags:    layers.IPv4DontFragment,
	}
	udppkt := &layers.UDP{
		SrcPort: layers.UDPPort(srcport),
		DstPort: layers.UDPPort(dstport),
	}
	err = udppkt.SetNetworkLayerForChecksum(ipv4pkt)
	if err != nil {
		return nil, nil, nil, err
	}
	err = gopacket.SerializeLayers(buf, opts,
		gopacket.Payload(d.Device.LinkLayerContents),
		ipv4pkt,
		udppkt,
		gopacket.Payload(payload),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return &iph, udpBytes, gopacket.NewPacket(buf.Bytes(), d.Device.LinkLayerType, gopacket.Default), nil
}

type pkt struct {
	Header  *ipv4.Header
	Payload []byte
	Port    int
	Gopkt   gopacket.Packet
}

func (d trUDPv4) NewIPv4TracePackets(src, dst net.IP) <-chan pkt {
	numPackets := int(d.NumPaths) * int(d.MaxTTL-d.MinTTL)
	ret := make(chan pkt, numPackets)

	go func() {
		var (
			srcPort, dstPort, basePort uint16
		)
		if d.UseSrcPort {
			basePort = d.SrcPort
		} else {
			basePort = d.DstPort
		}
		for ttl := d.MinTTL; ttl <= d.MaxTTL; ttl++ {
			for port := basePort; port < basePort+d.NumPaths; port++ {
				if d.UseSrcPort {
					srcPort = port
					dstPort = d.DstPort
				} else {
					srcPort = d.SrcPort
					dstPort = port
				}
				iph, payload, gopkt, err := d.NewIPv4TracePacket(ttl, src, dst, srcPort, dstPort)
				if err == nil {
					ret <- pkt{Header: iph, Payload: payload, Port: int(dstPort), Gopkt: gopkt}
				}
			}
		}
		close(ret)
	}()
	return ret
}

func (d trUDPv4) Match(sent []*ProbeUDPv4, received []*ProbeResponseUDPv4) {
	for _, spu := range sent {
		if err := spu.Validate(); err != nil {
			continue
		}
		//sentIP := spu.IP()
		for _, rpu := range received {
			if err := rpu.Validate(); err != nil {
				continue
			}
			if !rpu.Matches(spu) {
				continue
			}
			if _, exists := d.Validation.TracerouteReceivedProbes[spu.ip.Dst.String()]; !exists {
				d.Validation.TracerouteReceivedProbes[spu.ip.Dst.String()] = make(map[int64]string)
			}
			d.Validation.TracerouteReceivedProbes[spu.ip.Dst.String()][int64(spu.ip.TTL)] = rpu.Header.Src.String()
			break
		}
	}
}

type trUDPv6 struct {
	Target      net.IP
	SrcPort     uint16
	DstPort     uint16
	UseSrcPort  bool
	NumPaths    uint16
	MinHopLimit uint8
	MaxHopLimit uint8
	Delay       time.Duration
	Timeout     time.Duration
	BrokenNAT   bool
	Device      *Device
	Validation  *Validation
}

func tracerouteValidation6(schedule *Schedule, dev *Device, validation *Validation) error {
	for _, probe := range schedule.Probes {
		dt := &trUDPv6{
			Target:      probe.DstAddr,
			SrcPort:     uint16(probe.SrcPort),
			DstPort:     uint16(probe.DstPort),
			UseSrcPort:  false,
			NumPaths:    1,
			MinHopLimit: 1,
			MaxHopLimit: 30,
			Delay:       time.Duration(50) * time.Millisecond,
			Timeout:     10 * time.Second,
			BrokenNAT:   false,
			Device:      dev,
			Validation:  validation,
		}
		if err := dt.Validate(); err != nil {
			return err
		}
		sent, received, err := dt.SendReceiveIPv6Trace()
		if err != nil {
			return err
		}
		dt.Match(sent, received)
	}
	return nil
}

func (d *trUDPv6) Validate() error {
	if d.Target.To16() == nil {
		return errors.New("invalid IPv6 address")
	}
	if d.UseSrcPort {
		if int(d.SrcPort)+int(d.NumPaths) > 0xffff {
			return errors.New("source port plus number of paths cannot exceed 65535")
		}
	} else {
		if int(d.DstPort)+int(d.NumPaths) > 0xffff {
			return errors.New("destination port plus number of paths cannot exceed 65535")
		}
	}
	if d.MaxHopLimit < d.MinHopLimit {
		return errors.New("invalid maximum Hop Limit, must be greater or equal than minimum Hop Limit")
	}
	if d.Delay < 1 {
		return errors.New("invalid delay, must be positive")
	}
	return nil
}

type pkt6 struct {
	UDPHeader, Payload []byte
	Src                net.IP
	SrcPort, DstPort   int
	HopLimit           int
	Gopkt              gopacket.Packet
}

func (d trUDPv6) NewIPv6TracePacket(hl uint8, src, dst net.IP, srcport, dstport uint16) ([]byte, []byte, gopacket.Packet, error) {
	plen := 10 + int(hl)
	magic := []byte("NSMNC")
	payload := bytes.Repeat(magic, plen/len(magic)+1)[:plen]

	udph := inet.UDP{
		Src: srcport,
		Dst: dstport,
		Len: uint16(inet.UDPHeaderLen + len(payload)),
	}
	udpb, err := udph.MarshalBinary()
	if err != nil {
		return nil, nil, nil, err
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ipv6 := &layers.IPv6{
		Version:    6,
		NextHeader: layers.IPProtocolUDP,
		HopLimit:   hl,
		SrcIP:      src.To16(),
		DstIP:      dst.To16(),
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(srcport),
		DstPort: layers.UDPPort(dstport),
	}
	err = udp.SetNetworkLayerForChecksum(ipv6)
	if err != nil {
		return nil, nil, nil, err
	}
	err = gopacket.SerializeLayers(buf, opts,
		gopacket.Payload(d.Device.LinkLayerContents),
		ipv6,
		udp,
		gopacket.Payload(payload),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return udpb, payload, gopacket.NewPacket(buf.Bytes(), d.Device.LinkLayerType, gopacket.Default), nil
}

func (d trUDPv6) NewIPv6TracePackets(src, dst net.IP) <-chan pkt6 {
	numPackets := int(d.NumPaths) * int(d.MaxHopLimit-d.MinHopLimit)
	ret := make(chan pkt6, numPackets)

	go func() {
		var srcPort, dstPort, basePort uint16
		if d.UseSrcPort {
			basePort = d.SrcPort
		} else {
			basePort = d.DstPort
		}
		for hl := d.MinHopLimit; hl <= d.MaxHopLimit; hl++ {
			for port := basePort; port < basePort+d.NumPaths; port++ {
				if d.UseSrcPort {
					srcPort = port
					dstPort = d.DstPort
				} else {
					srcPort = d.SrcPort
					dstPort = port
				}
				udpb, payload, gopkt, err := d.NewIPv6TracePacket(hl, src, dst, srcPort, dstPort)
				if err == nil {
					ret <- pkt6{UDPHeader: udpb, Payload: payload, Src: src, SrcPort: int(srcPort), DstPort: int(dstPort), HopLimit: int(hl), Gopkt: gopkt}
				}
			}
		}
		close(ret)
	}()
	return ret
}

func (d trUDPv6) Match(sent []*ProbeUDPv6, received []*ProbeResponseUDPv6) {
	for _, spu := range sent {
		if err := spu.Validate(); err != nil {
			continue
		}
		for _, rpu := range received {
			if err := rpu.Validate(); err != nil {
				continue
			}
			if !rpu.Matches(spu) {
				continue
			}
			if _, exists := d.Validation.TracerouteReceivedProbes[spu.RemoteAddr.String()]; !exists {
				d.Validation.TracerouteReceivedProbes[spu.RemoteAddr.String()] = make(map[int64]string)
			}
			d.Validation.TracerouteReceivedProbes[spu.RemoteAddr.String()][int64(spu.HopLimit)] = rpu.Addr.String()
			break
		}
	}
}
