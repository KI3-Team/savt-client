// nextclient: 原语解释器。循环: POST /next(带上一轮报告) -> 执行 send/listen/wait -> 汇报。
// send.srcMode: spoof=pcap照抄探测源(伪造) / auto=本机真实源(内核socket,打洞/真实源探测)
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"sav-next-client/common"
)

var server = flag.String("server", "http://47.242.254.61:41452", "nextserver url")

var (
	token   []byte
	vid     int64
	dev     *common.Device
	sockets = map[int]*net.UDPConn{} // port -> 内核socket(auto发送与udp监听共用)

	repMu   sync.Mutex // 并发监听goroutine写报告map

	sentMu sync.Mutex
	sent   = map[uint16]sentInfo{} // 内层IP ID -> 发送记录(供ICMP匹配)
)

type sentInfo struct {
	seq int64
	ttl int64
}

func main() {
	flag.Parse()
	log.SetFlags(log.Ltime)

	// 1. 创建会话
	var info common.SessionInfo
	postJSONinto(*server+"/api/session", nil, &info)
	token, _ = hex.DecodeString(info.TokenHex)
	vid = info.ID
	log.Printf("session %d created, rounds: %v", vid, info.Rounds)

	// 2. 出口设备发现(伪造发送需要二层帧头)
	srvUDP, err := urlToUDP(*server)
	if err != nil {
		log.Fatal("server addr: ", err)
	}
	if d, err := common.DiscoverDevice(srvUDP); err != nil {
		log.Fatal("device discovery: ", err)
	} else {
		dev = d
	}
	log.Printf("egress %s dev %s", dev.LocalIP, dev.PcapName)

	// 3. 轮次循环
	var report *common.ClientReport
	for round := 0; ; round++ {
		body := common.ClientReport{Round: int64(round)}
		if report != nil {
			body = *report
		}
		var resp common.NextResponse
		postJSONinto(fmt.Sprintf("%s/api/session/%d/next", *server, vid), body, &resp)
		if resp.Op == "finish" {
			raw, _ := json.MarshalIndent(resp.Result, "", " ")
			fmt.Printf("\n===== 测量结果 =====\n%s\n", raw)
			return
		}
		report = executeRound(round, resp.Name, resp.Actions)
	}
}

// executeRound 执行一轮原语动作,产出该轮报告。
// 所有监听(icmp+udp)先于其它动作并发启动(icmp的差错在发送期间返回;多端口udp监听需同时开窗)。
func executeRound(round int, name string, actions []common.Action) *common.ClientReport {
	log.Printf("-- round %d [%s]: %d actions", round, name, len(actions))
	rep := &common.ClientReport{Round: int64(round), UDP: map[int64]string{}, Hops: map[string]map[int64]string{}}
	var waiters []chan struct{}
	for _, a := range actions {
		if a.Op != "listen" {
			continue
		}
		isICMP := a.Proto == "icmp"
		// 多个udp监听也并发(不同端口各自开窗);单udp监听若与打洞同socket(同端口)则仍串行,
		// 但并发启动socket由getSocket复用,无冲突。
		if !isICMP && countListens(actions, "udp") <= 1 {
			continue // 单udp监听保持原有串行位置(打洞send先行,随后监听)
		}
		ch := make(chan struct{})
		waiters = append(waiters, ch)
		if isICMP {
			go func(a common.Action) { defer close(ch); doListenICMP(a, rep) }(a)
		} else {
			go func(a common.Action) { defer close(ch); doListenUDP(a, rep) }(a)
		}
	}
	for _, a := range actions {
		if a.Op == "listen" {
			isICMP := a.Proto == "icmp"
			if isICMP || countListens(actions, "udp") > 1 {
				continue // 已并发启动
			}
		}
		switch a.Op {
		case "send":
			doSend(a, rep)
		case "listen":
			doListenUDP(a, rep)
		case "wait":
			log.Printf("   wait %dms", a.DurationMs)
			time.Sleep(time.Duration(a.DurationMs) * time.Millisecond)
		default:
			log.Printf("   unknown op %q skipped", a.Op)
		}
	}
	for _, ch := range waiters {
		<-ch
	}
	return rep
}

func countListens(actions []common.Action, proto string) int {
	n := 0
	for _, a := range actions {
		if a.Op == "listen" && a.Proto == proto {
			n++
		}
	}
	return n
}

// ---- send 原语 ----

func doSend(a common.Action, rep *common.ClientReport) {
	if a.SrcMode == "auto" && len(a.Probes) == 0 {
		// 打洞: 内核socket真实源,单包(目的缺省为服务端)
		sk := getSocket(a.SrcPort)
		dstHost := a.DstAddr
		if dstHost == "" {
			u, _ := url.Parse(*server)
			dstHost = u.Hostname()
		}
		dst, err := net.ResolveUDPAddr("udp", net.JoinHostPort(dstHost, fmt.Sprintf("%d", a.DstPort)))
		if err != nil {
			log.Printf("   punch resolve: %v", err)
			return
		}
		for i := 0; i < max(1, a.Count); i++ {
			sk.WriteToUDP(common.EncodePunch(vid, token), dst)
		}
		log.Printf("   send(auto/punch) -> %s from :%d", dst, a.SrcPort)
		return
	}
	if len(a.Probes) == 0 {
		return
	}
	interval := time.Duration(a.IntervalMs) * time.Millisecond
	rememberProbes(a.Probes)
	ttls := []int{0}
	if a.TTLMax > a.TTLMin {
		ttls = nil
		for t := a.TTLMin; t <= a.TTLMax; t++ {
			ttls = append(ttls, t)
		}
	}
	var packets [][]byte
	var useDev *common.Device
	for _, p := range a.Probes {
		src := net.ParseIP(p.SrcAddr)
		if a.SrcMode == "auto" || src == nil {
			src = dev.LocalIP // AUTO: 本机真实源(在客户端就地解析)
		}
		dst := net.ParseIP(p.DstAddr)
		for _, ttl := range ttls {
			payload := common.EncodePayload(vid, p.Seq, token)
			var pkt []byte
			var ipID uint16
			var err error
			if src.To4() == nil || dst.To4() == nil {
				pkt, ipID, err = common.BuildUDPPacket6(dev, src, dst, p.SrcPort, p.DstPort, ttl, payload)
			} else {
				pkt, ipID, err = common.BuildUDPPacket(dev, src, dst, p.SrcPort, p.DstPort, ttl, payload)
			}
			if err != nil {
				log.Printf("   build: %v", err)
				continue
			}
			useDev = dev
			packets = append(packets, pkt)
			sentMu.Lock()
			sent[ipID] = sentInfo{seq: p.Seq, ttl: int64(ttl)}
			sentMu.Unlock()
		}
	}
	if len(packets) == 0 || useDev == nil {
		return
	}
	n := 0
	for i := 0; i < max(1, a.Count); i++ {
		if err := common.SendPackets(dev.PcapName, packets, interval); err != nil {
			log.Printf("   send: %v", err)
			return
		}
		n += len(packets)
	}
	log.Printf("   send(%s) %d probes x ttl%d x %d = %d pkts", a.SrcMode, len(a.Probes), len(ttls), a.Count, n)
}

// ---- listen 原语 ----

func doListenUDP(a common.Action, rep *common.ClientReport) {
	sk := getSocket(a.Port)
	deadline := time.Now().Add(time.Duration(a.DurationMs) * time.Millisecond)
	buf := make([]byte, 1500)
	got := 0
	for time.Now().Before(deadline) {
		sk.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, addr, err := sk.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		p := buf[:n]
		v, seq, ok := common.Peek(p)
		if !ok || v != vid || !common.Verify(p, token) || seq == common.PunchSeq {
			continue
		}
		repMu.Lock()
		rep.UDP[seq] = addr.IP.String()
		repMu.Unlock()
		got++
	}
	log.Printf("   listen(udp :%d %dms) received %d probes", a.Port, a.DurationMs, got)
}

func doListenICMP(a common.Action, rep *common.ClientReport) {
	network, v6 := "ip4:icmp", dev.LocalIP.To4() == nil
	if v6 {
		network = "ip6:ipv6-icmp"
	}
	conn, err := net.ListenPacket(network, dev.LocalIP.String())
	if err != nil {
		log.Printf("   listen(icmp): %v", err)
		return
	}
	defer conn.Close()
	deadline := time.Now().Add(time.Duration(a.DurationMs) * time.Millisecond)
	buf := make([]byte, 4096)
	got, raw := 0, 0
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}
		raw++
		var seq, hopKey int64
		var ok bool
		if v6 {
			seq, hopKey, ok = matchInner6(buf[8:n])
		} else {
			seq, hopKey, ok = matchInner(buf[8:n])
		}
		if !ok {
			continue
		}
		tag := fmt.Sprintf("seq-%d", seq)
		if p := findProbe(seq); p != nil {
			tag = p.Tag
		}
		repMu.Lock()
		if rep.Hops[tag] == nil {
			rep.Hops[tag] = map[int64]string{}
		}
		rep.Hops[tag][hopKey] = addr.String()
		repMu.Unlock()
		got++
	}
	log.Printf("   listen(icmp %dms) raw=%d matched=%d", a.DurationMs, raw, got)
}

// matchInner6: ICMPv6头8B + 内层IPv6头40B定长 + UDP 8B + 载荷;RFC4443引用至多1280B,载荷匹配可靠
func matchInner6(inner []byte) (seq int64, hopKey int64, ok bool) {
	if len(inner) < 40+8+common.PayloadLen || inner[0]>>4 != 6 || inner[6] != 17 {
		return 0, 0, false
	}
	payload := inner[48:]
	v, s, good := common.Peek(payload)
	if !good || v != vid || !common.Verify(payload, token) {
		return 0, 0, false
	}
	return s, int64(inner[7]), true // hop limit
}

// matchInner 优先载荷匹配(Linux路由器引用较长);不足时按内层IP ID对照本地发送记录,
// 并用发送时记录的TTL作跳键(RFC792只引用内层IP头+8字节,且各栈引用TTL语义不一)。
func matchInner(inner []byte) (seq int64, hopKey int64, ok bool) {
	ihl := int(inner[0]&0x0f) * 4
	if len(inner) >= ihl+8+common.PayloadLen {
		if payload := inner[ihl+8:]; len(payload) == common.PayloadLen {
			if v, s, good := common.Peek(payload); good && v == vid && common.Verify(payload, token) {
				return s, int64(inner[8]), true
			}
		}
	}
	sentMu.Lock()
	defer sentMu.Unlock()
	info, found := sent[binary.BigEndian.Uint16(inner[4:6])]
	if !found {
		return 0, 0, false
	}
	return info.seq, info.ttl, true
}

// ---- 辅助 ----

var allProbes []common.Probe

func findProbe(seq int64) *common.Probe {
	for i := range allProbes {
		if allProbes[i].Seq == seq {
			return &allProbes[i]
		}
	}
	return nil
}

var sockMu sync.Mutex

func getSocket(port int) *net.UDPConn {
	sockMu.Lock()
	defer sockMu.Unlock()
	if sk, ok := sockets[port]; ok {
		return sk
	}
	sk, err := net.ListenUDP("udp", &net.UDPAddr{IP: dev.LocalIP, Port: port})
	if err != nil {
		log.Fatalf("bind udp :%d: %v", port, err)
	}
	sockets[port] = sk
	return sk
}

func postJSONinto(url string, body any, out any) {
	raw, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("%s -> %s", url, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		log.Fatal(err)
	}
}

// 记录见过的探测(供ICMP匹配tag)
func rememberProbes(ps []common.Probe) { allProbes = append(allProbes, ps...) }

func urlToUDP(s string) (*net.UDPAddr, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	return net.ResolveUDPAddr("udp", u.Host)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
