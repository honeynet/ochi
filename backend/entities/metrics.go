package entities

type Metric struct {
	ID       string `json:"id,omitempty"`
	DstPort  int    `json:"dst_port,omitempty"`
	Count    int    `json:"count,omitempty"`
	LastSeen string `json:"last_seen,omitempty"`
}
