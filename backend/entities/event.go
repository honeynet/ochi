package entities

import (
	"github.com/jmoiron/sqlx/types"
)

type Event struct {
	ID            string         `json:"id,omitempty"`
	OwnerID       string         `json:"ownerID,omitempty"`
	Payload       string         `json:"payload,omitempty"`
	DstPort       int            `json:"dstPort,omitempty"`   // the connection destination port
	Rule          string         `json:"rule,omitempty"`      // the rule that matched the connection
	Handler       string         `json:"handler,omitempty"`   // the processing handler
	Transport     string         `json:"transport,omitempty"` // the transport used
	Scanner       string         `json:"scanner,omitempty"`   // name of the scanner if detected
	SensorID      string         `json:"sensorID,omitempty"`  // the id of the sensor
	SrcHost       string         `json:"srcHost,omitempty"`   // the source IP address
	SrcPort       string         `json:"srcPort,omitempty"`   // the source port
	Timestamp     string         `json:"timestamp,omitempty"` // the UTC timestamp of the connection
	Decoded       types.JSONText `json:"decoded,omitempty"`   // a decoded version of the payload if available
	StartedAt     string         `json:"startedAt,omitempty"`
	DurationMs    *int           `json:"durationMs,omitempty"`
	SrcPtr        string         `json:"srcPtr,omitempty"`
	DstHost       string         `json:"dstHost,omitempty"`
	SensorVersion string         `json:"sensorVersion,omitempty"`
	RuleName      string         `json:"ruleName,omitempty"`
	PayloadHash   string         `json:"payloadHash,omitempty"`
	FrameCount    *int           `json:"frameCount,omitempty"`
	EndReason     string         `json:"endReason,omitempty"`
}
