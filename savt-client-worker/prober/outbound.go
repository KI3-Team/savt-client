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
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func outboundValidation(validation *Validation, schedule *Schedule, dev *Device) error {
	packets := make([]gopacket.Packet, 0)
	for _, probe := range schedule.Probes {
		srcAddr := &net.UDPAddr{IP: probe.SrcAddr, Port: probe.SrcPort}
		dstAddr := &net.UDPAddr{IP: probe.DstAddr, Port: probe.DstPort}
		if srcAddr.IP.To4() != nil {
			packet, err := dev.NewProbeIPv4Packet(srcAddr, dstAddr, validation.ID, schedule.ID, probe.Seq, validation.Token)
			if err != nil {
				return err
			}
			packets = append(packets, packet)
		} else {
			packet, err := dev.NewProbeIPv6Packet(srcAddr, dstAddr, validation.ID, schedule.ID, probe.Seq, validation.Token)
			if err != nil {
				return err
			}
			packets = append(packets, packet)
		}
	}
	for i := 0; i < 3; i++ {
		err := dev.SendPackets(packets)
		if err != nil {
			return err
		}
	}
	return nil
}

var ipID = uint16(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(math.MaxUint16))

func (dev *Device) NewProbeIPv4Packet(srcAddr *net.UDPAddr, dstAddr *net.UDPAddr, vid int64, sid int64, seq int64, token []byte) (gopacket.Packet, error) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ipID++
	ipv4 := &layers.IPv4{
		Version:  4,
		Id:       ipID,
		TTL:      128,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    srcAddr.IP.To4(),
		DstIP:    dstAddr.IP.To4(),
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(srcAddr.Port),
		DstPort: layers.UDPPort(dstAddr.Port),
	}
	err := udp.SetNetworkLayerForChecksum(ipv4)
	if err != nil {
		return nil, err
	}
	payload := make([]byte, 24+sha1.Size)
	binary.BigEndian.PutUint64(payload, uint64(vid))
	binary.BigEndian.PutUint64(payload[8:16], uint64(sid))
	binary.BigEndian.PutUint64(payload[16:24], uint64(seq))
	mac := hmac.New(sha1.New, token)
	mac.Write(payload[0:24])
	copy(payload[24:24+sha1.Size], mac.Sum(nil))
	err = gopacket.SerializeLayers(buf, opts,
		gopacket.Payload(dev.LinkLayerContents),
		ipv4,
		udp,
		gopacket.Payload(payload),
	)
	if err != nil {
		return nil, err
	}
	return gopacket.NewPacket(buf.Bytes(), dev.LinkLayerType, gopacket.Default), nil
}

func (dev *Device) NewProbeIPv6Packet(srcAddr *net.UDPAddr, dstAddr *net.UDPAddr, vid int64, sid int64, seq int64, token []byte) (gopacket.Packet, error) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ipID++
	ipv6 := &layers.IPv6{
		Version:    6,
		NextHeader: layers.IPProtocolUDP,
		HopLimit:   128,
		SrcIP:      srcAddr.IP.To16(),
		DstIP:      dev.Endpoint.IP.To16(),
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(srcAddr.Port),
		DstPort: layers.UDPPort(dstAddr.Port),
	}
	err := udp.SetNetworkLayerForChecksum(ipv6)
	if err != nil {
		return nil, err
	}
	payload := make([]byte, 24+sha1.Size)
	binary.BigEndian.PutUint64(payload, uint64(vid))
	binary.BigEndian.PutUint64(payload[8:16], uint64(sid))
	binary.BigEndian.PutUint64(payload[16:24], uint64(seq))
	mac := hmac.New(sha1.New, token)
	mac.Write(payload[0:24])
	copy(payload[24:24+sha1.Size], mac.Sum(nil))
	err = gopacket.SerializeLayers(buf, opts,
		gopacket.Payload(dev.LinkLayerContents),
		ipv6,
		udp,
		gopacket.Payload(payload),
	)
	if err != nil {
		return nil, err
	}
	return gopacket.NewPacket(buf.Bytes(), dev.LinkLayerType, gopacket.Default), nil
}

func (dev *Device) SendPackets(packets []gopacket.Packet) error {
	handle, err := pcap.OpenLive(dev.Pcap.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()
	for _, packet := range packets {
		err := handle.WritePacketData(packet.Data())
		if err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}
