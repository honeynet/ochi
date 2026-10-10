package backend

import (
	"context"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/honeynet/ochi/backend/repos"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"golang.org/x/time/rate"
)

type server struct {
	cfg Config

	// subscriberMessageBuffer controls the max number
	// of messages that can be queued for a subscriber
	// before it is kicked.
	//
	// Defaults to 64.
	subscriberMessageBuffer int

	// publishLimiter controls the rate limit applied to the publish endpoint.
	// Configured from publish_rate_per_sec / publish_burst (defaults 100/s, burst 50).
	publishLimiter *rate.Limiter

	// mux routes the various endpoints to the appropriate handler.
	mux *chi.Mux

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}

	// the repositories
	uRepo      *repos.UserRepo
	queryRepo  *repos.QueryRepo
	eventRepo  *repos.EventRepo
	sensorRepo *repos.SensorRepo
	// http client
	httpClient *http.Client

	fs fs.FS
}

func openSQLite(path string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite", path)
	if err != nil {
		return nil, err
	}
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

// NewServer constructs a server with the defaults.
func NewServer(fsys fs.FS, configPath string) (*server, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	cs := &server{
		cfg:                     cfg,
		subscriberMessageBuffer: 64,
		subscribers:             make(map[*subscriber]struct{}),
		publishLimiter:          rate.NewLimiter(rate.Limit(cfg.PublishRatePerSec), cfg.PublishBurst),
		httpClient: &http.Client{
			Timeout: time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout: time.Second,
			},
		},
		fs: fsys,
	}

	db, err := openSQLite(cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	cs.uRepo, err = repos.NewUserRepo(db)
	if err != nil {
		return nil, err
	}

	cs.queryRepo, err = repos.NewQueryRepo(db)
	if err != nil {
		return nil, err
	}

	cs.eventRepo, err = repos.NewEventRepo(db)
	if err != nil {
		return nil, err
	}

	cs.sensorRepo, err = repos.NewSensorRepo(db)
	if err != nil {
		return nil, err
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

// Run initializes the server
func (cs *server) Run() error {
	l, err := net.Listen("tcp", cs.cfg.ListenAddress)
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
