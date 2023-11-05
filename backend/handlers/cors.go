package handlers

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// CorsOptionsHandler defines a global handler for HTTP OPTIONS requests.
func CorsOptionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Access-Control-Request-Method") != "" {
		// Set CORS headers
		header := w.Header()
		header.Set("Access-Control-Allow-Methods", header.Get("Allow"))
		header.Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		header.Set("Access-Control-Allow-Origin", "*")
	}

	// Adjust status code to 204
	w.WriteHeader(http.StatusNoContent)
}

// CorsMiddleware defines a middleware for setting common CORS headers.
// This is useful when running backend and frontend on different ports.
// TODO: possibly make `Access-Control-Allow-Origin` more restrictive
// by returning a configurable list of hosts instead of *.
func CorsMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next(w, r, ps)
	}
}
