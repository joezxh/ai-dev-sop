package console

import (
	"os/exec"
)

// lookGit returns the empty string + a non-nil error if the `git`
// binary is not on $PATH. Used by async-clone tests so we can skip
// gracefully on machines without git (CI without git installed).
func lookGit() (string, error) {
	return exec.LookPath("git")
}

// newCmd builds an exec.Cmd rooted at `dir`. Saves a bit of
// boilerplate in tests that shell out to git.
func newCmd(dir, name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.Dir = dir
	return c
}
