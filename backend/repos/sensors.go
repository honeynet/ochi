package repos

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/honeynet/ochi/backend/entities"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

type SensorRepo struct {
	getSensorsByUser *sqlx.Stmt
	getSensor        *sqlx.Stmt
	getSensorByToken *sqlx.Stmt
	addSensor        *sqlx.NamedStmt
	deleteSensor     *sqlx.Stmt

	// registered holds every sensor ID in the table so Exists can answer the publish
	// path without a query. mu is held for writing across each INSERT and DELETE, so
	// Exists never observes the set and the table disagreeing.
	mu         sync.RWMutex
	registered map[string]struct{}
}

func NewSensorRepo(db *sqlx.DB) (*SensorRepo, error) {
	r := &SensorRepo{}
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sensors (
		id TEXT PRIMARY KEY NOT NULL
		, user_id TEXT NOT NULL
		, name TEXT NOT NULL
		, token TEXT
		, UNIQUE (user_id, name)
		, FOREIGN KEY (user_id) REFERENCES users(id)
	)`)
	if err != nil {
		return nil, err
	}

	// A database created before per-sensor tokens has no token column; add it and give
	// every existing sensor a token, so each has the secret the config and rules
	// endpoints authenticate against. New databases already have the column from the
	// CREATE above, where the migration finds nothing to do.
	if err := migrateSensorTokens(db); err != nil {
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

	r.getSensorByToken, err = db.Preparex(`SELECT * FROM sensors WHERE token=?`)
	if err != nil {
		return nil, err
	}

	r.addSensor, err = db.PrepareNamed(`
		INSERT INTO sensors (id , user_id , name , token) VALUES (:id , :user_id , :name , :token)
	`)
	if err != nil {
		return nil, err
	}

	r.deleteSensor, err = db.Preparex(`DELETE FROM sensors WHERE id=? AND user_id=?`)
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

	// Every sensor gets a token at creation. A caller cannot choose it, the way it can
	// choose the id: the token is a credential, so it must be unguessable.
	if sensor.Token == "" {
		token, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		sensor.Token = token.String()
	}

	if _, err := r.addSensor.Exec(sensor); err != nil {
		return err
	}
	r.registered[sensor.ID] = struct{}{}
	return nil
}

// GetSensorByToken returns the sensor whose token matches, for authenticating a request
// made by the sensor itself. An unknown token returns the underlying no-rows error.
func (r *SensorRepo) GetSensorByToken(token string) (entities.Sensor, error) {
	var s entities.Sensor
	if err := r.getSensorByToken.Get(&s, token); err != nil {
		return entities.Sensor{}, err
	}
	return s, nil
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

// migrateSensorTokens adds the token column to an older sensors table and fills a token
// in for every sensor that has none, then enforces uniqueness. It is a no-op on a table
// that already has the column and tokens for all rows, so it is safe to run on every start.
func migrateSensorTokens(db *sqlx.DB) error {
	// Mapper maps result columns by json tag, not db tag, so scan PRAGMA rows into a map
	// keyed by column name rather than a tagged struct.
	rows, err := db.Queryx(`PRAGMA table_info(sensors)`)
	if err != nil {
		return fmt.Errorf("inspect sensors table: %w", err)
	}
	defer rows.Close()
	hasToken := false
	for rows.Next() {
		row := map[string]any{}
		if err := rows.MapScan(row); err != nil {
			return fmt.Errorf("inspect sensors table: %w", err)
		}
		if name, _ := row["name"].(string); name == "token" {
			hasToken = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("inspect sensors table: %w", err)
	}
	if !hasToken {
		if _, err := db.Exec(`ALTER TABLE sensors ADD COLUMN token TEXT`); err != nil {
			return fmt.Errorf("add token column: %w", err)
		}
	}

	var ids []string
	if err := db.Select(&ids, `SELECT id FROM sensors WHERE token IS NULL OR token = ''`); err != nil {
		return fmt.Errorf("find sensors without a token: %w", err)
	}
	for _, id := range ids {
		token, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		if _, err := db.Exec(`UPDATE sensors SET token = ? WHERE id = ?`, token.String(), id); err != nil {
			return fmt.Errorf("backfill token for sensor %s: %w", id, err)
		}
	}

	// Enforced after the backfill, so two NULL tokens in an older table do not collide
	// before they are filled in.
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_sensors_token ON sensors(token)`); err != nil {
		return fmt.Errorf("index sensor tokens: %w", err)
	}
	return nil
}
