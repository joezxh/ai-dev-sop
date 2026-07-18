package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Password errors. Exported so handlers can branch on them without
// leaking bcrypt's internal error strings.
var (
	ErrPasswordMismatch   = errors.New("password mismatch")
	ErrPasswordTooShort   = errors.New("password too short (min 8 chars)")
	ErrPasswordBadFormat  = errors.New("password hash unrecognised")
	ErrPasswordInvalidArg = errors.New("bcrypt invalid argument (likely cost out of range)")
)

// BcryptCost is the work-factor used by Hash. 12 is a 2026-era default
// (~250ms on commodity hardware) — high enough to deter offline brute
// force, low enough not to noticeably tax logins. Override via HashWith.
const BcryptCost = 12

// DefaultBcryptCost returns BcryptCost as an int for callers that
// take a command-line flag with a default. Exists so the CLI flag
// can mirror the package constant without a separate hardcoded
// number.
func DefaultBcryptCost() int { return BcryptCost }

// Hash returns a bcrypt hash of the plaintext password. The cost is
// BcryptCost; for one-off tuning (e.g. tests) use HashWith.
func Hash(plaintext string) (string, error) {
	return HashWith(plaintext, BcryptCost)
}

// HashWith is Hash with an explicit cost. The bcrypt package enforces
// cost in [4, 31]; we mirror that as ErrPasswordInvalidArg so callers
// can surface a friendly 400 instead of a stack trace.
func HashWith(plaintext string, cost int) (string, error) {
	if len(plaintext) < 8 {
		return "", ErrPasswordTooShort
	}
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return "", fmt.Errorf("%w: cost=%d", ErrPasswordInvalidArg, cost)
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plaintext), cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}
	return string(b), nil
}

// Verify reports whether plaintext matches the previously-hashed hash.
// Returns nil on success. The "must reset" sentinel hashes that v1 →
// v2 migration stamps on legacy users are deliberately incompatible:
// any non-bcrypt hash returns ErrPasswordBadFormat so the handler can
// force the password-reset path.
//
// Legacy sentinel detection: bcrypt hashes always start with "$2a$",
// "$2b$", or "$2y$". Anything else is a placeholder and must be
// rejected so login flows route the user to the change-password UI.
func Verify(hash, plaintext string) error {
	if !looksLikeBcrypt(hash) {
		return ErrPasswordBadFormat
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	if err == nil {
		return nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrPasswordMismatch
	}
	return fmt.Errorf("bcrypt compare: %w", err)
}

func looksLikeBcrypt(hash string) bool {
	if len(hash) < 4 {
		return false
	}
	// bcrypt prefixes: "$2a$", "$2b$", "$2y$", "$2x$"
	prefix := hash[:4]
	return prefix == "$2a$" || prefix == "$2b$" || prefix == "$2y$" || prefix == "$2x$"
}

// IsLegacySentinel reports whether the hash is one of the migration
// sentinels (e.g. "!v1-legacy-must-reset!") that the v2 migrator stamps
// onto users carried over from v1. Callers should force a password
// reset when this returns true.
func IsLegacySentinel(hash string) bool {
	switch hash {
	case "!v1-legacy-must-reset!", "":
		return true
	}
	return false
}
