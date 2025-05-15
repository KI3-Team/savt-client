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

package net

import (
	"bytes"
	"encoding/binary"
	"errors"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// UDPHeaderLen is the length of a UDP header
var UDPHeaderLen = 8

// UDP represents a UDP packet header
type UDP struct {
	Src          uint16
	Dst          uint16
	Len          uint16
	Csum         uint16
	Payload      []byte
	PseudoHeader []byte
}

// NewUDP creates a new UDP header from a sequence of bytes
func NewUDP(b []byte) (*UDP, error) {
	var h UDP
	if err := h.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return &h, nil
}

// Checksum calculates the checksum of a byte sequence
func Checksum(b []byte) uint16 {
	var sum uint32

	for ; len(b) >= 2; b = b[2:] {
		sum += uint32(b[0])<<8 | uint32(b[1])
	}
	if len(b) > 0 {
		sum += uint32(b[0]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	csum := ^uint16(sum)
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}

// IPv4HeaderToPseudoHeader converts an IPv4 header to a pseudo header for checksum calculation
func IPv4HeaderToPseudoHeader(hdr *ipv4.Header, udplen int) ([]byte, error) {
	if hdr == nil {
		return nil, errors.New("got nil IPv4 header")
	}
	if udplen < 0 || udplen > 0xffff {
		return nil, errors.New("UDP length out of range")
	}

	var pseudoheader [12]byte
	copy(pseudoheader[0:4], hdr.Src.To4())
	copy(pseudoheader[4:8], hdr.Dst.To4())
	pseudoheader[8] = 0
	pseudoheader[9] = byte(hdr.Protocol)
	binary.BigEndian.PutUint16(pseudoheader[10:12], uint16(udplen))

	return pseudoheader[:], nil
}

// IPv6HeaderToPseudoHeader converts an IPv6 header to a pseudo header for checksum calculation
func IPv6HeaderToPseudoHeader(hdr *ipv6.Header) ([]byte, error) {
	if hdr == nil {
		return nil, errors.New("got nil IPv6 header")
	}
	if hdr.PayloadLen < 0 || hdr.PayloadLen > 0xffffffff {
		return nil, errors.New("payload length out of range")
	}

	var pseudoheader [40]byte
	copy(pseudoheader[0:16], hdr.Src.To16())
	copy(pseudoheader[16:32], hdr.Dst.To16())
	binary.BigEndian.PutUint32(pseudoheader[32:36], uint32(hdr.PayloadLen))
	pseudoheader[39] = byte(hdr.NextHeader)

	return pseudoheader[:], nil
}

// MarshalBinary serializes the UDP header
func (h *UDP) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, h.Src); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, h.Dst); err != nil {
		return nil, err
	}

	// 计算长度
	if h.Len == 0 {
		headerLen := UDPHeaderLen
		payloadLen := len(h.Payload)
		totalLen := headerLen + payloadLen

		if totalLen < 0 || totalLen > 0xffff {
			return nil, errors.New("UDP packet too large")
		}
		h.Len = uint16(totalLen)
	}

	if h.Len < 8 || h.Len > 0xffff-20 {
		return nil, errors.New("invalid udp header len")
	}
	if err := binary.Write(&buf, binary.BigEndian, h.Len); err != nil {
		return nil, err
	}
	if h.Csum == 0 && h.PseudoHeader != nil {
		var b bytes.Buffer
		if err := binary.Write(&b, binary.BigEndian, h.PseudoHeader); err != nil {
			return nil, err
		}
		if err := binary.Write(&b, binary.BigEndian, buf.Bytes()); err != nil {
			return nil, err
		}
		if err := binary.Write(&b, binary.BigEndian, h.Payload); err != nil {
			return nil, err
		}
		h.Csum = Checksum(b.Bytes())
	}
	if err := binary.Write(&buf, binary.BigEndian, h.Csum); err != nil {
		return nil, err
	}
	ret := append(buf.Bytes(), h.Payload...)
	return ret, nil
}

// UnmarshalBinary deserializes bytes into a UDP header
func (h *UDP) UnmarshalBinary(b []byte) error {
	if len(b) < UDPHeaderLen {
		return errors.New("short udp header")
	}
	h.Src = binary.BigEndian.Uint16(b[:2])
	h.Dst = binary.BigEndian.Uint16(b[2:4])
	h.Len = binary.BigEndian.Uint16(b[4:6])
	h.Csum = binary.BigEndian.Uint16(b[6:8])
	return nil
}

// Next returns the next layer (implements Layer interface)
func (h *UDP) Next() Layer {
	return nil
}

// SetNext sets the next layer (implements Layer interface)
func (h *UDP) SetNext(l Layer) {
	// UDP is typically the final layer, so this is a no-op
}
