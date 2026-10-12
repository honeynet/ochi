package entities

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTags(t *testing.T) {
	got, err := NormalizeTags([]Tag{
		{Type: " malware ", Value: "mirai\t"},
		{Type: "malware", Value: "mirai"}, // the same tag once trimmed
		{Type: "protocol", Value: "smb"},
	})
	require.NoError(t, err)
	assert.Equal(t, []Tag{{"malware", "mirai"}, {"protocol", "smb"}}, got)
}

func TestNormalizeTags_Empty(t *testing.T) {
	got, err := NormalizeTags(nil)
	require.NoError(t, err)
	assert.NotNil(t, got, "no tags must be an empty list, not null, in JSON")
	assert.Empty(t, got)
}

func TestNormalizeTags_Rejects(t *testing.T) {
	long := strings.Repeat("x", MaxTagLength+1)
	tooMany := make([]Tag, MaxTagsPerQuery+1)
	for i := range tooMany {
		tooMany[i] = Tag{Type: "t", Value: strings.Repeat("v", i+1)}
	}

	for name, tags := range map[string][]Tag{
		"empty type":      {{Type: "", Value: "v"}},
		"blank value":     {{Type: "t", Value: "   "}},
		"overlong type":   {{Type: long, Value: "v"}},
		"overlong value":  {{Type: "t", Value: long}},
		"too many tags":   tooMany,
		"one bad of many": {{Type: "ok", Value: "ok"}, {Type: "", Value: "v"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NormalizeTags(tags)
			assert.Error(t, err)
		})
	}
}

func TestNormalizeTags_LengthCountsCharactersNotBytes(t *testing.T) {
	// 64 three-byte characters is within the limit even though it is 192 bytes.
	_, err := NormalizeTags([]Tag{{Type: "t", Value: strings.Repeat("ü", MaxTagLength)}})
	assert.NoError(t, err)
}

func TestNormalizeTags_DuplicatesDoNotCountTowardsTheLimit(t *testing.T) {
	tags := make([]Tag, MaxTagsPerQuery+5)
	for i := range tags {
		tags[i] = Tag{Type: "t", Value: "same"}
	}
	got, err := NormalizeTags(tags)
	require.NoError(t, err)
	assert.Len(t, got, 1)
}
