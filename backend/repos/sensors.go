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
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sensors(
		id TEXT PRIMARY KEY NOT NULL 
		, ownerid TEXT NOT NULL
		, name TEXT NOT NULL

	)`)

	if err != nil {
		return nil, err
	}

	r.getSensorsByUser, err = db.Preparex(` SELECT * sensors 
		WHERE 
		WHERE ownerid=?
	`)

	if err != nil {

		return nil, err
	}

	r.addSensor, err = db.PrepareNamed(`
		INSERT INTO sensors (id , ownerid , name) VALUES (:id , :ownerid , :name)
	`)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *SensorRepo) GetSensorsByOwnerId(ownerId string) ([]entities.Sensor, error) {

	ss := []entities.Sensor{}

	err := r.getSensorsByUser.Select(ownerId)

	return ss, err

}

func (r *SensorRepo) AddSensors(sensor entities.Sensor, userId string) error {

	s := entities.Sensor{Id: sensor.Id, OwnerId: userId, Name: sensor.Name}

	_, err := r.addSensor.Exec(s)

	if err != nil {
		return err
	}

	return nil

}
