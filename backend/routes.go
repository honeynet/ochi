package backend

import (
	"io/fs"
	"net/http"

	"github.com/honeynet/ochi/backend/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func newRouter(cs *server) (*chi.Mux, error) {
	h := &handlers.Handlers{
		FS:         cs.fs,
		JWTSecret:  cs.cfg.JWTSecret,
		HTTPClient: cs.httpClient,
		Users:      cs.uRepo,
		Queries:    cs.queryRepo,
		Events:     cs.eventRepo,
		Sensors:    cs.sensorRepo,
		Publish:    cs.publish,
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
	r.Get("/", h.IndexHandler)
	r.Get("/global.css", h.CSSHandler)
	r.Get("/myqueries", h.IndexHandler)
	r.Get("/myevents", h.IndexHandler)
	r.Get("/events/{id}", h.IndexHandler)

	build, err := fs.Sub(cs.fs, "build")
	if err != nil {
		return nil, err
	}
	r.Get("/build/*", http.StripPrefix("/build", http.FileServer(http.FS(build))).ServeHTTP)

	// websocket
	r.Get("/subscribe", cs.subscribeHandler)
	r.Post("/publish", handlers.TokenMiddleware(h.PublishHandler, cs.cfg.PublishToken))

	// user
	r.Post("/login", h.LoginHandler)
	r.Get("/session", handlers.BearerMiddleware(h.SessionHandler, cs.cfg.JWTSecret))

	// query
	r.Get("/queries", handlers.BearerMiddleware(h.GetQueriesHandler, cs.cfg.JWTSecret))
	r.Post("/queries", handlers.BearerMiddleware(h.CreateQueryHandler, cs.cfg.JWTSecret))
	r.Patch("/queries/{id}", handlers.BearerMiddleware(h.UpdateQueryHandler, cs.cfg.JWTSecret))
	r.Delete("/queries/{id}", handlers.BearerMiddleware(h.DeleteQueryHandler, cs.cfg.JWTSecret))

	// event
	r.Post("/api/events", handlers.BearerMiddleware(h.CreateEventHandler, cs.cfg.JWTSecret))
	r.Delete("/api/events", handlers.BearerMiddleware(h.DeleteEventsHandler, cs.cfg.JWTSecret))
	r.Delete("/api/events/{id}", handlers.BearerMiddleware(h.DeleteEventHandler, cs.cfg.JWTSecret))
	r.Get("/api/events", handlers.BearerMiddleware(h.GetEventsHandler, cs.cfg.JWTSecret))
	// Shared event links are unguessable IDs and must work without login.
	r.Get("/api/events/{id}", h.GetEventByIDHandler)

	// sensor
	r.Get("/sensors", handlers.BearerMiddleware(h.GetSensorsByUser, cs.cfg.JWTSecret))
	r.Post("/sensors", handlers.BearerMiddleware(h.AddSensor, cs.cfg.JWTSecret))
	r.Get("/download/{os}/{arch}", handlers.BearerMiddleware(h.DownloadBinaryHandler, cs.cfg.JWTSecret))

	return r, nil
}
