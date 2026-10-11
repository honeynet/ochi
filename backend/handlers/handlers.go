package handlers

import (
	"io/fs"
	"net/http"

	"github.com/honeynet/ochi/backend/repos"
)

// Handlers holds the dependencies shared by the HTTP handlers.
type Handlers struct {
	FS         fs.FS
	JWTSecret  string
	HTTPClient *http.Client

	Users   *repos.UserRepo
	Queries *repos.QueryRepo
	Events  *repos.EventRepo
	Sensors *repos.SensorRepo

	// PublishFunc fans a message out to subscribers; it returns false when rate limited.
	PublishFunc func(msg []byte) bool
}
