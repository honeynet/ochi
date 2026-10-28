package repos

import (
	"strings"
	"sync"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

type SensorRepo struct {
	getSensorsByUser *sqlx.Stmt
	getSensor        *sqlx.Stmt
	addSensor        *sqlx.NamedStmt
	deleteSensor     *sqlx.Stmt

	getAllStats *sqlx.Stmt
	deleteStat  *sqlx.Stmt
	db          *sqlx.DB

	// registered holds every sensor ID in the table so Exists can answer the publish
	// path without a query. mu is held for writing across each INSERT and DELETE, so
	// Exists never observes the set and the table disagreeing.
	mu         sync.RWMutex
	registered map[string]struct{}
}

func NewSensorRepo(db *sqlx.DB) (*SensorRepo, error) {
	r := &SensorRepo{
		db: db,
	}
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sensors (
		id TEXT PRIMARY KEY NOT NULL 
		, user_id TEXT NOT NULL
		, name TEXT NOT NULL
		, UNIQUE (user_id, name)
		, FOREIGN KEY (user_id) REFERENCES users(id)
	)`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sensor_stats (
		sensor_id TEXT PRIMARY KEY NOT NULL
		, event_count INTEGER NOT NULL DEFAULT 0 CHECK(event_count >= 0)
		, last_event_at TEXT
	)`)
	if err != nil {
		return nil, err
	}

	r.getSensorsByUser, err = db.Preparex(`
	SELECT * FROM sensors 
		WHERE 
		user_id=?
	`)
	if err != nil {
		return nil, err
	}

	r.getSensor, err = db.Preparex(`
	SELECT * FROM sensors
		WHERE
		id=? AND user_id=?
	`)
	if err != nil {
		return nil, err
	}

	r.addSensor, err = db.PrepareNamed(`
		INSERT INTO sensors (id , user_id , name) VALUES (:id , :user_id , :name)
	`)
	if err != nil {
		return nil, err
	}

	r.deleteSensor, err = db.Preparex(`DELETE FROM sensors WHERE id=? AND user_id=?`)
	if err != nil {
		return nil, err
	}

	r.getAllStats, err = db.Preparex(`SELECT sensor_id, event_count, last_event_at FROM sensor_stats`)
	if err != nil {
		return nil, err
	}

	r.deleteStat, err = db.Preparex(`DELETE FROM sensor_stats WHERE sensor_id=?`)
	if err != nil {
		return nil, err
	}

	var ids []string
	if err := db.Select(&ids, `SELECT id FROM sensors`); err != nil {
		return nil, err
	}
	r.registered = make(map[string]struct{}, len(ids))
	for _, id := range ids {
		r.registered[id] = struct{}{}
	}

	return r, nil
}

func (r *SensorRepo) GetSensorsByOwnerId(ownerId string) ([]entities.Sensor, error) {
	var ss []entities.Sensor
	if err := r.getSensorsByUser.Select(&ss, ownerId); err != nil {
		return nil, err
	}
	return ss, nil
}

func (r *SensorRepo) AddSensors(sensor entities.Sensor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, err := r.addSensor.Exec(sensor); err != nil {
		return err
	}
	r.registered[sensor.ID] = struct{}{}
	return nil
}

// GetSensor returns the sensor with the given id, only if it belongs to ownerId.
func (r *SensorRepo) GetSensor(id, ownerId string) (entities.Sensor, error) {
	var s entities.Sensor
	if err := r.getSensor.Get(&s, id, ownerId); err != nil {
		return entities.Sensor{}, err
	}
	return s, nil
}

// DeleteByOwner removes the sensor only when userID owns it and reports how many
// rows went. Ownership is part of the statement, so a sensor that does not exist
// and one belonging to somebody else both report 0 and stay indistinguishable to
// the caller.
func (r *SensorRepo) DeleteByOwner(id, userID string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.deleteSensor.Exec(id, userID)
	if err != nil {
		return 0, err
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	// Only a delete that removed a row may unregister the ID. A non-owner's attempt
	// removes nothing and must leave the owner's sensor able to publish.
	if deleted > 0 {
		delete(r.registered, id)
		_, _ = r.deleteStat.Exec(id)
	}
	return deleted, nil
}

// Exists reports whether a sensor with the given UUID is registered. It takes the
// full UUID, not the truncated form published to subscribers, and answers from
// memory: it runs for every published event, so it must not query the database.
func (r *SensorRepo) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.registered[id]
	return ok
}

// LoadAllStats loads all persisted sensor statistics from SQLite.
func (r *SensorRepo) LoadAllStats() ([]entities.SensorStat, error) {
	var rows []entities.SensorStat
	if err := r.getAllStats.Select(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// BatchUpsertStats commits a batch of deltas into SQLite within a single transaction.
func (r *SensorRepo) BatchUpsertStats(deltas map[string]int64, lastSeen map[string]string) error {
	if len(deltas) == 0 {
		return nil
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareNamed(`
		INSERT INTO sensor_stats (sensor_id, event_count, last_event_at)
		VALUES (:sensor_id, :event_count, :last_event_at)
		ON CONFLICT(sensor_id) DO UPDATE SET
			event_count = event_count + excluded.event_count,
			last_event_at = excluded.last_event_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for id, delta := range deltas {
		if delta <= 0 {
			continue
		}
		stat := entities.SensorStat{
			SensorID:    id,
			EventCount:  delta,
			LastEventAt: lastSeen[id],
		}
		if _, err := stmt.Exec(stat); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteStat removes the stats row for a sensor.
func (r *SensorRepo) DeleteStat(id string) error {
	_, err := r.deleteStat.Exec(id)
	return err
}
