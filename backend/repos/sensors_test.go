package repos

import (
	"fmt"
	"sync"
	"testing"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestGetSensors(t *testing.T) {
	r := initRepoForSensors(t)

	sensors, err := r.GetSensorsByOwnerId("test@test")
	require.NoError(t, err)
	require.Empty(t, sensors)

	err = r.AddSensors(entities.Sensor{
		ID:     "123",
		Name:   "test",
		UserID: "test@test",
	})
	require.NoError(t, err)

	sensors, err = r.GetSensorsByOwnerId("test@test")
	require.NoError(t, err)
	require.NotEmpty(t, sensors)
}

func TestSensorExists(t *testing.T) {
	r := initRepoForSensors(t)
	const id = "abcd1234-ffff-ffff-ffff-ffffffffffff"

	require.False(t, r.Exists(id))

	err := r.AddSensors(entities.Sensor{ID: id, Name: "test", UserID: "test@test"})
	require.NoError(t, err)
	require.True(t, r.Exists(id))

	// A truncated uuid is not a registered sensor.
	require.False(t, r.Exists(id[:8]))
}

func TestSensorExists_FollowsDeletes(t *testing.T) {
	r := initRepoForSensors(t)
	const id = "abcd1234-ffff-ffff-ffff-ffffffffffff"
	require.NoError(t, r.AddSensors(entities.Sensor{ID: id, Name: "test", UserID: "owner"}))

	// Somebody else trying to delete it removes no row, so it must stay registered.
	deleted, err := r.DeleteByOwner(id, "not-the-owner")
	require.NoError(t, err)
	require.Zero(t, deleted)
	require.True(t, r.Exists(id))

	deleted, err = r.DeleteByOwner(id, "owner")
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	require.False(t, r.Exists(id))
}

func TestSensorExists_FailedInsertDoesNotRegister(t *testing.T) {
	r := initRepoForSensors(t)
	require.NoError(t, r.AddSensors(entities.Sensor{ID: "first", Name: "dup", UserID: "owner"}))

	// (user_id, name) is unique, so this INSERT fails and must not register "second".
	require.Error(t, r.AddSensors(entities.Sensor{ID: "second", Name: "dup", UserID: "owner"}))
	require.False(t, r.Exists("second"))
}

func TestSensorExists_LoadsExistingSensors(t *testing.T) {
	db := openSensorTestDB(t)
	first, err := NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, first.AddSensors(entities.Sensor{ID: "persisted", Name: "test", UserID: "owner"}))

	// A repo opened later, as after a restart, starts out knowing the stored sensors.
	restarted, err := NewSensorRepo(db)
	require.NoError(t, err)
	require.True(t, restarted.Exists("persisted"))
}

func TestSensorExists_DoesNotQueryTheDatabase(t *testing.T) {
	db := openSensorTestDB(t)
	r, err := NewSensorRepo(db)
	require.NoError(t, err)
	require.NoError(t, r.AddSensors(entities.Sensor{ID: "registered", Name: "test", UserID: "owner"}))

	// Exists runs for every published event. With the database gone it still answers,
	// so it cannot be issuing a query.
	require.NoError(t, db.Close())
	require.True(t, r.Exists("registered"))
	require.False(t, r.Exists("unknown"))
}

func TestSensorExists_ConcurrentWithWrites(t *testing.T) {
	r := initRepoForSensors(t)
	const n = 50

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("sensor-%d", i)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = r.AddSensors(entities.Sensor{ID: id, Name: id, UserID: "owner"})
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				r.Exists(id)
			}
		}()
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		require.True(t, r.Exists(fmt.Sprintf("sensor-%d", i)))
	}
}

func openSensorTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Connect("sqlite", fmt.Sprintf("%s/test.db", t.TempDir()))
	require.NoError(t, err)
	// Release the handle so t.TempDir() can remove the file on Windows.
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func initRepoForSensors(t *testing.T) *SensorRepo {
	r, err := NewSensorRepo(openSensorTestDB(t))
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}
