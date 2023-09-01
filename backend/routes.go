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

	build, err := fs.Sub(cs.fs, "build")
	if err != nil {
		return nil, err
	}
	r.ServeFiles("/build/*filepath", http.FS(build))

	// websocket
	r.GET("/subscribe", cs.subscribeHandler)
	r.POST("/publish", tokenMiddleware(cs.publishHandler, os.Args[2]))

	// user
	r.POST("/login", cs.loginHandler)
	r.GET("/session", handlers.CorsMiddleware(bearerMiddleware(cs.sessionHandler, os.Args[3])))

	// query
	// TODO: make CorsMiddleware more generic instead of specifying it on every handler.
	r.GET("/queries", handlers.CorsMiddleware(bearerMiddleware(cs.getQueriesHandler, os.Args[3])))
	r.POST("/queries", handlers.CorsMiddleware(bearerMiddleware(cs.createQueryHandler, os.Args[3])))
	r.PATCH("/queries/:id", handlers.CorsMiddleware(bearerMiddleware(cs.updateQueryHandler, os.Args[3])))
	r.DELETE("/queries/:id", handlers.CorsMiddleware(bearerMiddleware(cs.deleteQueryHandler, os.Args[3])))

	// event
	r.POST("/events", handlers.CorsMiddleware(bearerMiddleware(cs.createEventHandler, os.Args[3])))
	r.DELETE("/events/:id", handlers.CorsMiddleware(bearerMiddleware(cs.deleteEventHandler, os.Args[3])))
	r.GET("/events", handlers.CorsMiddleware(bearerMiddleware(cs.getEventsHandler, os.Args[3])))
	r.GET("/events/:id", handlers.CorsMiddleware(bearerMiddleware(cs.getEventByIDHandler, os.Args[3])))

	return r, nil
}
