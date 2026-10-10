package repos

import (
	"fmt"
	"testing"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/require"
)

const MOCK_USER_ID = "user1"

func initEventRepo(t *testing.T) *EventRepo {
	tmp := t.TempDir()
	dbPath := fmt.Sprintf("%s/test.db", tmp)
	db, err := sqlx.Connect("sqlite", dbPath)
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

	rule := "rule"
	handler := "handler"
	scanner := "scanner"
	event := entities.Event{
		OwnerID: MOCK_USER_ID,
		Payload: "payload",
		//ConnKey:   []int{1, 1},
		DstPort:   443,
		Rule:      &rule,
		Handler:   &handler,
		Transport: "tcp",
		Scanner:   &scanner,
		SensorID:  "sensorID",
		SrcHost:   "srcHost",
		SrcPort:   "srcPort",
		Timestamp: "2023-08-18T23:04:17+00:00",
		Decoded:   types.JSONText(`{"foo": 1, "bar": 2}`),
	}
	event, err = r.Create(event)
	require.NoError(t, err)
	require.NotEmpty(t, event.ID)

	saved, err := r.FindByOwnerId(MOCK_USER_ID)
	require.NoError(t, err)
	require.Equal(t, len(saved), 1)

	require.Equal(t, event, saved[0])
	require.Empty(t, saved[0].DstHost)
	require.Nil(t, saved[0].DurationMs)
	require.Nil(t, saved[0].FrameCount)
	require.Empty(t, saved[0].TLS)

	err = r.Delete("Not found")
	require.Error(t, err)

	err = r.Delete(saved[0].ID)
	require.NoError(t, err)

	u1, err = r.FindByOwnerId("123")
	require.NoError(t, err)
	require.Empty(t, u1)
}

func TestEvent_NewEnvelopeFields(t *testing.T) {
	r := initEventRepo(t)
	durationMs := 1500
	frameCount := 4
	rule := "Rule: SMB"
	handler := "smb"
	startedAt := "2026-01-01T00:00:00Z"
	srcPtr := "scanner.example."
	dstHost := "198.51.100.8"
	sensorVersion := "1.2.3"
	ruleName := "smb"
	payloadHash := "deadbeef"
	endReason := "timeout"
	event := entities.Event{
		OwnerID:       MOCK_USER_ID,
		Payload:       "cGF5bG9hZA==",
		DstPort:       445,
		Rule:          &rule,
		Handler:       &handler,
		Transport:     "tcp",
		SensorID:      "sensorID",
		SrcHost:       "203.0.113.10",
		SrcPort:       "54321",
		Timestamp:     "2026-01-01T00:00:00Z",
		Decoded:       types.JSONText(`[{"direction":"read","path":"IPC$"}]`),
		StartedAt:     &startedAt,
		DurationMs:    &durationMs,
		SrcPtr:        &srcPtr,
		DstHost:       &dstHost,
		SensorVersion: &sensorVersion,
		RuleName:      &ruleName,
		PayloadHash:   &payloadHash,
		FrameCount:    &frameCount,
		EndReason:     &endReason,
		TLS:           entities.OptionalJSON(`{"serverName":"example.com","alpn":["h2"],"cipher":""}`),
	}
	event, err := r.Create(event)
	require.NoError(t, err)

	got, err := r.GetByID(event.ID)
	require.NoError(t, err)
	require.Equal(t, event, got)
	require.NotNil(t, got.DstHost)
	require.Equal(t, "198.51.100.8", *got.DstHost)
	require.NotNil(t, got.EndReason)
	require.Equal(t, "timeout", *got.EndReason)
	require.NotNil(t, got.FrameCount)
	require.Equal(t, 4, *got.FrameCount)
	require.NotNil(t, got.DurationMs)
	require.Equal(t, 1500, *got.DurationMs)
	require.JSONEq(t, `{"serverName":"example.com","alpn":["h2"],"cipher":""}`, string(got.TLS))
}

func TestEventRepo_MigratesLegacySchema(t *testing.T) {
	tmp := t.TempDir()
	db, err := sqlx.Connect("sqlite", fmt.Sprintf("%s/legacy.db", tmp))
	require.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE events (
		id TEXT PRIMARY KEY NOT NULL
		, ownerID TEXT NOT NULL
		, payload TEXT NOT NULL
		, dstPort INTEGER NOT NULL
		, rule TEXT
		, handler TEXT
		, transport TEXT NOT NULL
		, scanner TEXT
		, sensorID TEXT NOT NULL
		, srcHost TEXT NOT NULL
		, srcPort TEXT NOT NULL
		, timestamp TEXT NOT NULL
		, decoded JSON
		, CONSTRAINT id_unique UNIQUE (id)
	)`)
	require.NoError(t, err)

	r, err := NewEventRepo(db)
	require.NoError(t, err)

	dstHost := "10.0.0.9"
	endReason := "client_close"
	event, err := r.Create(entities.Event{
		OwnerID:   MOCK_USER_ID,
		Payload:   "payload",
		DstPort:   80,
		Transport: "tcp",
		SensorID:  "sensorID",
		SrcHost:   "1.2.3.4",
		SrcPort:   "1",
		Timestamp: "2026-01-01T00:00:00Z",
		DstHost:   &dstHost,
		EndReason: &endReason,
	})
	require.NoError(t, err)

	got, err := r.GetByID(event.ID)
	require.NoError(t, err)
	require.NotNil(t, got.DstHost)
	require.Equal(t, "10.0.0.9", *got.DstHost)
	require.NotNil(t, got.EndReason)
	require.Equal(t, "client_close", *got.EndReason)
}

func newTestEvent(owner, ts string) entities.Event {
	return entities.Event{
		OwnerID:   owner,
		Payload:   "payload",
		DstPort:   80,
		Transport: "tcp",
		SensorID:  "sensorID",
		SrcHost:   "srcHost",
		SrcPort:   "srcPort",
		Timestamp: ts,
	}
}

func TestEvent_Paging(t *testing.T) {
	r := initEventRepo(t)
	for i := 1; i <= 5; i++ {
		_, err := r.Create(newTestEvent(MOCK_USER_ID, fmt.Sprintf("2026-01-0%dT00:00:00Z", i)))
		require.NoError(t, err)
	}
	_, err := r.Create(newTestEvent("other", "2026-01-09T00:00:00Z"))
	require.NoError(t, err)

	total, err := r.CountByOwnerId(MOCK_USER_ID)
	require.NoError(t, err)
	require.Equal(t, 5, total)

	first, err := r.FindPageByOwnerId(MOCK_USER_ID, 2, 0)
	require.NoError(t, err)
	require.Len(t, first, 2)
	require.Equal(t, "2026-01-05T00:00:00Z", first[0].Timestamp)

	last, err := r.FindPageByOwnerId(MOCK_USER_ID, 2, 4)
	require.NoError(t, err)
	require.Len(t, last, 1)
	require.Equal(t, "2026-01-01T00:00:00Z", last[0].Timestamp)
}

func TestEvent_DeleteByOwner(t *testing.T) {
	r := initEventRepo(t)
	mine1, err := r.Create(newTestEvent(MOCK_USER_ID, "2026-01-01T00:00:00Z"))
	require.NoError(t, err)
	mine2, err := r.Create(newTestEvent(MOCK_USER_ID, "2026-01-02T00:00:00Z"))
	require.NoError(t, err)
	theirs, err := r.Create(newTestEvent("other", "2026-01-03T00:00:00Z"))
	require.NoError(t, err)

	deleted, err := r.DeleteByOwner(MOCK_USER_ID, []string{mine1.ID, theirs.ID, "missing"})
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)

	left, err := r.FindByOwnerId(MOCK_USER_ID)
	require.NoError(t, err)
	require.Len(t, left, 1)
	require.Equal(t, mine2.ID, left[0].ID)

	_, err = r.GetByID(theirs.ID)
	require.NoError(t, err)

	deleted, err = r.DeleteByOwner(MOCK_USER_ID, nil)
	require.NoError(t, err)
	require.EqualValues(t, 0, deleted)
}
