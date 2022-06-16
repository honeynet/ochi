package main

import (
	"database/sql"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
}

type userRepo struct {
	read   *sqlx.Stmt
	create *sqlx.NamedStmt
	db     *sqlx.DB
}

func newUserRepo(db *sqlx.DB) (*userRepo, error) {
	r := &userRepo{}
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
	r.read, err = db.Preparex("SELECT * FROM users WHERE email=?")
	if err != nil {
		return nil, err
	}
	r.create, err = db.PrepareNamed("INSERT INTO users (id, email) VALUES (:id, :email)")
	return r, err
}

func (r *userRepo) get(email string) (User, error) {
	u := User{
		Email: email,
	}
	err := r.read.Get(&u, email)
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

func (r *userRepo) close() {
	if r.db != nil {
		r.db.Close()
	}
}
