package repos

import (
	"fmt"
	"strings"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "modernc.org/sqlite"
)

type QueryRepo struct {
	findByOwnerQuery *sqlx.Stmt
	deleteQuery      *sqlx.Stmt
	createQuery      *sqlx.NamedStmt
	updateQuery      *sqlx.Stmt
	getByIdQuery     *sqlx.Stmt
	tagsForQuery     *sqlx.Stmt
	tagsForOwner     *sqlx.Stmt
	// db is only used to start transactions. It is shared with the other repos and
	// owned by the server, which closes it on shutdown, so the repo never closes it.
	db *sqlx.DB
}

// NewQueryRepo creates a query repo
func NewQueryRepo(db *sqlx.DB) (*QueryRepo, error) {
	r := &QueryRepo{db: db}
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
	// Tags live in their own table, linked to the query they label. SQLite only
	// honours the foreign key when foreign_keys is enabled, which it is not here, so
	// Delete removes a query's tags itself rather than relying on the cascade.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS query_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT
		, query_id TEXT NOT NULL REFERENCES queries(id) ON DELETE CASCADE
		, type TEXT NOT NULL
		, value TEXT NOT NULL
		, UNIQUE (query_id, type, value)
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
	r.tagsForQuery, err = db.Preparex("SELECT type, value FROM query_tags WHERE query_id=? ORDER BY id")
	if err != nil {
		return nil, err
	}
	// One query for every tag of every query an owner has, rather than one per query.
	r.tagsForOwner, err = db.Preparex(`SELECT t.query_id, t.type, t.value
		FROM query_tags t JOIN queries q ON q.id = t.query_id
		WHERE q.owner_id=?
		ORDER BY t.id`)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// FindByOwnerId returns a list of queries by owner ID, each with its tags.
func (r *QueryRepo) FindByOwnerId(ownerId string) ([]entities.Query, error) {
	qs := []entities.Query{}
	if err := r.findByOwnerQuery.Select(&qs, ownerId); err != nil {
		return qs, err
	}

	var rows []struct {
		QueryID string `json:"query_id"`
		Type    string `json:"type"`
		Value   string `json:"value"`
	}
	if err := r.tagsForOwner.Select(&rows, ownerId); err != nil {
		return qs, err
	}
	byQuery := make(map[string][]entities.Tag, len(qs))
	for _, row := range rows {
		byQuery[row.QueryID] = append(byQuery[row.QueryID], entities.Tag{Type: row.Type, Value: row.Value})
	}
	for i := range qs {
		qs[i].Tags = byQuery[qs[i].ID]
		if qs[i].Tags == nil {
			qs[i].Tags = []entities.Tag{}
		}
	}
	return qs, nil
}

// Create creates a new query with its tags.
func (r *QueryRepo) Create(owner_id, content, description string, active bool, tags []entities.Tag) (entities.Query, error) {
	id, err := uuid.NewRandom()
	q := entities.Query{}
	if err != nil {
		return q, err
	}
	if tags == nil {
		tags = []entities.Tag{}
	}
	q = entities.Query{ID: id.String(), Content: content, OwnerID: owner_id, Active: active, Description: description, Tags: tags}
	err = r.inTx(func(tx *sqlx.Tx) error {
		if _, err := tx.NamedStmt(r.createQuery).Exec(&q); err != nil {
			return err
		}
		return insertTags(tx, q.ID, q.Tags)
	})
	return q, err
}

// Update updates an existing query. A nil tags leaves its tags as they are; any other
// value, including an empty slice, replaces them.
func (r *QueryRepo) Update(id, content, description string, active bool, tags []entities.Tag) error {
	return r.inTx(func(tx *sqlx.Tx) error {
		res, err := tx.Stmtx(r.updateQuery).Exec(content, active, description, id)
		if err != nil {
			return err
		}
		if cnt, err := res.RowsAffected(); err != nil {
			return err
		} else if cnt == 0 {
			return fmt.Errorf("%s not found", id)
		}
		if tags == nil {
			return nil
		}
		if _, err := tx.Exec("DELETE FROM query_tags WHERE query_id=?", id); err != nil {
			return err
		}
		return insertTags(tx, id, tags)
	})
}

// GetByID finds a single query by given ID, with its tags.
func (r *QueryRepo) GetByID(id string) (entities.Query, error) {
	q := entities.Query{}
	if err := r.getByIdQuery.Get(&q, id); err != nil {
		return q, err
	}
	tags := []entities.Tag{}
	if err := r.tagsForQuery.Select(&tags, id); err != nil {
		return q, err
	}
	q.Tags = tags
	return q, nil
}

// Delete removes a query by ID, together with its tags.
func (r *QueryRepo) Delete(id string) error {
	return r.inTx(func(tx *sqlx.Tx) error {
		if _, err := tx.Exec("DELETE FROM query_tags WHERE query_id=?", id); err != nil {
			return err
		}
		res, err := tx.Stmtx(r.deleteQuery).Exec(id)
		if err != nil {
			return err
		}
		if cnt, err := res.RowsAffected(); err != nil {
			return err
		} else if cnt == 0 {
			return fmt.Errorf("%s not found", id)
		}
		return nil
	})
}

// inTx runs fn in a transaction, committing only when it returns nil, so a query
// and its tags are always written together or not at all.
func (r *QueryRepo) inTx(fn func(*sqlx.Tx) error) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func insertTags(tx *sqlx.Tx, queryID string, tags []entities.Tag) error {
	for _, t := range tags {
		if _, err := tx.Exec("INSERT INTO query_tags (query_id, type, value) VALUES (?, ?, ?)", queryID, t.Type, t.Value); err != nil {
			return err
		}
	}
	return nil
}
