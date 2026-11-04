package entities

type Sensor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Token is the sensor's own secret. The sensor presents it to fetch its config and
	// rules, and (in a follow-up) to publish. It is scanned from the database by this json
	// tag, so handlers that return a Sensor over the API must clear it first — see
	// clearSecret — to keep it out of responses.
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// WithoutSecret returns a copy safe to serialise to an API client: the token is removed so
// it is never carried in a response body.
func (s Sensor) WithoutSecret() Sensor {
	s.Token = ""
	return s
}
