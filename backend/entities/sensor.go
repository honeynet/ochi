package entities

type Sensor struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	User string `json:"user_id,omitempty"`
}
