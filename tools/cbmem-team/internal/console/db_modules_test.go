package console

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// =============================================================================
// Unit tests for db_modules.go (M4 storage layer).
//
// These don't spin up the HTTP router — they exercise the storage
// helpers directly so regressions in the SQL/leaf logic surface
// fast.
// =============================================================================

// newModulesTestDB returns a v2 schema DB ready for modules tests.
// Calls setupTestDB (which already runs v1 + v2 migrations).
func newModulesTestDB(t *testing.T) *DB {
	t.Helper()
	return setupTestDB(t)
}

// seedProjectV2 creates a team + project so modules have something
// to attach to. Returns project id.
func seedProjectV2(t *testing.T, db *DB, slug string) (teamID, projectID string) {
	t.Helper()
	ctx := context.Background()
	tm := &Team{ID: "team_" + slug, Name: "Team " + slug, Slug: slug, OwnerID: ""}
	if err := db.CreateTeam(ctx, tm); err != nil {
		t.Fatalf("create team: %v", err)
	}
	p := &Project{
		ID:     "proj_" + slug,
		TeamID: tm.ID,
		Slug:   slug,
		Name:   "Project " + slug,
		Path:   "/tmp/" + slug, // path is NOT NULL UNIQUE
	}
	if err := db.CreateProjectV2(ctx, p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return tm.ID, p.ID
}

func TestModulesDBCreateLeafAndInternal(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "alpha")

	// First module is a leaf by default.
	root := &Module{
		ID:        "mod_root",
		ProjectID: projectID,
		Name:      "root",
	}
	if err := db.CreateModule(ctx, root); err != nil {
		t.Fatalf("create root: %v", err)
	}
	if !root.IsLeaf {
		t.Fatalf("root should be leaf, got is_leaf=false")
	}

	// Adding a child should flip the root to internal.
	child := &Module{
		ID:        "mod_child",
		ProjectID: projectID,
		ParentID:  &root.ID,
		Name:      "child",
	}
	if err := db.CreateModule(ctx, child); err != nil {
		t.Fatalf("create child: %v", err)
	}
	got, err := db.GetModuleByID(ctx, root.ID)
	if err != nil {
		t.Fatalf("get root: %v", err)
	}
	if got.IsLeaf {
		t.Fatalf("root should NOT be leaf after child added")
	}
	if got2, err := db.GetModuleByID(ctx, child.ID); err != nil || !got2.IsLeaf {
		t.Fatalf("child should be leaf: leaf=%v err=%v", got2.IsLeaf, err)
	}
}

func TestModulesDBUniqueNameUnderParent(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "dup")

	// Root module.
	root := &Module{ID: "mod_r", ProjectID: projectID, Name: "r"}
	if err := db.CreateModule(ctx, root); err != nil {
		t.Fatalf("create root: %v", err)
	}
	// First child with name "x".
	a := &Module{ID: "mod_a", ProjectID: projectID, ParentID: &root.ID, Name: "x"}
	if err := db.CreateModule(ctx, a); err != nil {
		t.Fatalf("first create: %v", err)
	}
	// Second child under the same parent with the same name → must
	// collide. SQLite treats NULLs as distinct in UNIQUE indexes,
	// so we deliberately use a non-NULL parent_id here to make the
	// composite UNIQUE(project_id, parent_id, name) fire.
	dup := &Module{ID: "mod_b", ProjectID: projectID, ParentID: &root.ID, Name: "x"}
	if err := db.CreateModule(ctx, dup); !errors.Is(err, ErrModuleExists) {
		t.Fatalf("expected ErrModuleExists, got %v", err)
	}
}

func TestModulesDBUniqueSameNameDifferentParent(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "two")

	a := &Module{ID: "mod_a", ProjectID: projectID, Name: "x"}
	if err := db.CreateModule(ctx, a); err != nil {
		t.Fatalf("create a: %v", err)
	}
	b := &Module{ID: "mod_b", ProjectID: projectID, Name: "x", ParentID: &a.ID}
	if err := db.CreateModule(ctx, b); err != nil {
		t.Fatalf("create b under a: %v", err)
	}
}

func TestModulesDBTree(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "tree")

	// Root1 + Root2 at the top level.
	root1 := &Module{ID: "r1", ProjectID: projectID, Name: "root1"}
	root2 := &Module{ID: "r2", ProjectID: projectID, Name: "root2"}
	for _, m := range []*Module{root1, root2} {
		if err := db.CreateModule(ctx, m); err != nil {
			t.Fatalf("seed %s: %v", m.ID, err)
		}
	}
	// r1 → a → leaf, r1 → b → leaf, r2 → c → leaf.
	a := &Module{ID: "a", ProjectID: projectID, ParentID: &root1.ID, Name: "a"}
	b := &Module{ID: "b", ProjectID: projectID, ParentID: &root1.ID, Name: "b"}
	c := &Module{ID: "c", ProjectID: projectID, ParentID: &root2.ID, Name: "c"}
	for _, m := range []*Module{a, b, c} {
		if err := db.CreateModule(ctx, m); err != nil {
			t.Fatalf("seed %s: %v", m.ID, err)
		}
	}
	flat, err := db.ListModulesByProject(ctx, projectID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	tree := BuildModuleTree(flat)
	if len(tree) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree))
	}
	// Walk to confirm shape. Since we don't know which is first,
	// find by id.
	var byID func([]*ModuleNode, string) *ModuleNode
	byID = func(ns []*ModuleNode, id string) *ModuleNode {
		for _, n := range ns {
			if n.ID == id {
				return n
			}
			if found := byID(n.Children, id); found != nil {
				return found
			}
		}
		return nil
	}
	r1 := byID(tree, "r1")
	if r1 == nil || len(r1.Children) != 2 {
		t.Fatalf("r1 should have 2 children: %+v", r1)
	}
	// Leaves should not be nested.
	for _, child := range r1.Children {
		if len(child.Children) != 0 {
			t.Fatalf("grandchild should not exist: %+v", child)
		}
	}
}

func TestModulesDBMovePreventsCycle(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "cyc")

	// Build a -> b -> c (a has child b, b has child c).
	a := &Module{ID: "a", ProjectID: projectID, Name: "a"}
	if err := db.CreateModule(ctx, a); err != nil {
		t.Fatalf("a: %v", err)
	}
	b := &Module{ID: "b", ProjectID: projectID, ParentID: &a.ID, Name: "b"}
	if err := db.CreateModule(ctx, b); err != nil {
		t.Fatalf("b: %v", err)
	}
	c := &Module{ID: "c", ProjectID: projectID, ParentID: &b.ID, Name: "c"}
	if err := db.CreateModule(ctx, c); err != nil {
		t.Fatalf("c: %v", err)
	}
	// Trying to move `a` under `b` (a direct descendant) — must cycle.
	if err := db.MoveModule(ctx, "a", &b.ID); !errors.Is(err, ErrModuleCycle) {
		t.Fatalf("expected ErrModuleCycle (move under child), got %v", err)
	}
	// Same for `a` under `c` (transitive descendant).
	if err := db.MoveModule(ctx, "a", &c.ID); !errors.Is(err, ErrModuleCycle) {
		t.Fatalf("expected ErrModuleCycle (move under transitive child), got %v", err)
	}
	// Moving `c` to root should be fine.
	if err := db.MoveModule(ctx, "c", nil); err != nil {
		t.Fatalf("promote: %v", err)
	}
	// Now `b` has no children, so it should be a leaf again.
	bGot, _ := db.GetModuleByID(ctx, "b")
	if !bGot.IsLeaf {
		t.Fatalf("b should be leaf after c moved out: leaf=%v", bGot.IsLeaf)
	}
}

func TestModulesDBDeleteRefusesChildrenWithoutCascade(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "del")

	root := &Module{ID: "r", ProjectID: projectID, Name: "r"}
	if err := db.CreateModule(ctx, root); err != nil {
		t.Fatalf("root: %v", err)
	}
	child := &Module{ID: "c", ProjectID: projectID, ParentID: &root.ID, Name: "c"}
	if err := db.CreateModule(ctx, child); err != nil {
		t.Fatalf("child: %v", err)
	}
	// Without cascade: should refuse.
	if err := db.DeleteModule(ctx, root.ID, false); !errors.Is(err, ErrModuleHasChildren) {
		t.Fatalf("expected ErrModuleHasChildren, got %v", err)
	}
	// With cascade: should succeed and mark both deleted.
	if err := db.DeleteModule(ctx, root.ID, true); err != nil {
		t.Fatalf("cascade delete: %v", err)
	}
	if _, err := db.GetModuleByID(ctx, root.ID); !errors.Is(err, ErrModuleNotFound) {
		t.Fatalf("root should be gone, got err=%v", err)
	}
	if _, err := db.GetModuleByID(ctx, child.ID); !errors.Is(err, ErrModuleNotFound) {
		t.Fatalf("child should be gone, got err=%v", err)
	}
}

func TestModulesDBRequireLeaf(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "leaf")
	root := &Module{ID: "r", ProjectID: projectID, Name: "r"}
	if err := db.CreateModule(ctx, root); err != nil {
		t.Fatalf("root: %v", err)
	}
	child := &Module{ID: "c", ProjectID: projectID, ParentID: &root.ID, Name: "c"}
	if err := db.CreateModule(ctx, child); err != nil {
		t.Fatalf("child: %v", err)
	}
	if err := db.RequireLeafModule(ctx, "c"); err != nil {
		t.Fatalf("c should be leaf: %v", err)
	}
	if err := db.RequireLeafModule(ctx, "r"); !errors.Is(err, ErrModuleNotLeaf) {
		t.Fatalf("expected ErrModuleNotLeaf, got %v", err)
	}
	if err := db.RequireLeafModule(ctx, "missing"); !errors.Is(err, ErrModuleNotFound) {
		t.Fatalf("expected ErrModuleNotFound, got %v", err)
	}
}

func TestModulesDBLookupDefaultLeafAutoSeeds(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "seed")

	id, err := db.LookupDefaultLeafModule(ctx, projectID)
	if err != nil {
		t.Fatalf("lookup default: %v", err)
	}
	if id == "" {
		t.Fatalf("expected seeded leaf id, got empty")
	}
	// Second call should return the same id.
	id2, _ := db.LookupDefaultLeafModule(ctx, projectID)
	if id != id2 {
		t.Fatalf("expected stable leaf id, got %s vs %s", id, id2)
	}
}

func TestModulesDBUpdatePartial(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "upd")
	m := &Module{ID: "m", ProjectID: projectID, Name: "m", Description: "old"}
	if err := db.CreateModule(ctx, m); err != nil {
		t.Fatalf("create: %v", err)
	}
	newName := "renamed"
	newDesc := "new"
	if err := db.UpdateModule(ctx, "m", &newName, nil, &newDesc, nil); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := db.GetModuleByID(ctx, "m")
	if got.Name != "renamed" {
		t.Fatalf("name not updated: %s", got.Name)
	}
	if got.Description != "new" {
		t.Fatalf("desc not updated: %s", got.Description)
	}
}

func TestModulesDBCountDescendantsCapped(t *testing.T) {
	db := newModulesTestDB(t)
	ctx := context.Background()
	_, projectID := seedProjectV2(t, db, "cnt")

	prev := ""
	chainIDs := []string{}
	// Build a long chain: max + 1 above MaxModuleDepth so we can
	// confirm CountDescendants walks breadth-first and caps at the
	// limit rather than blowing the stack.
	const chainLen = MaxModuleDepth + 2
	for i := 0; i < chainLen; i++ {
		id := "m" + strings.Repeat("x", i+1)
		m := &Module{
			ID:        id,
			ProjectID: projectID,
			Name:      "n" + strings.Repeat("x", i+1),
		}
		if prev != "" {
			m.ParentID = &prev
		}
		if err := db.CreateModule(ctx, m); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
		prev = id
		chainIDs = append(chainIDs, id)
	}
	// Count from the first node (root of the chain). Total
	// descendants = chainLen - 1, but the cap stops the walk at
	// MaxModuleDepth hops so the result must be ≤ chainLen - 1.
	n, err := db.CountDescendants(ctx, chainIDs[0])
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected ≥1 descendant, got %d", n)
	}
	if n >= chainLen {
		t.Fatalf("expected capped count, got %d (no cap applied)", n)
	}
}