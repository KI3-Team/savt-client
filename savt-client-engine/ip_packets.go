package engine

import (
	"fmt"
	"log"
	"net"
	"time"

	"savt-client/savt-client-engine/common"
)

var packetDevices = map[string]*common.Device{}
var discoverPacketDevice = common.DiscoverDevice
var sendIPFrames = common.SendPacketsCount
var resolveLocalSource = localSourceFor

// Connecting a UDP socket asks the local routing table for its real source.
// No UDP data is sent. Never substitute the controller-observed public IP.
func localSourceFor(destination string) (string, error) {
	ip := net.ParseIP(destination)
	if ip == nil {
		return "", fmt.Errorf("invalid route destination %q", destination)
	}
	network := "udp4"
	if ip.To4() == nil {
		network = "udp6"
	}
	conn, err := net.DialUDP(network, nil, &net.UDPAddr{IP: ip, Port: 9})
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func doSendIP(a common.Action, rep *common.ClientReport) {
	if rep.SendErrors == nil {
		rep.SendErrors = map[string]string{}
	}
	fail := func(id string, err error) {
		repMu.Lock()
		rep.SendErrors[id] = err.Error()
		repMu.Unlock()
		log.Printf("   IP send %s: %v", id, err)
	}
	if a.Encoding != common.IPPacketEncoding {
		fail("send", fmt.Errorf("unsupported packet encoding %q", a.Encoding))
		return
	}
	for _, port := range a.BindUDPPorts {
		if getSocket(port) == nil {
			fail(fmt.Sprintf("bind/%d", port), fmt.Errorf("cannot bind receive socket"))
			return
		}
	}
	for _, p := range a.Packets {
		repeat := p.Repeat
		if repeat < 1 {
			repeat = 1
		}
		failed := func(err error) {
			fail(p.ID, err)
			repMu.Lock()
			rep.SendResults = append(rep.SendResults, common.SendResult{ID: p.ID, Requested: repeat, Error: err.Error()})
			repMu.Unlock()
		}
		dst, err := common.PacketDestination(p.Data)
		if err != nil {
			failed(err)
			continue
		}
		d := packetDevices[dst.String()]
		if d == nil {
			d, err = discoverPacketDevice(&net.UDPAddr{IP: dst, Port: 9})
			if err != nil {
				failed(err)
				continue
			}
			packetDevices[dst.String()] = d
		}
		frame, err := common.EncapsulateIP(d, p.Data)
		if err != nil {
			failed(err)
			continue
		}
		frames := make([][]byte, repeat)
		for i := range frames {
			frames[i] = frame
		}
		n, err := sendIPFrames(d.PcapName, frames, time.Duration(p.IntervalMs)*time.Millisecond)
		result := common.SendResult{ID: p.ID, Requested: repeat, Sent: n}
		if err != nil {
			result.Error = err.Error()
			fail(p.ID, err)
		}
		repMu.Lock()
		rep.SendResults = append(rep.SendResults, result)
		repMu.Unlock()
	}
	log.Printf("   send(IP) %d packet definitions, outcomes=%d", len(a.Packets), len(rep.SendResults))
}
