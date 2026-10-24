package backend

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pragmas set with db.Exec only reach one pooled connection. Holding several connections
// at once forces the pool to open new ones, which must be configured too.
func TestOpenSQLite_PragmasOnEveryConnection(t *testing.T) {
	db, err := openSQLite(filepath.Join(t.TempDir(), "data.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	conns := make([]*sql.Conn, 3)
	for i := range conns {
		conns[i], err = db.Conn(ctx)
		require.NoError(t, err)
	}
	for i, c := range conns {
		var busy, sync int
		var mode string
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy))
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&sync))
		require.NoError(t, c.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode))
		assert.Equal(t, 5000, busy, "connection %d busy_timeout", i)
		assert.Equal(t, 1, sync, "connection %d synchronous (1 = NORMAL)", i)
		assert.Equal(t, "wal", strings.ToLower(mode), "connection %d journal_mode", i)
		require.NoError(t, c.Close())
	}
}

// A write on one connection while another holds the write lock must wait for it instead
// of failing at once with SQLITE_BUSY.
func TestOpenSQLite_ConcurrentWriteWaitsForLock(t *testing.T) {
	db, err := openSQLite(filepath.Join(t.TempDir(), "data.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("CREATE TABLE t (x INTEGER)")
	require.NoError(t, err)

	ctx := context.Background()
	holder, err := db.Conn(ctx)
	require.NoError(t, err)
	defer holder.Close()
	writer, err := db.Conn(ctx)
	require.NoError(t, err)
	defer writer.Close()

	_, err = holder.ExecContext(ctx, "BEGIN IMMEDIATE")
	require.NoError(t, err)
	_, err = holder.ExecContext(ctx, "INSERT INTO t VALUES (1)")
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, err := writer.ExecContext(ctx, "INSERT INTO t VALUES (2)")
		done <- err
	}()

	time.Sleep(200 * time.Millisecond)
	_, err = holder.ExecContext(ctx, "COMMIT")
	require.NoError(t, err)

	select {
	case err := <-done:
		require.NoError(t, err, "the second writer should wait for the lock, not fail")
	case <-time.After(5 * time.Second):
		t.Fatal("the second writer never finished")
	}
	var n int
	require.NoError(t, db.Get(&n, "SELECT COUNT(*) FROM t"))
	assert.Equal(t, 2, n)
}
