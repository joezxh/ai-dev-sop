// Tests for the M4 repo-pipeline runtime helpers. These are pure-function
// tests with no DB / network — the goal is to nail down the contract for
// the stages that downstream pipeline code relies on:
//   - URL parsing (parseGitHubTarget)
//   - text shaping (firstSentence)
//   - dedup (hashForDedup)
//   - similarity (jaccardTitleScore, tokenize)
//   - grading (HeuristicGrader.Grade)
//   - serialisation (marshalStageStatus, marshalCandidateTags)
//   - query parsing (atoiOrDefault)
package console

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// --- parseGitHubTarget ---------------------------------------------------

func TestParseGitHubTarget(t *testing.T) {
	cases := []struct {
		in        string
		wantOK    bool
		wantOwner string
		wantRepo  string
	}{
		{"https://github.com/golang/go", true, "golang", "go"},
		{"http://github.com/golang/go", true, "golang", "go"},
		{"github.com/golang/go", true, "golang", "go"},
		{"  https://github.com/golang/go/  ", true, "golang", "go"},
		{"https://github.com/golang/go/issues/123", true, "golang", "go"},
		{"https://gitlab.com/x/y", false, "", ""},
		{"", false, "", ""},
		{"https://github.com/", false, "", ""},
		{"https://github.com/onlyowner", false, "", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			owner, repo, ok := parseGitHubTarget(c.in)
			if ok != c.wantOK || owner != c.wantOwner || repo != c.wantRepo {
				t.Errorf("parseGitHubTarget(%q) = (%q,%q,%v), want (%q,%q,%v)",
					c.in, owner, repo, ok, c.wantOwner, c.wantRepo, c.wantOK)
			}
		})
	}
}

// --- firstSentence -------------------------------------------------------

func TestFirstSentence(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"   \n  ", ""},
		{"Hello world.", "Hello world."},
		{"First sentence. Second sentence.", "First sentence."},
		{"Only sentence without terminator", "Only sentence without terminator"},
		{"Multi\nline break terminates.", "Multi"},
		// 240-char safety cap: long body without dot or newline is truncated.
		{strings.Repeat("a", 300), strings.Repeat("a", 241)},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := firstSentence(c.in); got != c.want {
				t.Errorf("firstSentence(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// --- hashForDedup --------------------------------------------------------

func TestHashForDedup(t *testing.T) {
	// Determinism: same input -> same hash.
	a := hashForDedup("https://example.com/post-1")
	b := hashForDedup("https://example.com/post-1")
	if a != b {
		t.Errorf("expected deterministic hash, got %q vs %q", a, b)
	}
	// Stability: 12-char hex prefix of sha1(s).
	if len(a) != 12 {
		t.Errorf("expected 12-char hex hash, got len=%d (%q)", len(a), a)
	}
	for _, r := range a {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Errorf("hash contains non-hex rune %q in %q", r, a)
			break
		}
	}
	// Distinct inputs collide with negligible probability; if they do,
	// sha1 is broken.
	if hashForDedup("a") == hashForDedup("b") {
		t.Errorf("hash collision for distinct inputs")
	}
}

// --- tokenize + jaccardTitleScore ---------------------------------------

func TestTokenize(t *testing.T) {
	tk := tokenize("Hello, World! hello WORLD 2026")
	// Case-folded, punctuation stripped, length>1 only.
	want := map[string]struct{}{"hello": {}, "world": {}, "2026": {}}
	if len(tk) != len(want) {
		t.Fatalf("tokenize size = %d, want %d (%v)", len(tk), len(want), tk)
	}
	for k := range want {
		if _, ok := tk[k]; !ok {
			t.Errorf("tokenize missing %q (got %v)", k, tk)
		}
	}
}

func TestJaccardTitleScore(t *testing.T) {
	cases := []struct {
		a, b string
		want  float64
	}{
		// Identical -> 1.0
		{"Go testing tips", "Go testing tips", 1.0},
		// Empty input -> 0
		{"", "anything", 0.0},
		{"anything", "", 0.0},
		{"", "", 0.0},
		// Disjoint vocab -> 0
		{"alpha beta", "gamma delta", 0.0},
		// Overlap: {go, testing, tips} ∩ {go, testing, advanced} = {go, testing}
		//           union = {go, testing, tips, advanced} = 4 → 2/4 = 0.5
		{"go testing tips", "go testing advanced", 0.5},
		// Case-insensitive: same as above but mixed case.
		{"GO Testing Tips", "go TESTING advanced", 0.5},
		// Punctuation ignored: same set as "go testing advanced".
		{"go testing, tips!", "go-testing-advanced", 0.5},
	}
	for _, c := range cases {
		t.Run(c.a+" vs "+c.b, func(t *testing.T) {
			got := jaccardTitleScore(c.a, c.b)
			// float tolerance 1e-9 is overkill but harmless
			if diff := got - c.want; diff < -1e-9 || diff > 1e-9 {
				t.Errorf("jaccard(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

// --- HeuristicGrader -----------------------------------------------------

func TestHeuristicGrader_Grade(t *testing.T) {
	g := &HeuristicGrader{}
	ctx := context.Background()

	// Below-minimum body returns the floor (0.2) regardless of length.
	if got, _ := g.Grade(ctx, "", "short", ""); got != 0.2 {
		t.Errorf("short body grade = %v, want 0.2", got)
	}

	// Long + structured body + good title + summary should comfortably
	// exceed DefaultGradeThreshold (0.7).
	long := strings.Repeat("body ", 200) + "\n## Section\nMore text"
	if got, _ := g.Grade(ctx, "Heuristic grader for M4 repo pipeline", long, "summary here"); got < 0.7 {
		t.Errorf("expected ≥0.7, got %v", got)
	}

	// Score is capped at 0.95.
	giga := strings.Repeat("x", 5000) + "\n## h\n### h\n### h"
	if got, _ := g.Grade(ctx, "ok title", giga, "summary"); got > 0.95 {
		t.Errorf("score above cap: %v", got)
	}

	// Title length window: titles ≤8 chars forfeit the 0.1 bump.
	withShortTitle, _ := g.Grade(ctx, "short", long, "")
	withLongTitle, _ := g.Grade(ctx, "A title of the right length", long, "")
	if withLongTitle <= withShortTitle {
		t.Errorf("longer title should not score lower (%v vs %v)",
			withLongTitle, withShortTitle)
	}
}

// --- marshal helpers -----------------------------------------------------

func TestMarshalStageStatus(t *testing.T) {
	s, err := marshalStageStatus(nil)
	if err != nil {
		t.Fatalf("nil map: %v", err)
	}
	if s != "{}" {
		t.Errorf("nil map should serialise to %q, got %q", "{}", s)
	}

	s, err = marshalStageStatus(map[string]string{"crawl": "ok", "grade": "fail"})
	if err != nil {
		t.Fatalf("non-empty map: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["crawl"] != "ok" || got["grade"] != "fail" {
		t.Errorf("round-trip mismatch: %v", got)
	}
}

func TestMarshalCandidateTags(t *testing.T) {
	s, err := marshalCandidateTags(nil)
	if err != nil {
		t.Fatalf("nil slice: %v", err)
	}
	if s != "[]" {
		t.Errorf("nil slice should serialise to %q, got %q", "[]", s)
	}

	s, err = marshalCandidateTags([]string{"go", "testing"})
	if err != nil {
		t.Fatalf("non-empty slice: %v", err)
	}
	var got []string
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 || got[0] != "go" || got[1] != "testing" {
		t.Errorf("round-trip mismatch: %v", got)
	}
}

// --- atoiOrDefault -------------------------------------------------------

func TestAtoiOrDefault(t *testing.T) {
	cases := []struct {
		in   string
		def  int
		want int
	}{
		{"", 50, 50},
		{"42", 50, 42},
		{"0", 50, 0}, // 0 is a valid value; should NOT fall back to default
		{"-3", 50, 50}, // negative digits rejected, fall back to default
		{"abc", 50, 50}, // non-digit rejected
		{"12abc", 50, 50},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := atoiOrDefault(c.in, c.def); got != c.want {
				t.Errorf("atoiOrDefault(%q,%d) = %d, want %d", c.in, c.def, got, c.want)
			}
		})
	}

	// Overflow contract: the function does not check for overflow — it
	// returns whatever wraps. Callers that need a bound must clamp after
	// the call. The M4 candidate handler clamps limit to [1,500], so an
	// absurd `?limit=99999999999999999999` should reach that clamp and
	// fall back to 50 — but NOT inside atoiOrDefault itself.
	t.Run("overflow returns non-default", func(t *testing.T) {
		got := atoiOrDefault("99999999999999999999", 50)
		if got == 50 {
			t.Errorf("overflow value should not silently equal default, got %d", got)
		}
	})
}