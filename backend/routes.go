package backend

import (
	"io/fs"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

func newRouter(cs *server) (*httprouter.Router, error) {
	r := httprouter.New()

	// static
	r.GET("/", cs.indexHandler)
	r.GET("/global.css", cs.cssHandler)

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
	r.GET("/session", bearerMiddleware(cs.sessionHandler, os.Args[3]))

	return r, nil
}
