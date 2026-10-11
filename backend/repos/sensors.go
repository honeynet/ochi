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

	getAllStats  *sqlx.Stmt
	deleteStat   *sqlx.Stmt
	db           *sqlx.DB

	statsMu             sync.RWMutex
	statsTotals         map[string]int64
	statsLastSeen       map[string]string
	statsDeltas         map[string]int64
	statsDirtyBroadcast map[string]struct{}
}

func NewSensorRepo(db *sqlx.DB) (*SensorRepo, error) {
	r := &SensorRepo{
		db:                  db,
		statsTotals:         make(map[string]int64),
		statsLastSeen:       make(map[string]string),
		statsDeltas:         make(map[string]int64),
		statsDirtyBroadcast: make(map[string]struct{}),
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
		, event_count INTEGER NOT NULL DEFAULT 0
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
	if _, err := r.addSensor.Exec(sensor); err != nil {
		return err
	}
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
	res, err := r.deleteSensor.Exec(id, userID)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected > 0 {
		_, _ = r.deleteStat.Exec(id)
		r.statsMu.Lock()
		delete(r.statsTotals, id)
		delete(r.statsLastSeen, id)
		delete(r.statsDeltas, id)
		delete(r.statsDirtyBroadcast, id)
		r.statsMu.Unlock()
	}
	return affected, nil
}

// LoadStats loads previously persisted sensor statistics into the in-memory cache.
func (r *SensorRepo) LoadStats() error {
	var rows []entities.SensorStat
	if err := r.getAllStats.Select(&rows); err != nil {
		return err
	}
	r.statsMu.Lock()
	defer r.statsMu.Unlock()
	for _, row := range rows {
		r.statsTotals[row.SensorID] = row.EventCount
		r.statsLastSeen[row.SensorID] = row.LastEventAt
	}
	return nil
}

// IncrementStat records an event in memory and schedules it for batched flush and broadcast.
func (r *SensorRepo) IncrementStat(sensorID, eventTime string) entities.SensorStat {
	r.statsMu.Lock()
	defer r.statsMu.Unlock()

	r.statsTotals[sensorID]++
	r.statsDeltas[sensorID]++
	if eventTime != "" {
		r.statsLastSeen[sensorID] = eventTime
	}
	r.statsDirtyBroadcast[sensorID] = struct{}{}

	return entities.SensorStat{
		SensorID:    sensorID,
		EventCount:  r.statsTotals[sensorID],
		LastEventAt: r.statsLastSeen[sensorID],
	}
}

// GetStatsForSensors retrieves current stats for the requested sensor IDs from memory.
// It also matches by prefix when an event used a truncated 8-character ID or full UUID.
func (r *SensorRepo) GetStatsForSensors(sensorIDs []string) map[string]entities.SensorStat {
	r.statsMu.RLock()
	defer r.statsMu.RUnlock()

	res := make(map[string]entities.SensorStat, len(sensorIDs))
	for _, id := range sensorIDs {
		count := r.statsTotals[id]
		lastSeen := r.statsLastSeen[id]

		if count == 0 {
			if len(id) >= 8 {
				prefix := id[:8]
				if c, ok := r.statsTotals[prefix]; ok && c > 0 {
					count = c
					lastSeen = r.statsLastSeen[prefix]
				}
			}
			if count == 0 {
				for totalID, c := range r.statsTotals {
					if (strings.HasPrefix(totalID, id) || strings.HasPrefix(id, totalID)) && c > 0 {
						count += c
						if lastSeen == "" || r.statsLastSeen[totalID] > lastSeen {
							lastSeen = r.statsLastSeen[totalID]
						}
					}
				}
			}
		}

		res[id] = entities.SensorStat{
			SensorID:    id,
			EventCount:  count,
			LastEventAt: lastSeen,
		}
	}
	return res
}

// PopDirtyStatsForBroadcast returns all sensor stats modified since the last broadcast tick.
func (r *SensorRepo) PopDirtyStatsForBroadcast() []entities.SensorStat {
	r.statsMu.Lock()
	defer r.statsMu.Unlock()

	if len(r.statsDirtyBroadcast) == 0 {
		return nil
	}

	res := make([]entities.SensorStat, 0, len(r.statsDirtyBroadcast))
	for id := range r.statsDirtyBroadcast {
		res = append(res, entities.SensorStat{
			SensorID:    id,
			EventCount:  r.statsTotals[id],
			LastEventAt: r.statsLastSeen[id],
		})
	}
	r.statsDirtyBroadcast = make(map[string]struct{})
	return res
}

// FlushStats commits in-memory deltas into SQLite in a single transaction.
func (r *SensorRepo) FlushStats() error {
	r.statsMu.Lock()
	if len(r.statsDeltas) == 0 {
		r.statsMu.Unlock()
		return nil
	}

	pendingDeltas := r.statsDeltas
	r.statsDeltas = make(map[string]int64)
	pendingLastSeen := make(map[string]string, len(pendingDeltas))
	for id := range pendingDeltas {
		pendingLastSeen[id] = r.statsLastSeen[id]
	}
	r.statsMu.Unlock()

	tx, err := r.db.Beginx()
	if err != nil {
		r.revertDeltas(pendingDeltas)
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
		r.revertDeltas(pendingDeltas)
		return err
	}
	defer stmt.Close()

	for id, delta := range pendingDeltas {
		if delta == 0 {
			continue
		}
		stat := entities.SensorStat{
			SensorID:    id,
			EventCount:  delta,
			LastEventAt: pendingLastSeen[id],
		}
		if _, err := stmt.Exec(stat); err != nil {
			r.revertDeltas(pendingDeltas)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		r.revertDeltas(pendingDeltas)
		return err
	}

	return nil
}

func (r *SensorRepo) revertDeltas(pending map[string]int64) {
	r.statsMu.Lock()
	defer r.statsMu.Unlock()
	for id, delta := range pending {
		r.statsDeltas[id] += delta
	}
}
