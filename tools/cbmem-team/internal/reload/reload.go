// Package reload contains the hot-reload logic for cbmem-team.
// It provides a Reloader interface and a default implementation that
// reloads configuration and restarts services.
package reload

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"cbmem-team/internal/config"
)

// Reloader is the interface for hot-reloadable services.
// Reload is called when the config file changes.
type Reloader interface {
	Reload(ctx context.Context, cfg *config.Config) error
}

// Manager manages hot reloading of services.
type Manager struct {
	mu       sync.Mutex
	watcher  *config.Watcher
	services []Reloader
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewManager creates a new hot reload manager.
func NewManager() *Manager {
	return &Manager{
		stopCh: make(chan struct{}),
	}
}

// Register adds a service to the reload manager.
func (m *Manager) Register(svc Reloader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services = append(m.services, svc)
}

// Start begins watching the config file and triggers reloads.
func (m *Manager) Start(ctx context.Context, cfg *config.Config, configPath string) error {
	m.mu.Lock()
	if m.watcher != nil {
		m.mu.Unlock()
		return errors.New("hot reload already started")
	}
	m.mu.Unlock()

	watcher, err := config.NewWatcher(configPath)
	if err != nil {
		return err
	}
	m.watcher = watcher

	m.wg.Add(1)
	go m.run(ctx, cfg)

	return nil
}

func (m *Manager) run(ctx context.Context, cfg *config.Config) {
	defer m.wg.Done()

	debounce := time.NewTicker(500 * time.Millisecond)
	defer debounce.Stop()

	for {
		select {
		case <-m.watcher.C():
			// Debounce: wait a bit to avoid rapid reloads.
			select {
			case <-debounce.C:
			case <-ctx.Done():
				return
			}

			m.doReload(ctx, cfg)

		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) doReload(ctx context.Context, cfg *config.Config) {
	log.Println("hot reload: config changed, reloading...")

	// Reload config from file.
	newCfg := config.LoadConfig(cfg.Listen) // Keep current listen/CLI overrides via a copy
	// In practice, we reload the file and merge with the original CLI overrides.
	// For now, we just notify services with the current config.

	m.mu.Lock()
	services := make([]Reloader, len(m.services))
	copy(services, m.services)
	m.mu.Unlock()

	var lastErr error
	for _, svc := range services {
		if err := svc.Reload(ctx, newCfg); err != nil {
			log.Printf("hot reload: service %T reload failed: %v", svc, err)
			lastErr = err
		}
	}

	if lastErr != nil {
		log.Printf("hot reload: completed with errors")
	} else {
		log.Println("hot reload: completed successfully")
	}
}

// Stop stops the reload manager.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watcher == nil {
		return nil
	}

	close(m.stopCh)
	m.wg.Wait()

	err := m.watcher.Close()
	m.watcher = nil
	return err
}
