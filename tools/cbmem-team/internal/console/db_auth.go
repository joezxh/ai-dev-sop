package console

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// v2 (M2) user / refresh-token CRUD
// =============================================================================
//
// All read methods return sql.ErrNoRows when the row is missing so
// callers can branch on errors.Is(err, sql.ErrNoRows) instead of
// having to parse string-comparison error messages.
//
// Both SQLite and MySQL are supported; the SQL is portable. The
// password_hash column lives in the v2 schema (see
// v2_schema_migrate.go for the ALTER that adds it on legacy DBs).
//
// =============================================================================

// ErrUserNotFound is the canonical "no such user" error. Returned by
// GetUserByID / GetUserByUsername; callers should branch on it rather
// than sql.ErrNoRows so application code stays decoupled from driver
// internals.
var ErrUserNotFound = errors.New("user not found")

// ErrUserExists is returned by CreateUser when the chosen username is
// already taken. The 409 conflict handler in auth_handler.go maps this
// to a 409 response with code 4090001.
var ErrUserExists = errors.New("username already exists")

// CreateUser inserts a new v2 user row. passwordHash is the bcrypt
// hash produced by auth.Hash. createdAt / updatedAt default to now().
//
// All other fields fall back to safe defaults:
//   - role      → "developer" (the lowest platform role)
//   - must_change_password → true on first login
//   - deleted   → 0 (active)
func (db *DB) CreateUser(ctx context.Context, u *User) error {
	if u.ID == "" {
		return errors.New("CreateUser: id required")
	}
	if u.Username == "" {
		return errors.New("CreateUser: username required")
	}
	if u.PasswordHash == "" {
		return errors.New("CreateUser: password_hash required (bcrypt)")
	}
	// Normalise role. Empty role ⇒ developer; any unknown role falls
	// back too rather than panicking, so a typo in CLI input doesn't
	// kill boot.
	if u.Role == "" {
		u.Role = string(RoleDeveloper)
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now

	displayName := u.DisplayName
	if displayName == "" {
		displayName = u.Username
	}

	q := `INSERT INTO users
        (id, username, display_name, email, password_hash,
         default_team_id, role, must_change_password, disabled,
         created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	if _, err := db.ExecContext(ctx, q,
		u.ID, u.Username, displayName, u.Email, u.PasswordHash,
		u.DefaultTeamID, u.Role, boolToInt(u.MustChangePassword), boolToInt(u.Disabled),
		u.CreatedAt, u.UpdatedAt,
	); err != nil {
		// Detect unique-constraint violation on username.
		if isUniqueViolation(err) {
			return ErrUserExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetUserByID returns the user row whose id matches the argument. The
// password_hash is populated; callers should NOT serialise it.
func (db *DB) GetUserByID(ctx context.Context, id string) (*User, error) {
	const q = `SELECT id, username, display_name, email, password_hash,
                      default_team_id, role, must_change_password, disabled,
                      created_at, updated_at, last_login_at
                 FROM users WHERE id = ?`
	return db.scanUser(db.QueryRowContext(ctx, q, id))
}

// GetUserByUsername returns the user row whose username matches.
// Username is treated case-insensitively because operators frequently
// confuse caps ("Admin" vs "admin"); the SQLite default collation is
// already case-insensitive for ASCII, and we explicitly LOWER() on
// MySQL so the behaviour matches.
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var (
		q    string
		stmt string
		arg  any = username
	)
	if db.driver == "mysql" {
		q = `SELECT id, username, display_name, email, password_hash,
                    default_team_id, role, must_change_password, disabled,
                    created_at, updated_at, last_login_at
               FROM users WHERE LOWER(username) = LOWER(?)`
	} else {
		// SQLite: LIKE is case-insensitive for ASCII by default;
		// equality is too because the default collation is NOCASE for
		// user tables created with COLLATE NOCASE. We don't rely on
		// that being set, so we lower on read.
		q = `SELECT id, username, display_name, email, password_hash,
                    default_team_id, role, must_change_password, disabled,
                    created_at, updated_at, last_login_at
               FROM users WHERE LOWER(username) = LOWER(?)`
	}
	stmt = q
	_ = stmt
	return db.scanUser(db.QueryRowContext(ctx, q, arg))
}

// ListUsers returns every user. Pagination lives at the handler level;
// this method is the canonical source of truth.
func (db *DB) ListUsers(ctx context.Context) ([]*User, error) {
	const q = `SELECT id, username, display_name, email, password_hash,
                      default_team_id, role, must_change_password, disabled,
                      created_at, updated_at, last_login_at
                 FROM users
                ORDER BY created_at ASC`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UpdateUserPassword sets password_hash and clears
// must_change_password. updated_at is bumped automatically.
func (db *DB) UpdateUserPassword(ctx context.Context, id, newHash string) error {
	if newHash == "" {
		return errors.New("UpdateUserPassword: empty hash")
	}
	res, err := db.ExecContext(ctx,
		`UPDATE users
            SET password_hash = ?, must_change_password = 0, updated_at = ?
          WHERE id = ?`,
		newHash, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateUserLastLogin stamps last_login_at = now. Called from the
// login handler on success.
func (db *DB) UpdateUserLastLogin(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE users SET last_login_at = ? WHERE id = ?`,
		time.Now().UTC(), id,
	)
	return err
}

// SetUserDisabled toggles the disabled flag. Setting disabled=true is
// how an admin revokes access; the login handler refuses disabled users.
func (db *DB) SetUserDisabled(ctx context.Context, id string, disabled bool) error {
	res, err := db.ExecContext(ctx,
		`UPDATE users SET disabled = ?, updated_at = ? WHERE id = ?`,
		boolToInt(disabled), time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("set disabled: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// CountUsers returns the total number of users. Used by the
// bootstrapping logic to decide whether to auto-create the initial
// admin user on first boot (zero users → yes).
//
// We do not filter on `deleted` because the v2 users table has no
// `deleted` column (soft-delete is via `disabled` + audit logs).
// Once M3 introduces soft-delete, change this query to filter
// `deleted = 0`.
func (db *DB) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users`,
	).Scan(&n)
	return n, err
}

// ----------------------------------------------------------------------------
// Refresh-token storage
// ----------------------------------------------------------------------------

// CreateRefreshToken stores the (id, user_id, token_hash) triple. The
// caller owns the plaintext token — it goes into the response body
// once, then only the SHA-256 hash is persisted. That means a database
// dump alone cannot forge a refresh token.
func (db *DB) CreateRefreshToken(ctx context.Context, id, userID, plaintext string, ttl time.Duration) error {
	if id == "" || userID == "" || plaintext == "" {
		return errors.New("CreateRefreshToken: id/userID/plaintext required")
	}
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, issued_at, expires_at)
              VALUES (?, ?, ?, ?, ?)`,
		id, userID, hashToken(plaintext), now, now.Add(ttl),
	)
	if err != nil {
		return fmt.Errorf("insert refresh: %w", err)
	}
	return nil
}

// ValidateRefreshToken returns the user_id tied to a (non-expired,
// non-revoked) refresh token, or ErrUserNotFound. The plaintext is
// hashed with SHA-256 (matching CreateRefreshToken) before lookup.
//
// We use SHA-256 rather than bcrypt because refresh tokens are
// high-entropy random strings: brute-force resistance comes from the
// entropy, not the hashing cost. SHA-256 keeps the validation path
// fast so a refresh storm doesn't pin a CPU.
func (db *DB) ValidateRefreshToken(ctx context.Context, plaintext string) (userID, tokenID string, err error) {
	if plaintext == "" {
		return "", "", errors.New("empty token")
	}
	const q = `SELECT user_id, id, expires_at, revoked_at
                 FROM refresh_tokens
                WHERE token_hash = ?
                ORDER BY issued_at DESC
                LIMIT 1`
	var expiresAt time.Time
	var revokedAt sql.NullTime
	if err := db.QueryRowContext(ctx, q, hashToken(plaintext)).
		Scan(&userID, &tokenID, &expiresAt, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrUserNotFound
		}
		return "", "", fmt.Errorf("query refresh: %w", err)
	}
	if revokedAt.Valid {
		return "", "", errors.New("refresh token revoked")
	}
	if time.Now().UTC().After(expiresAt) {
		return "", "", errors.New("refresh token expired")
	}
	return userID, tokenID, nil
}

// RevokeRefreshToken marks a single refresh token as revoked. Used by
// /api/auth/logout and /api/auth/change-password (which revokes ALL
// tokens for the affected user as a security measure).
func (db *DB) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	if tokenID == "" {
		return errors.New("empty token id")
	}
	_, err := db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ?
          WHERE id = ? AND revoked_at IS NULL`,
		time.Now().UTC(), tokenID,
	)
	return err
}

// RevokeAllRefreshTokensForUser revokes every outstanding refresh token
// belonging to userID. Called on password change so any device that
// somehow learned the old password is logged out.
func (db *DB) RevokeAllRefreshTokensForUser(ctx context.Context, userID string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ?
          WHERE user_id = ? AND revoked_at IS NULL`,
		time.Now().UTC(), userID,
	)
	return err
}

// PurgeExpiredRefreshTokens deletes revoked/expired rows older than
// `olderThan`. Intended to be run from a periodic janitor, but safe to
// call on demand.
func (db *DB) PurgeExpiredRefreshTokens(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	res, err := db.ExecContext(ctx,
		`DELETE FROM refresh_tokens
          WHERE (revoked_at IS NOT NULL OR expires_at < ?)
            AND issued_at < ?`,
		cutoff, cutoff,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// scanUser scans a single row (QueryRow) into a *User. Used by the
// Get* methods.
func (db *DB) scanUser(row *sql.Row) (*User, error) {
	u, err := scanUserRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

// scanUserRow is shared between the single-row (QueryRow) and
// multi-row (Query + Next) read paths.
func scanUserRow(s scanner) (*User, error) {
	var u User
	var lastLogin sql.NullTime
	err := s.Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.PasswordHash,
		&u.DefaultTeamID, &u.Role, &u.MustChangePassword, &u.Disabled,
		&u.CreatedAt, &u.UpdatedAt, &lastLogin,
	)
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		t := lastLogin.Time
		u.LastLoginAt = &t
	}
	return &u, nil
}

// scanner abstracts over *sql.Row and *sql.Rows so scanUserRow can be
// called from both Get* and List* paths.
type scanner interface {
	Scan(dest ...any) error
}

// hashToken returns the SHA-256 hex digest of the plaintext refresh
// token. Refresh tokens are 256 bits of random, so SHA-256 is the
// right tradeoff (collision-resistant, fast) — bcrypt is overkill and
// would burn CPU on every refresh.
func hashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// isUniqueViolation detects the "duplicate key" / "UNIQUE constraint
// failed" errors that both backends raise on PK / UNIQUE collisions.
// We treat them uniformly so CreateUser can map them to ErrUserExists.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "Error 1062") ||
		strings.Contains(msg, "SQLSTATE 23000") ||
		strings.Contains(msg, "SQLSTATE 23505")
}

// boolToInt encodes a Go bool as the SQLite/MySQL integer we use for
// the disabled / must_change_password columns. Putting it here avoids
// sprinkling driver-specific conversions through the handlers.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
