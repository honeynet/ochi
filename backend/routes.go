package backend

import (
	"io/fs"
	"net/http"

	"github.com/honeynet/ochi/backend/handlers"

	"github.com/julienschmidt/httprouter"
)

func newRouter(cs *server) (*httprouter.Router, error) {
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

	r := httprouter.New()
	// Set CORS headers
	r.GlobalOPTIONS = http.HandlerFunc(handlers.CorsOptionsHandler)

	// static
	r.GET("/", h.IndexHandler)
	r.GET("/global.css", h.CSSHandler)
	r.GET("/myqueries", h.IndexHandler)
	r.GET("/myevents", h.IndexHandler)
	r.GET("/events/:id", h.IndexHandler)

	build, err := fs.Sub(cs.fs, "build")
	if err != nil {
		return nil, err
	}
	r.ServeFiles("/build/*filepath", http.FS(build))

	// websocket
	r.GET("/subscribe", cs.subscribeHandler)
	r.POST("/publish", handlers.TokenMiddleware(h.PublishHandler, cs.cfg.PublishToken))

	// user
	r.POST("/login", h.LoginHandler)
	r.GET("/session", handlers.CorsMiddleware(handlers.BearerMiddleware(h.SessionHandler, cs.cfg.JWTSecret)))

	// query
	// TODO: make CorsMiddleware more generic instead of specifying it on every handler.
	r.GET("/queries", handlers.CorsMiddleware(handlers.BearerMiddleware(h.GetQueriesHandler, cs.cfg.JWTSecret)))
	r.POST("/queries", handlers.CorsMiddleware(handlers.BearerMiddleware(h.CreateQueryHandler, cs.cfg.JWTSecret)))
	r.PATCH("/queries/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(h.UpdateQueryHandler, cs.cfg.JWTSecret)))
	r.DELETE("/queries/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(h.DeleteQueryHandler, cs.cfg.JWTSecret)))

	// event
	r.POST("/api/events", handlers.CorsMiddleware(handlers.BearerMiddleware(h.CreateEventHandler, cs.cfg.JWTSecret)))
	r.DELETE("/api/events", handlers.CorsMiddleware(handlers.BearerMiddleware(h.DeleteEventsHandler, cs.cfg.JWTSecret)))
	r.DELETE("/api/events/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(h.DeleteEventHandler, cs.cfg.JWTSecret)))
	r.GET("/api/events", handlers.CorsMiddleware(handlers.BearerMiddleware(h.GetEventsHandler, cs.cfg.JWTSecret)))
	// Shared event links are unguessable IDs and must work without login.
	r.GET("/api/events/:id", handlers.CorsMiddleware(h.GetEventByIDHandler))

	// sensor
	r.GET("/sensors", handlers.CorsMiddleware(handlers.BearerMiddleware(h.GetSensorsByUser, cs.cfg.JWTSecret)))
	r.POST("/sensors", handlers.CorsMiddleware(handlers.BearerMiddleware(h.AddSensor, cs.cfg.JWTSecret)))
	r.GET("/download/:os/:arch", handlers.CorsMiddleware(
		handlers.BearerMiddleware(h.DownloadBinaryHandler, cs.cfg.JWTSecret)))

	return r, nil
}
