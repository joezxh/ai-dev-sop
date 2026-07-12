package mcp

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// sseHub tracks live SSE subscribers so a future invocation_log writer
// can push real-time events without coupling to the http writer.
//
// One channel per (user_id, session_id) tuple; subscribers receive
// `invocation` events formatted as SSE `data:` lines.
type sseHub struct {
	mu   sync.RWMutex
	subs map[string]map[chan sseEvent]struct{} // session_id → set of chans
}

type sseEvent struct {
	Event string `json:"event"`
	Data  string `json:"data"`
	At    string `json:"at"`
}

var hub = &sseHub{subs: map[string]map[chan sseEvent]struct{}{}}

// publish fans out an event to all subscribers of session_id. Returns
// the number of subscribers notified. Safe to call from any goroutine.
func (h *sseHub) publish(sessionID, event, data string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	subs := h.subs[sessionID]
	for ch := range subs {
		select {
		case ch <- sseEvent{Event: event, Data: data, At: time.Now().UTC().Format(time.RFC3339)}:
		default:
			// Drop on slow consumer rather than block the publisher.
		}
	}
	return len(subs)
}

// SSEHandler returns the long-lived /mcp/sse endpoint. Clients connect
// with GET /mcp/sse?session_id=<id>. The handler emits a `ready` event
// immediately, then streams invocation events as they occur for that
// session. The hub keeps the channel registry bounded by client
// disconnects (detected by the http.Request.Context() Done channel).
func SSEHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
			return
		}
		v, _ := c.Get("user_id")
		uid, _ := v.(string)

		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering
		c.Writer.WriteHeader(http.StatusOK)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
			return
		}

		ch := make(chan sseEvent, 32)
		hub.mu.Lock()
		if hub.subs[sessionID] == nil {
			hub.subs[sessionID] = map[chan sseEvent]struct{}{}
		}
		hub.subs[sessionID][ch] = struct{}{}
		hub.mu.Unlock()
		defer func() {
			hub.mu.Lock()
			delete(hub.subs[sessionID], ch)
			if len(hub.subs[sessionID]) == 0 {
				delete(hub.subs, sessionID)
			}
			hub.mu.Unlock()
			close(ch)
		}()

		fmt.Fprintf(c.Writer, "event: ready\ndata: {\"session_id\":%q,\"user_id\":%q}\n\n",
			sessionID, uid)
		flusher.Flush()

		// 25-second keep-alive comment so reverse proxies don't time the
		// connection out. RFC 8895 says comments are fine in event streams.
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()

		ctx := c.Request.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprintf(c.Writer, ": keep-alive %d\n\n", time.Now().Unix())
				flusher.Flush()
			case ev := <-ch:
				fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Event, ev.Data)
				flusher.Flush()
			}
		}
	}
}
