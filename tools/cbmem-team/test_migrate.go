package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cbmem-team/internal/console"
)

func main() {
	// Exactly match what main.go does:
	// DataDir defaults to /var/lib/cbmem-team on Unix, but on Windows?
	dataDir := "/var/lib/cbmem-team"
	if runtimeOS := os.Getenv("GOOS"); runtimeOS == "" {
		// Default
	}

	dbPath := filepath.Join(dataDir, "cbmem-team.db")
	fmt.Println("DB path:", dbPath)

	os.Remove(dbPath)
	os.Remove(dbPath + "-wal")
	os.Remove(dbPath + "-shm")

	db, err := console.Open(dbPath)
	if err != nil {
		log.Fatal("open:", err)
	}
	defer db.Close()
	fmt.Println("DB opened, driver:", db.Driver())

	fmt.Println("\nCalling Migrate (same as Mount does)...")
	err = db.Migrate(context.Background(),
		console.M1ExtraSchema(),
		console.M2ExtraSchema(),
		console.M3ExtraSchema(),
		console.M4ExtraSchema())
	if err != nil {
		fmt.Println("Migrate FAILED:", err)
		return
	}
	fmt.Println("Migrate OK")

	fmt.Println("\nCalling MigrateSessionsAutoSyncColumns...")
	err = db.MigrateSessionsAutoSyncColumns(context.Background())
	if err != nil {
		fmt.Println("MigrateSessionsAutoSyncColumns FAILED:", err)
		return
	}
	fmt.Println("MigrateSessionsAutoSyncColumns OK")

	fmt.Println("\nCalling MigrateV2...")
	err = db.MigrateV2(context.Background(), nil)
	if err != nil {
		fmt.Println("MigrateV2 FAILED:", err)
		return
	}
	fmt.Println("MigrateV2 OK")

	// Verify
	rows, err := db.QueryContext(context.Background(), "PRAGMA table_info(sessions)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fmt.Println("\nSessions columns:")
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt interface{}
		rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		fmt.Printf("  %s %s\n", name, ctype)
	}
}
