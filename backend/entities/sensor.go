package entities

type Sensor struct {
	ID     string `json:"id" db:"id"`
	Name   string `json:"name" db:"name"`
	UserID string `json:"user_id" db:"user_id"`
}

type SensorStat struct {
	SensorID    string `json:"sensor_id" db:"sensor_id"`
	EventCount  int64  `json:"event_count" db:"event_count"`
	LastEventAt string `json:"last_event_at,omitempty" db:"last_event_at"`
}
