package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession(t *testing.T) {
	ja := NewTokenAuth("test")
	user := User{ID: "test_id", Email: "test@example.com"}

	tokenString, err := NewToken(ja, user)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	token, err := ja.Decode(tokenString)
	require.NoError(t, err)

	var userID string
	require.NoError(t, token.Get("user_id", &userID))
	assert.Equal(t, user.ID, userID)

	exp, ok := token.Expiration()
	require.True(t, ok, "token must carry an expiry")
	assert.WithinDuration(t, time.Now().Add(TokenTTL), exp, time.Minute)
}

func TestSession_RejectsAnotherSecret(t *testing.T) {
	tokenString, err := NewToken(NewTokenAuth("test"), User{ID: "test_id"})
	require.NoError(t, err)

	_, err = NewTokenAuth("another-secret").Decode(tokenString)
	require.Error(t, err)
}
