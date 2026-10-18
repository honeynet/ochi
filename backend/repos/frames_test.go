package repos

import (
	"testing"

	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/require"
)

func TestTotalFrameCount(t *testing.T) {
	three := 3
	nested := types.JSONText(`[
		{"direction":"read","command":"socks4-connect"},
		{"direction":"write","status":"granted"},
		{"direction":"read","command":"tunnel","http":[
			{"direction":"read","command":"GET"},
			{"direction":"write","status":"200"}
		]}
	]`)
	got := totalFrameCount(nested, &three)
	require.NotNil(t, got)
	require.Equal(t, 5, *got)

	flat := types.JSONText(`[{"direction":"read","methods":[0,2]},{"direction":"write"}]`)
	require.Equal(t, &three, totalFrameCount(flat, &three), "number arrays are not frames")

	require.Equal(t, &three, totalFrameCount(types.JSONText(`{"payload":"x"}`), &three))
	require.Nil(t, totalFrameCount(nil, nil))
}
