package widgets

import (
	"strconv"

	pb "savt-client/savt-client-api/savt"
)

// FormatResultASN supports new four-octet values and older worker/history data.
func FormatResultASN(result *pb.MeasurementResult) string {
	if result == nil {
		return "-"
	}
	if result.ClientAsn != nil {
		value := *result.ClientAsn
		if value == 0 || value == ^uint32(0) {
			return "-"
		}
		return strconv.FormatUint(uint64(value), 10)
	}
	if result.Asn <= 0 {
		return "-"
	}
	return strconv.FormatInt(int64(result.Asn), 10)
}
