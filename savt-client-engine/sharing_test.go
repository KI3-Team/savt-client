package engine

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"savt-client/savt-client-engine/common"
	"testing"
)

func TestSharingRequestPresenceAndSnapshot(t *testing.T) {
	for _, value := range []bool{false, true} {
		m, err := NewMeasurementWithSharing([]string{"4", "6"}, value)
		if err != nil {
			t.Fatal(err)
		}
		request := m.SessionRequest()
		data, _ := json.Marshal(request)
		var fields map[string]any
		json.Unmarshal(data, &fields)
		if got, exists := fields["isPublic"]; !exists || got != value {
			t.Fatalf("lost explicit %v", value)
		}
		*request.IsPublic = !value
		request.Families[0] = "bad"
		if *m.SessionRequest().IsPublic != value || m.SessionRequest().Families[0] != "4" {
			t.Fatal("request mutated job snapshot")
		}
	}
	old, _ := NewMeasurement([]string{"4"})
	data, _ := json.Marshal(old.SessionRequest())
	var fields map[string]any
	json.Unmarshal(data, &fields)
	if _, exists := fields["isPublic"]; exists {
		t.Fatal("absent legacy caller became known")
	}
}
func TestSharingAcknowledgementBeforePrimitives(t *testing.T) {
	oldServer, oldDiscover, oldResolve := Server, discoverDevice, resolveLocalSource
	defer func() { Server = oldServer; discoverDevice = oldDiscover; resolveLocalSource = oldResolve }()
	for _, echo := range []bool{false, true} {
		discovered := 0
		discoverDevice = func(*net.UDPAddr) (*common.Device, error) { discovered++; return &common.Device{}, nil }
		resolveLocalSource = func(string) (string, error) { return "192.0.2.1", nil }
		m, _ := NewMeasurementWithSharing([]string{"4"}, false)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/session":
				var req common.MeasurementRequest
				json.NewDecoder(r.Body).Decode(&req)
				if req.IsPublic == nil || *req.IsPublic {
					t.Error("worker lost false")
				}
				info := common.SessionInfo{ID: 1, MeasurementID: m.ID}
				if echo {
					info.IsPublic = req.IsPublic
				}
				json.NewEncoder(w).Encode(info)
			case "/api/session/1/next":
				json.NewEncoder(w).Encode(common.NextResponse{Op: "complete", Result: &common.Result{ID: 1}})
			default:
				t.Error("unexpected path")
			}
		}))
		Server = server.URL
		result := m.RunOnce(false, "4")
		server.Close()
		if echo && (result == nil || discovered != 1) {
			t.Fatal("acknowledged false failed")
		}
		if !echo && (result != nil || discovered != 0) {
			t.Fatal("unacknowledged preference reached primitives")
		}
	}
}
