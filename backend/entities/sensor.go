package entities

type Sensor struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	OwnerId string `json:"ownerid,omitempty"`
}
