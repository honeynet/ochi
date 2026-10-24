package repos

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/honeynet/ochi/backend/entities"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "modernc.org/sqlite"
)

type UserRepo struct {
	readEmail *sqlx.Stmt
	readUser  *sqlx.Stmt
	create    *sqlx.NamedStmt
	readRoles *sqlx.Stmt
	grantRole *sqlx.Stmt
	listUsers *sqlx.Stmt
	listRoles *sqlx.Stmt
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
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS user_roles (
		user_id TEXT NOT NULL
		, role TEXT NOT NULL
		, PRIMARY KEY (user_id, role)
		, FOREIGN KEY (user_id) REFERENCES users(id)
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
	if err != nil {
		return nil, err
	}
	r.readRoles, err = db.Preparex("SELECT role FROM user_roles WHERE user_id=? ORDER BY role")
	if err != nil {
		return nil, err
	}
	r.grantRole, err = db.Preparex("INSERT OR IGNORE INTO user_roles (user_id, role) VALUES (?, ?)")
	if err != nil {
		return nil, err
	}
	r.listUsers, err = db.Preparex("SELECT id, email FROM users ORDER BY email")
	if err != nil {
		return nil, err
	}
	r.listRoles, err = db.Preparex("SELECT user_id, role FROM user_roles ORDER BY role")
	if err != nil {
		return nil, err
	}
	return r, nil
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

// List returns every user with their roles, ordered by email.
func (r *UserRepo) List() ([]entities.User, error) {
	users := []entities.User{}
	if err := r.listUsers.Select(&users); err != nil {
		return nil, err
	}
	var grants []struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := r.listRoles.Select(&grants); err != nil {
		return nil, err
	}
	byUser := make(map[string][]entities.Role, len(grants))
	for _, g := range grants {
		byUser[g.UserID] = append(byUser[g.UserID], entities.Role(g.Role))
	}
	for i := range users {
		users[i].Roles = byUser[users[i].ID]
	}
	return users, nil
}

// Roles returns the roles assigned to a user. A user without roles gets an
// empty slice, not an error.
func (r *UserRepo) Roles(userID string) ([]entities.Role, error) {
	var roles []entities.Role
	if err := r.readRoles.Select(&roles, userID); err != nil {
		return nil, err
	}
	return roles, nil
}

// GrantRole assigns role to a user. Granting a role the user already holds is a
// no-op.
func (r *UserRepo) GrantRole(userID string, role entities.Role) error {
	if !entities.ValidRole(role) {
		return fmt.Errorf("unknown role %q", role)
	}
	_, err := r.grantRole.Exec(userID, role)
	return err
}

// Close the db
func (r *UserRepo) Close() {
	if r.db != nil {
		r.db.Close()
	}
}
