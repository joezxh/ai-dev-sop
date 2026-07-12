package devconf

import (
	"os"
	"strings"
	"testing"
)

// TestResolvePrecedence verifies the documented resolution order:
// flag > CBMEM_MYSQL_DSN > DSN > DefaultDevMySQLDSN.
func TestResolvePrecedence(t *testing.T) {
	const flagVal = "user:flag@tcp(host:3306)/flagdb?x=1"
	const envVal = "user:env@tcp(host:3306)/envdb?x=1"
	const legacyVal = "user:legacy@tcp(host:3306)/legacydb?x=1"

	cases := []struct {
		name string
		env  map[string]string // env keys to set (cleaned in t.Cleanup)
		flag string
		want string
	}{
		{
			name: "flag wins",
			flag: flagVal,
			env:  map[string]string{EnvMySQLDSN: envVal, EnvLegacyMySQLDSN: legacyVal},
			want: flagVal,
		},
		{
			name: "CBMEM_MYSQL_DSN second",
			env:  map[string]string{EnvMySQLDSN: envVal, EnvLegacyMySQLDSN: legacyVal},
			want: envVal,
		},
		{
			name: "DSN legacy third",
			env:  map[string]string{EnvLegacyMySQLDSN: legacyVal},
			want: legacyVal,
		},
		{
			name: "default last",
			want: DefaultDevMySQLDSN,
		},
		{
			name: "blank falls back through",
			env:  map[string]string{EnvMySQLDSN: "  "}, // whitespace only
			want: DefaultDevMySQLDSN,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				old, had := os.LookupEnv(k)
				os.Setenv(k, v)
				t.Cleanup(func() {
					if had {
						os.Setenv(k, old)
					} else {
						os.Unsetenv(k)
					}
				})
			}
			got := ResolveMySQLDSN(tc.flag)
			if got != tc.want {
				t.Fatalf("ResolveMySQLDSN(%q)\n  got:  %q\n  want: %q", tc.flag, got, tc.want)
			}
		})
	}
}

// TestResolveWithDBNoName covers the "DSN has no /db" branch — we must
// inject dbName at the right position, before any "?" query string.
func TestResolveWithDBNoName(t *testing.T) {
	in := "root:secret@tcp(host:3306)/?parseTime=true"
	got := ResolveMySQLDSNWithDB(in, "cbmem")
	want := "root:secret@tcp(host:3306)/cbmem?parseTime=true"
	if got != want {
		t.Fatalf("inject before ?:\n got:  %s\n want: %s", got, want)
	}
}

// TestResolveWithDBHasName covers the "DSN already names a DB" branch:
// we MUST NOT clobber the existing name. (This is the bug we want to
// prevent; otherwise create-db's second connection ended up pointing
// at the wrong schema.)
func TestResolveWithDBHasName(t *testing.T) {
	in := "root:secret@tcp(host:3306)/existing?parseTime=true"
	got := ResolveMySQLDSNWithDB(in, "cbmem")
	if got != in {
		t.Fatalf("must preserve existing db name:\n got:  %s\n want: %s", got, in)
	}
}

// TestResolveWithDBNoQuery covers DSNs that have no `?opts` at all.
// Whichever form we emit (`/cbmem` or `/cbmem/`) is fine for the Go MySQL
// driver; pin to one so the regression catches any future change.
func TestResolveWithDBNoQuery(t *testing.T) {
	in := "root:secret@tcp(host:3306)/"
	got := ResolveMySQLDSNWithDB(in, "cbmem")
	want := "root:secret@tcp(host:3306)/cbmem"
	if got != want {
		t.Fatalf("append /dbName:\n got:  %s\n want: %s", got, want)
	}
}

// TestResolveWithDBEmpty ensures we hand back something usable when the
// caller forgets to pass dbName.
func TestResolveWithDBEmpty(t *testing.T) {
	got := ResolveMySQLDSNWithDB(DefaultDevMySQLDSN, "")
	if !strings.HasPrefix(got, DefaultDevMySQLDSN) {
		t.Fatalf("must return at least the unresolved DSN, got %q", got)
	}
}