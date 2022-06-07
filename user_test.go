package main

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUser(t *testing.T) {
	os.Remove("./test.db")
	db, err := sqlx.Connect("sqlite3", "./test.db")
	require.NoError(t, err)
	defer os.Remove("./test.db")
	r, err := newRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	u1, err := r.user("test@test")
	require.NoError(t, err)
	require.NotEmpty(t, u1)
	u2, err := r.user("test@test")
	require.NoError(t, err)
	require.NotEmpty(t, u2)
	require.Equal(t, u1, u2)
}
