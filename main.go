package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/glaslos/ochi/entities"
	"github.com/glaslos/ochi/repos"
	"github.com/jmoiron/sqlx"
	"golang.org/x/time/rate"
	"google.golang.org/api/idtoken"
	"nhooyr.io/websocket"
)

type server struct {
	// subscriberMessageBuffer controls the max number
	// of messages that can be queued for a subscriber
	// before it is kicked.
	//
	// Defaults to 16.
	subscriberMessageBuffer int

	// publishLimiter controls the rate limit applied to the publish endpoint.
	//
	// Defaults to one publish every 100ms with a burst of 8.
	publishLimiter *rate.Limiter

	// serveMux routes the various endpoints to the appropriate handler.
	serveMux http.ServeMux

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}

	// the repositories
	uRepo *repos.UserRepo

	// http client
	httpClient *http.Client
}

//go:embed public
var public embed.FS

// newServer constructs a server with the defaults.
func newServer() *server {
	cs := &server{
		subscriberMessageBuffer: 16,
		subscribers:             make(map[*subscriber]struct{}),
		publishLimiter:          rate.NewLimiter(rate.Every(time.Millisecond*100), 8),
		httpClient: &http.Client{
			Timeout: time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout: time.Second,
			},
		},
	}

	content, err := fs.Sub(public, "public")
	if err != nil {
		log.Fatal(err)
	}

	db, err := sqlx.Connect("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}

	cs.uRepo, err = repos.NewUserRepo(db)
	if err != nil {
		log.Fatal(err)
	}

	cs.serveMux.Handle("/", http.FileServer(http.FS(content)))
	cs.serveMux.HandleFunc("/subscribe", cs.subscribeHandler)
	cs.serveMux.HandleFunc("/publish", cs.publishHandler)
	cs.serveMux.HandleFunc("/login", cs.loginHandler)
	cs.serveMux.HandleFunc("/session", cs.sessionHandler)

	return cs
}

// subscriber represents a subscriber
type subscriber struct {
	msgs      chan []byte
	closeSlow func()
}

func (cs *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cs.serveMux.ServeHTTP(w, r)
}

// subscribeHandler accepts the WebSocket connection and then subscribes
// it to all future messages.
func (cs *server) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusInternalError, "")

	err = cs.subscribe(r.Context(), c)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		return
	}
}

// publishHandler reads the request body with a limit of 8192 bytes and then publishes
// the received message.
func (cs *server) publishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	token, ok := r.URL.Query()["token"]
	if !ok || len(token) == 0 || token[0] != os.Args[2] {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

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

// sessionHandler ...
func (cs *server) sessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	body := http.MaxBytesReader(w, r.Body, 1024)
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	claims, valid, err := ValidateToken(string(data), os.Args[3])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	user, err := cs.uRepo.Get(claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := NewToken(os.Args[3], user)
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

// loginHandler ...
func (cs *server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

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

	token, err := NewToken(os.Args[3], user)
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

// subscribe subscribes the given WebSocket to all broadcast messages.
// It creates a subscriber with a buffered msgs chan to give some room to slower
// connections and then registers the subscriber. It then listens for all messages
// and writes them to the WebSocket. If the context is cancelled or
// an error occurs, it returns and deletes the subscription.
//
// It uses CloseRead to keep reading from the connection to process control
// messages and cancel the context if the connection drops.
func (cs *server) subscribe(ctx context.Context, c *websocket.Conn) error {
	ctx = c.CloseRead(ctx)

	s := &subscriber{
		msgs: make(chan []byte, cs.subscriberMessageBuffer),
		closeSlow: func() {
			c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		},
	}
	cs.addSubscriber(s)
	defer cs.deleteSubscriber(s)

	for {
		select {
		case msg := <-s.msgs:
			err := writeTimeout(ctx, time.Second*5, c, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// publish publishes the msg to all subscribers.
func (cs *server) publish(msg []byte) {
	cs.subscribersMu.Lock()
	defer cs.subscribersMu.Unlock()

	cs.publishLimiter.Wait(context.Background())

	for s := range cs.subscribers {
		select {
		case s.msgs <- msg:
		default:
			go s.closeSlow()
		}
	}
}

// addSubscriber registers a subscriber.
func (cs *server) addSubscriber(s *subscriber) {
	cs.subscribersMu.Lock()
	cs.subscribers[s] = struct{}{}
	cs.subscribersMu.Unlock()
}

// deleteSubscriber deletes the given subscriber.
func (cs *server) deleteSubscriber(s *subscriber) {
	cs.subscribersMu.Lock()
	delete(cs.subscribers, s)
	cs.subscribersMu.Unlock()
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}

// run initializes the server
func run() error {
	if len(os.Args) < 4 {
		return errors.New("please provide an address to listen on as the first argument, token second, secret third")
	}

	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		return err
	}
	log.Printf("listening on http://%v", l.Addr())

	cs := newServer()
	s := &http.Server{
		Handler:      cs,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	defer cs.uRepo.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- s.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
