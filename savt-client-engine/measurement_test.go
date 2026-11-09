package engine

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"savt-client/savt-client-engine/common"
)

func TestMeasurementClientDualStack(t *testing.T) {
	old4, old6, oldDiscover, oldResolve := Server, Server6, discoverDevice, resolveLocalSource
	oldMetadata := sessionMetadata
	var metadataFamilies []string
	SetSessionMetadata(func(f string, info common.SessionInfo) {
		if info.ClientASN == nil || *info.ClientASN != 4000000000 {
			t.Error("missing creation ASN")
		}
		metadataFamilies = append(metadataFamilies, f)
	})
	t.Cleanup(func() { SetSessionMetadata(oldMetadata) })
	t.Cleanup(func() { Server, Server6, discoverDevice, resolveLocalSource = old4, old6, oldDiscover, oldResolve })
	discoverDevice = func(*net.UDPAddr) (*common.Device, error) { return &common.Device{}, nil }
	resolveLocalSource = func(string) (string, error) { return "192.168.31.166", nil }
	var requests []common.MeasurementRequest
	makeController := func(family string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/session":
				var req common.MeasurementRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				requests = append(requests, req)
				json.NewEncoder(w).Encode(common.SessionInfo{ClientMetadata: common.ClientMetadata{ClientASN: func() *uint32 { v := uint32(4000000000); return &v }(), ASNSource: "ipinfo", ASNDataVersion: "sha256:snapshot"}, ID: 12, TokenHex: "abcd", MeasurementID: req.MeasurementID})
			case "/api/session/12/next":
				var req common.ClientReport
				json.NewDecoder(r.Body).Decode(&req)
				if req.LocalSourceIP != "192.168.31.166" {
					t.Error("source missing")
				}
				receipt := common.MeasurementReceipt{MeasurementID: requests[0].MeasurementID}
				if family == "2001:db8::1" {
					receipt.Closed = true
					receipt.SessionID = 12345
					receipt.ReportToken = "canonical-ipv4-token"
				}
				json.NewEncoder(w).Encode(common.NextResponse{Op: "complete", Result: &common.Result{ID: 12, ClientIP: family}, Receipt: &receipt})
			default:
				t.Error("unexpected request (including standalone finish): " + r.URL.Path)
				http.Error(w, "unexpected", 404)
			}
		}))
	}
	v4 := makeController("192.0.2.1")
	defer v4.Close()
	v6 := makeController("2001:db8::1")
	defer v6.Close()
	Server, Server6 = v4.URL, v6.URL
	m, err := NewMeasurement([]string{"4", "6"})
	if err != nil {
		t.Fatal(err)
	}
	r4, r6 := m.RunOnce(false, "4"), m.RunOnce(false, "6")
	if len(metadataFamilies) != 2 || metadataFamilies[0] != "IPv4" || metadataFamilies[1] != "IPv6" {
		t.Fatal("creation callbacks missing")
	}
	if r4 == nil || r6 == nil {
		t.Fatal("missing result")
	}
	if r4.ClientASN == nil || r6.ClientASN == nil || *r4.ClientASN != 4000000000 || r6.ASNDataVersion != "sha256:snapshot" {
		t.Fatal("creation snapshot fallback missing")
	}
	receipt, err := m.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if receipt.SessionID != 12345 || r4.SessionID != r6.SessionID || r4.SessionID != 12345 {
		t.Fatalf("IDs %+v %+v", r4, r6)
	}
	if receipt.ReportToken != "canonical-ipv4-token" || r4.ReportToken != receipt.ReportToken || r6.ReportToken != receipt.ReportToken {
		t.Fatal("report token not applied to both results")
	}
	if len(requests) != 2 || requests[0].MeasurementID != requests[1].MeasurementID || requests[0].MeasurementToken != requests[1].MeasurementToken {
		t.Fatal("different job identities")
	}
	other, _ := NewMeasurement([]string{"4", "6"})
	if other.ID == m.ID || other.token == m.token {
		t.Fatal("job identity reused")
	}
}

func TestMeasurementFinishRetryAndPartial(t *testing.T) {
	m, _ := NewMeasurement([]string{"4", "6"})
	m.status["4"] = "failed"
	m.status["6"] = "completed"
	result := &common.Result{}
	m.results = append(m.results, result)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "lost response", 500)
			return
		}
		var req common.ClientReport
		json.NewDecoder(r.Body).Decode(&req)
		if req.MeasurementStatus["4"] != "failed" || req.MeasurementStatus["6"] != "completed" {
			t.Error("partial status lost")
		}
		json.NewEncoder(w).Encode(common.NextResponse{Op: "complete", Receipt: &common.MeasurementReceipt{Closed: true, MeasurementID: m.ID, SessionID: 99, ReportToken: "ipv6-report-token"}})
	}))
	defer srv.Close()
	m.endpoints["6"] = sessionEndpoint{Server: srv.URL, ID: 12, Source: "2001:db8::1"}
	if _, err := m.Finish(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || result.SessionID != 99 || result.ReportToken != "ipv6-report-token" {
		t.Fatal("receipt not retried/applied")
	}
}

func TestMeasurementRejectsUngroupedController(t *testing.T) {
	old4, old6 := Server, Server6
	t.Cleanup(func() { Server, Server6 = old4, old6 })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"id":1,"tokenHex":"abcd"}`) }))
	defer srv.Close()
	Server, Server6 = srv.URL, ""
	m, _ := NewMeasurement([]string{"4"})
	if r := m.RunOnce(false, "4"); r != nil {
		t.Fatal("claimed success without grouping support")
	}
	if _, err := m.Finish(); err == nil {
		t.Fatal("fabricated final ID")
	}
}
