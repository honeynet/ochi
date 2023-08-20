package entities

type Event struct {
	ID        string      `json:"id,omitempty"`
	OwnerID   string      `json:"ownerID,omitempty"`
	Payload   string      `json:"payload,omitempty"`
	ConnKey   []int       `json:"connKey,omitempty"`   // identifier based on IP and source port
	DstPort   int         `json:"dstPort,omitempty"`   // the connection destination port
	Rule      string      `json:"rule,omitempty"`      // the rule that matched the connection
	Handler   string      `json:"handler,omitempty"`   // the processing handler
	Transport string      `json:"transport,omitempty"` // the transport used
	Scanner   string      `json:"scanner,omitempty"`   // name of the scanner if detected
	SensorID  string      `json:"sensorID,omitempty"`  // the id of the sensor
	SrcHost   string      `json:"srcHost,omitempty"`   // the source IP address
	SrcPort   string      `json:"srcPort,omitempty"`   // the source port
	Timestamp string      `json:"timestamp,omitempty"` // the UTC timestamp of the connection
	Decoded   interface{} `json:"decoded,omitempty"`   // a decoded version of the payload if available
}
