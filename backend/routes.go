package backend

import (
	"io/fs"
	"net/http"
	"os"

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
	// This URL is rendered using client side rendering with routify
	// TODO: make this solution more generic
	r.GET("/myqueries", cs.indexHandler)
	r.GET("/events/:id", cs.indexHandler)

	build, err := fs.Sub(cs.fs, "build")
	if err != nil {
		return nil, err
	}
	r.ServeFiles("/build/*filepath", http.FS(build))

	// websocket
	r.GET("/subscribe", cs.subscribeHandler)
	r.POST("/publish", handlers.TokenMiddleware(cs.publishHandler, os.Args[2]))

	// user
	r.POST("/login", cs.loginHandler)
	r.GET("/session", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.sessionHandler, os.Args[3])))

	// query
	// TODO: make CorsMiddleware more generic instead of specifying it on every handler.
	r.GET("/queries", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.getQueriesHandler, os.Args[3])))
	r.POST("/queries", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.createQueryHandler, os.Args[3])))
	r.PATCH("/queries/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.updateQueryHandler, os.Args[3])))
	r.DELETE("/queries/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.deleteQueryHandler, os.Args[3])))

	// event
	r.POST("/api/events", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.createEventHandler, os.Args[3])))
	r.DELETE("/api/events/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.deleteEventHandler, os.Args[3])))
	r.GET("/api/events", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.getEventsHandler, os.Args[3])))
	r.GET("/api/events/:id", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.getEventByIDHandler, os.Args[3])))

	// metrics
	r.GET("/api/metrics", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.getMetricsHandler, os.Args[3])))
	r.GET("/api/metrics/:dstPort", handlers.CorsMiddleware(handlers.BearerMiddleware(cs.getMetricHandler, os.Args[3])))

	return r, nil
}
