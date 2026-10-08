package backend

import (
	"io/fs"
	"net/http"

	"github.com/honeynet/ochi/backend/handlers"

	"github.com/julienschmidt/httprouter"
)

func newRouter(cs *server) (*httprouter.Router, error) {
	r := httprouter.New()
	// Set CORS headers
	r.GlobalOPTIONS = http.HandlerFunc(handlers.CorsOptionsHandler)

	// static
	r.GET("/", cs.indexHandler)
	r.GET("/global.css", cs.cssHandler)
	r.GET("/myqueries", cs.indexHandler)
	r.GET("/myevents", cs.indexHandler)
	r.GET("/events/:id", cs.indexHandler)

	build, err := fs.Sub(cs.fs, "build")
	if err != nil {
		return nil, err
	}
	r.ServeFiles("/build/*filepath", http.FS(build))

	// websocket
	r.GET("/subscribe", cs.subscribeHandler)
	r.POST("/publish", handlers.TokenMiddleware(cs.publishHandler, cs.cfg.PublishToken))

	// user
	r.POST("/login", cs.loginHandler)
	r.GET("/session", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.sessionHandler, cs.cfg.JWTSecret)))

	// query
	// TODO: make CorsMiddleware more generic instead of specifying it on every handler.
	r.GET("/queries", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.getQueriesHandler, cs.cfg.JWTSecret)))
	r.POST("/queries", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.createQueryHandler, cs.cfg.JWTSecret)))
	r.PATCH("/queries/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.updateQueryHandler, cs.cfg.JWTSecret)))
	r.DELETE("/queries/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.deleteQueryHandler, cs.cfg.JWTSecret)))

	// event
	r.POST("/api/events", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.createEventHandler, cs.cfg.JWTSecret)))
	r.DELETE("/api/events/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.deleteEventHandler, cs.cfg.JWTSecret)))
	r.GET("/api/events", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.getEventsHandler, cs.cfg.JWTSecret)))
	// Shared event links are unguessable IDs and must work without login.
	r.GET("/api/events/:id", handlers.CorsMiddleware(cs.getEventByIDHandler))

	// sensor
	r.GET("/sensors", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.getSensorsByUser, cs.cfg.JWTSecret)))
	r.POST("/sensors", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.addSensor, cs.cfg.JWTSecret)))
	r.GET("/download/:os/:arch", handlers.CorsMiddleware(
		handlers.BearerMiddleware(cs.downloadBinaryHandler, cs.cfg.JWTSecret)))

	return r, nil
}
