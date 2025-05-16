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

// Package net provides network protocol implementations for the SAV-T prober.
// It includes implementations for IPv4, IPv6, ICMP, ICMPv6, UDP and other protocols
// needed for network testing and validation.
package net

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// ICMP represents an ICMP packet structure
type ICMP struct {
	Type     ICMPType
	Code     ICMPCode
	Checksum uint16
	Unused   uint32
	Payload  []byte
}

// ICMPHeaderLen is the length of an ICMP header
var ICMPHeaderLen = 8

// ICMPType defines ICMP message types
type ICMPType uint8

// ICMP message types
var (
	ICMPEchoReply                     ICMPType
	ICMPDestUnreachable               ICMPType = 3
	ICMPSourceQuench                  ICMPType = 4
	ICMPRedirect                      ICMPType = 5
	ICMPAlternateHostAddr             ICMPType = 6
	ICMPEchoRequest                   ICMPType = 8
	ICMPRouterAdv                     ICMPType = 9
	ICMPRouterSol                     ICMPType = 10
	ICMPTimeExceeded                  ICMPType = 11
	ICMPParamProblem                  ICMPType = 12
	ICMPTimestampReq                  ICMPType = 13
	ICMPTimestampReply                ICMPType = 14
	ICMPAddrMaskReq                   ICMPType = 17
	ICMPAddrMaskReply                 ICMPType = 18
	ICMPTraceroute                    ICMPType = 30
	ICMPConversionErr                 ICMPType = 31
	ICMPMobileHostRedirect            ICMPType = 32
	ICMPIPv6WhereAreYou               ICMPType = 33
	ICMPIPv6IAmHere                   ICMPType = 34
	ICMPMobileRegistrationReq         ICMPType = 35
	ICMPMobileRegistrationReply       ICMPType = 36
	ICMPDomainNameReq                 ICMPType = 37
	ICMPDomainNameReply               ICMPType = 38
	ICMPSkipAlgoDiscoveryProtocol     ICMPType = 39
	ICMPPhoturis                      ICMPType = 40
	ICMPExperimentalMobilityProtocols ICMPType = 41
)

// ICMPCode defines ICMP code types
type ICMPCode uint8

// NewICMP creates a new ICMP packet from bytes
func NewICMP(b []byte) (*ICMP, error) {
	var i ICMP
	if err := i.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return &i, nil
}

// ComputeChecksum calculates the checksum for an ICMP packet
func (i ICMP) ComputeChecksum() (uint16, error) {
	var bc bytes.Buffer
	if err := binary.Write(&bc, binary.BigEndian, i.Type); err != nil {
		return 0, err
	}
	if err := binary.Write(&bc, binary.BigEndian, i.Code); err != nil {
		return 0, err
	}
	if err := binary.Write(&bc, binary.BigEndian, i.Payload); err != nil {
		return 0, err
	}
	return Checksum(bc.Bytes()), nil
}

// MarshalBinary serializes the ICMP packet
func (i ICMP) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, i.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, i.Code); err != nil {
		return nil, err
	}
	csum, err := i.ComputeChecksum()
	if err != nil {
		return nil, err
	}
	i.Checksum = csum
	if err := binary.Write(&b, binary.BigEndian, i.Checksum); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, i.Unused); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, i.Payload); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// UnmarshalBinary deserializes bytes into an ICMP packet
func (i *ICMP) UnmarshalBinary(b []byte) error {
	if len(b) < ICMPHeaderLen {
		return errors.New("short icmp header")
	}
	i.Type = ICMPType(b[0])
	i.Code = ICMPCode(b[1])
	i.Checksum = binary.BigEndian.Uint16(b[2:4])
	payload := b[ICMPHeaderLen:]
	if len(payload) > 0 {
		i.Payload = payload
	}
	return nil
}
