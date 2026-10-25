package repos

import (
	"path/filepath"
	"testing"

	"github.com/honeynet/ochi/backend/entities"

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

func TestUserRoles(t *testing.T) {
	db, err := sqlx.Connect("sqlite", filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	r, err := NewUserRepo(db)
	require.NoError(t, err)

	admin, err := r.Find("admin@test")
	require.NoError(t, err)
	plain, err := r.Find("plain@test")
	require.NoError(t, err)

	roles, err := r.Roles(admin.ID)
	require.NoError(t, err)
	require.Empty(t, roles)

	require.NoError(t, r.GrantRole(admin.ID, entities.RoleAdmin))
	require.NoError(t, r.GrantRole(admin.ID, entities.RoleAdmin), "granting twice is a no-op")
	require.Error(t, r.GrantRole(admin.ID, entities.Role("superuser")), "unknown roles are refused")

	roles, err = r.Roles(admin.ID)
	require.NoError(t, err)
	require.Equal(t, []entities.Role{entities.RoleAdmin}, roles)

	users, err := r.List()
	require.NoError(t, err)
	require.Equal(t, []entities.User{
		{ID: admin.ID, Email: "admin@test", Roles: []entities.Role{entities.RoleAdmin}},
		{ID: plain.ID, Email: "plain@test"},
	}, users)
}
