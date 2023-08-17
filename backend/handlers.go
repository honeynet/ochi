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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(w, fh); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (cs *server) cssHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fh, err := cs.fs.Open("global.css")
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

// query handlers

// getQueriesHandler returns a list of queries belonging to ther user.
func (cs *server) getQueriesHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID := r.Context().Value(userID("userID")).(string)
	queries, err := cs.queryRepo.FindByOwnerId(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(queries); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// createQueryHandler creates a new query.
func (cs *server) createQueryHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID := r.Context().Value(userID("userID")).(string)
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var t entities.Query
	err := decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	query, err := cs.queryRepo.Create(userID, t.Content, t.Description, t.Active)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(query); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// udpateQueryHandler updates an existing query making sure the user owns the query.
func (cs *server) updateQueryHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	userID := r.Context().Value(userID("userID")).(string)
	id := p.ByName("id")
	q, err := cs.queryRepo.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if userID != q.OwnerID {
		http.Error(w, "Unauthorized", http.StatusInternalServerError)
		return
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if id != q.ID {
		http.Error(w, "Ids dont match", http.StatusBadRequest)
		return
	}
	err = cs.queryRepo.Update(q.ID, q.Content, q.Description, q.Active)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// deleteQueryHandler deletes a query making sure the user owns the query.
func (cs *server) deleteQueryHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	userID := r.Context().Value(userID("userID")).(string)
	id := p.ByName("id")
	q, err := cs.queryRepo.GetByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if userID != q.OwnerID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err = cs.queryRepo.Delete(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
