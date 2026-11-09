package main

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
	"os"
	"path/filepath"
	pb "savt-client/savt-client-api/savt"
	"savt-client/savt-client-engine/common"
	"testing"
	"time"
)

func TestResultMappingFullASNPrefixAndToken(t *testing.T) {
	for _, asn := range []uint32{4538, 4000000000} {
		for _, prefix := range []int{0, 8, -1} {
			r := &common.Result{ClientMetadata: common.ClientMetadata{ClientASN: &asn}, SpoofPrefixMeasured: prefix >= 0, SpoofPrefixLen: &prefix, ReportToken: "report"}
			got := resultToMeasurementResult(r)
			if got.ClientAsn == nil || *got.ClientAsn != asn || int(got.SpoofablePrefixLength) != prefix {
				t.Fatalf("bad mapping %+v", got)
			}
			if asn < 2147483648 && got.Asn != int32(asn) || asn >= 2147483648 && got.Asn != 0 {
				t.Fatal("legacy ASN overflow")
			}
			if resultReportToken(nil, r) != "report" || resultReportToken(r, &common.Result{ReportToken: "other"}) != "" {
				t.Fatal("token mapping")
			}
		}
	}
	if m := resultToMeasurementResult(&common.Result{}); m.ClientAsn != nil || m.SpoofablePrefixLength != -1 {
		t.Fatal("unknown became known")
	}
}

func TestHistoryMetadataOverGRPCAndOldArchives(t *testing.T) {
	before := *dataDir
	*dataDir = t.TempDir()
	t.Cleanup(func() { *dataDir = before })
	_, history := daemonDirs()
	os.MkdirAll(history, 0700)
	asn := uint32(4000000000)
	prefix := 0
	start := time.Date(2026, 10, 15, 1, 17, 34, 0, time.UTC)
	rec := map[string]any{"token": "report-token", "start_time": start, "duration_seconds": 191, "status": 2, "result": &common.Result{ClientIP: "192.0.2.1", ClientMetadata: common.ClientMetadata{ClientASN: &asn, ASNSource: "ipinfo"}, SpoofPrefixMeasured: true, SpoofPrefixLen: &prefix}, "result6": nil}
	data, _ := json.Marshal(rec)
	os.WriteFile(filepath.Join(history, "new.json"), data, 0600)
	old := []byte(`{"status":2,"token":"old-token","start_time":{"seconds":1792027054},"duration_seconds":90,"job_type":1,"ipv4":{"client_address":"192.0.2.2","asn":4538,"spoofable_prefix_length":8}}`)
	os.WriteFile(filepath.Join(history, "old.json"), old, 0600)
	job := historyFileToJob(history, "new.json")
	if job.Token != "report-token" || job.StartTime == nil || !job.StartTime.AsTime().Equal(start) || job.DurationSeconds != 191 || job.ProgressBar != 100 || job.Ipv6 != nil {
		t.Fatalf("new history %+v", job)
	}
	older := historyFileToJob(history, "old.json")
	if older.Token != "old-token" || older.JobType != pb.Jobtype_SCHEDULED || older.Ipv4.GetAsn() != 4538 || older.Ipv4.GetSpoofablePrefixLength() != 8 {
		t.Fatalf("old history %+v", older)
	}
	// Read history through the actual protobuf service, with no measurement or scheduler.
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	pb.RegisterSavtIpcServer(server, &grpcServer{})
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///fixture", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	response, err := pb.NewSavtIpcClient(conn).GetHistory(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, got := range response.Jobs {
		if got.Token != "report-token" {
			continue
		}
		found = true
		if got.Ipv4.ClientAsn == nil || *got.Ipv4.ClientAsn != 4000000000 || got.Ipv4.SpoofablePrefixLength != 0 {
			t.Fatal("wire metadata lost")
		}
	}
	if !found {
		t.Fatal("history missing")
	}
	after, _ := os.ReadFile(filepath.Join(history, "old.json"))
	if string(after) != string(old) {
		t.Fatal("old archive rewritten")
	}
}
