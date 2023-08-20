package repos

import (
	"fmt"
	"strings"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "github.com/mattn/go-sqlite3"
)

type QueryRepo struct {
	findByOwnerQuery *sqlx.Stmt
	deleteQuery      *sqlx.Stmt
	createQuery      *sqlx.NamedStmt
	updateQuery      *sqlx.Stmt
	getByIdQuery     *sqlx.Stmt
	db               *sqlx.DB
}

// NewQueryRepo creates a query repo
func NewQueryRepo(db *sqlx.DB) (*QueryRepo, error) {
	r := &QueryRepo{}
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS queries (
		id TEXT PRIMARY KEY NOT NULL
		, content TEXT NOT NULL
		, owner_id TEXT NOT NULL
		, active INTEGER NOT NULL
		, description TEXT NOT NULL
		, CONSTRAINT id_unique UNIQUE (id)
	)`)
	if err != nil {
		return nil, err
	}
	r.findByOwnerQuery, err = db.Preparex("SELECT * FROM queries WHERE owner_id=?")
	if err != nil {
		return nil, err
	}
	r.createQuery, err = db.PrepareNamed(`INSERT INTO queries
			(id, owner_id, content, active, description)
			VALUES
			(:id, :owner_id, :content, :active, :description)`)
	if err != nil {
		return nil, err
	}
	r.updateQuery, err = db.Preparex(
		`UPDATE queries
		SET content=?, active=?, description=?
		WHERE id=?`)
	if err != nil {
		return nil, err
	}
	r.getByIdQuery, err = db.Preparex("SELECT * FROM queries WHERE id=?")
	if err != nil {
		return nil, err
	}
	r.deleteQuery, err = db.Preparex("DELETE FROM queries WHERE id=?")
	if err != nil {
		return nil, err
	}
	return r, nil
}

// FindByOwnerId returns a list of queries by owner ID
func (r *QueryRepo) FindByOwnerId(ownerId string) ([]entities.Query, error) {
	qs := []entities.Query{}
	err := r.findByOwnerQuery.Select(&qs, ownerId)
	return qs, err
}

// Create creates a new query
func (r *QueryRepo) Create(owner_id, content, description string, active bool) (entities.Query, error) {
	id, err := uuid.NewRandom()
	q := entities.Query{}
	if err != nil {
		return q, err
	}
	q = entities.Query{ID: id.String(), Content: content, OwnerID: owner_id, Active: active, Description: description}
	_, err = r.createQuery.Exec(&q)
	if err != nil {
		return q, err
	}
	return q, nil
}

// Update updates an existing query
func (r *QueryRepo) Update(id, content, description string, active bool) error {
	res, err := r.updateQuery.Exec(content, active, description, id)
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

// GetByID finds a single query by given ID
func (r *QueryRepo) GetByID(id string) (entities.Query, error) {
	q := entities.Query{}
	err := r.getByIdQuery.Get(&q, id)
	return q, err
}

// Delete removes a query by ID
func (r *QueryRepo) Delete(id string) error {
	res, err := r.deleteQuery.Exec(id)
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
func (r *QueryRepo) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
