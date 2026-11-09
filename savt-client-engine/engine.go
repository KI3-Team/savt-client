// Package engine executes IP send/listen/wait primitives. All measurement
// interpretation and IP packet construction belong to the controller.
package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"savt-client/savt-client-engine/common"
	"strings"
	"sync"
	"time"
)

var Server, Server6 string
var serverURL string
var dev *common.Device
var sockets = map[int]*net.UDPConn{}
var repMu, sockMu sync.Mutex

func MeasuredFamily(serverURL string) string {
	u, err := url.Parse(serverURL)
	if err == nil && strings.Contains(u.Hostname(), ":") {
		return "IPv6"
	}
	return "IPv4"
}
func RunOnce(verbose bool, family string) *common.Result { return runOnce(verbose, family, nil) }
func runOnce(verbose bool, family string, m *Measurement) *common.Result {
	serverURL = ""
	packetDevices = map[string]*common.Device{}
	closeSockets()
	defer closeSockets()
	req := common.MeasurementRequest{}
	if m != nil {
		req = m.request()
	}
	var info common.SessionInfo
	for _, cand := range ResolveServerCandidatesFamily(family) {
		if err := postJSONintoErr(cand+"/api/session", req, &info); err != nil {
			log.Printf("server %s: %v", cand, err)
			continue
		}
		if req.IsPublic != nil && (info.IsPublic == nil || *info.IsPublic != *req.IsPublic) {
			log.Printf("server %s did not acknowledge sharing preference", cand)
			continue
		}
		if m != nil && info.MeasurementID != m.ID {
			log.Printf("server %s does not support measurement grouping", cand)
			continue
		}
		serverURL = cand
		break
	}
	if serverURL == "" {
		log.Printf("no reachable IPv%s controller", family)
		return nil
	}
	if sessionMetadata != nil {
		sessionMetadata("IPv"+strings.TrimPrefix(family, "IPv"), info)
	}
	endpoint := sessionEndpoint{Server: serverURL, ID: info.ID}
	if m != nil {
		m.endpoints[family] = endpoint
	}
	udp, err := urlToUDP(serverURL)
	if err != nil {
		log.Print(err)
		return nil
	}
	source, err := resolveLocalSource(udp.IP.String())
	if err != nil {
		log.Printf("local source: %v", err)
		return nil
	}
	endpoint.Source = source
	if m != nil {
		m.endpoints[family] = endpoint
	}
	dev, err = discoverDevice(udp)
	if err != nil {
		log.Printf("device discovery: %v", err)
		return nil
	}
	packetDevices[udp.IP.String()] = dev
	log.Printf("server %s session %d egress %s dev %s", serverURL, info.ID, source, dev.PcapName)
	routes := map[string]string{}
	body := common.ClientReport{Round: 0, LocalSourceIP: source}
	for {
		body.LocalSourceIP, body.RouteSources = source, routes
		if m != nil {
			body.MeasurementToken = m.token
		}
		endpoint.Round, endpoint.Routes = body.Round, routes
		if m != nil {
			m.endpoints[family] = endpoint
		}
		var resp common.NextResponse
		if err := postNext(endpoint, body, &resp); err != nil {
			log.Printf("round %d: %v", body.Round, err)
			return nil
		}
		if resp.Receipt != nil && m != nil {
			m.applyReceipt(*resp.Receipt)
		}
		if resp.Op == "complete" || resp.Op == "finish" {
			if resp.Result != nil && resp.Result.ASNSource == "" && info.ASNSource != "" {
				resp.Result.ClientMetadata = info.ClientMetadata
			}
			if verbose {
				raw, _ := json.MarshalIndent(resp.Result, "", " ")
				fmt.Printf("\n===== Measurement results =====\n%s\n", raw)
			}
			return resp.Result
		}
		if resp.Op != "actions" || int64(resp.Round) != body.Round {
			log.Printf("invalid next response: %s round %d", resp.Op, resp.Round)
			return nil
		}
		// Route overrides are learned from actual send packets, with no target-list
		// preparation request. Ask the server to compile again before any injection.
		changed := false
		for _, a := range resp.Actions {
			if a.Op == "send" {
				for _, p := range a.Packets {
					dst, e := common.PacketDestination(p.Data)
					if e != nil {
						continue
					}
					local, e := resolveLocalSource(dst.String())
					if e == nil && local != source && routes[dst.String()] != local {
						routes[dst.String()] = local
						changed = true
					}
				}
			}
		}
		if changed {
			continue
		}
		if roundProgress != nil {
			roundProgress(MeasuredFamily(serverURL), resp.Name, resp.Round, len(info.Rounds), false)
		}
		body = *executeRound(resp.Round, resp.Name, resp.Actions, routes)
		body.Round = int64(resp.Round + 1)
		if roundProgress != nil {
			roundProgress(MeasuredFamily(serverURL), resp.Name, resp.Round, len(info.Rounds), true)
		}
	}
}

var roundProgress func(family, name string, round, total int, done bool)

var sessionMetadata func(family string, info common.SessionInfo)

func SetSessionMetadata(fn func(string, common.SessionInfo)) { sessionMetadata = fn }

func SetProgress(fn func(family, name string, round, total int, done bool)) { roundProgress = fn }
func LastServerFamily() string {
	if serverURL == "" {
		return ""
	}
	return MeasuredFamily(serverURL)
}

var beginCapture = startCapture

func executeRound(round int, name string, actions []common.Action, routes map[string]string) *common.ClientReport {
	log.Printf("-- round %d [%s]: %d actions", round, name, len(actions))
	rep := &common.ClientReport{Round: int64(round + 1), SendErrors: map[string]string{}}
	waiters, err := beginCapture(actions, rep, routes)
	if err != nil {
		rep.SendErrors["capture"] = err.Error()
		log.Printf("capture: %v", err)
		return rep
	}
	for _, a := range actions {
		switch a.Op {
		case "send":
			doSendIP(a, rep)
		case "listen": // Capture is already active.
		case "wait":
			time.Sleep(time.Duration(a.DurationMs) * time.Millisecond)
		default:
			rep.SendErrors["action"] = fmt.Sprintf("unsupported operation %q", a.Op)
		}
	}
	for _, done := range waiters {
		<-done
	}
	return rep
}
func closeSockets() {
	sockMu.Lock()
	defer sockMu.Unlock()
	for p, sk := range sockets {
		sk.Close()
		delete(sockets, p)
	}
}
func getSocket(port int) *net.UDPConn {
	sockMu.Lock()
	defer sockMu.Unlock()
	if sk := sockets[port]; sk != nil {
		return sk
	}
	network, ip := "udp4", net.IPv4zero
	if dev.LocalIP.To4() == nil {
		network, ip = "udp6", net.IPv6zero
	}
	sk, err := net.ListenUDP(network, &net.UDPAddr{IP: ip, Port: port})
	if err != nil {
		log.Printf("bind UDP %d: %v", port, err)
		return nil
	}
	sockets[port] = sk
	return sk
}
func postNext(endpoint sessionEndpoint, body common.ClientReport, out *common.NextResponse) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = postJSONintoErr(fmt.Sprintf("%s/api/session/%d/next", endpoint.Server, endpoint.ID), body, out)
		if err == nil {
			return nil
		}
		if e, ok := err.(httpStatusError); ok && e.code < 500 {
			return err
		}
		if attempt < 2 {
			time.Sleep(250 * time.Millisecond)
		}
	}
	return err
}

type httpStatusError struct {
	code   int
	status string
}

func (e httpStatusError) Error() string { return e.status }

// ---- 服务器地址解析(机制与老架构 savt-client-api 一致) ----

type serviceConfig struct {
	Version string `json:"version"`
	Env     string `json:"env"`
}

// systemConfigDir 系统配置目录(与老架构 GetSysConfig 相同)
func systemConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("PROGRAMDATA"), "savt-client")
	case "darwin":
		return "/Library/Application Support/savt-client"
	default:
		return "/var/lib/savt-client"
	}
}

// loadServiceConfig 按老架构的搜索顺序读 service.json: ../ → ./ → 系统配置目录
func loadServiceConfig() (serviceConfig, bool) {
	for _, p := range []string{
		filepath.Join("..", "service.json"),
		"service.json",
		filepath.Join(systemConfigDir(), "service.json"),
	} {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var c serviceConfig
		if err := json.Unmarshal(data, &c); err != nil || c.Env == "" {
			continue
		}
		log.Printf("config %s: env=%s version=%s", p, c.Env, c.Version)
		return c, true
	}
	return serviceConfig{}, false
}

// resolveServerCandidates 解析候选服务器: --server 优先; 否则 env→端口 + 域名DNS(与老架构一致)
func ResolveServerCandidates() []string {
	return ResolveServerCandidatesFamily("")
}

// resolveServerCandidatesFamily 按栈族过滤候选(""/"4"/"6"); --server 指定时原样返回。
func ResolveServerCandidatesFamily(family string) []string {
	if family == "6" && Server6 != "" {
		return []string{Server6}
	}
	if Server != "" && Server6 == "" {
		return []string{Server}
	}
	if Server != "" && (family == "" || family == "4") {
		return []string{Server}
	}
	cfg, ok := loadServiceConfig()
	if !ok {
		log.Printf("no --server specified and service.json not found (searched ./, ../, %s)", systemConfigDir())
		return nil
	}
	port := "31452" // 非 test 环境
	if cfg.Env == "test" {
		port = "41452"
	}
	var cands []string
	for _, dom := range []string{"v4.sav-t.ki3.org.cn", "v6.sav-t.ki3.org.cn"} {
		ips, err := net.LookupIP(dom)
		if err != nil {
			time.Sleep(500 * time.Millisecond) // 瞬时抖动重试一次(守护模式不容失败)
			ips, err = net.LookupIP(dom)
		}
		if err != nil {
			log.Printf("DNS %s: %v", dom, err)
			continue
		}
		for _, ip := range ips {
			if v4 := ip.To4(); v4 != nil {
				if family == "6" {
					continue
				}
				cands = append(cands, fmt.Sprintf("http://%s:%s", v4, port))
			} else {
				if family == "4" {
					continue
				}
				cands = append(cands, fmt.Sprintf("http://[%s]:%s", ip, port))
			}
		}
	}
	if len(cands) == 0 {
		log.Printf("cannot resolve controller hostnames (v4/v6.sav-t.ki3.org.cn); skipping this measurement [family=%s]", family)
	}
	return cands
}

// postJSONintoErr 同 postJSONinto 但返回错误(供多候选探测)
var discoverDevice = common.DiscoverDevice

var controlClient = &http.Client{Timeout: 30 * time.Second}

func postJSONintoErr(u string, body any, out any) error {
	raw, _ := json.Marshal(body)
	resp, err := controlClient.Post(u, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return httpStatusError{resp.StatusCode, resp.Status}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

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
