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

	db, err := sqlx.Connect("sqlite3", dbPath)
	require.NoError(t, err)

	// defer os.Remove("./querytest.db")
	r, err := NewSensorRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}
