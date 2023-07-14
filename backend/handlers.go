package backend

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/honeynet/ochi/backend/entities"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/api/idtoken"
)

func (cs *server) indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fh, err := cs.fs.Open("index.html")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(w, fh); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// publishHandler reads the request body with a limit of 8192 bytes and then publishes
// the received message.
func (cs *server) publishHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body := http.MaxBytesReader(w, r.Body, 8192)
	msg, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	cs.publish(msg)

	w.WriteHeader(http.StatusAccepted)
}

type response struct {
	User  entities.User `json:"user,omitempty"`
	Token string        `json:"token,omitempty"`
}

// sessionHandler creates a new token for the user
func (cs *server) sessionHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID := r.Context().Value(userID("userID")).(string)
	user, err := cs.uRepo.Get(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := entities.NewToken(os.Args[3], user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response{user, token}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// loginHandler validates a token with Google
func (cs *server) loginHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body := http.MaxBytesReader(w, r.Body, 8192)
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	ctx := context.Background()
	val, err := idtoken.NewValidator(ctx, idtoken.WithHTTPClient(cs.httpClient))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload, err := val.Validate(ctx, string(data), "610036027764-0lveoeejd62j594aqab5e24o2o82r8uf.apps.googleusercontent.com")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var user entities.User
	if emailInt, ok := payload.Claims["email"]; ok {
		if email, ok := emailInt.(string); ok {
			user, err = cs.uRepo.Find(email)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}

	token, err := entities.NewToken(os.Args[3], user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response{user, token}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
