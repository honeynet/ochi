package entities

type User struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
	// Roles is not a column of the users table; it is filled from user_roles
	// where a caller needs it.
	Roles []Role `json:"roles,omitempty"`
}
