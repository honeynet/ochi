package entities

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionalJSON_OmittedWhenAbsent(t *testing.T) {
	var ev Event
	require.NoError(t, json.Unmarshal([]byte(`{"sensorID":"abc","tls":null}`), &ev))
	require.Empty(t, ev.TLS)

	out, err := json.Marshal(ev)
	require.NoError(t, err)
	require.NotContains(t, string(out), `"tls"`)

	v, err := ev.TLS.Value()
	require.NoError(t, err)
	require.Nil(t, v)
}

func TestOptionalJSON_RoundTrip(t *testing.T) {
	var ev Event
	require.NoError(t, json.Unmarshal([]byte(`{"tls":{"serverName":"example.com"}}`), &ev))

	out, err := json.Marshal(ev)
	require.NoError(t, err)
	require.JSONEq(t, `{"tls":{"serverName":"example.com"}}`, string(out))

	_, err = OptionalJSON(`{bad`).Value()
	require.Error(t, err)
}
