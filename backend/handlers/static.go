package handlers

import (
	"io"
	"net/http"
)

func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	fh, err := h.FS.Open("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(w, fh); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) CSS(w http.ResponseWriter, r *http.Request) {
	fh, err := h.FS.Open("global.css")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "text/css")
	if _, err := io.Copy(w, fh); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
