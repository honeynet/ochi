package main

import (
	"testing"

	"github.com/glaslos/ochi/entities"
	"github.com/stretchr/testify/require"
)

func TestSession(t *testing.T) {
	secret := "test"
	user := entities.User{ID: "test_id"}
	tokenString, err := NewToken(secret, user)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)
	claims, valid, err := ValidateToken(tokenString, secret)
	require.NoError(t, err)
	require.True(t, valid)
	require.Equal(t, user.ID, claims.UserID)
}
