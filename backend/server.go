package backend

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/honeynet/ochi/backend/repos"
	"nhooyr.io/websocket"

	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/time/rate"
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

	// mux routes the various endpoints to the appropriate handler.
	mux *httprouter.Router

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}

	// the repositories
	uRepo     *repos.UserRepo
	queryRepo *repos.QueryRepo

	// http client
	httpClient *http.Client

	fs fs.FS
}

// NewServer constructs a server with the defaults.
func NewServer(fsys fs.FS) (*server, error) {
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
		fs: fsys,
	}

	db, err := sqlx.Connect("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}

	cs.uRepo, err = repos.NewUserRepo(db)
	if err != nil {
		log.Fatal(err)
	}

	cs.queryRepo, err = repos.NewQueryRepo(db)
	if err != nil {
		log.Fatal(err)
	}

	cs.mux, err = newRouter(cs)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func (cs *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cs.mux.ServeHTTP(w, r)
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}

// run initializes the server
func (cs *server) Run() error {
	if len(os.Args) < 4 {
		return errors.New("please provide an address to listen on as the first argument, token second, secret third")
	}

	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		return err
	}
	log.Printf("listening on http://%v", l.Addr())

	srv := &http.Server{
		Handler:      cs,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	defer cs.uRepo.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- srv.Serve(l)
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

	return srv.Shutdown(ctx)
}
