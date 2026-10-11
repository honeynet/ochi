package handlers

import (
	"io/fs"
	"net/http"

	"github.com/honeynet/ochi/backend/repos"

	"github.com/go-chi/jwtauth/v5"
)

// Handlers holds the dependencies shared by the HTTP handlers.
type Handlers struct {
	FS         fs.FS
	TokenAuth  *jwtauth.JWTAuth
	HTTPClient *http.Client

	Users   *repos.UserRepo
	Queries *repos.QueryRepo
	Events  *repos.EventRepo
	Sensors *repos.SensorRepo

	// PublishFunc fans a message out to subscribers; it returns false when rate limited.
	PublishFunc func(msg []byte) bool
}
