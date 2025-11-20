package entities

type Tag struct {
    Type  string `json:"type"`
    Value string `json:"value"`
}

type Query struct {
	ID          string `json:"id,omitempty"`
	Content     string `json:"content,omitempty"`
	OwnerID     string `json:"owner_id,omitempty"`
	Active      bool   `json:"active"` // TODO: currently unused
	Description string `json:"description,omitempty"`
	Tags []Tag `json:"tags" db:"tags"`
}
