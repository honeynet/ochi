package backend

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// subscriber represents a subscriber
type subscriber struct {
	msgs      chan []byte
	closeSlow func()
	// handlers restricts delivery to events whose handler is in the set.
	// nil means no filtering.
	handlers map[string]struct{}
}

// parseHandlerFilter reads the repeatable, comma-separated "handler" query
// parameter into a set. It returns nil when no handler was requested.
func parseHandlerFilter(q url.Values) map[string]struct{} {
	var set map[string]struct{}
	for _, v := range q["handler"] {
		for _, h := range strings.Split(v, ",") {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}
			if set == nil {
				set = make(map[string]struct{})
			}
			set[h] = struct{}{}
		}
	}
	return set
}

// subscribeHandler accepts the WebSocket connection and then subscribes
// it to all future messages.
func (cs *server) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusInternalError, "")

	err = cs.subscribe(r.Context(), c, parseHandlerFilter(r.URL.Query()))
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

// subscribe subscribes the given WebSocket to all broadcast messages.
// It creates a subscriber with a buffered msgs chan to give some room to slower
// connections and then registers the subscriber. It then listens for all messages
// and writes them to the WebSocket. If the context is cancelled or
// an error occurs, it returns and deletes the subscription.
//
// It uses CloseRead to keep reading from the connection to process control
// messages and cancel the context if the connection drops.
func (cs *server) subscribe(ctx context.Context, c *websocket.Conn, handlers map[string]struct{}) error {
	ctx = c.CloseRead(ctx)

	s := &subscriber{
		handlers: handlers,
		msgs:     make(chan []byte, cs.subscriberMessageBuffer),
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
// It returns false when the publish rate limit is exceeded (caller should 429).
func (cs *server) publish(msg []byte) bool {
	if !cs.publishLimiter.Allow() {
		return false
	}

	cs.subscribersMu.Lock()
	subs := make([]*subscriber, 0, len(cs.subscribers))
	for s := range cs.subscribers {
		subs = append(subs, s)
	}
	cs.subscribersMu.Unlock()

	var (
		handler string
		decoded bool
	)
	for _, s := range subs {
		if s.handlers != nil {
			if !decoded {
				var ev struct {
					Handler string `json:"handler"`
				}
				_ = json.Unmarshal(msg, &ev)
				handler, decoded = ev.Handler, true
			}
			if _, ok := s.handlers[handler]; !ok {
				continue
			}
		}
		select {
		case s.msgs <- msg:
		default:
			go s.closeSlow()
		}
	}
	return true
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
