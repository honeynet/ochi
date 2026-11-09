package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/honeynet/ochi/backend/entities"
)

// StatsRepository persists and retrieves sensor statistics from database.
type StatsRepository interface {
	LoadAllStats() ([]entities.SensorStat, error)
	BatchUpsertStats(deltas map[string]int64, lastSeen map[string]string) error
	IsOwner(sensorID, userID string) bool
}

type statsSubscriber struct {
	userID    string
	msgs      chan []byte
	closeSlow func()
}

// StatsConfig configures intervals and buffer sizes for StatsService.
type StatsConfig struct {
	FlushInterval     time.Duration
	BroadcastInterval time.Duration
	BufferSize        int
}

// DefaultStatsConfig returns sensible defaults for StatsService.
func DefaultStatsConfig() StatsConfig {
	return StatsConfig{
		FlushInterval:     5 * time.Second,
		BroadcastInterval: 500 * time.Millisecond,
		BufferSize:        64,
	}
}

// StatsService manages in-memory event counting, periodic persistence, and live updates.
type StatsService struct {
	repo StatsRepository
	cfg  StatsConfig

	flushMu        sync.Mutex
	mu             sync.RWMutex
	totals         map[string]int64
	lastSeen       map[string]string
	deltas         map[string]int64
	dirtyBroadcast map[string]struct{}
	tombstones     map[string]struct{}

	subsMu sync.Mutex
	subs   map[*statsSubscriber]struct{}

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewStatsService creates a new StatsService configured with cfg.
func NewStatsService(repo StatsRepository, cfg StatsConfig) *StatsService {
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}
	if cfg.BroadcastInterval <= 0 {
		cfg.BroadcastInterval = 500 * time.Millisecond
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 64
	}
	return &StatsService{
		repo:           repo,
		cfg:            cfg,
		totals:         make(map[string]int64),
		lastSeen:       make(map[string]string),
		deltas:         make(map[string]int64),
		dirtyBroadcast: make(map[string]struct{}),
		tombstones:     make(map[string]struct{}),
		subs:           make(map[*statsSubscriber]struct{}),
	}
}

// Start loads initial stats from the database and launches periodic background tasks.
func (s *StatsService) Start(ctx context.Context) error {
	if s.repo == nil {
		return errors.New("stats repository is required")
	}

	rows, err := s.repo.LoadAllStats()
	if err != nil {
		return err
	}
	s.mu.Lock()
	for _, row := range rows {
		if row.EventCount >= 0 {
			s.totals[row.SensorID] = row.EventCount
			s.lastSeen[row.SensorID] = row.LastEventAt
		}
	}
	s.mu.Unlock()

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(2)
	go s.flusher(runCtx)
	go s.broadcaster(runCtx)

	return nil
}

// Stop stops background workers and commits any pending deltas to the database.
// It bounds the wait for background tasks so shutdown cannot hang indefinitely.
func (s *StatsService) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}

	waitCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
	case <-time.After(5 * time.Second):
		log.Printf("stats service shutdown timed out waiting for background tasks")
	}

	return s.Flush()
}

// IncrementStat increments the event count in memory and marks it for broadcast and flush.
// It guards against overflow past math.MaxInt64 and guarantees the counter remains non-negative.
func (s *StatsService) IncrementStat(sensorID, eventTime string) entities.SensorStat {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, tombstoned := s.tombstones[sensorID]; tombstoned {
		return entities.SensorStat{SensorID: sensorID}
	}

	curTotal := s.totals[sensorID]
	if curTotal < 0 {
		curTotal = 0
	}
	if curTotal < math.MaxInt64 {
		s.totals[sensorID] = curTotal + 1
	}

	curDelta := s.deltas[sensorID]
	if curDelta < 0 {
		curDelta = 0
	}
	if curDelta < math.MaxInt64 {
		s.deltas[sensorID] = curDelta + 1
	}

	if eventTime != "" {
		s.lastSeen[sensorID] = eventTime
	}
	s.dirtyBroadcast[sensorID] = struct{}{}

	return entities.SensorStat{
		SensorID:    sensorID,
		EventCount:  s.totals[sensorID],
		LastEventAt: s.lastSeen[sensorID],
	}
}

// GetStatsForSensors retrieves current stats for the requested sensor IDs from memory,
// matching on exact ID only.
func (s *StatsService) GetStatsForSensors(sensorIDs []string) map[string]entities.SensorStat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]entities.SensorStat, len(sensorIDs))
	for _, id := range sensorIDs {
		count := s.totals[id]
		if count < 0 {
			count = 0
		}

		res[id] = entities.SensorStat{
			SensorID:    id,
			EventCount:  count,
			LastEventAt: s.lastSeen[id],
		}
	}
	return res
}

// DeleteSensor removes a sensor's in-memory statistics and tombstones it so that
// any in-flight flushes or increments do not re-create or restore it.
func (s *StatsService) DeleteSensor(sensorID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tombstones[sensorID] = struct{}{}
	delete(s.totals, sensorID)
	delete(s.lastSeen, sensorID)
	delete(s.deltas, sensorID)
	delete(s.dirtyBroadcast, sensorID)
}

// PopDirtyStatsForBroadcast extracts and clears dirty stats accumulated since the last check.
func (s *StatsService) PopDirtyStatsForBroadcast() []entities.SensorStat {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.dirtyBroadcast) == 0 {
		return nil
	}

	dirty := make([]entities.SensorStat, 0, len(s.dirtyBroadcast))
	for id := range s.dirtyBroadcast {
		dirty = append(dirty, entities.SensorStat{
			SensorID:    id,
			EventCount:  s.totals[id],
			LastEventAt: s.lastSeen[id],
		})
	}
	s.dirtyBroadcast = make(map[string]struct{})
	return dirty
}

// Flush commits pending in-memory deltas to the database in a batch.
// It swaps pending deltas under a short in-memory lock so concurrent calls to
// IncrementStat (/api/publish) are never blocked by database disk I/O.
func (s *StatsService) Flush() error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.mu.Lock()
	if len(s.deltas) == 0 {
		s.mu.Unlock()
		return nil
	}

	pendingDeltas := make(map[string]int64, len(s.deltas))
	pendingLastSeen := make(map[string]string, len(s.deltas))
	for id, delta := range s.deltas {
		if _, tombstoned := s.tombstones[id]; !tombstoned {
			pendingDeltas[id] = delta
			pendingLastSeen[id] = s.lastSeen[id]
		}
	}
	s.deltas = make(map[string]int64)
	s.mu.Unlock()

	if len(pendingDeltas) == 0 {
		return nil
	}

	if s.repo == nil {
		return nil
	}

	if err := s.repo.BatchUpsertStats(pendingDeltas, pendingLastSeen); err != nil {
		s.revertDeltas(pendingDeltas)
		return err
	}

	return nil
}

func (s *StatsService) revertDeltas(pending map[string]int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, delta := range pending {
		if _, tombstoned := s.tombstones[id]; tombstoned {
			continue
		}
		if delta > 0 {
			if s.deltas[id] <= math.MaxInt64-delta {
				s.deltas[id] += delta
			} else {
				s.deltas[id] = math.MaxInt64
			}
		}
	}
}

func (s *StatsService) flusher(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Flush(); err != nil {
				log.Printf("failed to flush sensor stats: %v", err)
			}
		}
	}
}

func (s *StatsService) broadcaster(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.BroadcastInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dirty := s.PopDirtyStatsForBroadcast()
			if len(dirty) == 0 {
				continue
			}
			s.BroadcastStats(dirty)
		}
	}
}

// BroadcastStats sends dirty stats to active subscribers, filtering by sensor owner.
func (s *StatsService) BroadcastStats(dirty []entities.SensorStat) {
	if len(dirty) == 0 {
		return
	}

	s.subsMu.Lock()
	subs := make([]*statsSubscriber, 0, len(s.subs))
	for sub := range s.subs {
		subs = append(subs, sub)
	}
	s.subsMu.Unlock()

	if len(subs) == 0 {
		return
	}

	userMsgs := make(map[string][]byte)
	for _, sub := range subs {
		if sub.userID == "" {
			continue
		}
		msg, ok := userMsgs[sub.userID]
		if !ok {
			var userStats []entities.SensorStat
			for _, st := range dirty {
				if s.repo != nil && s.repo.IsOwner(st.SensorID, sub.userID) {
					userStats = append(userStats, st)
				}
			}
			if len(userStats) > 0 {
				encoded, err := json.Marshal(userStats)
				if err == nil {
					msg = encoded
				}
			}
			userMsgs[sub.userID] = msg
		}
		if len(msg) > 0 {
			select {
			case sub.msgs <- msg:
			default:
				go sub.closeSlow()
			}
		}
	}
}

// PublishStats sends a message to all active WebSocket stats subscribers.
func (s *StatsService) PublishStats(msg []byte) {
	s.subsMu.Lock()
	subs := make([]*statsSubscriber, 0, len(s.subs))
	for sub := range s.subs {
		subs = append(subs, sub)
	}
	s.subsMu.Unlock()

	for _, sub := range subs {
		select {
		case sub.msgs <- msg:
		default:
			go sub.closeSlow()
		}
	}
}

// Subscribe attaches an authenticated WebSocket connection to the stats broadcast stream.
func (s *StatsService) Subscribe(ctx context.Context, c *websocket.Conn, userID string) error {
	ctx = c.CloseRead(ctx)

	sub := &statsSubscriber{
		userID: userID,
		msgs:   make(chan []byte, s.cfg.BufferSize),
		closeSlow: func() {
			c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		},
	}

	s.subsMu.Lock()
	s.subs[sub] = struct{}{}
	s.subsMu.Unlock()

	defer func() {
		s.subsMu.Lock()
		delete(s.subs, sub)
		s.subsMu.Unlock()
	}()

	for {
		select {
		case msg := <-sub.msgs:
			ctxTimeout, cancel := context.WithTimeout(ctx, time.Second)
			err := c.Write(ctxTimeout, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
