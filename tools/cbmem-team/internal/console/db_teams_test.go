package console

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"cbmem-team/internal/auth"
)

// =============================================================================
// DB layer tests for M3 (teams + projects v2).
//
// These tests exercise the storage layer directly. The HTTP layer is
// covered in teams_handler_test.go and projects_v2_handler_test.go.
// =============================================================================

// mustCreateUser inserts a user with a unique id and returns it. The
// password_hash is bcrypt of a long-enough secret so legacy-sentinel
// logic doesn't trip up the createUser test path.
func mustCreateUser(t *testing.T, db *DB, id, role string) *User {
	t.Helper()
	hash, err := auth.Hash("password123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := &User{
		ID:           id,
		Username:     "u_" + id,
		PasswordHash: hash,
		Role:         role,
	}
	if err := db.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("create user %s: %v", id, err)
	}
	return u
}

// setupTestDB returns a DB with BOTH v1 and v2 migrations applied.
// Most M3 tests need teams/projects v2 schema, so we call MigrateV2
// after Migrate so the v2 tables exist on a fresh :memory: handle.
func setupTestDB(t *testing.T) *DB {
	t.Helper()
	db := newTestDB(t)
	if err := db.MigrateV2(context.Background(), nil); err != nil {
		t.Fatalf("migrate v2: %v", err)
	}
	return db
}

func TestTeamsDBCreateGetList(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	tm := &Team{
		ID:      "team_alpha",
		Name:    "Alpha",
		Slug:    "alpha",
		OwnerID: "",
	}
	if err := db.CreateTeam(ctx, tm); err != nil {
		t.Fatalf("create team: %v", err)
	}

	got, err := db.GetTeamByID(ctx, "team_alpha")
	if err != nil {
		t.Fatalf("get team: %v", err)
	}
	if got.Name != "Alpha" || got.OwnerID != "" {
		t.Fatalf("team=%+v", got)
	}

	got2, err := db.GetTeamBySlug(ctx, "alpha")
	if err != nil || got2.ID != "team_alpha" {
		t.Fatalf("get by slug: id=%s err=%v", got2.ID, err)
	}

	list, err := db.ListTeams(ctx)
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}
	found := false
	for _, t := range list {
		if t.ID == "team_alpha" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("team_alpha not in list (n=%d)", len(list))
	}
}

func TestTeamsDBSlugUnique(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	if err := db.CreateTeam(ctx, &Team{ID: "team_a", Name: "A", Slug: "shared"}); err != nil {
		t.Fatalf("first: %v", err)
	}
	err := db.CreateTeam(ctx, &Team{ID: "team_b", Name: "B", Slug: "shared"})
	if !errors.Is(err, ErrTeamExists) {
		t.Fatalf("expected ErrTeamExists, got %v", err)
	}
}

func TestTeamsDBUpdateDelete(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	if err := db.CreateTeam(ctx, &Team{ID: "team_x", Name: "X", Slug: "x", OwnerID: ""}); err != nil {
		t.Fatal(err)
	}
	desc := "freshly described"
	if err := db.UpdateTeam(ctx, "team_x", nil, &desc); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := db.GetTeamByID(ctx, "team_x")
	if got.Description != desc {
		t.Fatalf("desc=%q", got.Description)
	}
	// Update with nil name + description leaves name untouched.
	if err := db.UpdateTeam(ctx, "team_x", nil, nil); err != nil {
		t.Fatalf("noop update: %v", err)
	}
	got, _ = db.GetTeamByID(ctx, "team_x")
	if got.Name != "X" || got.Description != desc {
		t.Fatalf("post-noop name=%q desc=%q", got.Name, got.Description)
	}

	if err := db.DeleteTeam(ctx, "team_x"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := db.GetTeamByID(ctx, "team_x"); !errors.Is(err, ErrTeamNotFound) {
		t.Fatalf("expected ErrTeamNotFound after delete, got %v", err)
	}
}

func TestTeamsDBMembers(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	mustCreateUser(t, db, "u_alice", string(RoleDeveloper))
	mustCreateUser(t, db, "u_bob", string(RoleLead))
	if err := db.CreateTeam(ctx, &Team{ID: "team_alpha", Name: "Alpha", Slug: "alpha", OwnerID: "u_alice"}); err != nil {
		t.Fatal(err)
	}

	if err := db.AddTeamMember(ctx, "team_alpha", "u_bob", ProjectRoleDeveloper); err != nil {
		t.Fatalf("add: %v", err)
	}
	// duplicate add
	if err := db.AddTeamMember(ctx, "team_alpha", "u_bob", ProjectRoleDeveloper); !errors.Is(err, ErrMemberExists) {
		t.Fatalf("expected ErrMemberExists, got %v", err)
	}
	if err := db.UpdateTeamMemberRole(ctx, "team_alpha", "u_bob", ProjectRoleMaintainer); err != nil {
		t.Fatalf("update role: %v", err)
	}
	if err := db.RemoveTeamMember(ctx, "team_alpha", "u_bob"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := db.RemoveTeamMember(ctx, "team_alpha", "u_bob"); !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("expected ErrMemberNotFound, got %v", err)
	}
}

func TestTeamsDBIsTeamMember(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	mustCreateUser(t, db, "u1", string(RoleDeveloper))
	mustCreateUser(t, db, "u2", string(RoleDeveloper))
	if err := db.CreateTeam(ctx, &Team{ID: "team_t", Name: "T", Slug: "t", OwnerID: "u1"}); err != nil {
		t.Fatal(err)
	}
	isMember, err := db.IsTeamMember(ctx, "team_t", "u1")
	if err != nil || !isMember {
		t.Fatalf("owner should be implicit member: is=%v err=%v", isMember, err)
	}
	isMember, _ = db.IsTeamMember(ctx, "team_t", "u2")
	if isMember {
		t.Fatalf("u2 should not be a member yet")
	}
	_ = db.AddTeamMember(ctx, "team_t", "u2", ProjectRoleDeveloper)
	isMember, _ = db.IsTeamMember(ctx, "team_t", "u2")
	if !isMember {
		t.Fatalf("u2 should be a member after AddTeamMember")
	}
}

func TestTeamsDBListForUser(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	mustCreateUser(t, db, "u_admin", string(RoleAdmin))
	mustCreateUser(t, db, "u_dev", string(RoleDeveloper))

	// Two teams: one owned by dev, one they're just a member of.
	if err := db.CreateTeam(ctx, &Team{ID: "team_own", Name: "Own", Slug: "own", OwnerID: "u_dev"}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateTeam(ctx, &Team{ID: "team_member", Name: "Member", Slug: "member", OwnerID: "u_admin"}); err != nil {
		t.Fatal(err)
	}
	if err := db.AddTeamMember(ctx, "team_member", "u_dev", ProjectRoleDeveloper); err != nil {
		t.Fatal(err)
	}
	// A third team they have nothing to do with.
	if err := db.CreateTeam(ctx, &Team{ID: "team_other", Name: "Other", Slug: "other", OwnerID: "u_admin"}); err != nil {
		t.Fatal(err)
	}

	got, err := db.ListTeamsForUser(ctx, "u_dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("u_dev should see exactly 2 teams, got %d (%+v)", len(got), got)
	}
	seen := map[string]bool{}
	for _, tm := range got {
		seen[tm.ID] = true
	}
	if !seen["team_own"] || !seen["team_member"] {
		t.Fatalf("missing expected team: %+v", seen)
	}
}

func TestProjectsV2DBCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	mustCreateUser(t, db, "u_owner", string(RoleAdmin))
	if err := db.CreateTeam(ctx, &Team{ID: "team_t", Name: "T", Slug: "t", OwnerID: "u_owner"}); err != nil {
		t.Fatal(err)
	}
	p := &Project{
		ID:      "proj_one",
		TeamID:  "team_t",
		Name:    "One",
		Slug:    "one",
		Path:    "/repo/one",
		Status:  ProjectStatusReady,
	}
	if err := db.CreateProjectV2(ctx, p); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := db.GetProjectV2ByID(ctx, "proj_one")
	if err != nil || got.Name != "One" || got.Path != "/repo/one" {
		t.Fatalf("get: %+v err=%v", got, err)
	}

	// Path uniqueness across teams
	if err := db.CreateProjectV2(ctx, &Project{
		ID: "proj_two", TeamID: "team_t", Name: "Two", Slug: "two", Path: "/repo/one", Status: ProjectStatusReady,
	}); !errors.Is(err, ErrProjectExists) {
		t.Fatalf("expected ErrProjectExists (path collision), got %v", err)
	}

	// (team_id, slug) uniqueness within the team
	if err := db.CreateProjectV2(ctx, &Project{
		ID: "proj_three", TeamID: "team_t", Name: "Three", Slug: "one", Path: "/repo/three", Status: ProjectStatusReady,
	}); !errors.Is(err, ErrProjectExists) {
		t.Fatalf("expected ErrProjectExists (slug collision), got %v", err)
	}

	list, err := db.ListProjectsV2(ctx, "team_t")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: n=%d err=%v", len(list), err)
	}

	if err := db.SetProjectStatus(ctx, "proj_one", ProjectStatusIndexing); err != nil {
		t.Fatalf("set status: %v", err)
	}
	got, _ = db.GetProjectV2ByID(ctx, "proj_one")
	if got.Status != ProjectStatusIndexing {
		t.Fatalf("status=%q", got.Status)
	}

	// Update: changing name only.
	newName := "One Updated"
	if err := db.UpdateProjectV2(ctx, "proj_one", &newName, nil, nil); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = db.GetProjectV2ByID(ctx, "proj_one")
	if got.Name != newName || got.Slug != "one" {
		t.Fatalf("post-update name=%q slug=%q", got.Name, got.Slug)
	}

	if err := db.DeleteProjectV2(ctx, "proj_one"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := db.GetProjectV2ByID(ctx, "proj_one"); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound after delete, got %v", err)
	}
}

func TestProjectsV2AccessibleByUser(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	mustCreateUser(t, db, "u_admin", string(RoleAdmin))
	mustCreateUser(t, db, "u_in", string(RoleDeveloper))
	mustCreateUser(t, db, "u_out", string(RoleDeveloper))
	if err := db.CreateTeam(ctx, &Team{ID: "team_t", Name: "T", Slug: "t", OwnerID: "u_admin"}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateProjectV2(ctx, &Project{
		ID: "proj_a", TeamID: "team_t", Name: "A", Slug: "a",
		Path: "/repo/a", Status: ProjectStatusReady,
	}); err != nil {
		t.Fatal(err)
	}
	_ = db.AddTeamMember(ctx, "team_t", "u_in", ProjectRoleDeveloper)

	cases := []struct {
		userID, role string
		want         bool
	}{
		{"u_admin", string(RoleAdmin), true},
		{"u_in", string(RoleDeveloper), true},
		{"u_out", string(RoleDeveloper), false},
	}
	for _, c := range cases {
		got, err := db.ProjectAccessibleByUser(ctx, "proj_a", c.userID, c.role)
		if err != nil || got != c.want {
			t.Fatalf("user=%s role=%s got=%v want=%v err=%v", c.userID, c.role, got, c.want, err)
		}
	}
}

func TestComputeProjectPath(t *testing.T) {
	root := t.TempDir()
	p, err := ComputeProjectPath(root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p, filepath.Join(root, "demo")) {
		t.Fatalf("path=%q", p)
	}
	if _, err := ComputeProjectPath("", "demo"); err == nil {
		t.Fatal("expected error on empty root")
	}
	if _, err := ComputeProjectPath(root, ""); err == nil {
		t.Fatal("expected error on empty slug")
	}
}

func TestSanitizeSlug(t *testing.T) {
	cases := []struct{ in, out string }{
		{"hello", "hello"},
		{"hello world", "hello_world"},
		{"a/b\\c", "a_b_c"},
		{"foo:bar*?", "foo_bar__"},
	}
	for _, c := range cases {
		if got := SanitizeSlug(c.in); got != c.out {
			t.Fatalf("SanitizeSlug(%q)=%q want %q", c.in, got, c.out)
		}
	}
}

func TestValidateGitURL(t *testing.T) {
	if err := ValidateGitURL(""); err != nil {
		t.Fatalf("empty should be allowed: %v", err)
	}
	if err := ValidateGitURL("https://github.com/x/y.git"); err != nil {
		t.Fatalf("https should pass: %v", err)
	}
	if err := ValidateGitURL("git@github.com:x/y.git"); err != nil {
		t.Fatalf("ssh should pass: %v", err)
	}
	if err := ValidateGitURL("ftp://example.com/x"); err == nil {
		t.Fatal("ftp should be rejected")
	}
}

func TestValidProjectRole(t *testing.T) {
	if !validProjectRole(ProjectRoleOwner) {
		t.Fatal("owner should be valid")
	}
	if validProjectRole(ProjectRole("godmode")) {
		t.Fatal("godmode should be invalid")
	}
}