package entities

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Query struct {
	ID          string `json:"id,omitempty"`
	Content     string `json:"content,omitempty"`
	OwnerID     string `json:"owner_id,omitempty"`
	Active      bool   `json:"active"` // TODO: currently unused
	Description string `json:"description,omitempty"`
	Tags        []Tag  `json:"tags"`
}

// Tag labels a saved query with a type and a value, for example type "malware"
// and value "mirai".
type Tag struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

const (
	// MaxTagsPerQuery and MaxTagLength keep tags short labels rather than a place
	// to store arbitrary data.
	MaxTagsPerQuery = 20
	MaxTagLength    = 64
)

// NormalizeTags trims each tag's type and value and drops exact repeats, keeping
// the first occurrence in order. It rejects a tag with an empty or overlong part,
// and more than MaxTagsPerQuery distinct tags.
func NormalizeTags(tags []Tag) ([]Tag, error) {
	out := make([]Tag, 0, len(tags))
	seen := make(map[Tag]struct{}, len(tags))
	for _, t := range tags {
		t = Tag{Type: strings.TrimSpace(t.Type), Value: strings.TrimSpace(t.Value)}
		if t.Type == "" || t.Value == "" {
			return nil, fmt.Errorf("tag type and value must not be empty")
		}
		if utf8.RuneCountInString(t.Type) > MaxTagLength || utf8.RuneCountInString(t.Value) > MaxTagLength {
			return nil, fmt.Errorf("tag type and value must be at most %d characters", MaxTagLength)
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) > MaxTagsPerQuery {
		return nil, fmt.Errorf("a query can have at most %d tags", MaxTagsPerQuery)
	}
	return out, nil
}
