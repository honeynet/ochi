package backend

import (
	"io/fs"
	"net/http"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/honeynet/ochi/backend/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/jwtauth/v5"
)

func newRouter(cs *server) (*chi.Mux, error) {
	tokenAuth := entities.NewTokenAuth(cs.cfg.JWTSecret)

	h := &handlers.Handlers{
		FS:          cs.fs,
		TokenAuth:   tokenAuth,
		HTTPClient:  cs.httpClient,
		Users:       cs.uRepo,
		Queries:     cs.queryRepo,
		Events:      cs.eventRepo,
		Sensors:     cs.sensorRepo,
		PublishFunc: cs.publish,

		RequireRegisteredSensors: cs.cfg.RequireRegisteredSensors,
		AdminEmails:              cs.cfg.AdminEmails,
	}

	r := chi.NewRouter()
	// CORS is useful when running backend and frontend on different ports.
	// TODO: possibly make AllowedOrigins more restrictive by using a
	// configurable list of hosts instead of *.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet, http.MethodHead, http.MethodPost,
			http.MethodPut, http.MethodPatch, http.MethodDelete,
		},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		ExposedHeaders: []string{"X-Total-Count"},
	}))

	// static
	r.Get("/", h.Index)
	r.Get("/global.css", h.CSS)
	r.Get("/myqueries", h.Index)
	r.Get("/myevents", h.Index)
	r.Get("/mysensors", h.Index)
	r.Get("/users", h.Index)
	r.Get("/events/{id}", h.Index)

	build, err := fs.Sub(cs.fs, "build")
	if err != nil {
		return nil, err
	}
	r.Get("/build/*", http.StripPrefix("/build", http.FileServer(http.FS(build))).ServeHTTP)

	r.Route("/api", func(r chi.Router) {
		// Public. Everything here guards itself: the publish token, the Google
		// login exchange, or an unguessable id in the path.
		r.Get("/subscribe", cs.subscribeHandler)
		r.Post("/publish", handlers.TokenMiddleware(h.Publish, cs.cfg.PublishToken))
		r.Post("/login", h.Login)
		// Shared event links are unguessable IDs and must work without login.
		r.Get("/events/{id}", h.GetEventByID)
		// Unauthenticated: guarded by the unguessable sensor id in the path plus the owner's user_id.
		r.Get("/download/{sensor_id}", h.DownloadBinary)

		// Everything below needs a valid session token. Verifier decodes it onto
		// the request context, Authenticator turns a missing or bad one into a 401
		// before any handler runs.
		r.Group(func(r chi.Router) {
			r.Use(jwtauth.Verifier(tokenAuth))
			r.Use(handlers.Authenticator)

			// user
			r.Get("/session", h.Session)

			// query
			r.Get("/queries", h.GetQueries)
			r.Post("/queries", h.CreateQuery)
			r.Patch("/queries/{id}", h.UpdateQuery)
			r.Delete("/queries/{id}", h.DeleteQuery)

			// event
			r.Post("/events", h.CreateEvent)
			r.Delete("/events", h.DeleteEvents)
			r.Delete("/events/{id}", h.DeleteEvent)
			r.Get("/events", h.GetEvents)

			// sensor
			r.Get("/sensors", h.GetSensorsByUser)
			r.Post("/sensors", h.AddSensor)
			r.Delete("/sensors/{id}", h.DeleteSensor)

			// admin
			r.With(h.RequirePermission(entities.PermListUsers)).Get("/users", h.ListUsers)
		})
	})

	return r, nil
}
