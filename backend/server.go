package backend

import (
	"context"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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

	// db is the single SQLite handle shared by every repo, closed on shutdown.
	db *sqlx.DB
	// the repositories
	uRepo      *repos.UserRepo
	queryRepo  *repos.QueryRepo
	eventRepo  *repos.EventRepo
	sensorRepo *repos.SensorRepo
	// http client
	httpClient *http.Client

	fs fs.FS
}

// sqlitePragmas are passed in the DSN, so the driver runs them on every connection the
// pool opens. Running them with db.Exec only reached whichever connection executed the
// statement: the others had no busy_timeout and failed at once with SQLITE_BUSY while
// another connection held the write lock.
var sqlitePragmas = []string{"journal_mode(WAL)", "busy_timeout(5000)", "synchronous(NORMAL)"}

func openSQLite(path string) (*sqlx.DB, error) {
	dsn := path
	for i, pragma := range sqlitePragmas {
		sep := "&"
		if i == 0 && !strings.Contains(path, "?") {
			sep = "?"
		}
		dsn += sep + "_pragma=" + pragma
	}
	return sqlx.Connect("sqlite", dsn)
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
	cs.db = db

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

// Run serves until the process is asked to stop. It shuts down gracefully on
// SIGINT, SIGTERM or SIGHUP: the deploy script quits the screen session (SIGHUP)
// and then sends SIGTERM, neither of which the old handler caught, so a redeploy
// killed in-flight requests and never closed the database.
func (cs *server) Run() error {
	l, err := net.Listen("tcp", cs.cfg.ListenAddress)
	if err != nil {
		_ = cs.closeDB()
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	return cs.serve(ctx, l)
}

// serve runs the HTTP server on l until ctx is cancelled or Serve fails, then
// drains in-flight requests and closes the database. It always closes the
// database before returning, whichever path it takes.
func (cs *server) serve(ctx context.Context, l net.Listener) error {
	log.Printf("listening on http://%v", l.Addr())

	srv := &http.Server{
		Handler:      cs,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	errc := make(chan error, 1)
	go func() {
		errc <- srv.Serve(l)
	}()

	select {
	case err := <-errc:
		// Serve stopped on its own (a listen error, say), not on a signal.
		log.Printf("failed to serve: %v", err)
		_ = cs.closeDB()
		return err
	case <-ctx.Done():
		log.Print("terminating")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := srv.Shutdown(shutdownCtx)
	if closeErr := cs.closeDB(); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}

// closeDB closes the shared SQLite handle, which checkpoints the WAL. It is safe
// to call when the handle was never opened.
func (cs *server) closeDB() error {
	if cs.db == nil {
		return nil
	}
	err := cs.db.Close()
	cs.db = nil
	if err != nil {
		log.Printf("closing database: %v", err)
	}
	return err
}
