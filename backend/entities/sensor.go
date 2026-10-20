package entities

type Sensor struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	UserID string `json:"user_id"`
}

type SensorStat struct {
	SensorID    string `json:"sensor_id"`
	EventCount  int64  `json:"event_count"`
	LastEventAt string `json:"last_event_at,omitempty"`
}
