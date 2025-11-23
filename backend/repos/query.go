package repos

import (
	"encoding/json"
	"fmt"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// internal struct to map DB columns
type queryDB struct {
	ID          string `db:"id"`
	OwnerID     string `db:"owner_id"`
	Content     string `db:"content"`
	Description string `db:"description"`
	Tags        string `db:"tags"`
	Active      bool   `db:"active"`
}

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
	r := &QueryRepo{db: db}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS queries (
        id TEXT PRIMARY KEY NOT NULL,
        content TEXT NOT NULL,
        owner_id TEXT NOT NULL,
        active INTEGER NOT NULL,
        description TEXT NOT NULL,
        tags TEXT DEFAULT '[]',
        CONSTRAINT id_unique UNIQUE (id)
	)`)
	if err != nil {
		return nil, err
	}

	r.findByOwnerQuery, err = db.Preparex("SELECT * FROM queries WHERE owner_id=?")
	if err != nil {
		return nil, err
	}
	r.createQuery, err = db.PrepareNamed(`INSERT INTO queries
    	(id, owner_id, content, active, description, tags)
    	VALUES
    	(:id, :owner_id, :content, :active, :description, :tags)`)
	if err != nil {
		return nil, err
	}
	r.updateQuery, err = db.Preparex(
		`UPDATE queries
		 SET content=?, active=?, description=?, tags=?
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
	rows := []queryDB{}
	err := r.findByOwnerQuery.Select(&rows, ownerId)
	if err != nil {
		return nil, err
	}

	qs := make([]entities.Query, len(rows))
	for i, row := range rows {
		var tags []entities.Tag
		if err := json.Unmarshal([]byte(row.Tags), &tags); err != nil {
			return nil, err
		}
		qs[i] = entities.Query{
			ID:          row.ID,
			OwnerID:     row.OwnerID,
			Content:     row.Content,
			Description: row.Description,
			Active:      row.Active,
			Tags:        tags,
		}
	}
	return qs, nil
}

// Create creates a new query
func (r *QueryRepo) Create(owner_id, content, description string, active bool, tags []entities.Tag) (entities.Query, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return entities.Query{}, err
	}

	q := entities.Query{
		ID:          id.String(),
		Content:     content,
		OwnerID:     owner_id,
		Active:      active,
		Description: description,
		Tags:        tags,
	}

	tagsJSON, err := json.Marshal(q.Tags)
	if err != nil {
		return q, err
	}

	qDB := queryDB{
		ID:          q.ID,
		OwnerID:     q.OwnerID,
		Content:     q.Content,
		Description: q.Description,
		Active:      q.Active,
		Tags:        string(tagsJSON),
	}

	_, err = r.createQuery.Exec(&qDB)
	if err != nil {
		return q, err
	}

	return q, nil
}

// Update updates an existing query
func (r *QueryRepo) Update(id, content, description string, active bool, tags []entities.Tag) error {
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return err
	}

	res, err := r.updateQuery.Exec(content, active, description, string(tagsJSON), id)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("%s not found", id)
	}
	return nil
}

// GetByID finds a single query by ID
func (r *QueryRepo) GetByID(id string) (entities.Query, error) {
	row := queryDB{}
	err := r.getByIdQuery.Get(&row, id)
	if err != nil {
		return entities.Query{}, err
	}

	var tags []entities.Tag
	if err := json.Unmarshal([]byte(row.Tags), &tags); err != nil {
		return entities.Query{}, err
	}

	return entities.Query{
		ID:          row.ID,
		OwnerID:     row.OwnerID,
		Content:     row.Content,
		Description: row.Description,
		Active:      row.Active,
		Tags:        tags,
	}, nil
}

// Delete removes a query by ID
func (r *QueryRepo) Delete(id string) error {
	res, err := r.deleteQuery.Exec(id)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("%s not found", id)
	}
	return nil
}

// Close closes DB connection
func (r *QueryRepo) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
