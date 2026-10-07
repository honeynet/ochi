package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// OptionalJSON is a JSON value stored as NULL when absent. Unlike
// types.JSONText it does not turn a missing value into {}, so it can be
// omitted from responses with omitempty.
type OptionalJSON json.RawMessage

// MarshalJSON returns the raw JSON, or null when empty.
func (j OptionalJSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON stores a copy of data, treating null as absent.
func (j *OptionalJSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("OptionalJSON: UnmarshalJSON on nil pointer")
	}
	if string(data) == "null" {
		*j = nil
		return nil
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// Value returns NULL when empty, otherwise the validated JSON.
func (j OptionalJSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	if !json.Valid(j) {
		return nil, errors.New("OptionalJSON: invalid JSON")
	}
	return []byte(j), nil
}

// Scan stores src, leaving j empty for NULL.
func (j *OptionalJSON) Scan(src interface{}) error {
	switch t := src.(type) {
	case nil:
		*j = nil
	case string:
		*j = OptionalJSON(t)
	case []byte:
		*j = append((*j)[0:0], t...)
	default:
		return errors.New("OptionalJSON: incompatible type")
	}
	return nil
}
