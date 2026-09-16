// Package common 定义 sav-next 的原语类型、载荷编解码与探测生成工具。
// 载荷不变式: vid(8) + seq(8) + HMAC-SHA1(token)(20) = 36字节,seq 全局唯一(含保留值 PunchSeq)。
package common

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"math/big"
	"math/rand"
	"net"
)

const (
	PunchSeq   int64 = -1 // 打洞包保留seq
	PayloadLen       = 16 + sha1.Size
)

// ---- 载荷编解码(所有原语共用的真实性骨干) ----

func EncodePayload(vid, seq int64, token []byte) []byte {
	p := make([]byte, PayloadLen)
	binary.BigEndian.PutUint64(p, uint64(vid))
	binary.BigEndian.PutUint64(p[8:16], uint64(seq))
	m := hmac.New(sha1.New, token)
	m.Write(p[0:16])
	copy(p[16:], m.Sum(nil))
	return p
}

// Peek 仅解出 vid/seq(未验HMAC),用于会话路由
func Peek(p []byte) (vid, seq int64, ok bool) {
	if len(p) != PayloadLen {
		return 0, 0, false
	}
	return int64(binary.BigEndian.Uint64(p[:8])), int64(binary.BigEndian.Uint64(p[8:16])), true
}

func Verify(p, token []byte) bool {
	if len(p) != PayloadLen || len(token) == 0 {
		return false
	}
	m := hmac.New(sha1.New, token)
	m.Write(p[0:16])
	return hmac.Equal(m.Sum(nil), p[16:])
}

// ---- 探测与动作 ----

type Probe struct {
	Seq     int64  `json:"seq"`
	Tag     string `json:"tag"` // normal/internal/private/neighbor-N/constant/nattest/tracefilter/traceroute
	SrcAddr string `json:"srcAddr,omitempty"` // srcMode=spoof 时逐字节使用
	DstAddr string `json:"dstAddr"`
	SrcPort int    `json:"srcPort"`
	DstPort int    `json:"dstPort"`
	TTL     int    `json:"ttl,omitempty"`
}

// Action 是编排下发的原语实例。send 的 srcMode: spoof(照抄probes源) | auto(本机真实源)。
// TTL 阶梯: ttlMin..ttlMax(仅 send);listen: proto=udp|icmp。
type Action struct {
	Op         string  `json:"op"`
	SrcMode    string  `json:"srcMode,omitempty"`
	ProbeSet   string  `json:"probeSet,omitempty"` // 服务端生成的探测集合名
	Probes     []Probe `json:"probes,omitempty"`
	DstAddr    string  `json:"dstAddr,omitempty"` // send(auto,无probes): 打洞等单包目标
	DstPort    int     `json:"dstPort,omitempty"`
	SrcPort    int     `json:"srcPort,omitempty"`
	TTLMin     int     `json:"ttlMin,omitempty"`
	TTLMax     int     `json:"ttlMax,omitempty"`
	Count      int     `json:"count,omitempty"`
	IntervalMs int     `json:"intervalMs,omitempty"`
	Proto      string  `json:"proto,omitempty"`
	Port       int     `json:"port,omitempty"`
	DurationMs int     `json:"durationMs,omitempty"`
}

type SpooferSpec struct {
	ProbeSet   string `json:"probeSet"`
	Count      int    `json:"count"`
	IntervalMs int    `json:"intervalMs"`
}

type Round struct {
	Name    string       `json:"name"`
	DelayMs int          `json:"delayMs,omitempty"` // spoofer派发延迟(等客户端listen就位)
	Client  []Action     `json:"client"`
	Spoofer *SpooferSpec `json:"spoofer,omitempty"`
}

type Strategy struct {
	Rounds []Round          `json:"rounds"`
	Reacts []map[string]string `json:"reacts,omitempty"` // 如 {"on":"nattest","do":"echo"}
}

// ---- 会话协议 ----

type SessionInfo struct {
	ID       int64    `json:"id"`
	TokenHex string   `json:"tokenHex"`
	Rounds   []string `json:"rounds"`
}

type ClientReport struct {
	Round int64                `json:"round"`
	UDP   map[int64]string     `json:"udp,omitempty"`  // seq -> 实收源(listen udp)
	Hops  map[string]map[int64]string `json:"hops,omitempty"` // tag -> hop序 -> 路由器IP(listen icmp)
}

type ProbeOutcome struct {
	Seq     int64  `json:"seq"`
	Tag     string `json:"tag"`
	Src     string `json:"src"`
	Dst     string `json:"dst"`
	Status  string `json:"status"` // received|rewritten|blocked
	RevAddr string `json:"revAddr,omitempty"`
}

type Result struct {
	ID              int64    `json:"id"`
	ClientIP        string   `json:"clientIP"`
	CreateTime      string   `json:"createTime"`
	InboundMapped   string   `json:"inboundMapped,omitempty"`
	Probes          []ProbeOutcome `json:"probes"`
	OutboundPublic  string   `json:"outboundPublic,omitempty"`
	OutboundPrivate string   `json:"outboundPrivate,omitempty"`
	InboundValid    *bool    `json:"inboundValid,omitempty"`
	InboundPublic   string   `json:"inboundPublic,omitempty"`
	InboundPrivate  string   `json:"inboundPrivate,omitempty"`
	NATtest         bool     `json:"nattest"`
	NATMapping      string   `json:"natMapping,omitempty"`  // eim|adm(由两观测点打洞映射比对)
	NATFiltering    string   `json:"natFiltering,omitempty"` // eif|adf(由入向对照组)
	Paths           map[string]map[int64]string `json:"paths,omitempty"` // tracefilter(服务端ICMP)
	Hops            map[string]map[int64]string `json:"hops,omitempty"` // traceroute(客户端ICMP)
}

type NextResponse struct {
	Op      string    `json:"op"` // actions | finish
	Round   int       `json:"round,omitempty"`
	Name    string    `json:"name,omitempty"`
	Actions []Action  `json:"actions,omitempty"`
	Result  *Result   `json:"result,omitempty"`
}

// ---- 探测生成工具 ----

// FlipBit 从LSB起第bit位翻转(按地址族自动选v4/v6)
func FlipBit(ip net.IP, bit int) net.IP {
	if ip.To4() != nil {
		b := append(net.IP{}, ip.To4()...)
		b[len(b)-1-(bit-1)/8] ^= 1 << uint((bit-1)%8)
		return b
	}
	b := append(net.IP{}, ip.To16()...)
	b[len(b)-1-(bit-1)/8] ^= 1 << uint((bit-1)%8)
	return b
}

func IsV6(ipStr string) bool { return net.ParseIP(ipStr) != nil && net.ParseIP(ipStr).To4() == nil }

func RandomPrivateV4() net.IP {
	r := rand.New(rand.NewSource(rand.Int63()))
	pfx := [][2]byte{{10, 0}, {172, 16 + byte(r.Intn(16))}, {192, 168}}
	c := pfx[r.Intn(3)]
	return net.IPv4(c[0], c[1], byte(r.Intn(256)), byte(r.Intn(256)))
}

func RandomULA6() net.IP {
	b := make([]byte, 16)
	b[0] = 0xfd
	for i := 1; i < 16; i++ {
		b[i] = byte(rand.Intn(256))
	}
	return b
}

func RandomPrivate(ip net.IP) net.IP {
	if ip.To4() != nil {
		return RandomPrivateV4()
	}
	return RandomULA6()
}

func RandomToken() []byte {
	t := make([]byte, sha1.Size)
	n, _ := rand.Read(t)
	if n != sha1.Size {
		for i := range t {
			t[i] = byte(rand.Intn(256))
		}
	}
	return t
}

var _ = big.NewInt
