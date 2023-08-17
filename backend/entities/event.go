package entities

type Event struct {
	ID      string `json:"id,omitempty"`
	OwnerID string `json:"ownerID,omitempty"`
	Payload string `json:"payload,omitempty"`
	// TODO: find out how to work with int slice in sqlite
	// ConnKey   []int  `json:"connKey,omitempty"`   // identifier based on IP and source port
	DstPort   int    `json:"dstPort,omitempty"`   // the connection destination port
	Rule      string `json:"rule,omitempty"`      // the rule that matched the connection
	Handler   string `json:"handler,omitempty"`   // the processing handler
	Scanner   string `json:"scanner,omitempty"`   // name of the scanner if detected
	SensorID  string `json:"sensorID,omitempty"`  // the id of the sensor
	SrcHost   string `json:"srcHost,omitempty"`   // the source IP address
	SrcPort   string `json:"srcPort,omitempty"`   // the source port
	Timestamp string `json:"timestamp,omitempty"` // the UTC timestamp of the connection
	// TODO: decide how to store untyped decoded value in SQL
	// Decoded   interface{} `json:"decoded,omitempty"`   // a decoded version of the payload if available
}
