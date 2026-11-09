package engine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"time"

	"savt-client/savt-client-engine/common"
)

// Measurement owns one client job, including both address families. Engine
// execution remains sequential; the controller owns aggregation and storage.
type Measurement struct {
	isPublic  *bool
	ID        string
	token     string
	families  []string
	status    map[string]string
	endpoints map[string]sessionEndpoint
	receipt   common.MeasurementReceipt
	results   []*common.Result
}

func NewMeasurement(families []string) (*Measurement, error) {
	unique := map[string]bool{}
	for _, f := range families {
		if f != "4" && f != "6" {
			return nil, fmt.Errorf("invalid family %q", f)
		}
		unique[f] = true
	}
	fs := []string{}
	for f := range unique {
		fs = append(fs, f)
	}
	sort.Strings(fs)
	if len(fs) == 0 {
		return nil, fmt.Errorf("empty measurement")
	}
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	m := &Measurement{ID: hex.EncodeToString(b[:16]), token: hex.EncodeToString(b[16:]), families: fs, status: map[string]string{}, endpoints: map[string]sessionEndpoint{}}
	for _, f := range fs {
		m.status[f] = "skipped"
	}
	return m, nil
}

// NewMeasurementWithSharing snapshots the existing client preference for one job.
func NewMeasurementWithSharing(families []string, isPublic bool) (*Measurement, error) {
	m, err := NewMeasurement(families)
	if err != nil {
		return nil, err
	}
	m.isPublic = &isPublic
	return m, nil
}

// SessionRequest is the metadata submitted for each address family. Each call
// returns independent values so a caller cannot mutate the job's preference.
func (m *Measurement) SessionRequest() common.MeasurementRequest {
	var sharing *bool
	if m.isPublic != nil {
		v := *m.isPublic
		sharing = &v
	}
	return common.MeasurementRequest{IsPublic: sharing, MeasurementID: m.ID, MeasurementToken: m.token, Families: append([]string(nil), m.families...)}
}
func (m *Measurement) request() common.MeasurementRequest { return m.SessionRequest() }

func (m *Measurement) RunOnce(verbose bool, family string) *common.Result {
	if _, ok := m.status[family]; !ok {
		log.Printf("measurement family %s was not declared", family)
		return nil
	}
	m.status[family] = "failed"
	res := runOnce(verbose, family, m)
	if res != nil && res.Status == "failed" {
		log.Printf("measurement has IP send failures: %v", res.SendErrors)
		return nil
	}
	if res != nil && res.Status != "failed" {
		m.status[family] = "completed"
		m.results = append(m.results, res)
		m.applyReceipt(m.receipt)
	}
	return res
}

type sessionEndpoint struct {
	Server string
	ID     int64
	Source string
	Round  int64
	Routes map[string]string
}

func (m *Measurement) applyReceipt(receipt common.MeasurementReceipt) {
	if receipt.MeasurementID != m.ID {
		return
	}
	m.receipt = receipt
	for _, res := range m.results {
		res.MeasurementID = m.ID
		res.SessionID = receipt.SessionID
		res.ReportToken = receipt.ReportToken
	}
}

// Finalize only reports outstanding failed/skipped families through /next.
// On the normal success path, the last round has already closed the group.
func (m *Measurement) Finalize() (common.MeasurementReceipt, error) {
	if m.receipt.Closed {
		m.applyReceipt(m.receipt)
		return m.receipt, nil
	}
	keys := []string{}
	for family := range m.endpoints {
		keys = append(keys, family)
	}
	sort.Strings(keys)
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		for _, f := range keys {
			endpoint := m.endpoints[f]
			if endpoint.Source == "" {
				continue
			}
			body := common.ClientReport{Round: endpoint.Round, LocalSourceIP: endpoint.Source, RouteSources: endpoint.Routes, MeasurementToken: m.token, MeasurementStatus: m.status}
			var response common.NextResponse
			err := postNext(endpoint, body, &response)
			if err == nil && response.Receipt != nil && response.Receipt.Closed {
				m.applyReceipt(*response.Receipt)
				if len(m.results) > 0 && m.receipt.SessionID <= 0 {
					return m.receipt, fmt.Errorf("invalid final session ID")
				}
				log.Printf("measurement %s automatically finalized: sessionId=%d status=%v", m.ID, m.receipt.SessionID, m.receipt.Status)
				return m.receipt, nil
			}
			if err == nil {
				err = fmt.Errorf("controller has not closed measurement")
			}
			lastErr = err
		}
		if attempt < 2 {
			time.Sleep(250 * time.Millisecond)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no available measurement session")
	}
	return common.MeasurementReceipt{}, fmt.Errorf("measurement %s finalization: %w", m.ID, lastErr)
}

// Kept as a Go API alias for older callers; it sends no /finish HTTP request.
func (m *Measurement) Finish() (common.MeasurementReceipt, error) { return m.Finalize() }
