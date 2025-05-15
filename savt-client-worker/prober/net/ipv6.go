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
	"net"
)

// Version6 is the IPv6 version number
var Version6 = 6

// IPv6HeaderLen is the length of an IPv6 header
var IPv6HeaderLen = 40

// NewIPv6 creates a new IPv6 header from a sequence of bytes
func NewIPv6(b []byte) (*IPv6, error) {
	var h IPv6
	if err := h.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return &h, nil
}

// IPv6 represents an IPv6 packet header
type IPv6 struct {
	Version      int
	TrafficClass int
	FlowLabel    int
	PayloadLen   int
	NextHeader   IPProto
	HopLimit     int
	Src          net.IP
	Dst          net.IP
	next         Layer
	IPinICMP     bool
}

// Next returns the next layer
func (h IPv6) Next() Layer {
	return h.next
}

// SetNext sets the next layer
func (h *IPv6) SetNext(l Layer) {
	h.next = l
}

// MarshalBinary serializes the layer
func (h IPv6) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	if h.Version == 0 {
		h.Version = Version6
	}
	if h.Version != Version6 {
		return nil, errors.New("invalid IPv6 version")
	}
	if h.TrafficClass < 0 || h.TrafficClass > 0xff {
		return nil, errors.New("invalid IPv6 traffic class")
	}
	if h.FlowLabel < 0 || h.FlowLabel > 0x0fffff {
		return nil, errors.New("invalid IPv6 flow label")
	}
	var (
		payload []byte
		err     error
	)
	if h.next != nil {
		payload, err = h.next.MarshalBinary()
		if err != nil {
			return nil, err
		}
	}
	if h.PayloadLen == 0 {
		h.PayloadLen = len(payload)
	}
	if h.PayloadLen < 0 || h.PayloadLen > 0xffff {
		return nil, errors.New("invalid IPv6 payload length")
	}
	if h.NextHeader < 0 || h.NextHeader > 0xff {
		return nil, errors.New("invalid IPv6 next header")
	}
	if h.HopLimit < 0 || h.HopLimit > 0xff {
		return nil, errors.New("invalid IPv6 hop limit")
	}
	if h.Src == nil {
		h.Src = net.IPv6zero
	}
	if h.Dst == nil {
		h.Dst = net.IPv6zero
	}

	// Use temporary variables to ensure values are in valid range
	versionAndFields := uint32((h.Version << 28) | (h.TrafficClass << 20) | h.FlowLabel)
	if err := binary.Write(&b, binary.BigEndian, versionAndFields); err != nil {
		return nil, err
	}

	payloadLen := uint16(h.PayloadLen)
	if err := binary.Write(&b, binary.BigEndian, payloadLen); err != nil {
		return nil, err
	}

	nextHeader := uint8(h.NextHeader)
	if err := binary.Write(&b, binary.BigEndian, nextHeader); err != nil {
		return nil, err
	}

	hopLimit := uint8(h.HopLimit)
	if err := binary.Write(&b, binary.BigEndian, hopLimit); err != nil {
		return nil, err
	}

	if err := binary.Write(&b, binary.BigEndian, []byte(h.Src.To16())); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, []byte(h.Dst.To16())); err != nil {
		return nil, err
	}
	ret := b.Bytes()
	ret = append(ret, payload...)
	return ret, nil
}

// UnmarshalBinary deserializes the layer
func (h *IPv6) UnmarshalBinary(b []byte) error {
	if len(b) < IPv6HeaderLen {
		return errors.New("short ipv6 header")
	}
	var (
		u8   byte
		u16  [2]byte
		u32  [4]byte
		u128 [16]byte
	)
	buf := bytes.NewBuffer(b)
	if n, err := buf.Read(u32[:]); err != nil || n != len(u32) {
		return err
	}
	h.Version = int(u32[0] >> 4)
	h.TrafficClass = int(u32[0]&0xf)<<4 | int(u32[1]>>4)
	h.FlowLabel = int(u32[1]&0xf)<<16 | int(u32[2])<<8 | int(u32[3])
	if n, err := buf.Read(u16[:]); err != nil || n != len(u16) {
		return err
	}
	h.PayloadLen = int(binary.BigEndian.Uint16(u16[:]))
	u8, _ = buf.ReadByte()
	h.NextHeader = IPProto(u8)
	u8, _ = buf.ReadByte()
	h.HopLimit = int(u8)
	if n, err := buf.Read(u128[:]); err != nil || n != len(u128) {
		return err
	}
	h.Src = append([]byte{}, u128[:]...)
	if n, err := buf.Read(u128[:]); err != nil || n != len(u128) {
		return err
	}
	h.Dst = append([]byte{}, u128[:]...)
	if len(b) < h.PayloadLen && !h.IPinICMP {
		return errors.New("invalid IPv6 packet: payload too short")
	}
	payload := b[IPv6HeaderLen : IPv6HeaderLen+h.PayloadLen]
	if h.NextHeader == ProtoUDP {
		// Process UDP protocol
		udp, err := NewUDP(payload)
		if err == nil {
			h.next = udp
		} else {
			h.next = &Raw{Data: payload}
		}
	} else {
		h.next = &Raw{Data: payload}
	}
	return nil
}
