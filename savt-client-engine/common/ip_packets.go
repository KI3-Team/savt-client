package common

import (
	"encoding/binary"
	"fmt"
	"net"
	"runtime"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

const IPPacketEncoding = "ip-base64-v1"

// Data starts at the IP header. JSON encodes []byte as Base64.
// ID is control-plane metadata; the client never appends it to Data.
type IPPacket struct {
	ID         string `json:"id"`
	Data       []byte `json:"data"`
	Repeat     int    `json:"repeat,omitempty"`
	IntervalMs int    `json:"intervalMs,omitempty"`
}

// Captured bytes begin at the outer IP header and are never reserialized.
type ReceivedIPPacket struct {
	Data      []byte `json:"data"`
	Timestamp int64  `json:"timestampUnixNano"`
	Interface string `json:"interface,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}
type SendResult struct {
	ID        string `json:"id"`
	Requested int    `json:"requested"`
	Sent      int    `json:"sent"`
	Error     string `json:"error,omitempty"`
}

// Retained solely for older protocol readers.
type ICMPObservation struct {
	Family string `json:"family"`
	Source string `json:"source"`
	Data   []byte `json:"data"`
}

// ParseIP reads a complete outer IP header and locates the transport payload.
// It rejects truncated/fragmented captures instead of manufacturing observations.
func ParseIP(data []byte) (src, dst net.IP, proto byte, transport []byte, err error) {
	dst, err = PacketDestination(data)
	if err != nil {
		return
	}
	offset, end := 0, len(data)
	if data[0]>>4 == 4 {
		offset = int(data[0]&15) * 4
		end = int(binary.BigEndian.Uint16(data[2:4]))
		if offset < 20 || end < offset || end > len(data) || binary.BigEndian.Uint16(data[6:8])&0x3fff != 0 {
			err = fmt.Errorf("invalid or fragmented IPv4 packet")
			return
		}
		src, proto = net.IP(data[12:16]), data[9]
	} else {
		offset, end = 40, 40+int(binary.BigEndian.Uint16(data[4:6]))
		src, proto = net.IP(data[8:24]), data[6]
		if end > len(data) {
			err = fmt.Errorf("truncated IPv6 packet")
			return
		}
		for proto == 0 || proto == 43 || proto == 60 || proto == 44 || proto == 51 {
			if offset+8 > end {
				err = fmt.Errorf("short IPv6 extension")
				return
			}
			next, size := data[offset], (int(data[offset+1])+1)*8
			if proto == 44 {
				if binary.BigEndian.Uint16(data[offset+2:offset+4])&0xfff9 != 0 {
					err = fmt.Errorf("fragmented IPv6 packet")
					return
				}
				size = 8
			} else if proto == 51 {
				size = (int(data[offset+1]) + 2) * 4
			}
			if offset+size > end {
				err = fmt.Errorf("short IPv6 extension")
				return
			}
			offset, proto = offset+size, next
		}
	}
	transport = data[offset:end]
	return
}

// PacketDestination only reads the IP header. It does not normalize lengths,
// checksums, options or transport data, including deliberately unusual fields.
func PacketDestination(data []byte) (net.IP, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty IP packet")
	}
	switch data[0] >> 4 {
	case 4:
		if len(data) < 20 {
			return nil, fmt.Errorf("short IPv4 header")
		}
		return net.IP(data[16:20]), nil
	case 6:
		if len(data) < 40 {
			return nil, fmt.Errorf("short IPv6 header")
		}
		return net.IP(data[24:40]), nil
	default:
		return nil, fmt.Errorf("unsupported IP version")
	}
}

// EncapsulateIP preserves every supplied IP byte. Loopback address-family
// headers and Ethernet EtherType belong to the client, as does the next hop.
func EncapsulateIP(dev *Device, data []byte) ([]byte, error) {
	dst, err := PacketDestination(data)
	if err != nil {
		return nil, err
	}
	header := append([]byte(nil), dev.LinkLayerContents...)
	v6 := dst.To4() == nil
	switch dev.LinkLayerType {
	case layers.LayerTypeEthernet:
		if len(header) < 14 {
			return nil, fmt.Errorf("short Ethernet header")
		}
		offset := 12
		for {
			etherType := binary.BigEndian.Uint16(header[offset : offset+2])
			if etherType != 0x8100 && etherType != 0x88a8 {
				break
			}
			offset += 4
			if offset+2 > len(header) {
				return nil, fmt.Errorf("short VLAN header")
			}
		}
		etherType := uint16(0x0800)
		if v6 {
			etherType = 0x86dd
		}
		binary.BigEndian.PutUint16(header[offset:offset+2], etherType)
	case layers.LayerTypeLoopback:
		family := uint32(2)
		if v6 {
			family = 24
			switch runtime.GOOS {
			case "darwin":
				family = 30
			case "linux":
				family = 10
			case "freebsd":
				family = 28
			}
		}
		order := binary.ByteOrder(binary.LittleEndian)
		if len(header) >= 4 {
			if binary.LittleEndian.Uint32(header[:4]) > 255 {
				order = binary.BigEndian
			}
			if v6 && order.Uint32(header[:4]) != 2 {
				family = order.Uint32(header[:4])
			}
		}
		header = make([]byte, 4)
		order.PutUint32(header, family)
	default:
		// RAW/tunnel devices have no link header; discovery supplies any other
		// supported platform-specific link header unchanged.
	}
	frame := make([]byte, 0, len(header)+len(data))
	frame = append(frame, header...)
	return append(frame, data...), nil
}

// BuildUDPIP is a server-side compiler helper. The client never calls it for
// negotiated IP send actions. The caller explicitly allocates the packet ID.
func BuildUDPIP(src, dst net.IP, sport, dport, hop int, id uint32, payload []byte) ([]byte, error) {
	if src == nil || dst == nil || (src.To4() == nil) != (dst.To4() == nil) {
		return nil, fmt.Errorf("IP family mismatch")
	}
	if sport < 0 || sport > 65535 || dport < 0 || dport > 65535 || hop < 0 || hop > 255 {
		return nil, fmt.Errorf("invalid UDP/IP field")
	}
	udp := &layers.UDP{SrcPort: layers.UDPPort(sport), DstPort: layers.UDPPort(dport)}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if dst.To4() != nil {
		ip := &layers.IPv4{Version: 4, IHL: 5, Id: uint16(id), TTL: uint8(hop), Protocol: layers.IPProtocolUDP, SrcIP: src.To4(), DstIP: dst.To4()}
		if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
			return nil, err
		}
		if err := gopacket.SerializeLayers(buf, opts, ip, udp, gopacket.Payload(payload)); err != nil {
			return nil, err
		}
	} else {
		ip := &layers.IPv6{Version: 6, FlowLabel: id & 0xfffff, HopLimit: uint8(hop), NextHeader: layers.IPProtocolUDP, SrcIP: src.To16(), DstIP: dst.To16()}
		if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
			return nil, err
		}
		if err := gopacket.SerializeLayers(buf, opts, ip, udp, gopacket.Payload(payload)); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// QuoteKey can be recovered from an IP header plus the first eight transport
// bytes. Source IP/port, checksums and remaining hop count may change in transit.
func QuoteKey(data []byte) (string, error) {
	dst, err := PacketDestination(data)
	if err != nil {
		return "", err
	}
	var offset int
	var proto byte
	var id uint32
	if data[0]>>4 == 4 {
		offset = int(data[0]&15) * 4
		if offset < 20 || len(data) < offset+8 {
			return "", fmt.Errorf("short IPv4 quote")
		}
		if binary.BigEndian.Uint16(data[6:8])&0x1fff != 0 {
			return "", fmt.Errorf("noninitial fragment")
		}
		proto, id = data[9], uint32(binary.BigEndian.Uint16(data[4:6]))
	} else {
		offset, proto, id = 40, data[6], binary.BigEndian.Uint32(data[:4])&0xfffff
		for proto == 0 || proto == 43 || proto == 60 || proto == 44 || proto == 51 {
			if len(data) < offset+8 {
				return "", fmt.Errorf("short IPv6 extension")
			}
			next, size := data[offset], (int(data[offset+1])+1)*8
			if proto == 44 {
				if binary.BigEndian.Uint16(data[offset+2:offset+4])&0xfff8 != 0 {
					return "", fmt.Errorf("noninitial fragment")
				}
				size = 8
			} else if proto == 51 {
				size = (int(data[offset+1]) + 2) * 4
			}
			if offset+size > len(data) {
				return "", fmt.Errorf("short IPv6 extension")
			}
			offset, proto = offset+size, next
		}
		if len(data) < offset+8 {
			return "", fmt.Errorf("short IPv6 quote")
		}
	}
	var transport string
	switch proto {
	case 17: // UDP length is stable; source port/checksum can be rewritten by NAT.
		transport = fmt.Sprintf("%d/%d", binary.BigEndian.Uint16(data[offset+2:offset+4]), binary.BigEndian.Uint16(data[offset+4:offset+6]))
	case 6: // TCP sequence is in the minimum quoted eight bytes.
		transport = fmt.Sprintf("%d/%d", binary.BigEndian.Uint16(data[offset+2:offset+4]), binary.BigEndian.Uint32(data[offset+4:offset+8]))
	case 1, 58:
		transport = fmt.Sprintf("%d/%d/%d", data[offset], data[offset+1], binary.BigEndian.Uint16(data[offset+6:offset+8]))
	default:
		transport = fmt.Sprintf("%x", data[offset:offset+8])
	}
	return fmt.Sprintf("%d/%s/%d/%d/%s", data[0]>>4, dst.String(), proto, id, transport), nil
}

func ICMPQuote(family string, data []byte) ([]byte, bool) {
	if len(data) < 8 {
		return nil, false
	}
	if family == "4" && data[0] != 3 && data[0] != 11 && data[0] != 12 {
		return nil, false
	}
	if family == "6" && (data[0] < 1 || data[0] > 4) {
		return nil, false
	}
	if family != "4" && family != "6" {
		return nil, false
	}
	return data[8:], true
}
