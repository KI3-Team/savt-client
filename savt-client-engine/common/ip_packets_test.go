package common

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net"
	"testing"

	"github.com/google/gopacket/layers"
)

func TestIPBytesRemainUnchanged(t *testing.T) {
	for _, version := range []byte{4, 6} {
		for _, protocol := range []byte{1, 6, 17, 58} {
			data := make([]byte, 80)
			data[0] = version << 4
			// Deliberately unusual lengths/checksums must survive, too.
			data[2], data[3], data[10], data[11] = 0x12, 0x34, 0xab, 0xcd
			if version == 4 {
				data[0] |= 5
				data[9] = protocol
				copy(data[16:20], net.ParseIP("192.0.2.1").To4())
			} else {
				data[6] = protocol
				copy(data[24:40], net.ParseIP("2001:db8::1").To16())
			}
			var packet IPPacket
			encoded, _ := json.Marshal(IPPacket{ID: "test", Data: data})
			if err := json.Unmarshal(encoded, &packet); err != nil {
				t.Fatal(err)
			}
			for _, device := range []*Device{
				{LinkLayerType: layers.LayerTypeEthernet, LinkLayerContents: make([]byte, 14)},
				{LinkLayerType: layers.LayerTypeLoopback, LinkLayerContents: []byte{2, 0, 0, 0}},
				{LinkLayerType: layers.LayerTypeIPv4},
			} {
				before := append([]byte(nil), data...)
				frame, err := EncapsulateIP(device, packet.Data)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(frame[len(frame)-len(data):], before) || !bytes.Equal(packet.Data, before) {
					t.Fatal("IP bytes changed")
				}
			}
		}
	}
}

func TestEthernetVLANAndLoopbackFamily(t *testing.T) {
	head := make([]byte, 18)
	binary.BigEndian.PutUint16(head[12:14], 0x8100)
	binary.BigEndian.PutUint16(head[14:16], 123)
	p := make([]byte, 48)
	p[0] = 0x60
	frame, err := EncapsulateIP(&Device{LinkLayerType: layers.LayerTypeEthernet, LinkLayerContents: head}, p)
	if err != nil {
		t.Fatal(err)
	}
	if binary.BigEndian.Uint16(frame[16:18]) != 0x86dd || binary.BigEndian.Uint16(frame[14:16]) != 123 {
		t.Fatal("VLAN/EtherType changed incorrectly")
	}
	frame, err = EncapsulateIP(&Device{LinkLayerType: layers.LayerTypeLoopback, LinkLayerContents: []byte{0, 0, 0, 30}}, p)
	if err != nil || binary.BigEndian.Uint32(frame[:4]) != 30 {
		t.Fatal("loopback family or byte order changed")
	}
}

func TestShortICMPQuoteDoesNotUseRemainingTTL(t *testing.T) {
	for _, family := range []string{"4", "6"} {
		src, dst := net.ParseIP("192.168.1.8"), net.ParseIP("198.51.100.1")
		header := 20
		if family == "6" {
			src, dst, header = net.ParseIP("2001:db8:1::8"), net.ParseIP("2001:db8:2::1"), 40
		}
		packet, err := BuildUDPIP(src, dst, 41499, 31452, 12, 12345, make([]byte, 36))
		if err != nil {
			t.Fatal(err)
		}
		key, err := QuoteKey(packet)
		if err != nil {
			t.Fatal(err)
		}
		quote := append([]byte(nil), packet[:header+8]...)
		if family == "4" {
			quote[8] = 0
			copy(quote[12:16], net.ParseIP("203.0.113.8").To4())
		} else {
			quote[7] = 1
		}
		binary.BigEndian.PutUint16(quote[header:header+2], 55555)
		binary.BigEndian.PutUint16(quote[header+6:header+8], 0xffff)
		got, err := QuoteKey(quote)
		if err != nil || got != key {
			t.Fatalf("NAT or remaining TTL changed key: %q %q %v", key, got, err)
		}
		for n := 0; n < header+8; n++ {
			if _, err := QuoteKey(quote[:n]); err == nil {
				t.Fatalf("accepted short quote %d", n)
			}
		}
	}
}
