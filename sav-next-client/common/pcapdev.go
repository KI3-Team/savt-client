package common

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const SnapLen = 1514

// Device: 出口设备 + 学到的以太网帧头(pcap注入必需)
type Device struct {
	PcapName          string
	LinkLayerContents []byte
	LinkLayerType     gopacket.LayerType
	LocalIP           net.IP
}

// DiscoverDevice 让内核向 dst 发一个包,再用pcap抓回来学二层帧头
func DiscoverDevice(dst *net.UDPAddr) (*Device, error) {
	conn, err := net.DialUDP("udp", nil, dst)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	local := conn.LocalAddr().(*net.UDPAddr)

	var pcapDev *pcap.Interface
	devs, _ := pcap.FindAllDevs()
	for i := range devs {
		for _, a := range devs[i].Addresses {
			if a.IP.Equal(local.IP) {
				pcapDev = &devs[i]
				break
			}
		}
	}
	if pcapDev == nil {
		return nil, fmt.Errorf("no pcap device for %s", local.IP)
	}

	handle, err := pcap.OpenLive(pcapDev.Name, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	defer handle.Close()
	if err := handle.SetBPFFilter(fmt.Sprintf("udp src port %d", local.Port)); err != nil {
		return nil, err
	}
	if _, err := conn.Write(make([]byte, 64)); err != nil {
		return nil, err
	}
	src := gopacket.NewPacketSource(handle, handle.LinkType())
	pkt, err := src.NextPacket()
	if err != nil {
		return nil, err
	}
	// 判断是否RAW类链路(隧道/点对点设备):不看DLT常量(Linux的DLT_RAW=12与gopacket的101不一致),
	// 而是看解码出的第一层是否直接是IP层——是则没有以太网头,pcap注入应直接从IP头开始
	var llc []byte
	if first := pkt.Layers(); len(first) > 0 {
		switch first[0].LayerType() {
		case layers.LayerTypeIPv4, layers.LayerTypeIPv6:
			// RAW链路:无二层头
		default:
			var ll gopacket.Layer = pkt.LinkLayer()
			if ll == nil {
				ll = first[0]
			}
			llc = ll.LayerContents()
		}
	}
	return &Device{
		PcapName:          pcapDev.Name,
		LinkLayerContents: llc,
		LinkLayerType:     gopacket.LayerType(handle.LinkType()),
		LocalIP:           local.IP,
	}, nil
}

var ipIDCounter = uint16(time.Now().UnixNano() % 65535)

// BuildUDPPacket 构造 以太网+IPv4+UDP+载荷 的完整帧(源地址逐字节使用,即伪造的实现)。
// 返回所用IP ID,供ICMP差错内层匹配使用(RFC792只引用内层IP头+8字节,载荷匹配不可靠)。
func BuildUDPPacket(dev *Device, srcIP, dstIP net.IP, sport, dport int, ttl int, payload []byte) ([]byte, uint16, error) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ipIDCounter++
	if ttl <= 0 {
		ttl = 128
	}
	ip := &layers.IPv4{Version: 4, Id: ipIDCounter, TTL: uint8(ttl), Protocol: layers.IPProtocolUDP,
		SrcIP: srcIP.To4(), DstIP: dstIP.To4()}
	udp := &layers.UDP{SrcPort: layers.UDPPort(sport), DstPort: layers.UDPPort(dport)}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		return nil, 0, err
	}
	layers_ := []gopacket.SerializableLayer{ip, udp, gopacket.Payload(payload)}
	if len(dev.LinkLayerContents) > 0 {
		layers_ = append([]gopacket.SerializableLayer{gopacket.Payload(dev.LinkLayerContents)}, layers_...)
	}
	if err := gopacket.SerializeLayers(buf, opts, layers_...); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), ipIDCounter, nil
}

// BuildUDPPacket6 构造 以太网+IPv6+UDP+载荷 (UDP over IPv6 校验和必填,gopacket自动计算)
func BuildUDPPacket6(dev *Device, srcIP, dstIP net.IP, sport, dport int, hopLimit int, payload []byte) ([]byte, uint16, error) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	ipIDCounter++
	if hopLimit <= 0 {
		hopLimit = 128
	}
	ip := &layers.IPv6{Version: 6, NextHeader: layers.IPProtocolUDP, HopLimit: uint8(hopLimit),
		SrcIP: srcIP.To16(), DstIP: dstIP.To16(), FlowLabel: uint32(ipIDCounter)}
	udp6 := &layers.UDP{SrcPort: layers.UDPPort(sport), DstPort: layers.UDPPort(dport)}
	if err := udp6.SetNetworkLayerForChecksum(ip); err != nil {
		return nil, 0, err
	}
	layers6 := []gopacket.SerializableLayer{ip, udp6, gopacket.Payload(payload)}
	if len(dev.LinkLayerContents) > 0 {
		layers6 = append([]gopacket.SerializableLayer{gopacket.Payload(dev.LinkLayerContents)}, layers6...)
	}
	if err := gopacket.SerializeLayers(buf, opts, layers6...); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), ipIDCounter, nil
}
func SendPackets(pcapName string, packets [][]byte, interval time.Duration) error {
	handle, err := pcap.OpenLive(pcapName, SnapLen, false, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()
	for _, p := range packets {
		if err := handle.WritePacketData(p); err != nil {
			return err
		}
		if interval > 0 {
			time.Sleep(interval)
		}
	}
	return nil
}

// EncodePunch 构造打洞载荷(seq=PunchSeq)
func EncodePunch(vid int64, token []byte) []byte {
	return EncodePayload(vid, PunchSeq, token)
}

var _ = binary.BigEndian
