// Package repos manages per-user git clones on the server.
//
// Each (userID, repoURL) pair is mapped to a deterministic local path:
//
//	${DataDir}/users/${userID}/repos/${safeName}/
//
// safeName is derived from the URL host + path so that two users cloning the
// same upstream do not collide. The clone is performed synchronously on
// first request; subsequent requests run `git fetch` to keep the mirror
// up-to-date without blocking the request goroutine.
package repos

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Repo struct {
	UserID    string    `json:"user_id"`
	URL       string    `json:"url"`
	Name      string    `json:"name"`
	LocalPath string    `json:"local_path"`
	ClonedAt  time.Time `json:"cloned_at"`
	LastPull  time.Time `json:"last_pull,omitempty"`
	LastErr   string    `json:"last_error,omitempty"`
}

type Manager struct {
	dataDir string
	mu      sync.Mutex
	// in-memory mirror of on-disk state; source of truth is the directory listing
	byUser map[string][]*Repo
}

// New returns a Manager rooted at dataDir/users/<id>/repos.
// Existing clones on disk are discovered on construction so that restarts
// don't lose track of repos already cloned.
func New(dataDir string) (*Manager, error) {
	m := &Manager{
		dataDir: dataDir,
		byUser:  map[string][]*Repo{},
	}
	usersRoot := filepath.Join(dataDir, "users")
	entries, err := os.ReadDir(usersRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return m, nil
		}
		return nil, fmt.Errorf("read users dir: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		userID := e.Name()
		reposDir := filepath.Join(usersRoot, userID, "repos")
		rEntries, err := os.ReadDir(reposDir)
		if err != nil {
			continue // user might have no repos yet
		}
		for _, re := range rEntries {
			if !re.IsDir() {
				continue
			}
			r := &Repo{
				UserID:    userID,
				Name:      re.Name(),
				LocalPath: filepath.Join(reposDir, re.Name()),
			}
			// recover the original URL from a sidecar file written by Clone
			if data, err := os.ReadFile(filepath.Join(r.LocalPath, ".cbmem-repo-url")); err == nil {
				r.URL = strings.TrimSpace(string(data))
			}
			if st, err := os.Stat(filepath.Join(r.LocalPath, ".git")); err == nil {
				r.ClonedAt = st.ModTime()
			}
			m.byUser[userID] = append(m.byUser[userID], r)
		}
	}
	return m, nil
}

// SafeName returns a filesystem-safe unique-ish name for a URL.
// We keep the host + repo path so duplicates across hosts do not collide.
func SafeName(rawURL string) (string, error) {
	// Handle SCP-style SSH URLs: git@github.com:foo/bar.git
	// Convert to a parseable form by replacing the first colon after '@'
	// with a slash.
	norm := rawURL
	if strings.HasPrefix(norm, "git@") {
		// e.g. git@github.com:foo/bar.git -> we treat host as github.com, path as foo/bar
		atIdx := strings.Index(norm, "@")
		colonIdx := strings.Index(norm, ":")
		if atIdx >= 0 && colonIdx > atIdx {
			norm = "ssh://" + strings.Replace(norm, ":", "/", 1)
		}
	}

	u, err := url.Parse(norm)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("url must include scheme and host: %q", rawURL)
	}
	host := u.Host
	p := strings.Trim(u.Path, "/")
	if p == "" {
		return "", fmt.Errorf("url must include a path: %q", rawURL)
	}
	// strip ".git" suffix for nicer names
	p = strings.TrimSuffix(p, ".git")
	name := host + "__" + strings.ReplaceAll(p, "/", "_")
	// sanitise
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if len(out) > 120 {
		out = out[len(out)-120:]
	}
	return out, nil
}

// Clone performs `git clone` for the given (userID, url) pair. Idempotent:
// if the local directory already exists and is a git repo, we fall back
// to a `git fetch` so admins can re-trigger a sync.
//
// Returns the local path on success; admins should add that path to
// user.ProjectPaths so /mcp requests can target it.
func (m *Manager) Clone(ctx context.Context, userID, rawURL string) (*Repo, error) {
	name, err := SafeName(rawURL)
	if err != nil {
		return nil, err
	}
	userRoot := filepath.Join(m.dataDir, "users", userID)
	if err := os.MkdirAll(userRoot, 0o750); err != nil {
		return nil, fmt.Errorf("mkdir user root: %w", err)
	}
	reposDir := filepath.Join(userRoot, "repos")
	if err := os.MkdirAll(reposDir, 0o750); err != nil {
		return nil, fmt.Errorf("mkdir repos: %w", err)
	}
	localPath := filepath.Join(reposDir, name)

	r := &Repo{
		UserID:    userID,
		URL:       rawURL,
		Name:      name,
		LocalPath: localPath,
	}

	// Sidecar file records the original URL even after a clone strips it
	urlSidecar := filepath.Join(localPath, ".cbmem-repo-url")

	gitDir := filepath.Join(localPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Already cloned - fetch latest
		fetchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(fetchCtx, "git", "-C", localPath, "fetch", "--prune", "--tags", "origin")
		out, ferr := cmd.CombinedOutput()
		if ferr != nil {
			r.LastErr = fmt.Sprintf("fetch failed: %v: %s", ferr, truncate(string(out), 512))
			m.record(userID, r)
			return r, fmt.Errorf("git fetch: %w: %s", ferr, truncate(string(out), 512))
		}
		r.LastPull = time.Now().UTC()
		m.record(userID, r)
		return r, nil
	}

	// Fresh clone. Use a 5-minute ceiling so a hung HTTPS doesn't lock admins out.
	cloneCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Ensure parent exists; git clone requires it to be empty or absent.
	if err := os.MkdirAll(localPath, 0o750); err != nil {
		return nil, fmt.Errorf("mkdir clone target: %w", err)
	}
	cmd := exec.CommandContext(cloneCtx, "git", "clone", "--no-local", rawURL, localPath)
	// Strip GIT_ASKPASS / SSH_ASKPASS so any interactive prompt surfaces in logs.
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Best-effort cleanup so a retry doesn't trip over a half-cloned dir.
		_ = os.RemoveAll(localPath)
		r.LastErr = fmt.Sprintf("clone failed: %v: %s", err, truncate(string(out), 512))
		m.record(userID, r)
		return r, fmt.Errorf("git clone: %w: %s", err, truncate(string(out), 512))
	}
	if err := os.WriteFile(urlSidecar, []byte(rawURL), 0o600); err != nil {
		// non-fatal - clone succeeded; we just lose URL metadata
		r.LastErr = "sidecar write failed: " + err.Error()
	}
	r.ClonedAt = time.Now().UTC()
	m.record(userID, r)
	return r, nil
}

// List returns a defensive copy of all known repos for a user.
func (m *Manager) List(userID string) []*Repo {
	m.mu.Lock()
	defer m.mu.Unlock()
	src := m.byUser[userID]
	out := make([]*Repo, len(src))
	for i, r := range src {
		cp := *r
		out[i] = &cp
	}
	return out
}

// record inserts or replaces a Repo in the in-memory mirror.
func (m *Manager) record(userID string, r *Repo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, existing := range m.byUser[userID] {
		if existing.LocalPath == r.LocalPath {
			m.byUser[userID][i] = r
			return
		}
	}
	m.byUser[userID] = append(m.byUser[userID], r)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
