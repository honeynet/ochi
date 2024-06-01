package repos

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/honeynet/ochi/backend/entities"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func initTestMetricsDB(t *testing.T) (*sqlx.DB, func()) {
	os.Remove("./metrics.db")
	db, err := sqlx.Connect("sqlite3", "./metrics.db")
	require.NoError(t, err)
	return db, func() {
		os.Remove("./metrics.db")
	}
}

func TestInsertMetrics(t *testing.T) {
	db, cleanup := initTestMetricsDB(t)
	defer cleanup()
	mdb, err := NewMetricsRepo(db)
	require.NoError(t, err)
	require.NotNil(t, mdb)
}

func TestGetMetric(t *testing.T) {
	db, cleanup := initTestMetricsDB(t)
	defer cleanup()
	mdb, err := NewMetricsRepo(db)
	require.NoError(t, err)
	require.NotNil(t, mdb)
	metric := entities.Metric{
		ID:       uuid.New().String(),
		DstPort:  80,
		Count:    1,
		LastSeen: time.Now().Format(time.RFC3339),
	}
	err = mdb.InsertMetric(metric)
	require.NoError(t, err)
	found, err := mdb.GetMetric(80)
	require.NoError(t, err)
	require.NotEmpty(t, found)
	require.Equal(t, metric, found)
}

func TestGetMetrics(t *testing.T) {
	db, cleanup := initTestMetricsDB(t)
	defer cleanup()
	mdb, err := NewMetricsRepo(db)
	require.NoError(t, err)
	require.NotNil(t, mdb)
	metric := entities.Metric{
		ID:       uuid.New().String(),
		DstPort:  80,
		Count:    1,
		LastSeen: time.Now().Format(time.RFC3339),
	}
	err = mdb.InsertMetric(metric)
	require.NoError(t, err)
	metric = entities.Metric{
		ID:       uuid.New().String(),
		DstPort:  443,
		Count:    1,
		LastSeen: time.Now().Format(time.RFC3339),
	}
	err = mdb.InsertMetric(metric)
	require.NoError(t, err)
	metrics, err := mdb.GetMetrics()
	require.NoError(t, err)
	require.NotEmpty(t, metrics)
	require.Equal(t, 2, len(metrics))
}
