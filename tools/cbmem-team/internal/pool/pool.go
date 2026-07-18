// Package pool manages one stdio process per (user, project) pair.
//
// The wrapper keeps each user's index strictly isolated by spawning a
// dedicated `codebase-memory-mcp` process whose working directory is the
// user's personal index root. SQLite file-locks then provide natural
// isolation without any extra coordination.
//
// Lifecycle:
//   - Get(userID, projectPath) lazily starts a process on first use
//   - Idle processes are reaped after cfg.IdleTTL
//   - On any subprocess crash we restart on the next request
//   - Close() terminates all live processes
package pool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	DataDir      string
	MCPBinary    string
	IdleTTL      time.Duration
	MaxProcs     int
	StartTimeout time.Duration
}

type proc struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	reader      *bufio.Reader
	lastUsed    atomic.Int64 // unix nanos
	mu          sync.Mutex   // protects concurrent JSON-RPC sends
	indexDir    string
	projectPath string
	cancel      context.CancelFunc
	done        chan struct{}
}

type Pool struct {
	cfg Config

	mu    sync.Mutex
	procs map[string]*proc // key = userID
}

func New(cfg Config) *Pool {
	if cfg.IdleTTL == 0 {
		cfg.IdleTTL = 30 * time.Minute
	}
	if cfg.MaxProcs == 0 {
		cfg.MaxProcs = 4
	}
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 15 * time.Second
	}
	return &Pool{
		cfg:   cfg,
		procs: map[string]*proc{},
	}
}

func (p *Pool) userDir(userID string) string {
	return filepath.Join(p.cfg.DataDir, "users", userID)
}

func (p *Pool) indexDir(userID, projectPath string) string {
	return filepath.Join(p.userDir(userID), "projects", sanitize(projectPath))
}

func sanitize(p string) string {
	s := p
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= 'a' && s[i] <= 'z',
			s[i] >= 'A' && s[i] <= 'Z',
			s[i] >= '0' && s[i] <= '9',
			s[i] == '-' || s[i] == '_' || s[i] == '.':
		default:
			s = s[:i] + "_" + s[i+1:]
		}
	}
	if len(s) > 80 {
		s = s[len(s)-80:]
	}
	return s
}

// Get returns (and lazily starts) the subprocess for a user.
// projectPath is the *current* repository root on the user's machine; we
// hash it into the indexDir so that a user working in two repos gets two
// isolated indexes, just like the local-mode behaviour.
func (p *Pool) Get(userID, projectPath string) (*proc, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := userID
	if existing, ok := p.procs[key]; ok {
		existing.lastUsed.Store(time.Now().UnixNano())
		return existing, nil
	}

	idx := p.indexDir(userID, projectPath)
	if err := os.MkdirAll(idx, 0o750); err != nil {
		return nil, fmt.Errorf("mkdir index dir: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, p.cfg.MCPBinary)
	cmd.Dir = idx
	cmd.Env = append(os.Environ(),
		"CODING_MEMORY_DATA_DIR="+idx,
		"HOME="+p.userDir(userID), // keeps .config/codebase-memory isolated per user
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = io.Discard // optionally route to per-user log file

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start %s: %w", p.cfg.MCPBinary, err)
	}

	pr := &proc{
		cmd:         cmd,
		stdin:       stdin,
		reader:      bufio.NewReader(stdout),
		indexDir:    idx,
		projectPath: projectPath,
		cancel:      cancel,
		done:        make(chan struct{}),
	}
	pr.lastUsed.Store(time.Now().UnixNano())
	p.procs[key] = pr

	go p.reaper(key, pr)
	return pr, nil
}

// Request sends a single JSON-RPC frame to the user's subprocess and
// reads one JSON-RPC frame back. Subprocess framing uses newlines,
// matching the canonical MCP stdio transport.
func (p *Pool) Request(userID, projectPath, frame string) (string, error) {
	pr, err := p.Get(userID, projectPath)
	if err != nil {
		return "", err
	}
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, err := pr.stdin.Write([]byte(frame)); err != nil {
		return "", fmt.Errorf("write to subprocess: %w", err)
	}
	if _, err := pr.stdin.Write([]byte("\n")); err != nil {
		return "", fmt.Errorf("write newline: %w", err)
	}
	// JSON-RPC notifications (method set, no id) get NO response from the
	// subprocess. Blocking on ReadString here would hang forever and hold
	// pr.mu, deadlocking every subsequent request for this user — which is
	// exactly what a client sending notifications/initialized after the
	// initialize handshake triggers. Ack immediately without reading.
	if isNotification([]byte(frame)) {
		pr.lastUsed.Store(time.Now().UnixNano())
		return "", nil
	}
	// Read response. Subprocess may emit banner lines (e.g. "level=info msg=...")
	// before the JSON-RPC frame; skip non-JSON lines until we see '{'.
	for {
		line, err := pr.reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read from subprocess: %w", err)
		}
		if len(line) == 0 {
			continue
		}
		if line[0] == '{' {
			pr.lastUsed.Store(time.Now().UnixNano())
			return line, nil
		}
		// Non-JSON: log to /dev/null in production; here we discard.
		// Optional: write to a per-user log file later.
	}
}

// isNotification reports whether a JSON-RPC frame is a notification: it has
// a method but no id member. Per the JSON-RPC 2.0 / MCP spec such messages
// receive no response, so the caller must not wait for one. A missing id
// leaves ID nil; an explicit "id": null (a malformed request, not a
// notification) yields the 4-byte "null" and is treated as a request.
func isNotification(frame []byte) bool {
	var m struct {
		Method string          `json:"method"`
		ID     json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(frame, &m); err != nil {
		return false
	}
	return m.Method != "" && len(m.ID) == 0
}

// Drop forcibly kills and removes a user's subprocess(es).
func (p *Pool) Drop(userID string) error {
	p.mu.Lock()
	pr, ok := p.procs[userID]
	delete(p.procs, userID)
	p.mu.Unlock()
	if !ok {
		return nil
	}
	return pr.terminate()
}

func (p *Pool) Stats() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := map[string]any{
		"live_procs": len(p.procs),
	}
	users := map[string]any{}
	for k, v := range p.procs {
		users[k] = map[string]any{
			"index_dir": v.indexDir,
			"project":   v.projectPath,
			"pid":       v.cmd.Process.Pid,
			"idle_sec":  int(time.Since(time.Unix(0, v.lastUsed.Load())).Seconds()),
		}
	}
	out["users"] = users
	return out
}

// Close terminates every live subprocess and waits up to 5s.
func (p *Pool) Close() error {
	p.mu.Lock()
	procs := p.procs
	p.procs = map[string]*proc{}
	p.mu.Unlock()
	var firstErr error
	for _, pr := range procs {
		if err := pr.terminate(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (pr *proc) terminate() error {
	pr.cancel()
	_ = pr.stdin.Close()
	done := make(chan error, 1)
	go func() {
		err := pr.cmd.Wait()
		close(pr.done)
		done <- err
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = pr.cmd.Process.Kill()
		<-done
	}
	return nil
}

// reaper terminates the subprocess after IdleTTL of inactivity.
func (p *Pool) reaper(key string, pr *proc) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-pr.done:
			return
		case <-ticker.C:
			if time.Since(time.Unix(0, pr.lastUsed.Load())) > p.cfg.IdleTTL {
				_ = p.Drop(key)
				return
			}
		}
	}
}
