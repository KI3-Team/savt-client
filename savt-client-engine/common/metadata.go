package common

// ClientMetadata is a server-provided snapshot. The client does not infer it.
type ClientMetadata struct {
	ClientASN      *uint32 `json:"clientASN"`
	ASNSource      string  `json:"asnSource,omitempty"`
	ASNDataVersion string  `json:"asnDataVersion,omitempty"`
}
