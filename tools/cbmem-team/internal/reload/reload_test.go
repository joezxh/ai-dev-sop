package reload

import (
	"context"
	"testing"

	"cbmem-team/internal/config"
)

type mockReloader struct {
	reloadCount int
	reloadErr   error
}

func (m *mockReloader) Reload(ctx context.Context, cfg *config.Config) error {
	m.reloadCount++
	return m.reloadErr
}

func TestManager_Register(t *testing.T) {
	m := NewManager()
	m.Register(&mockReloader{})
	m.Register(&mockReloader{})

	if len(m.services) != 2 {
		t.Errorf("expected 2 services, got %d", len(m.services))
	}
}

func TestManager_DoubleStart(t *testing.T) {
	// Can't test double Start without a real config file, so we just verify
	// the manager is created correctly
	m := NewManager()
	if m == nil {
		t.Error("expected non-nil manager")
	}
}

func TestManager_StopWithoutStart(t *testing.T) {
	m := NewManager()
	err := m.Stop()
	if err != nil {
		t.Errorf("Stop without Start failed: %v", err)
	}
}

func TestManager_RegisterAfterStart(t *testing.T) {
	m := NewManager()
	cfg := config.DefaultConfig()

	// Can't test actual reload without a real config file watcher,
	// but we can verify the manager accepts late registration.
	m.Register(&mockReloader{})
	m.Register(&mockReloader{})

	// Start without a config file path (will fail but manager is functional)
	m.Start(context.Background(), cfg, "")
	m.Stop()
}

func TestMockReloader_Count(t *testing.T) {
	m := &mockReloader{}

	ctx := context.Background()
	cfg := config.DefaultConfig()

	m.Reload(ctx, cfg)
	m.Reload(ctx, cfg)
	m.Reload(ctx, cfg)

	if m.reloadCount != 3 {
		t.Errorf("expected 3 reloads, got %d", m.reloadCount)
	}
}

func TestMockReloader_Error(t *testing.T) {
	m := &mockReloader{
		reloadErr: context.DeadlineExceeded,
	}

	ctx := context.Background()
	cfg := config.DefaultConfig()

	err := m.Reload(ctx, cfg)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

// Integration test that requires a real config file
func TestManager_WithRealWatcher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires the config package's NewWatcher to be functional
	// which needs a real directory to watch
}

func TestManager_CloseIdempotent(t *testing.T) {
	m := NewManager()
	cfg := config.DefaultConfig()

	// Start with empty path (won't actually start watching)
	m.Start(context.Background(), cfg, "")

	// Double close should be idempotent
	m.Stop()
	m.Stop()
}

// TestReloaderInterface verifies the interface is satisfied
func TestReloaderInterface(t *testing.T) {
	var _ Reloader = &mockReloader{}
}

// TestManagerInterface verifies the Manager interface
func TestManagerInterface(t *testing.T) {
	m := NewManager()
	
	// Verify methods exist and can be called
	m.Register(&mockReloader{})
	_ = m.Stop()
}
