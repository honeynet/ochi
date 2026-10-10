package repos

import (
	"fmt"
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

	found, err := r.Exists(id)
	require.NoError(t, err)
	require.False(t, found)

	err = r.AddSensors(entities.Sensor{ID: id, Name: "test", UserID: "test@test"})
	require.NoError(t, err)

	found, err = r.Exists(id)
	require.NoError(t, err)
	require.True(t, found)

	// A truncated uuid is not a registered sensor.
	found, err = r.Exists(id[:8])
	require.NoError(t, err)
	require.False(t, found)
}

func initRepoForSensors(t *testing.T) *SensorRepo {
	tmp := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmp)

	db, err := sqlx.Connect("sqlite", dbPath)
	require.NoError(t, err)
	// Release the handle so t.TempDir() can remove the file on Windows.
	t.Cleanup(func() { _ = db.Close() })

	r, err := NewSensorRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}
