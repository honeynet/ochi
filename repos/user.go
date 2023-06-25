package repos

import (
	"database/sql"
	"strings"

	"github.com/honeynet/ochi/entities"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "github.com/mattn/go-sqlite3"
)

type UserRepo struct {
	readEmail *sqlx.Stmt
	readUser  *sqlx.Stmt
	create    *sqlx.NamedStmt
	db        *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) (*UserRepo, error) {
	r := &UserRepo{}
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY NOT NULL
		, email TEXT NOT NULL
		, CONSTRAINT id_unique UNIQUE (id)
		, CONSTRAINT email_unique UNIQUE (email)
	)`)
	if err != nil {
		return nil, err
	}
	r.readEmail, err = db.Preparex("SELECT * FROM users WHERE email=?")
	if err != nil {
		return nil, err
	}
	r.readUser, err = db.Preparex("SELECT * FROM users WHERE id=?")
	if err != nil {
		return nil, err
	}
	r.create, err = db.PrepareNamed("INSERT INTO users (id, email) VALUES (:id, :email)")
	return r, err
}

// Get a user
func (r *UserRepo) Get(id string) (entities.User, error) {
	u := entities.User{
		ID: id,
	}
	err := r.readUser.Get(&u, id)
	return u, err
}

// Find a user
func (r *UserRepo) Find(email string) (entities.User, error) {
	u := entities.User{Email: email}
	err := r.readEmail.Get(&u, email)
	if err == sql.ErrNoRows {
		var id uuid.UUID
		id, err = uuid.NewRandom()
		if err != nil {
			return u, err
		}
		u.ID = id.String()
		_, err = r.create.Exec(&u)
		if err != nil {
			return u, err
		}
	}
	return u, err
}

// Close the db
func (r *UserRepo) Close() {
	if r.db != nil {
		r.db.Close()
	}
}
