package repos

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUser(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlx.Connect("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	r, err := NewUserRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	u1, err := r.Find("test@test")
	require.NoError(t, err)
	require.NotEmpty(t, u1)
	u2, err := r.Find("test@test")
	require.NoError(t, err)
	require.NotEmpty(t, u2)
	require.Equal(t, u1, u2)
}
