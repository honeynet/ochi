package repos

import (
	"encoding/json"

	"github.com/jmoiron/sqlx/types"
)

// totalFrameCount counts the frames of a decoded event including frames nested
// inside other frames (a socks tunnel read carries the http handler's frames
// under "http"). The sensor's frameCount covers only the top level, and list
// pages omit decoded, so the stored count is raised to the total. It returns
// fallback unchanged when nothing is nested or decoded is not a frame array.
func totalFrameCount(decoded types.JSONText, fallback *int) *int {
	var frames []map[string]json.RawMessage
	if len(decoded) == 0 || json.Unmarshal(decoded, &frames) != nil {
		return fallback
	}
	total, nested := countFrames(frames)
	if nested == 0 {
		return fallback
	}
	return &total
}

func countFrames(frames []map[string]json.RawMessage) (total, nested int) {
	total = len(frames)
	for _, frame := range frames {
		for _, raw := range frame {
			var inner []map[string]json.RawMessage
			if json.Unmarshal(raw, &inner) != nil || len(inner) == 0 || !allDirected(inner) {
				continue
			}
			t, _ := countFrames(inner)
			total += t
			nested += t
		}
	}
	return total, nested
}

func allDirected(frames []map[string]json.RawMessage) bool {
	for _, frame := range frames {
		var dir string
		if json.Unmarshal(frame["direction"], &dir) != nil {
			return false
		}
	}
	return true
}
