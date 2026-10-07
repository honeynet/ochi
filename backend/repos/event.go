package repos

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "modernc.org/sqlite"
)

type EventRepo struct {
	createEvent      *sqlx.NamedStmt
	getByIdEvent     *sqlx.Stmt
	findByOwnerEvent *sqlx.Stmt
	deleteEvent      *sqlx.Stmt
	db               *sqlx.DB
}

var eventEnvelopeColumns = []string{
	"dstHost TEXT",
	"endReason TEXT",
	"frameCount INTEGER",
	"payloadHash TEXT",
	"sensorVersion TEXT",
	"startedAt TEXT",
	"durationMs INTEGER",
	"srcPtr TEXT",
	"ruleName TEXT",
	"tls JSON",
}

func addColumnIfMissing(db *sqlx.DB, table, columnDef string) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, columnDef))
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return err
}

// NewEventRepo creates an event repo and migrates the events table.
func NewEventRepo(db *sqlx.DB) (*EventRepo, error) {
	r := &EventRepo{}
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS events (
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
		, dstHost TEXT
		, endReason TEXT
		, frameCount INTEGER
		, payloadHash TEXT
		, sensorVersion TEXT
		, startedAt TEXT
		, durationMs INTEGER
		, srcPtr TEXT
		, ruleName TEXT
		, tls JSON
		, CONSTRAINT id_unique UNIQUE (id)
	)`)
	if err != nil {
		return nil, err
	}
	for _, col := range eventEnvelopeColumns {
		if err := addColumnIfMissing(db, "events", col); err != nil {
			return nil, err
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_events_payloadHash ON events (payloadHash)`); err != nil {
		return nil, err
	}
	r.createEvent, err = db.PrepareNamed(`INSERT INTO events
			(id, ownerID, payload, dstPort, rule, handler, transport, scanner, sensorID, srcHost, srcPort, timestamp, decoded, dstHost, endReason, frameCount, payloadHash, sensorVersion, startedAt, durationMs, srcPtr, ruleName, tls)
			VALUES
			(:id, :ownerID, :payload, :dstPort, :rule, :handler, :transport, :scanner, :sensorID, :srcHost, :srcPort, :timestamp, :decoded, :dstHost, :endReason, :frameCount, :payloadHash, :sensorVersion, :startedAt, :durationMs, :srcPtr, :ruleName, :tls)`)
	if err != nil {
		return nil, err
	}
	r.getByIdEvent, err = db.Preparex("SELECT * FROM events WHERE id=?")
	if err != nil {
		return nil, err
	}
	r.findByOwnerEvent, err = db.Preparex("SELECT * FROM events WHERE ownerID=?")
	if err != nil {
		return nil, err
	}
	r.deleteEvent, err = db.Preparex("DELETE FROM events WHERE id=?")
	if err != nil {
		return nil, err
	}
	return r, nil
}

// FindByOwnerId returns a list of events by owner ID
func (r *EventRepo) FindByOwnerId(ownerId string) ([]entities.Event, error) {
	evs := []entities.Event{}
	err := r.findByOwnerEvent.Select(&evs, ownerId)
	return evs, err
}

// Create creates a new event
func (r *EventRepo) Create(event entities.Event) (entities.Event, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return event, err
	}
	event.ID = id.String()
	_, err = r.createEvent.Exec(event)
	if err != nil {
		return event, err
	}
	return event, nil
}

// GetByID finds a single event by given ID
func (r *EventRepo) GetByID(id string) (entities.Event, error) {
	ev := entities.Event{}
	err := r.getByIdEvent.Get(&ev, id)
	return ev, err
}

// Delete removes an event by ID
func (r *EventRepo) Delete(id string) error {
	res, err := r.deleteEvent.Exec(id)
	if err != nil {
		return err
	}
	if cnt, err := res.RowsAffected(); err != nil {
		return err
	} else if cnt == 0 {
		return fmt.Errorf("%s not found", id)
	}
	return nil
}

// Close closes DB connection held by this repo.
func (r *EventRepo) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
