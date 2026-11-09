package widgets

import (
	pb "savt-client/savt-client-api/savt"
	"testing"
)

func TestASNDisplayAndPrefixZero(t *testing.T) {
	for _, tc := range []struct {
		result *pb.MeasurementResult
		want   string
	}{
		{nil, "-"}, {&pb.MeasurementResult{}, "-"}, {&pb.MeasurementResult{Asn: -1}, "-"}, {&pb.MeasurementResult{Asn: 4538}, "4538"},
		{&pb.MeasurementResult{ClientAsn: asnPointer(4000000000)}, "4000000000"}, {&pb.MeasurementResult{Asn: 4538, ClientAsn: asnPointer(0)}, "-"},
	} {
		if got := FormatResultASN(tc.result); got != tc.want {
			t.Fatalf("got %s want %s", got, tc.want)
		}
	}
	if formatSpoofablePrefixLength(0) != "0" || formatSpoofablePrefixLength(-1) != "-" {
		t.Fatal("prefix /0 confused with unavailable")
	}
}
func asnPointer(v uint32) *uint32 { return &v }
