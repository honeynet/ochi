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

func initRepoForSensors(t *testing.T) *SensorRepo {
	tmp := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmp)

	db, err := sqlx.Connect("sqlite", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	r, err := NewSensorRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}

func TestSensorRepo_Stats(t *testing.T) {
	r := initRepoForSensors(t)

	// Increment stats
	s1 := "s-11111111"
	stat1 := r.IncrementStat(s1, "2026-10-11T12:00:00Z")
	require.Equal(t, int64(1), stat1.EventCount)
	require.Equal(t, s1, stat1.SensorID)

	// Increment again
	stat1 = r.IncrementStat(s1, "2026-10-11T12:01:00Z")
	require.Equal(t, int64(2), stat1.EventCount)

	// GetStats
	stats := r.GetStatsForSensors([]string{s1, "unknown"})
	require.Equal(t, int64(2), stats[s1].EventCount)
	require.Equal(t, int64(0), stats["unknown"].EventCount)

	// Pop dirty for broadcast
	dirty := r.PopDirtyStatsForBroadcast()
	require.Len(t, dirty, 1)
	require.Equal(t, s1, dirty[0].SensorID)

	// Pop again should be empty
	require.Empty(t, r.PopDirtyStatsForBroadcast())

	// Flush to database
	require.NoError(t, r.FlushStats())

	// Check persistent row exists
	var row entities.SensorStat
	require.NoError(t, r.getAllStats.Get(&row))
	require.Equal(t, s1, row.SensorID)
	require.Equal(t, int64(2), row.EventCount)
}
