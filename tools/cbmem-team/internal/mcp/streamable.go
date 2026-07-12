package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"cbmem-team/internal/console"
	"cbmem-team/internal/pool"
	"cbmem-team/internal/repos"
	"cbmem-team/internal/store"
)

// StreamableHandler returns a gin.HandlerFunc that serves the v2
// Streamable HTTP transport:
//
//   - POST /mcp?transport=streamable&as=<user>&project=<path>
//   - Accepts: application/json (single JSON-RPC) or text/event-stream (NDJSON)
//   - Returns: text/event-stream (SSE format) so the client can render
//     tool output incrementally.
//
// The pool.Request call is still blocking per-request; streamable's value
// here is the SSE response format + the latency/error capture hook in
// invocation_log.go (which records into tool_invocation_logs regardless
// of transport).
//
// v1 clients that POST without ?transport= still get the v1 stdio-like
// response, so existing IDE mcp.json configs are not broken.
func StreamableHandler(p *pool.Pool, users *store.Registry, repos *repos.Manager, db *console.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Debug breadcrumb: lets operators confirm the request reached the
		// streamable path AND which user/project it claims. Cheap, single
		// line, no payload. Triggers only on POST /mcp?transport=streamable
		// because v1 clients without ?transport= fall through before this
		// line runs.
		uidPre, _ := c.Get("user_id")
		log.Printf("streamable: transport=%s auth_present=%t user_id=%v project=%q ip=%s",
			c.Query("transport"),
			c.GetHeader("Authorization") != "",
			uidPre,
			c.Query("project"),
			c.ClientIP(),
		)

		if c.Query("transport") != "streamable" {
			// Fall through to v1 — kept here so the route is single-sourced.
			stdioLikeHandler(p, users, repos)(c)
			return
		}

		v, ok := c.Get("user_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no user_id in token"})
			return
		}
		userID, _ := v.(string)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "empty user_id"})
			return
		}
		u, exists := users.Get(userID)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "user not registered"})
			return
		}
		if u.Disabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "user disabled"})
			return
		}
		project := c.Query("project")
		if project == "" {
			project = c.GetHeader("X-Project-Path")
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Rate-limit gate. We compute the tool_id from the JSON-RPC body
		// (params.name for the methods/call envelope) so per-tool
		// counters are accurate. Falls back to "unknown" for non-tool
		// RPCs (initialize, ping) which still count toward a single
		// shared bucket so a runaway client can't bypass limits by
		// spamming pings.
		toolID := jsonRPCMethod(body)
		if toolID == "" {
			toolID = "unknown"
		}
		if db != nil {
			rl := console.Check(c.Request.Context(), db, toolID, userID)
			if !rl.Allowed {
				// 429 for short-term throttles, 503 for breaker open.
				status := http.StatusTooManyRequests
				if rl.Reason == "breaker" {
					status = http.StatusServiceUnavailable
				}
				c.Header("Retry-After", fmt.Sprintf("%d", int(rl.RetryAfter.Seconds()+0.999)))
				c.JSON(status, gin.H{
					"error":       "rate limited",
					"reason":      rl.Reason,
					"retry_after": rl.RetryAfter.String(),
					"tool_id":     toolID,
				})
				return
			}
		}

		// Start timing + write to tool_invocation_logs on completion.
		inv := newInvocation("streamable", userID, project, body, c)
		started := time.Now()

		resp, err := p.Request(userID, project, string(body))
		ended := time.Now()

		inv.markComplete(resp, err, ended.Sub(started))

		// Feed the breaker.
		if err != nil {
			console.ObserveError(toolID, userID)
		} else {
			console.ObserveSuccess(toolID, userID)
		}
		console.Release(toolID, userID)

		// Streamable is self-persisting; the CaptureInvocations
		// middleware skips us via the skip_capture_invocations sentinel.
		c.Set("skip_capture_invocations", true)

		// Persist asynchronously to keep the SSE response hot-path short.
		if db != nil {
			go func() {
				if err := inv.persist(context.Background(), db); err != nil {
					// best-effort; never block the user on log writes
				}
			}()
		}

		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		// SSE framing — one JSON-RPC response per `data:` line.
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.WriteHeader(http.StatusOK)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			// fallback: dump the whole body as one SSE event
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", resp)
			return
		}

		// Stream line-by-line so multi-event responses render progressively.
		scanner := bufio.NewScanner(strings.NewReader(resp))
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			flusher.Flush()
		}
	}
}

// stdioLikeHandler is the v1 path, factored out so the v2 handler can
// share its access-control + dispatch logic. Behaviour is identical to
// the original Handler in mcp.go.
func stdioLikeHandler(p *pool.Pool, users *store.Registry, repos *repos.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user_id")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no user_id in token"})
			return
		}
		userID, _ := v.(string)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "empty user_id"})
			return
		}
		u, exists := users.Get(userID)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "user not registered"})
			return
		}
		if u.Disabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "user disabled"})
			return
		}
		project := c.Query("project")
		if project == "" {
			project = c.GetHeader("X-Project-Path")
		}
		allowed := append([]string{}, u.ProjectPaths...)
		if repos != nil {
			for _, r := range repos.List(userID) {
				allowed = append(allowed, r.LocalPath)
			}
		}
		if len(allowed) > 0 && !anyMatch(allowed, project) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "project not in allow-list",
				"allowed_prefix": allowed,
			})
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := p.Request(userID, project, string(body))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", []byte(resp))
	}
}

// jsonRPCMethod pulls `method` out of a JSON-RPC body without bringing
// in the full schema. Used by invocation_log to record tool_id.
func jsonRPCMethod(body []byte) string {
	var r struct {
		Method string `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return ""
	}
	if r.Params.Name != "" {
		return r.Params.Name
	}
	return r.Method
}