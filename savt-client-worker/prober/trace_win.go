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

	inet "savt-client/savt-client-worker/prober/net"

	"github.com/google/gopacket/pcap"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/windows"
)

func (d UDPv4) SendIPv4Trace() error {
	handle, err := pcap.OpenLive(d.Device.Pcap.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()

	for p := range d.NewIPv4TracePackets() {
		for i := 0; i < 3; i++ {
			err = handle.WritePacketData(p.Data())
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
			err = handle.WritePacketData(p.Data())
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

	socket, err := windows.Socket(windows.AF_INET, windows.SOCK_RAW, windows.IPPROTO_ICMP)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create socket: %w", err)
	}
	defer windows.Closesocket(socket)
	var addr windows.SockaddrInet4
	addr.Addr = [4]byte{0, 0, 0, 0} // 0.0.0.0 represents all local network interfaces
	addr.Port = 0                   // 0 means the system will assign a port number

	// Bind socket to the local address
	err = windows.Bind(socket, &addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to bind socket: %w", err)
	}

	numPackets := int(d.NumPaths) * int(d.MaxTTL-d.MinTTL)

	recvErrors := make(chan error)
	recvChan := make(chan []*ProbeResponseUDPv4, 1)
	go func(errch chan error, rc chan []*ProbeResponseUDPv4) {
		howLong := d.Delay*time.Duration(numPackets) + d.Timeout
		received, err := d.ListenFor(socket, howLong)
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
			err = handle.WritePacketData(p.Gopkt.Data())
			if err != nil {
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

func (d trUDPv4) ListenFor(socket windows.Handle, howLong time.Duration) ([]*ProbeResponseUDPv4, error) {
	packets := make([]*ProbeResponseUDPv4, 0)
	deadline := time.Now().Add(howLong)

	recvTimeout := time.Millisecond * 100 // Set the timeout for each receive operation
	windows.SetsockoptInt(socket, windows.SOL_SOCKET, windows.SO_RCVTIMEO, int(recvTimeout.Milliseconds()))
	for {
		if deadline.Sub(time.Now()) <= 0 {
			break
		}

		buf := make([]byte, 1024)
		n, _, err := windows.Recvfrom(socket, buf, 0)
		if err != nil {
			if errno, ok := err.(windows.Errno); ok {
				if errno == windows.WSAETIMEDOUT {
					// Receive timeout, continue the loop
					continue
				}
				//fmt.Printf("Error receiving from socket: %v\n", err)
				return nil, fmt.Errorf("failed to receive package: %w", err)
			} else {
				//fmt.Printf("Unexpected error type: %v\n", err)
				return nil, fmt.Errorf("unexpected error type: %w", err)
			}
		}

		if n > 0 {
			//fmt.Printf("Received %d bytes from socket\n", n)
			receivedAt := time.Now()
			ipHeader, err := ipv4.ParseHeader(buf[:n])
			if err != nil {
				//fmt.Printf("Failed to parse IPv4 header: %v\n", err)
				continue
			}
			packets = append(packets, &ProbeResponseUDPv4{
				Header:    ipHeader,
				Payload:   buf[ipHeader.Len:n],
				Timestamp: receivedAt,
			})
		}
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

	// Create a raw socket for ICMPv6 protocol
	socket, err := windows.Socket(windows.AF_INET6, windows.SOCK_RAW, windows.IPPROTO_ICMPV6)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create socket: %w", err)
	}
	defer windows.Closesocket(socket)
	var addr windows.SockaddrInet6
	addr.Addr = [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	//copy(addr.Addr[:], localUDPAddr.IP.To16())
	addr.Port = 0 // 0 means the system will assign a port number

	// Bind socket to the local address
	err = windows.Bind(socket, &addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to bind socket: %w", err)
	}

	numPackets := int(d.NumPaths) * int(d.MaxHopLimit-d.MinHopLimit)

	recvErrors := make(chan error)
	recvChan := make(chan []*ProbeResponseUDPv6, 1)

	go func(errch chan error, rc chan []*ProbeResponseUDPv6) {
		howLong := d.Delay*time.Duration(numPackets) + d.Timeout
		received, err := d.ListenFor(socket, howLong)
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
			err = handle.WritePacketData(p.Gopkt.Data())
			if err != nil {
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

func (d trUDPv6) ListenFor(socket windows.Handle, howLong time.Duration) ([]*ProbeResponseUDPv6, error) {
	packets := make([]*ProbeResponseUDPv6, 0)
	deadline := time.Now().Add(howLong)

	recvTimeout := time.Millisecond * 100 // Set the timeout for each receive operation
	windows.SetsockoptInt(socket, windows.SOL_SOCKET, windows.SO_RCVTIMEO, int(recvTimeout.Milliseconds()))
	for {
		if deadline.Sub(time.Now()) <= 0 {
			break
		}

		data := make([]byte, 4096)
		n, addr, err := windows.Recvfrom(socket, data, 0)
		if err != nil {
			if errno, ok := err.(windows.Errno); ok {
				if errno == windows.WSAETIMEDOUT {
					//fmt.Println("Receive timed out, continuing...")
					continue
				}
				//fmt.Printf("Error receiving from socket: %v\n", err)
				return nil, fmt.Errorf("failed to receive package: %w", err)
			} else {
				//fmt.Printf("Unexpected error type: %v\n", err)
				return nil, fmt.Errorf("unexpected error type: %w", err)
			}
		}
		if n < 0 {
			continue
		}
		//fmt.Printf("received! %d\n", n)
		receivedAt := time.Now()
		srcIP := net.IP(addr.(*windows.SockaddrInet6).Addr[0:])
		packets = append(packets, &ProbeResponseUDPv6{
			Data:      data[:n],
			Addr:      srcIP,
			Timestamp: receivedAt,
		})

	}
	return packets, nil
}
