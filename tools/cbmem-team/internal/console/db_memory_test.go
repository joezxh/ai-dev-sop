package console

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cbmem-team/internal/auth"
)

// =============================================================================
// M5 (Memory + Template + Summarize/Distill tasks) — DB-layer tests.
// =============================================================================
//
// The v2 schema is seeded by setupTestDB (which calls newTestDB +
// MigrateV2). The migration also runs migrateV2Phase6, so the five
// built-in templates are pre-seeded before each test starts.
// =============================================================================

// findTemplateIDByName is a test helper that returns the int64 ID of
// a template by its unique name. It fails the test if not found.
func findTemplateIDByName(t *testing.T, db *DB, name string) int64 {
	t.Helper()
	tpl, err := db.GetMemoryTemplateByName(context.Background(), name)
	if err != nil {
		t.Fatalf("lookup template %q: %v", name, err)
	}
	return tpl.ID
}

func TestMemoryDBTemplatesSeeded(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	rows, err := db.ListMemoryTemplates(ctx, false)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("expected 5 built-in templates, got %d", len(rows))
	}
	want := map[string]bool{
		"Architecture Decision Record (ADR)": false,
		"Lessons Learned":                    false,
		"Reusable Code Snippet":              false,
		"Operational Runbook":                false,
		"Lightweight Decision":               false,
	}
	for _, r := range rows {
		want[r.Name] = true
		if !r.IsBuiltin {
			t.Errorf("template %s should be marked built-in", r.Name)
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing seeded template %q", name)
		}
	}
}

func TestMemoryDBCreateAndGetMemory(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	// Setup: user + team + project + leaf module.
	seedMem5Fixtures(t, db)
	adrID := findTemplateIDByName(t, db, "Architecture Decision Record (ADR)")
	tpl, err := db.GetMemoryTemplate(ctx, adrID)
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	if tpl.BodyTemplate == "" {
		t.Fatal("ADR body_template should not be empty")
	}
	mem := &Memory{
		ID:         "mem_alpha",
		TeamID:     "team_alpha",
		ProjectID:  "proj_alpha",
		ModuleID:   "mod_leaf_alpha",
		UserID:     "u_owner",
		Title:      "Use MemPalace for vector retrieval",
		Content:    "Replace Sql FTS5 with MemPalace for semantic search.",
		TemplateID: adrID,
		Tags:       []string{"retrieval", "mem-palace"},
		Hall:       "discoveries",
	}
	if err := db.CreateMemory(ctx, mem); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := db.GetMemory(ctx, "mem_alpha")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != mem.Title || got.Hall != "discoveries" {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "retrieval" || got.Tags[1] != "mem-palace" {
		t.Fatalf("tags roundtrip: %+v", got.Tags)
	}
	if got.TemplateID != adrID {
		t.Fatalf("template_id roundtrip: got %d, want %d", got.TemplateID, adrID)
	}
}

func TestMemoryDBRejectsNonLeafModule(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	mem := &Memory{
		ID:     "mem_bad",
		TeamID: "team_alpha", ProjectID: "proj_alpha",
		ModuleID: "mod_internal_alpha", // NOT a leaf
		UserID:   "u_owner",
		Title:    "Should fail", Content: "Should fail",
	}
	err := db.CreateMemory(ctx, mem)
	if err == nil {
		t.Fatal("expected non-leaf rejection, got nil")
	}
	if !strings.Contains(err.Error(), "leaf") {
		t.Fatalf("expected leaf error, got: %v", err)
	}
}

func TestMemoryDBRejectsBadHall(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	mem := &Memory{
		ID: "mem_hall", TeamID: "team_alpha", ProjectID: "proj_alpha",
		ModuleID: "mod_leaf_alpha", UserID: "u_owner",
		Title: "T", Content: "C", Hall: "unknown",
	}
	err := db.CreateMemory(ctx, mem)
	if !errors.Is(err, ErrInvalidHall) {
		t.Fatalf("want ErrInvalidHall, got %v", err)
	}
}

func TestMemoryDBListFilter(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	for _, hall := range []string{"facts", "discoveries", "advice"} {
		m := &Memory{
			ID: "mem_" + hall, TeamID: "team_alpha", ProjectID: "proj_alpha",
			ModuleID: "mod_leaf_alpha", UserID: "u_owner",
			Title: hall, Content: hall, Hall: hall,
			Tags: []string{"alpha", hall},
		}
		if err := db.CreateMemory(ctx, m); err != nil {
			t.Fatalf("seed %s: %v", hall, err)
		}
	}
	// Filter by hall.
	f := NewMemoryFilter()
	f.TeamID = "team_alpha"
	f.Hall = "discoveries"
	rows, total, err := db.ListMemories(ctx, f)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != "mem_discoveries" {
		t.Fatalf("expected one discoveries row, got total=%d len=%d", total, len(rows))
	}
	// Filter by tag substring.
	f = NewMemoryFilter()
	f.Tag = "advice"
	rows, _, err = db.ListMemories(ctx, f)
	if err != nil {
		t.Fatalf("list tag: %v", err)
	}
	if len(rows) != 1 || rows[0].Hall != "advice" {
		t.Fatalf("expected one advice row, got %+v", rows)
	}
}

func TestMemoryDBUpdatePartial(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	mem := &Memory{
		ID: "mem_up", TeamID: "team_alpha", ProjectID: "proj_alpha",
		ModuleID: "mod_leaf_alpha", UserID: "u_owner",
		Title: "Original", Content: "Original", Hall: "facts",
	}
	if err := db.CreateMemory(ctx, mem); err != nil {
		t.Fatal(err)
	}
	newTitle := "Updated"
	newHall := "events"
	if err := db.UpdateMemory(ctx, "mem_up", &newTitle, nil, &newHall, []string{"only-one"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := db.GetMemory(ctx, "mem_up")
	if got.Title != "Updated" || got.Hall != "events" {
		t.Fatalf("partial update wrong: %+v", got)
	}
	if got.Content != "Original" || len(got.Tags) != 1 || got.Tags[0] != "only-one" {
		t.Fatalf("non-nil fields overwritten: %+v", got)
	}
}

func TestMemoryDBSoftDelete(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	mem := &Memory{
		ID: "mem_del", TeamID: "team_alpha", ProjectID: "proj_alpha",
		ModuleID: "mod_leaf_alpha", UserID: "u_owner",
		Title: "T", Content: "C", Hall: "facts",
	}
	if err := db.CreateMemory(ctx, mem); err != nil {
		t.Fatal(err)
	}
	if err := db.SoftDeleteMemory(ctx, "mem_del"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetMemory(ctx, "mem_del"); err != ErrMemoryNotFound {
		t.Fatalf("want ErrMemoryNotFound, got %v", err)
	}
	// After soft-delete, list with default filter should hide it.
	f := NewMemoryFilter()
	f.TeamID = "team_alpha"
	rows, total, _ := db.ListMemories(ctx, f)
	if total != 0 || len(rows) != 0 {
		t.Fatalf("soft-deleted row leaked into list: total=%d", total)
	}
}

func TestMemoryDBTaskCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	// Summarize
	st := &SummarizeTask{
		ID:         "sum_test",
		UserID:     "u_owner",
		SourceIDs:  `["s1","s2"]`,
		Depth:      "shallow",
		TargetWing: "wing_default",
	}
	if err := db.CreateSummarizeTask(ctx, st); err != nil {
		t.Fatalf("create summarize: %v", err)
	}
	got, err := db.GetSummarizeTask(ctx, "sum_test")
	if err != nil {
		t.Fatalf("get summarize: %v", err)
	}
	if got.Depth != "shallow" || got.UserID != "u_owner" {
		t.Fatalf("roundtrip wrong: %+v", got)
	}
	rows, err := db.ListSummarizeTasks(ctx, "u_owner", 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("list summarize: err=%v len=%d", err, len(rows))
	}
	// Depth validation
	bad := &SummarizeTask{ID: "sum_bad", UserID: "u", SourceIDs: `[]`, Depth: "wrong"}
	if err := db.CreateSummarizeTask(ctx, bad); err == nil {
		t.Fatal("expected depth validation to fail")
	}
	// Distill
	dt := &DistillTask{
		ID:         "dis_test",
		UserID:     "u_owner",
		SourceIDs:  `["s1"]`,
		RulesJSON:  `{"min_value_score":0.8}`,
		TargetWing: "wing_default",
	}
	if err := db.CreateDistillTask(ctx, dt); err != nil {
		t.Fatalf("create distill: %v", err)
	}
	dg, err := db.GetDistillTask(ctx, "dis_test")
	if err != nil {
		t.Fatalf("get distill: %v", err)
	}
	if dg.MemPalaceSynced {
		t.Fatal("new distill task should not be mempalace_synced")
	}
}

// =============================================================================
// Fixtures — user + team + project + internal module + leaf module.
// Mirrors the path most M5 tests need.
// =============================================================================

func seedMem5Fixtures(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	hash, err := authHash("password123")
	if err != nil {
		t.Fatal(err)
	}
	u := &User{ID: "u_owner", Username: "owner", PasswordHash: hash, Role: "admin"}
	if err := db.CreateUser(ctx, u); err != nil {
		t.Fatalf("create user: %v", err)
	}
	tm := &Team{ID: "team_alpha", Name: "Alpha", Slug: "alpha", OwnerID: "u_owner"}
	if err := db.CreateTeam(ctx, tm); err != nil {
		t.Fatalf("create team: %v", err)
	}
	if err := db.AddTeamMember(ctx, "team_alpha", "u_owner", ProjectRoleOwner); err != nil {
		t.Fatalf("add owner: %v", err)
	}
	p := &Project{
		ID:      "proj_alpha",
		TeamID:  "team_alpha",
		Slug:    "alpha",
		Name:    "Alpha",
		Path:    "/tmp/cbmem-alpha",
		GitURL:  "",
		Status:  "ready",
		OwnerID: "u_owner",
	}
	if err := db.CreateProjectV2(ctx, p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	// Modules: one internal node, one leaf. CreateModule always
	// makes is_leaf=1 (and flips the parent to non-leaf when a
	// parent_id is supplied), so we use the parent→child path:
	// create A as a leaf, then create B with parent=A; A becomes
	// non-leaf and B remains a leaf.
	if err := db.CreateModule(ctx, &Module{
		ID: "mod_internal_alpha", ProjectID: "proj_alpha",
		Name: "Internal", IsLeaf: true,
	}); err != nil {
		t.Fatalf("create parent-as-leaf: %v", err)
	}
	if err := db.CreateModule(ctx, &Module{
		ID: "mod_leaf_alpha", ProjectID: "proj_alpha",
		ParentID: ptrString("mod_internal_alpha"),
		Name: "Leaf", IsLeaf: true,
	}); err != nil {
		t.Fatalf("create child leaf: %v", err)
	}
}

// authHash is a one-line wrapper so the test file doesn't import auth everywhere.
func authHash(pw string) (string, error) { return auth.Hash(pw) }