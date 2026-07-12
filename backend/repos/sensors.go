package repos

import (
	"strings"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

type SensorRepo struct {
	getSensorsByUser *sqlx.Stmt
	addSensor        *sqlx.NamedStmt
}

func NewSensorRepo(db *sqlx.DB) (*SensorRepo, error) {
	r := &SensorRepo{}
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

	r.getSensorsByUser, err = db.Preparex(`
	SELECT * FROM sensors 
		WHERE 
		user_id=?
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
