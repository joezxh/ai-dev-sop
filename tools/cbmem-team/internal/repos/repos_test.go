package repos

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeName(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  bool
	}{
		{"https://github.com/foo/bar.git", "github.com__foo_bar", false},
		{"https://github.com/foo/bar", "github.com__foo_bar", false},
		{"git@github.com:foo/bar.git", "github.com__foo_bar", false},
		{"https://gitlab.example.com/group/sub/repo.git", "gitlab.example.com__group_sub_repo", false},
		{"ftp://example.com/repo", "example.com__repo", false},
		{"", "", true},
		{"/just/a/path", "", true},
	}
	for _, tc := range cases {
		got, err := SafeName(tc.in)
		if (err != nil) != tc.err {
			t.Errorf("SafeName(%q) err=%v want_err=%v", tc.in, err, tc.err)
			continue
		}
		if !tc.err && got != tc.want {
			t.Errorf("SafeName(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCloneFetchIdempotent(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skipf("git not available: %v", err)
	}
	dir := t.TempDir()

	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, git, src, "init", "-b", "main")
	mustGit(t, git, src, "config", "user.email", "test@example.com")
	mustGit(t, git, src, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, git, src, "add", "README.md")
	mustGit(t, git, src, "commit", "-m", "init")

	srcURL := "file://" + filepath.ToSlash(src)

	dataDir := filepath.Join(dir, "data")
	m, err := New(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	r1, err := m.Clone(context.Background(), "alice", srcURL)
	if err != nil {
		t.Fatalf("first clone: %v", err)
	}
	if r1.LastErr != "" {
		t.Fatalf("first clone side error: %s", r1.LastErr)
	}
	if _, err := os.Stat(filepath.Join(r1.LocalPath, ".git")); err != nil {
		t.Fatalf("expected .git dir: %v", err)
	}

	r2, err := m.Clone(context.Background(), "alice", srcURL)
	if err != nil {
		t.Fatalf("second clone (should fetch): %v", err)
	}
	if r2.LastPull.IsZero() {
		t.Errorf("expected LastPull to be set on second call (fetch path)")
	}
	if !strings.HasPrefix(r2.URL, "file://") {
		t.Errorf("URL not preserved: %q", r2.URL)
	}

	got := m.List("alice")
	if len(got) != 1 {
		t.Errorf("List(alice) = %d, want 1", len(got))
	}
}

func mustGit(t *testing.T, git, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(git, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", strings.Join(args, " "), err, out)
	}
}
