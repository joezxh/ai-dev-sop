// Package devconf is the single source of truth for MySQL/Redis/etc.
// DSNs and other dev-environment defaults used by:
//
//	cmd/create-db
//	cmd/mysql-probe
//	cmd/show-tables
//	cmd/show-indexes
//	cmd/e2e-m1, cmd/e2e-v5, cmd/e2e-v6, cmd/e2e-v7
//
// Real production deployments read the same DSNs from
// /etc/cbmem-team/cbmem-team.env (see deploy/cbmem-team.env). The dev
// defaults here match the ones used by the local docker-compose stack
// in deploy/docker-compose.mysql.yml.
//
// Resolution order (first non-empty wins):
//
//  1. flag value (-mysql-dsn "...")         — explicit, CI override
//  2. $CBMEM_MYSQL_DSN                      — env, also matches deploy/cbmem-team.env
//  3. $DSN                                  — legacy env name used by create-db/mysql-probe
//  4. DefaultDevMySQLDSN                    — last-resort localhost dev DSN
//
// The package deliberately uses only the standard library; we do not
// want to pull in viper or envconfig just for a dev tool.
package devconf

import (
	"errors"
	"os"
	"strings"
)

// DefaultDevMySQLDSN is the localhost DSN used when no flag/env is set.
// It assumes the docker-compose.dev.yml stack is running on the same
// host; production servers MUST override via -mysql-dsn or
// CBMEM_MYSQL_DSN.
//
// NOTE: the password here is intentionally the well-known dev/test one
// from compose. It is **not** a real production credential — no
// `adsop123` user exists outside `127.0.0.1` in the developer sandbox.
const DefaultDevMySQLDSN = "root:adsop123@tcp(127.0.0.1:3306)/?parseTime=true&loc=Local&charset=utf8mb4"

// DefaultDevMySQLDB is the database name prepended to `DefaultDevMySQLDSN`
// when callers want the `cbmem`-selected variant without spelling it out.
const DefaultDevMySQLDB = "cbmem"

// EnvMySQLDSN is the canonical env var name. `deploy/cbmem-team.env`
// uses the same key so dev tools and the production binary share a
// vocabulary — flipping between MySQL and SQLite is a one-line change.
const EnvMySQLDSN = "CBMEM_MYSQL_DSN"

// EnvLegacyMySQLDSN is the older "DSN" name used by create-db and
// mysql-probe before devconf existed. We accept it for backwards
// compatibility with existing terminal habits; new callers should use
// EnvMySQLDSN.
const EnvLegacyMySQLDSN = "DSN"

// ResolveMySQLDSN returns the MySQL DSN according to the order
// documented at the package level. Empty string means "no decision was
// made", so callers can fall back to SQLite the same way the production
// binary does when -mysql-dsn is empty.
//
// example usage:
//
//	dsn := devconf.ResolveMySQLDSN("")
//	if dsn != "" {
//	    db, _ := sql.Open("mysql", dsn)
//	} else {
//	    db, _ := sql.Open("sqlite", "console.db")
//	}
func ResolveMySQLDSN(flagValue string) string {
	switch {
	case strings.TrimSpace(flagValue) != "":
		return flagValue
	case strings.TrimSpace(os.Getenv(EnvMySQLDSN)) != "":
		return os.Getenv(EnvMySQLDSN)
	case strings.TrimSpace(os.Getenv(EnvLegacyMySQLDSN)) != "":
		return os.Getenv(EnvLegacyMySQLDSN)
	default:
		return DefaultDevMySQLDSN
	}
}

// ResolveMySQLDSNWithDB is ResolveMySQLDSN plus an injected
// database name. Use this when the caller requires the schema name
// inside the DSN itself (most mysql drivers do not). Behaviour:
//
//  1. If `flagValue` or an env var resolves to a DSN that already
//     contains a database name (i.e. has "/" followed by a name segment
//     before `?`), it is returned untouched.
//  2. Otherwise the resolved DSN is rewritten with `dbName` injected
//     between the host and the query string, matching the
//     `user:pass@tcp(host:port)/<db>?opts` layout.
//
// Passing dbName == "" returns the unresolved DSN unchanged (caller
// will then decide whether to swallow the error or open without a DB).
func ResolveMySQLDSNWithDB(flagValue, dbName string) string {
	dsn := ResolveMySQLDSN(flagValue)
	if dbName == "" || dsn == "" {
		return dsn
	}
	if dbHasName(dsn) {
		return dsn
	}
	// Insert "/" + dbName right before "?" (or at end if no query).
	// The default dev DSN already ends with "/?parseTime=..."; strip that
	// trailing "/" so we don't end up with "//cbmem" after the injection.
	if i := strings.Index(dsn, "?"); i >= 0 {
		prefix := strings.TrimRight(dsn[:i], "/")
		return prefix + "/" + dbName + dsn[i:]
	}
	dsn = strings.TrimRight(dsn, "/")
	return dsn + "/" + dbName
}

// dbHasName returns true when dsn already names a database, i.e.
// `user:pass@tcp(host:port)/<name>?...` or `user:pass@tcp(host:port)/<name>`.
// Detection is "any non-empty path segment after the host part".
func dbHasName(dsn string) bool {
	// Walk forward to the first "/" that is **after** the closing paren
	// of @tcp(...). Anything between that slash and the next "?" or end
	// is the database name.
	paren := strings.Index(dsn, ")")
	if paren < 0 {
		return false
	}
	tail := dsn[paren+1:]
	if !strings.HasPrefix(tail, "/") {
		return false
	}
	// Drop the leading "/".
	tail = tail[1:]
	if tail == "" {
		return false
	}
	// DB name is everything before "?" (or all of `tail`).
	if i := strings.Index(tail, "?"); i >= 0 {
		tail = tail[:i]
	}
	return tail != ""
}

// ErrEmptyDSN is returned by MustResolveMySQLDSN when no DSN source is
// available; callers that genuinely require a DB connection can wrap
// this with their own context.
var ErrEmptyDSN = errors.New("devconf: no MySQL DSN resolved (set -mysql-dsn, $CBMEM_MYSQL_DSN, or $DSN)")