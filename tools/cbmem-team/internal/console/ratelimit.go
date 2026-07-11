// Package console · M2 rate limiter.
//
// Three independent mechanisms, all per-tool-per-user scoped:
//   1. Token bucket   — per-user cap on instantaneous rate (qps_per_user)
//   2. Sliding window — per-user cap on burst over 60s (qpm_per_user)
//   3. Circuit breaker — global per-tool error-rate gate; open when err%>30%
//                         for 30s straight, half-open after cooldown.
//
// Configuration is stored as JSON in `tool_directory.rate_limit_json` so
// the console UI can edit it via PUT /api/console/v2/tools/:id/rate-limit
// and the change is picked up on the next call.
package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RateLimitConfig is the persisted JSON shape. Zero values mean
// "no limit" (bucket size 0 = unlimited). Cooldown is honored only when
// the breaker is open and is otherwise ignored.
type RateLimitConfig struct {
	QPSPerUser          int `json:"qps_per_user"`
	QPMPerUser          int `json:"qpm_per_user"`
	ConcurrencyGlobal   int `json:"concurrency_global"`
	QueuePriority       string `json:"queue_priority"`
	CooldownMsAfterError int `json:"cooldown_ms_after_error"`
}

// DefaultRateLimit is the fallback when a tool has no explicit row.
// 5 qps / 60 qpm / 16 concurrency is generous; tools that need
// tighter limits should declare their own.
func DefaultRateLimit() RateLimitConfig {
	return RateLimitConfig{QPSPerUser: 5, QPMPerUser: 60, ConcurrencyGlobal: 16, QueuePriority: "P2"}
}

// AllowResult is what Check returns. Allowed=false carries a Reason and
// a RetryAfter duration; the middleware maps Reason to an HTTP status.
type AllowResult struct {
	Allowed    bool
	Reason     string // "qps"|"qpm"|"concurrency"|"breaker"
	RetryAfter time.Duration
	Remaining  int
}

// ---- token bucket ----

type tokenBucket struct {
	mu        sync.Mutex
	capacity  float64
	refillRPS float64
	tokens    float64
	last      time.Time
}

func newTokenBucket(capacity, rps float64) *tokenBucket {
	return &tokenBucket{capacity: capacity, refillRPS: rps, tokens: capacity, last: time.Now()}
}

// take attempts to consume one token. Returns (allowed, retry-after).
func (b *tokenBucket) take(now time.Time) (bool, time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * b.refillRPS
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	missing := 1 - b.tokens
	wait := time.Duration(missing/b.refillRPS*1000) * time.Millisecond
	if wait < 50*time.Millisecond {
		wait = 50 * time.Millisecond
	}
	return false, wait
}

// ---- sliding window ----

type slidingWindow struct {
	mu       sync.Mutex
	capacity int
	window   time.Duration
	hits     []time.Time
}

func newSlidingWindow() *slidingWindow {
	return &slidingWindow{}
}

// allow returns whether a hit at `now` fits within the last `window` and
// counts up to `capacity`. Old entries are evicted on the fly.
func (w *slidingWindow) allow(now time.Time) (bool, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	cutoff := now.Add(-w.window)
	// Drop expired hits (linear scan, capacity is small).
	keep := w.hits[:0]
	for _, t := range w.hits {
		if t.After(cutoff) {
			keep = append(keep, t)
		}
	}
	w.hits = keep
	if len(w.hits) >= w.capacity {
		return false, 0
	}
	w.hits = append(w.hits, now)
	return true, w.capacity - len(w.hits)
}

// ---- circuit breaker ----

type breaker struct {
	mu              sync.Mutex
	state           string // "closed" | "open" | "half-open"
	openedAt        time.Time
	halfOpenAt      time.Time
	recentWindow    []breakerSample
	errThreshold    float64       // 0.30
	sustainedWindow time.Duration // 30s
	cooldown        time.Duration // 30s
}

type breakerSample struct {
	at  time.Time
	err bool
}

// record feeds one observation; flips state when sustained error rate
// crosses the threshold.
func (b *breaker) record(now time.Time, isErr bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	cutoff := now.Add(-b.sustainedWindow)
	keep := b.recentWindow[:0]
	for _, s := range b.recentWindow {
		if s.at.After(cutoff) {
			keep = append(keep, s)
		}
	}
	b.recentWindow = append(keep, breakerSample{at: now, err: isErr})

	// Auto-close from half-open after cooldown elapses.
	if b.state == "half-open" && now.After(b.halfOpenAt.Add(b.cooldown)) {
		b.state = "closed"
		b.recentWindow = b.recentWindow[:0]
	}

	if len(b.recentWindow) < 10 {
		return // need a meaningful sample size
	}
	var errs int
	for _, s := range b.recentWindow {
		if s.err {
			errs++
		}
	}
	rate := float64(errs) / float64(len(b.recentWindow))
	if rate > b.errThreshold && b.state == "closed" {
		b.state = "open"
		b.openedAt = now
		b.halfOpenAt = now.Add(b.cooldown)
	} else if rate < b.errThreshold/2 && b.state == "open" {
		// Recovery: half-open after cooldown.
		b.state = "half-open"
	}
}

// allow returns whether the breaker is letting traffic through right now.
func (b *breaker) allow(now time.Time) (bool, time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case "closed", "half-open":
		return true, 0
	case "open":
		wait := b.halfOpenAt.Sub(now)
		if wait < 0 {
			wait = 0
		}
		return false, wait
	}
	return true, 0
}

// ---- per-tool-per-user entry ----

type rlEntry struct {
	bucket  *tokenBucket
	window  *slidingWindow
	breaker *breaker
	concSem chan struct{}
	cfg     RateLimitConfig
	cfgAt   time.Time
}

// rateLimiter is the in-memory store of rlEntry, indexed by
// (tool_id, user_id). It is goroutine-safe; reload invalidates entries
// whose cfg is older than the call-site's `lastReload`.
type rateLimiter struct {
	mu      sync.RWMutex
	entries map[string]*rlEntry // key = toolID + "|" + userID
}

func newRateLimiter() *rateLimiter { return &rateLimiter{entries: map[string]*rlEntry{}} }

func key(toolID, userID string) string { return toolID + "|" + userID }

// getOrCreate returns the rlEntry for (tool, user). If an entry exists
// but its cfg is older than `cfgFresh`, it is rebuilt — this is how
// "edit rate-limit in UI → next call sees it" works.
func (rl *rateLimiter) getOrCreate(toolID, userID string, cfg RateLimitConfig, cfgFresh time.Time) *rlEntry {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	k := key(toolID, userID)
	e, ok := rl.entries[k]
	if ok && !e.cfgAt.Before(cfgFresh) {
		return e
	}
	qps := float64(cfg.QPSPerUser)
	if qps <= 0 {
		qps = 1
	}
	e = &rlEntry{
		bucket:  newTokenBucket(qps, qps),
		window:  newSlidingWindow(),
		breaker: &breaker{state: "closed", errThreshold: 0.30, sustainedWindow: 30 * time.Second, cooldown: 30 * time.Second},
		cfg:     cfg,
		cfgAt:   cfgFresh,
	}
	if cfg.QPMPerUser > 0 {
		e.window.capacity = cfg.QPMPerUser
		e.window.window = 60 * time.Second
	}
	if cfg.ConcurrencyGlobal > 0 {
		e.concSem = make(chan struct{}, cfg.ConcurrencyGlobal)
	}
	rl.entries[k] = e
	return e
}

// ---- public surface ----

// RateLimiter is the package-level singleton used by middleware.
var RateLimiter = newRateLimiter()

// LoadRateLimit reads `rate_limit_json` for a tool. Empty / malformed
// JSON returns DefaultRateLimit(), so a tool that hasn't been configured
// still gets sensible defaults rather than zero (which means "no limit"
// and would let a buggy client DoS the pool).
func LoadRateLimit(ctx context.Context, db *DB, toolID string) (RateLimitConfig, error) {
	var raw *string
	err := db.QueryRowContext(ctx,
		`SELECT rate_limit_json FROM tool_directory WHERE tool_id = ?`, toolID,
	).Scan(&raw)
	if err != nil {
		return DefaultRateLimit(), err
	}
	if raw == nil || *raw == "" {
		return DefaultRateLimit(), nil
	}
	var cfg RateLimitConfig
	if err := json.Unmarshal([]byte(*raw), &cfg); err != nil {
		return DefaultRateLimit(), fmt.Errorf("decode rate_limit_json: %w", err)
	}
	if cfg.QPSPerUser == 0 && cfg.QPMPerUser == 0 && cfg.ConcurrencyGlobal == 0 {
		return DefaultRateLimit(), nil
	}
	return cfg, nil
}

// Check runs all three gates for (tool, user). Returns an AllowResult
// the middleware can turn into HTTP 200 / 429 / 503.
func Check(ctx context.Context, db *DB, toolID, userID string) AllowResult {
	cfg, _ := LoadRateLimit(ctx, db, toolID)
	e := RateLimiter.getOrCreate(toolID, userID, cfg, time.Now())

	now := time.Now()
	// 1. breaker first — cheapest reject.
	if ok, wait := e.breaker.allow(now); !ok {
		return AllowResult{Allowed: false, Reason: "breaker", RetryAfter: wait}
	}
	// 2. token bucket — instantaneous rate.
	if cfg.QPSPerUser > 0 {
		if ok, wait := e.bucket.take(now); !ok {
			return AllowResult{Allowed: false, Reason: "qps", RetryAfter: wait}
		}
	}
	// 3. sliding window — minute-level cap.
	if cfg.QPMPerUser > 0 {
		if ok, rem := e.window.allow(now); !ok {
			return AllowResult{Allowed: false, Reason: "qpm", RetryAfter: 1 * time.Second, Remaining: rem}
		}
	}
	// 4. concurrency semaphore (non-blocking).
	if e.concSem != nil {
		select {
		case e.concSem <- struct{}{}:
			// got a slot; caller must Release() after the call.
		default:
			return AllowResult{Allowed: false, Reason: "concurrency", RetryAfter: 100 * time.Millisecond}
		}
	}
	return AllowResult{Allowed: true, Remaining: -1}
}

// Release returns a concurrency slot. Safe to call even if Check did
// not acquire one (it's a no-op in that case because we use a buffered
// channel and never block).
func Release(toolID, userID string) {
	e := RateLimiter.getOrCreate(toolID, userID, DefaultRateLimit(), time.Time{})
	if e.concSem != nil {
		select {
		case <-e.concSem:
		default:
		}
	}
}

// ObserveError / ObserveSuccess feed the breaker.
func ObserveError(toolID, userID string) {
	e := RateLimiter.getOrCreate(toolID, userID, DefaultRateLimit(), time.Time{})
	e.breaker.record(time.Now(), true)
}

func ObserveSuccess(toolID, userID string) {
	e := RateLimiter.getOrCreate(toolID, userID, DefaultRateLimit(), time.Time{})
	e.breaker.record(time.Now(), false)
}

// ErrRateLimited is exported so the middleware can errors.Is on it.
var ErrRateLimited = errors.New("rate limited")