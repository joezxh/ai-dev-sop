// Package store manages the persistent user registry.
//
// Each user has:
//   - id: stable unique handle (used in JWT and as filesystem name)
//   - display_name
//   - project_paths: list of repository roots that the user is allowed to index
//   - max_procs: optional override of pool capacity
//
// The registry is JSON on disk at ${DataDir}/users.json so admins can
// edit / version-control it. Concurrency is guarded by an internal RWMutex.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type User struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"display_name"`
	ProjectPaths []string  `json:"project_paths"`
	MaxProcs     int       `json:"max_procs,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	Disabled     bool      `json:"disabled,omitempty"`
}

type Registry struct {
	mu   sync.RWMutex
	byID map[string]*User
	path string
}

func LoadUsers(path string) (*Registry, error) {
	r := &Registry{byID: map[string]*User{}, path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var list []*User
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for _, u := range list {
		if u.ID == "" {
			return nil, errors.New("user with empty id")
		}
		r.byID[u.ID] = u
	}
	return r, nil
}

func (r *Registry) List() []*User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*User, 0, len(r.byID))
	for _, u := range r.byID {
		cp := *u
		out = append(out, &cp)
	}
	return out
}

func (r *Registry) Get(id string) (*User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, false
	}
	cp := *u
	return &cp, true
}

func (r *Registry) Upsert(u *User) error {
	if u.ID == "" {
		return errors.New("user.id required")
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[u.ID] = u
	return r.flushLocked()
}

func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byID, id)
	return r.flushLocked()
}

func (r *Registry) flushLocked() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	list := make([]*User, 0, len(r.byID))
	for _, u := range r.byID {
		list = append(list, u)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}
