package console

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// 建表后再 migrate 必须幂等
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate idempotent: %v", err)
	}
	// 应能查到 users 表
	var name string
	if err := db.Conn().QueryRowContext(context.Background(),
		"SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&name); err != nil {
		t.Fatalf("users table missing: %v", err)
	}
}
