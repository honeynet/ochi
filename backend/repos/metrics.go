package repos

import (
	"strings"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

type MetricsRepo struct {
	insertMetric *sqlx.NamedStmt
	getMetric    *sqlx.Stmt
	getMetrics   *sqlx.Stmt
	db           *sqlx.DB
}

// NewMetricsRepo creates a metrics repo
func NewMetricsRepo(db *sqlx.DB) (*MetricsRepo, error) {
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	r := &MetricsRepo{
		db: db,
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS metrics (
		id TEXT PRIMARY KEY NOT NULL
		, dst_port INTEGER NOT NULL
		, count INTEGER NOT NULL
		, last_seen TEXT NOT NULL
	)`)
	if err != nil {
		return nil, err
	}
	r.insertMetric, err = db.PrepareNamed(`INSERT INTO metrics
			(id, dst_port, count, last_seen)
			VALUES
			(:id, :dst_port, :count, :last_seen)
			ON CONFLICT(id) DO UPDATE SET count = count + :count, last_seen = :last_seen
	`)
	if err != nil {
		return nil, err
	}
	r.getMetric, err = db.Preparex("SELECT * FROM metrics WHERE dst_port=?")
	if err != nil {
		return nil, err
	}

	r.getMetrics, err = db.Preparex("SELECT * FROM metrics")
	if err != nil {
		return nil, err
	}

	return r, nil
}

// InsertMetric inserts metrics
func (r *MetricsRepo) InsertMetric(metric entities.Metric) error {
	_, err := r.insertMetric.Exec(metric)
	return err
}

// GetMetric gets metrics
func (r *MetricsRepo) GetMetric(dstPort int) (entities.Metric, error) {
	var metric entities.Metric
	err := r.getMetric.Get(&metric, dstPort)
	return metric, err
}

// GetMetrics gets all metrics
func (r *MetricsRepo) GetMetrics() ([]entities.Metric, error) {
	var metrics []entities.Metric
	err := r.getMetrics.Select(&metrics)
	return metrics, err
}
