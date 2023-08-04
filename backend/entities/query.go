package entities

type Query struct {
	ID          string `json:"id,omitempty"`
	Content     string `json:"content,omitempty"`
	OwnerID     string `json:"owner_id,omitempty"`
	Active      bool   `json:"active,omitempty"`
	Description string `json:"description,omitempty"`
}
