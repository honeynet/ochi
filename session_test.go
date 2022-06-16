package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSession(t *testing.T) {
	secret := "test"
	tokenString, err := NewToken(secret)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)
	valid, err := ValidateToken(tokenString, secret)
	require.NoError(t, err)
	require.True(t, valid)
}
