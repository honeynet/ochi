package services

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/honeynet/ochi/backend/entities"
)

// StatsRepository persists and retrieves sensor statistics from database.
type StatsRepository interface {
	LoadAllStats() ([]entities.SensorStat, error)
	BatchUpsertStats(deltas map[string]int64, lastSeen map[string]string) error
}

type statsSubscriber struct {
	msgs      chan []byte
	closeSlow func()
}

// StatsService manages in-memory event counting, periodic persistence, and live updates.
type StatsService struct {
	repo StatsRepository

	mu             sync.RWMutex
	totals         map[string]int64
	lastSeen       map[string]string
	deltas         map[string]int64
	dirtyBroadcast map[string]struct{}

	subsMu sync.Mutex
	subs   map[*statsSubscriber]struct{}

	flushInterval     time.Duration
	broadcastInterval time.Duration
	bufferSize        int

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewStatsService creates a new StatsService with default intervals.
func NewStatsService(repo StatsRepository) *StatsService {
	return &StatsService{
		repo:              repo,
		totals:            make(map[string]int64),
		lastSeen:          make(map[string]string),
		deltas:            make(map[string]int64),
		dirtyBroadcast:    make(map[string]struct{}),
		subs:              make(map[*statsSubscriber]struct{}),
		flushInterval:     5 * time.Second,
		broadcastInterval: 500 * time.Millisecond,
		bufferSize:        64,
	}
}

// SetFlushInterval overrides the default periodic flush interval.
func (s *StatsService) SetFlushInterval(d time.Duration) {
	s.flushInterval = d
}

// SetBroadcastInterval overrides the default broadcast interval.
func (s *StatsService) SetBroadcastInterval(d time.Duration) {
	s.broadcastInterval = d
}

// Start loads initial stats from the database and launches periodic background tasks.
func (s *StatsService) Start(ctx context.Context) error {
	if s.repo != nil {
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
	}

	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(2)
	go s.flusher(runCtx)
	go s.broadcaster(runCtx)

	return nil
}

// Stop stops background workers and commits any pending deltas to the database.
func (s *StatsService) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	return s.Flush()
}

// IncrementStat increments the event count in memory and marks it for broadcast and flush.
// It guards against overflow past math.MaxInt64 and guarantees the counter remains non-negative.
func (s *StatsService) IncrementStat(sensorID, eventTime string) entities.SensorStat {
	s.mu.Lock()
	defer s.mu.Unlock()

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

// GetStatsForSensors retrieves current stats for the requested sensor IDs from memory.
// It matches by prefix when an event used a truncated 8-character ID or full UUID.
func (s *StatsService) GetStatsForSensors(sensorIDs []string) map[string]entities.SensorStat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]entities.SensorStat, len(sensorIDs))
	for _, id := range sensorIDs {
		count := s.totals[id]
		lastSeen := s.lastSeen[id]

		if count == 0 {
			if len(id) >= 8 {
				prefix := id[:8]
				if c, ok := s.totals[prefix]; ok && c > 0 {
					count = c
					lastSeen = s.lastSeen[prefix]
				}
			}
			if count == 0 {
				for totalID, c := range s.totals {
					if (strings.HasPrefix(totalID, id) || strings.HasPrefix(id, totalID)) && c > 0 {
						if count <= math.MaxInt64-c {
							count += c
						} else {
							count = math.MaxInt64
						}
						if lastSeen == "" || s.lastSeen[totalID] > lastSeen {
							lastSeen = s.lastSeen[totalID]
						}
					}
				}
			}
		}

		if count < 0 {
			count = 0
		}

		res[id] = entities.SensorStat{
			SensorID:    id,
			EventCount:  count,
			LastEventAt: lastSeen,
		}
	}
	return res
}

// DeleteSensor removes a sensor's in-memory statistics.
func (s *StatsService) DeleteSensor(sensorID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
// On failure, pending deltas are preserved for retry.
func (s *StatsService) Flush() error {
	s.mu.Lock()
	if len(s.deltas) == 0 {
		s.mu.Unlock()
		return nil
	}

	pendingDeltas := s.deltas
	s.deltas = make(map[string]int64)
	pendingLastSeen := make(map[string]string, len(pendingDeltas))
	for id := range pendingDeltas {
		pendingLastSeen[id] = s.lastSeen[id]
	}
	s.mu.Unlock()

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
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.Flush()
		}
	}
}

func (s *StatsService) broadcaster(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.broadcastInterval)
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
			msg, err := json.Marshal(dirty)
			if err != nil {
				continue
			}
			s.PublishStats(msg)
		}
	}
}

// PublishStats sends a message to all active WebSocket stats subscribers.
func (s *StatsService) PublishStats(msg []byte) bool {
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
	return true
}

// Subscribe attaches a WebSocket connection to the stats broadcast stream.
func (s *StatsService) Subscribe(ctx context.Context, c *websocket.Conn) error {
	ctx = c.CloseRead(ctx)

	sub := &statsSubscriber{
		msgs: make(chan []byte, s.bufferSize),
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
			ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
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
