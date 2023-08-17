package repos

import (
	"fmt"
	"testing"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const MOCK_USER_ID = "user1"

func initEventRepo(t *testing.T) *EventRepo {
	tmp := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmp)
	db, err := sqlx.Connect("sqlite3", dbPath)
	require.NoError(t, err)

	// defer os.Remove("./querytest.db")
	r, err := NewEventRepo(db)
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}

func TestEvent(t *testing.T) {
	r := initEventRepo(t)
	u1, err := r.FindByOwnerId("test@test")
	require.NoError(t, err)
	require.Empty(t, u1)

	q, err := r.Create("payload", MOCK_USER_ID, "rule",
		"handler", "scanner", "sensorID", "srcHost",
		"srcPort", "2023-08-18T23:04:17+00:00", 443)
	require.NoError(t, err)
	require.NotEmpty(t, q.ID)

	saved, err := r.FindByOwnerId(MOCK_USER_ID)
	require.NoError(t, err)
	require.Equal(t, len(saved), 1)

	require.Equal(t, entities.Event{
		ID:        saved[0].ID,
		OwnerID:   MOCK_USER_ID,
		Payload:   "payload",
		DstPort:   443,
		Rule:      "rule",
		Handler:   "handler",
		Scanner:   "scanner",
		SensorID:  "sensorID",
		SrcHost:   "srcHost",
		SrcPort:   "srcPort",
		Timestamp: "2023-08-18T23:04:17+00:00",
	}, saved[0])

	err = r.Delete("Not found")
	require.Error(t, err)

	err = r.Delete(saved[0].ID)
	require.NoError(t, err)

	u1, err = r.FindByOwnerId("123")
	require.NoError(t, err)
	require.Empty(t, u1)
}