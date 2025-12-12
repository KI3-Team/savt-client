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
	"fmt"
	"log"
	"net"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/pcap"
)

const SnapLen = 1514

type Device struct {
	Net               *net.Interface
	Pcap              *pcap.Interface
	LinkLayerContents []byte
	LinkLayerType     gopacket.LayerType
	Endpoint          *net.UDPAddr
	PrivateIP         net.IP
	PublicIP          net.IP
}

func newPcapDev(ip net.IP) (*pcap.Interface, error) {
	devs, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, nil
	}
	for _, dev := range devs {
		for _, address := range dev.Addresses {
			if ip.Equal(address.IP) {
				return &dev, nil
			}
		}
	}
	if ip.IsLoopback() {
		return &devs[len(devs)-1], nil
	}
	return nil, nil
}

func newNetDev(ip net.IP) (*net.Interface, error) {
	devs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, nil
	}
	for _, dev := range devs {
		addrs, err := dev.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if ip.Equal(addr.(*net.IPNet).IP) {
				return &dev, nil
			}
		}
	}
	if ip.IsLoopback() {
		return &devs[len(devs)-1], nil
	}
	return nil, nil
}

func newDev(srcAddr, dstAddr *net.UDPAddr, validation *Validation) (*Device, error) {
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing UDP connection: %v", err)
		}
	}()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	pcapDev, err := newPcapDev(localAddr.IP)
	if err != nil {
		return nil, err
	}
	if pcapDev == nil {
		return nil, fmt.Errorf("no pcap device available")
	}
	netDev, err := newNetDev(localAddr.IP)
	if err != nil {
		return nil, err
	}
	if netDev == nil {
		return nil, fmt.Errorf("no net device available")
	}
	handle, err := pcap.OpenLive(pcapDev.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		log.Printf("You may need to run this program with administrator privileges.\n")
		return nil, err
	}
	err = handle.SetBPFFilter(fmt.Sprintf("udp src port %d", localAddr.Port))
	if err != nil {
		return nil, err
	}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	payload := make([]byte, 8+net.IPv4len+sha1.Size)
	binary.BigEndian.PutUint64(payload, uint64(validation.ID))
	copy(payload[8:8+net.IPv4len], localAddr.IP.To4())
	mac := hmac.New(sha1.New, validation.Token)
	mac.Write(payload[0 : 8+net.IPv4len])
	copy(payload[8+net.IPv4len:8+net.IPv4len+sha1.Size], mac.Sum(nil))
	_, err = conn.Write(payload)
	if err != nil {
		return nil, err
	}
	packet, err := packetSource.NextPacket()
	if err != nil {
		return nil, err
	}
	handle.Close()
	var linkLayer gopacket.Layer = packet.LinkLayer()
	if linkLayer == nil {
		linkLayer = packet.Layers()[0]
	}
	return &Device{
		Net:               netDev,
		Pcap:              pcapDev,
		LinkLayerContents: linkLayer.LayerContents(),
		LinkLayerType:     linkLayer.LayerType(),
		Endpoint:          dstAddr,
		PrivateIP:         localAddr.IP,
	}, nil
}

func newIPv4Dev(dstAddr *net.UDPAddr, validation *Validation) (*Device, error) {
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	return newDev(srcAddr, dstAddr, validation)
}

func newIPv6Dev(dstAddr *net.UDPAddr, validation *Validation) (*Device, error) {
	srcAddr := &net.UDPAddr{IP: net.IPv6zero, Port: 0}
	return newDev(srcAddr, dstAddr, validation)
}
