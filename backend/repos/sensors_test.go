package repos

import (
	"fmt"
	"math"
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

func TestSensorRepo_StatsPersistence(t *testing.T) {
	r := initRepoForSensors(t)

	s1 := "s-11111111"
	s2 := "s-22222222"

	// Initial batch upsert
	deltas := map[string]int64{
		s1: 5,
		s2: 10,
	}
	lastSeen := map[string]string{
		s1: "2026-10-12T12:00:00Z",
		s2: "2026-10-12T12:05:00Z",
	}
	require.NoError(t, r.BatchUpsertStats(deltas, lastSeen))

	// LoadAllStats
	stats, err := r.LoadAllStats()
	require.NoError(t, err)
	require.Len(t, stats, 2)

	statsMap := make(map[string]entities.SensorStat)
	for _, s := range stats {
		statsMap[s.SensorID] = s
	}
	require.Equal(t, int64(5), statsMap[s1].EventCount)
	require.Equal(t, "2026-10-12T12:00:00Z", statsMap[s1].LastEventAt)
	require.Equal(t, int64(10), statsMap[s2].EventCount)
	require.Equal(t, "2026-10-12T12:05:00Z", statsMap[s2].LastEventAt)

	// Second batch upsert adds deltas
	deltas2 := map[string]int64{
		s1: 3,
	}
	lastSeen2 := map[string]string{
		s1: "2026-10-12T12:10:00Z",
	}
	require.NoError(t, r.BatchUpsertStats(deltas2, lastSeen2))

	stats, err = r.LoadAllStats()
	require.NoError(t, err)
	statsMap = make(map[string]entities.SensorStat)
	for _, s := range stats {
		statsMap[s.SensorID] = s
	}
	require.Equal(t, int64(8), statsMap[s1].EventCount)
	require.Equal(t, "2026-10-12T12:10:00Z", statsMap[s1].LastEventAt)

	// DeleteStat
	require.NoError(t, r.DeleteStat(s1))
	stats, err = r.LoadAllStats()
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Equal(t, s2, stats[0].SensorID)
}

func TestSensorRepo_StatsBoundaryAndConstraint(t *testing.T) {
	r := initRepoForSensors(t)

	const sBoundary = "s-boundary-max"

	// Upsert large int64 boundary value (within SQLite signed 64-bit INTEGER range)
	const largeCount int64 = math.MaxInt64 / 2
	require.NoError(t, r.BatchUpsertStats(map[string]int64{sBoundary: largeCount}, map[string]string{sBoundary: "2026-10-12T12:00:00Z"}))

	stats, err := r.LoadAllStats()
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Equal(t, largeCount, stats[0].EventCount)

	// SQLite check constraint verifies non-negative counts
	_, err = r.db.Exec(`INSERT INTO sensor_stats (sensor_id, event_count) VALUES ('invalid-neg', -1)`)
	require.Error(t, err, "negative counts must be rejected by check constraint")
}
