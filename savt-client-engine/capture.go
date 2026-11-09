package engine

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"savt-client/savt-client-engine/common"
)

// extractIP copies the original bytes after the link header, including IPv6
// extension headers. Ethernet padding is outside the IP packet.
func extractIP(frame []byte, link layers.LinkType) ([]byte, bool, error) {
	pkt := gopacket.NewPacket(frame, link, gopacket.DecodeOptions{Lazy: false, NoCopy: true})
	offset := 0
	for _, layer := range pkt.Layers() {
		if layer.LayerType() == layers.LayerTypeIPv4 || layer.LayerType() == layers.LayerTypeIPv6 {
			data := frame[offset:]
			if _, err := common.PacketDestination(data); err != nil {
				return nil, false, err
			}
			size := 0
			if data[0]>>4 == 4 {
				size = int(binary.BigEndian.Uint16(data[2:4]))
			} else {
				size = 40 + int(binary.BigEndian.Uint16(data[4:6]))
			}
			if size == 0 {
				size = len(data)
			}
			truncated := size > len(data)
			if size < len(data) {
				data = data[:size]
			}
			return append([]byte(nil), data...), truncated, nil
		}
		offset += len(layer.LayerContents())
	}
	return nil, false, fmt.Errorf("frame has no IP header")
}

// All capture handles are installed before any packet injection. The UDP
// sockets only keep receive ports open; parsing belongs entirely to the server.
func startCapture(actions []common.Action, rep *common.ClientReport, sources map[string]string) ([]<-chan struct{}, error) {
	maxWindow := time.Duration(0)
	terms := []string{}
	for _, a := range actions {
		if a.Op != "listen" {
			continue
		}
		window := time.Duration(a.DurationMs) * time.Millisecond
		if window > maxWindow {
			maxWindow = window
		}
		switch a.Proto {
		case "icmp":
			terms = append(terms, "icmp or icmp6")
		case "udp":
			if getSocket(a.Port) == nil {
				return nil, fmt.Errorf("cannot bind UDP receive port %d", a.Port)
			}
			terms = append(terms, fmt.Sprintf("udp dst port %d", a.Port))
		default:
			return nil, fmt.Errorf("unsupported listen protocol %q", a.Proto)
		}
	}
	if maxWindow == 0 {
		return nil, nil
	}
	localIPs := map[string]bool{dev.LocalIP.String(): true}
	for _, source := range sources {
		localIPs[source] = true
	}
	devices := map[string]bool{dev.PcapName: true}
	available, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}
	for _, d := range available {
		for _, a := range d.Addresses {
			if localIPs[a.IP.String()] {
				devices[d.Name] = true
			}
		}
	}
	hosts := []string{}
	for ip := range localIPs {
		if net.ParseIP(ip) != nil {
			hosts = append(hosts, "dst host "+ip)
		}
	}
	filter := "(" + strings.Join(terms, " or ") + ") and (" + strings.Join(hosts, " or ") + ")"
	type capture struct {
		h    *pcap.Handle
		name string
	}
	opened := []capture{}
	for name := range devices {
		h, e := pcap.OpenLive(name, 131072, false, 100*time.Millisecond)
		if e == nil {
			e = h.SetBPFFilter(filter)
		}
		if e != nil {
			if h != nil {
				h.Close()
			}
			for _, c := range opened {
				c.h.Close()
			}
			return nil, fmt.Errorf("capture %s: %w", name, e)
		}
		opened = append(opened, capture{h, name})
	}
	deadline := time.Now().Add(maxWindow)
	done := []<-chan struct{}{}
	for _, c := range opened {
		ch := make(chan struct{})
		done = append(done, ch)
		go func(c capture) {
			defer close(ch)
			defer c.h.Close()
			count, size := 0, 0
			for time.Now().Before(deadline) {
				data, ci, e := c.h.ReadPacketData()
				if e == pcap.NextErrorTimeoutExpired {
					continue
				}
				if e != nil {
					repMu.Lock()
					rep.SendErrors["capture/"+c.name] = e.Error()
					repMu.Unlock()
					break
				}
				ip, truncated, e := extractIP(data, c.h.LinkType())
				if e != nil {
					continue
				}
				repMu.Lock()
				if len(rep.ReceivedPackets) >= 8192 || captureBytes(rep)+len(ip) > 8<<20 {
					rep.SendErrors["capture/limit"] = "receive capture limit exceeded"
				} else {
					rep.ReceivedPackets = append(rep.ReceivedPackets, common.ReceivedIPPacket{Data: ip, Timestamp: ci.Timestamp.UnixNano(), Interface: c.name, Truncated: truncated || ci.CaptureLength < ci.Length})
					count++
					size += len(ip)
				}
				repMu.Unlock()
			}
			log.Printf("   capture(%s %dms) %d IP packets, %d bytes", c.name, maxWindow.Milliseconds(), count, size)
		}(c)
	}
	return done, nil
}
func captureBytes(rep *common.ClientReport) int {
	n := 0
	for _, p := range rep.ReceivedPackets {
		n += len(p.Data)
	}
	return n
}
