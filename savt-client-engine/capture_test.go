package engine

import (
	"bytes"
	"encoding/binary"
	"github.com/google/gopacket/layers"
	"net"
	"savt-client/savt-client-engine/common"
	"testing"
)

func TestCapturePreservesCompleteIPBytes(t *testing.T) {
	for _, family := range []string{"4", "6"} {
		src, dst := net.ParseIP("192.0.2.1"), net.ParseIP("192.0.2.2")
		ether := uint16(0x0800)
		if family == "6" {
			src, dst = net.ParseIP("2001:db8::1"), net.ParseIP("2001:db8::2")
			ether = 0x86dd
		}
		ip, _ := common.BuildUDPIP(src, dst, 123, 456, 12, 999, []byte("unaltered received bytes"))
		header := make([]byte, 18)
		binary.BigEndian.PutUint16(header[12:14], 0x8100)
		binary.BigEndian.PutUint16(header[16:18], ether)
		frame := append(header, ip...)
		frame = append(frame, make([]byte, 20)...)
		got, truncated, err := extractIP(frame, layers.LinkTypeEthernet)
		if err != nil || truncated || !bytes.Equal(got, ip) {
			t.Fatalf("capture changed IPv%s bytes: %v", family, err)
		}
		frame = frame[:18+len(ip)-4]
		got, truncated, err = extractIP(frame, layers.LinkTypeEthernet)
		if err != nil || !truncated || !bytes.Equal(got, ip[:len(ip)-4]) {
			t.Fatal("truncated capture lost bytes/flag")
		}
	}
}
