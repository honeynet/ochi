package backend

import (
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

func newRouter(cs *server) *httprouter.Router {
	r := httprouter.New()

	// static
	r.GET("/", cs.indexHandler)
	r.ServeFiles("/src/*filepath", http.FS(cs.fs))

	// websocket
	r.GET("/subscribe", cs.subscribeHandler)
	r.POST("/publish", tokenMiddleware(cs.publishHandler, os.Args[2]))

	// user
	r.POST("/login", cs.loginHandler)
	r.GET("/session", bearerMiddleware(cs.sessionHandler, os.Args[3]))

	return r
}
