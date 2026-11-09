package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"savt-client/savt-client-engine/common"
)

func TestClientReportsRouteSourceAndSendsServerBytes(t *testing.T) {
	oldServer, oldDiscover, oldPacketDiscover, oldSend, oldResolve := Server, discoverDevice, discoverPacketDevice, sendIPFrames, resolveLocalSource
	t.Cleanup(func() {
		Server, discoverDevice, discoverPacketDevice, sendIPFrames, resolveLocalSource = oldServer, oldDiscover, oldPacketDiscover, oldSend, oldResolve
	})
	source, target := "192.168.31.166", "198.51.100.1"
	data, _ := common.BuildUDPIP(net.ParseIP(source), net.ParseIP(target), 41499, 31452, 12, 111, []byte("server selected data"))
	data[10], data[11] = 0xab, 0xcd // client must not repair this checksum
	device := &common.Device{PcapName: "test", LocalIP: net.ParseIP(source)}
	discoverDevice = func(*net.UDPAddr) (*common.Device, error) { return device, nil }
	discoverPacketDevice = func(dst *net.UDPAddr) (*common.Device, error) {
		if dst.IP.String() != target {
			t.Fatal("wrong next hop target")
		}
		return device, nil
	}
	resolveLocalSource = func(dst string) (string, error) {
		if dst != target && dst != "127.0.0.1" {
			t.Fatal(dst)
		}
		return source, nil
	}
	sent := 0
	sendIPFrames = func(name string, frames [][]byte, interval time.Duration) (int, error) {
		if len(frames) != 3 || interval != 50*time.Millisecond {
			t.Fatal("send schedule changed")
		}
		for _, f := range frames {
			if !bytes.Equal(f, data) {
				t.Fatal("server bytes changed")
			}
			sent++
		}
		return len(frames), nil
	}
	executed := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/session" {
			var req common.MeasurementRequest
			json.NewDecoder(r.Body).Decode(&req)

			json.NewEncoder(w).Encode(common.SessionInfo{ID: 1, TokenHex: "abcd", Rounds: []string{"test"}, Encoding: common.IPPacketEncoding})
			return
		}
		var rep common.ClientReport
		json.NewDecoder(r.Body).Decode(&rep)
		if rep.LocalSourceIP != source {
			t.Error("private source was not reported in every next")
		}
		if !executed {
			executed = true
			json.NewEncoder(w).Encode(common.NextResponse{Op: "actions", Actions: []common.Action{{Op: "send", Encoding: common.IPPacketEncoding, Packets: []common.IPPacket{{ID: "p12", Data: data, Repeat: 3, IntervalMs: 50}}}}})
		} else {
			json.NewEncoder(w).Encode(common.NextResponse{Op: "finish", Result: &common.Result{ID: 1}})
		}
	}))
	defer srv.Close()
	Server = srv.URL
	if RunOnce(false, "4") == nil || sent != 3 {
		t.Fatal("measurement/send incomplete")
	}
}

func TestIPSendFailureIsReported(t *testing.T) {
	old, oldSend := packetDevices, sendIPFrames
	t.Cleanup(func() { packetDevices, sendIPFrames = old, oldSend })
	p, _ := common.BuildUDPIP(net.ParseIP("192.168.1.1"), net.ParseIP("198.51.100.1"), 1, 2, 3, 4, nil)
	packetDevices = map[string]*common.Device{"198.51.100.1": {}}
	sendIPFrames = func(string, [][]byte, time.Duration) (int, error) { return 0, fmt.Errorf("injection failed") }
	rep := &common.ClientReport{}
	doSendIP(common.Action{Encoding: common.IPPacketEncoding, Packets: []common.IPPacket{{ID: "failure", Data: p}}}, rep)
	if rep.SendErrors["failure"] == "" {
		t.Fatal("failure silently became a blocked probe")
	}
}
