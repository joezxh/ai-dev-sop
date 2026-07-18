package auth

import (
	"errors"
	"testing"
)

func TestHashAndVerifyRoundTrip(t *testing.T) {
	// Cost=4 keeps the test under a millisecond while still exercising
	// the bcrypt code path.
	const pw = "correct-horse-battery-staple"
	h, err := HashWith(pw, 4)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !looksLikeBcrypt(h) {
		t.Fatalf("hash %q does not look like a bcrypt hash", h)
	}
	if err := Verify(h, pw); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	h, err := HashWith("hunter22", 4)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	err = Verify(h, "hunter23")
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch, got %v", err)
	}
}

func TestHashRejectsShortPassword(t *testing.T) {
	_, err := HashWith("short", 4)
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestHashRejectsBadCost(t *testing.T) {
	_, err := HashWith("longenoughpw", 1)
	if !errors.Is(err, ErrPasswordInvalidArg) {
		t.Fatalf("expected ErrPasswordInvalidArg, got %v", err)
	}
	_, err = HashWith("longenoughpw", 99)
	if !errors.Is(err, ErrPasswordInvalidArg) {
		t.Fatalf("expected ErrPasswordInvalidArg (cost too high), got %v", err)
	}
}

func TestVerifyRejectsLegacySentinel(t *testing.T) {
	// The migration stamps "!v1-legacy-must-reset!" on every v1 row.
	// Verify should refuse it so the login flow can route the user to
	// change-password.
	err := Verify("!v1-legacy-must-reset!", "anything")
	if !errors.Is(err, ErrPasswordBadFormat) {
		t.Fatalf("expected ErrPasswordBadFormat for sentinel, got %v", err)
	}
	if err := Verify("", "anything"); !errors.Is(err, ErrPasswordBadFormat) {
		t.Fatalf("expected ErrPasswordBadFormat for empty hash, got %v", err)
	}
}

func TestIsLegacySentinel(t *testing.T) {
	if !IsLegacySentinel("!v1-legacy-must-reset!") {
		t.Errorf("expected sentinel detection on v1 marker")
	}
	if !IsLegacySentinel("") {
		t.Errorf("expected sentinel detection on empty hash")
	}
	if IsLegacySentinel("$2a$10$abc") {
		t.Errorf("real bcrypt hash should not be flagged as sentinel")
	}
}

func TestLooksLikeBcrypt(t *testing.T) {
	if !looksLikeBcrypt("$2a$10$abc") {
		t.Error("$2a$ prefix should match")
	}
	if !looksLikeBcrypt("$2b$10$abc") {
		t.Error("$2b$ prefix should match")
	}
	if !looksLikeBcrypt("$2y$10$abc") {
		t.Error("$2y$ prefix should match")
	}
	if looksLikeBcrypt("not-a-hash") {
		t.Error("plain string should not match")
	}
	if looksLikeBcrypt("") {
		t.Error("empty string should not match")
	}
	if looksLikeBcrypt("$1$") {
		t.Error("md5crypt prefix should not match")
	}
}
