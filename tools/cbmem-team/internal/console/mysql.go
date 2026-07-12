package console

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

// OpenEither returns a DB backed by either MySQL (when mysqlDSN is set) or
// SQLite (when mysqlDSN is empty and sqlitePath is non-empty). The caller's
// existing code paths keep working unchanged; only the wiring in main.go
// needs to switch on the new flag.
//
// Both backends share the same Go *sql.DB contract, so handler code is
// portable. The differences between dialects are documented in
// docs/superpowers/specs/2026-07-11-cbmem-team-v2-research.md §3.2.
func OpenEither(sqlitePath, mysqlDSN string, opts MySQLOptions) (*DB, error) {
	if mysqlDSN != "" {
		return OpenMySQL(mysqlDSN, opts)
	}
	if sqlitePath == "" {
		return nil, errors.New("OpenEither: either sqlitePath or mysqlDSN required")
	}
	return Open(sqlitePath)
}

// MySQLOptions tunes the connection pool. The zero value yields sane
// defaults that match MySQL 8.0 defaults; pass via flags to override.
type MySQLOptions struct {
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  time.Duration
}

func (o MySQLOptions) withDefaults() MySQLOptions {
	if o.MaxOpenConns <= 0 {
		o.MaxOpenConns = 16
	}
	if o.MaxIdleConns <= 0 {
		o.MaxIdleConns = 4
	}
	if o.ConnMaxLife <= 0 {
		o.ConnMaxLife = 30 * time.Minute
	}
	return o
}

// OpenMySQL opens a MySQL 8.0+ console DB.
//
// Required DSN fields: parseTime=true, loc=Local, charset=utf8mb4. The DSN
// must NOT include "multiStatements=true" except for ETL runs — production
// traffic uses single-statement SQL everywhere, so multiStatements stays
// off as a default and is enabled only by the ETL wrapper.
func OpenMySQL(dsn string, opts MySQLOptions) (*DB, error) {
	opts = opts.withDefaults()
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLife)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return &DB{DB: sqlDB, driver: "mysql", dsn: dsn}, nil
}

// PingMySQL is used by `cbmem-team mysql-ping` to verify a DSN without
// opening the rest of the runtime. It opens the connection, pings, closes.
func PingMySQL(ctx context.Context, dsn string) error {
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return sqlDB.PingContext(ctx)
}

// EscapePath is a small helper preserved for SQLite callers — kept here
// instead of in db.go so the mysql-only build doesn't accidentally depend
// on url import order.
func EscapePath(p string) string { return url.PathEscape(p) }
