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
		FS:          cs.fs,
		JWTSecret:   cs.cfg.JWTSecret,
		HTTPClient:  cs.httpClient,
		Users:       cs.uRepo,
		Queries:     cs.queryRepo,
		Events:      cs.eventRepo,
		Sensors:     cs.sensorRepo,
		PublishFunc: cs.publish,
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
	r.Get("/events/{id}", h.Index)

	build, err := fs.Sub(cs.fs, "build")
	if err != nil {
		return nil, err
	}
	r.Get("/build/*", http.StripPrefix("/build", http.FileServer(http.FS(build))).ServeHTTP)

	r.Route("/api", func(r chi.Router) {
		// websocket
		r.Get("/subscribe", cs.subscribeHandler)
		r.Post("/publish", handlers.TokenMiddleware(h.Publish, cs.cfg.PublishToken))

		// user
		r.Post("/login", h.Login)
		r.Get("/session", handlers.BearerMiddleware(h.Session, cs.cfg.JWTSecret))

		// query
		r.Get("/queries", handlers.BearerMiddleware(h.GetQueries, cs.cfg.JWTSecret))
		r.Post("/queries", handlers.BearerMiddleware(h.CreateQuery, cs.cfg.JWTSecret))
		r.Patch("/queries/{id}", handlers.BearerMiddleware(h.UpdateQuery, cs.cfg.JWTSecret))
		r.Delete("/queries/{id}", handlers.BearerMiddleware(h.DeleteQuery, cs.cfg.JWTSecret))

		// event
		r.Post("/events", handlers.BearerMiddleware(h.CreateEvent, cs.cfg.JWTSecret))
		r.Delete("/events", handlers.BearerMiddleware(h.DeleteEvents, cs.cfg.JWTSecret))
		r.Delete("/events/{id}", handlers.BearerMiddleware(h.DeleteEvent, cs.cfg.JWTSecret))
		r.Get("/events", handlers.BearerMiddleware(h.GetEvents, cs.cfg.JWTSecret))
		// Shared event links are unguessable IDs and must work without login.
		r.Get("/events/{id}", h.GetEventByID)

		// sensor
		r.Get("/sensors", handlers.BearerMiddleware(h.GetSensorsByUser, cs.cfg.JWTSecret))
		r.Post("/sensors", handlers.BearerMiddleware(h.AddSensor, cs.cfg.JWTSecret))
		// Unauthenticated: guarded by the unguessable sensor id in the path plus the owner's user_id.
		r.Get("/download/{sensor_id}", h.DownloadBinary)
	})

	return r, nil
}
