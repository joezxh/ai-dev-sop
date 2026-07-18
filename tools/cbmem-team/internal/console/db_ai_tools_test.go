package console

import (
	"context"
	"testing"
)

// =============================================================================
// M6 (AI Tools) — DB-layer tests.
// =============================================================================

func TestAIToolCreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)

	t1 := &AITool{
		ID:              "ait_cursor",
		TeamID:          "team_alpha",
		Name:            "Cursor CLI",
		Slug:            "cursor",
		Description:     "Cursor developer tool.",
		Endpoint:        "http://localhost:9222/api/ask",
		Protocol:        "http",
		RequiredRole:    "developer",
		TimeoutSeconds:  120,
		Enabled:         true,
	}
	if err := db.CreateAITool(ctx, t1); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := db.GetAIToolByID(ctx, "ait_cursor")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Cursor CLI" || got.Protocol != "http" {
		t.Fatalf("roundtrip wrong: %+v", got)
	}
	if !got.Enabled {
		t.Error("should be enabled")
	}
}

func TestAIToolRejectsBadProtocol(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	t1 := &AITool{
		ID: "ait_bad", TeamID: "team_alpha", Name: "X",
		Slug: "x", Protocol: "ftp",
	}
	if err := db.CreateAITool(ctx, t1); err == nil {
		t.Fatal("expected bad protocol rejection")
	}
}

func TestAIToolRejectsBadRole(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	t1 := &AITool{
		ID: "ait_bad", TeamID: "team_alpha", Name: "X",
		Slug: "x", Protocol: "http", RequiredRole: "viewer",
	}
	if err := db.CreateAITool(ctx, t1); err == nil {
		t.Fatal("expected bad required_role rejection")
	}
}

func TestAIToolListAndUpdate(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	for _, s := range []string{"cursor", "qoder", "superpowers"} {
		t1 := &AITool{
			ID: "ait_" + s, TeamID: "team_alpha", Name: s,
			Slug: s, Protocol: "http", TimeoutSeconds: 60,
		}
		if err := db.CreateAITool(ctx, t1); err != nil {
			t.Fatalf("create %s: %v", s, err)
		}
	}
	rows, err := db.ListAITools(ctx, "team_alpha")
	if err != nil || len(rows) != 3 {
		t.Fatalf("list: got %d rows, err=%v", len(rows), err)
	}
	newTimeout := 300
	if err := db.UpdateAITool(ctx, "ait_cursor", AIToolUpdate{TimeoutSeconds: &newTimeout}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := db.GetAIToolByID(ctx, "ait_cursor")
	if got.TimeoutSeconds != 300 {
		t.Fatalf("update not persisted: timeout=%d", got.TimeoutSeconds)
	}
}

func TestAIToolPartialUpdate(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	t1 := &AITool{ID: "ait_p", TeamID: "team_alpha", Name: "P",
		Slug: "p", Protocol: "http", TimeoutSeconds: 60, Enabled: true}
	if err := db.CreateAITool(ctx, t1); err != nil {
		t.Fatal(err)
	}
	newName := "P-renamed"
	disabled := false
	if err := db.UpdateAITool(ctx, "ait_p", AIToolUpdate{
		Name:    &newName,
		Enabled: &disabled,
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := db.GetAIToolByID(ctx, "ait_p")
	if got.Name != "P-renamed" || got.Enabled {
		t.Fatalf("partial update wrong: name=%s enabled=%v", got.Name, got.Enabled)
	}
}

func TestAIToolDeleteWithInvocationsBlocked(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	t1 := &AITool{ID: "ait_del", TeamID: "team_alpha", Name: "Del",
		Slug: "del", Protocol: "http"}
	if err := db.CreateAITool(ctx, t1); err != nil {
		t.Fatal(err)
	}
	// Insert an invocation referencing the tool.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO ai_tool_invocations
             (id, tool_id, user_id, team_id, project_id, module_id,
              working_dir, input_json, status, started_at)
          VALUES (?, ?, ?, ?, ?, ?, ?, '{}', 'success', ?)`,
		"inv_x", "ait_del", "u_owner", "team_alpha",
		"proj_alpha", "mod_leaf_alpha", "/tmp", "2026-01-01T00:00:00Z",
	); err != nil {
		t.Fatalf("seed invocation: %v", err)
	}
	// Delete should be refused.
	if err := db.DeleteAITool(ctx, "ait_del"); err == nil {
		t.Fatal("expected delete to be blocked with invocations")
	}
}

func TestInvocationCRUD(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)

	t1 := &AITool{ID: "ait_i", TeamID: "team_alpha", Name: "I",
		Slug: "i", Protocol: "stdio", Command: "echo"}
	if err := db.CreateAITool(ctx, t1); err != nil {
		t.Fatal(err)
	}
	inv := &AIToolInvocation{
		ID:         "inv_test",
		ToolID:     "ait_i",
		UserID:     "u_owner",
		TeamID:     "team_alpha",
		ProjectID:  "proj_alpha",
		ModuleID:   "mod_leaf_alpha",
		WorkingDir: "/tmp",
		Status:     "running",
	}
	if err := db.CreateInvocation(ctx, inv); err != nil {
		t.Fatalf("create invocation: %v", err)
	}
	got, err := db.GetInvocation(ctx, "inv_test")
	if err != nil || got.Status != "running" {
		t.Fatalf("get invocation: err=%v status=%s", err, got.Status)
	}
	// Update to success.
	if err := db.UpdateInvocationStatus(ctx, "inv_test", "success",
		`{"output":"ok"}`, ""); err != nil {
		t.Fatal(err)
	}
	got, _ = db.GetInvocation(ctx, "inv_test")
	if got.Status != "success" || got.FinishedAt == nil {
		t.Fatalf("update wrong: status=%s finished=%v", got.Status, got.FinishedAt)
	}
}

func TestInvocationListFilter(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)
	t1 := &AITool{ID: "ait_f", TeamID: "team_alpha", Name: "F", Slug: "f", Protocol: "http"}
	if err := db.CreateAITool(ctx, t1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		inv := &AIToolInvocation{
			ID: "inv_f_" + string(rune('a'+i)),
			ToolID: "ait_f", UserID: "u_owner", TeamID: "team_alpha",
			ProjectID: "proj_alpha", ModuleID: "mod_leaf_alpha",
			WorkingDir: "/tmp", Status: "success",
		}
		if err := db.CreateInvocation(ctx, inv); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	rows, err := db.ListInvocations(ctx, ListInvocationsFilter{ToolID: "ait_f", Limit: 3})
	if err != nil || len(rows) != 3 {
		t.Fatalf("list: got %d rows, err=%v", len(rows), err)
	}
}

func TestAIToolRBAC(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	seedMem5Fixtures(t, db)

	// Developer-only tool.
	t1 := &AITool{ID: "ait_dev", TeamID: "team_alpha", Name: "Dev Tool",
		Slug: "dev", Protocol: "http", RequiredRole: "developer"}
	if err := db.CreateAITool(ctx, t1); err != nil {
		t.Fatal(err)
	}
	// Lead can access developer tools.
	ok, err := db.AIToolAccessibleByUser(ctx, "ait_dev", "u_owner", "lead")
	if err != nil || !ok {
		t.Fatalf("lead should access dev tool: ok=%v err=%v", ok, err)
	}
	// Project allow-list: empty = all allowed.
	ok, err = db.AIToolProjectAllowed(ctx, "ait_dev", "proj_alpha")
	if err != nil || !ok {
		t.Fatalf("project should be allowed: ok=%v err=%v", ok, err)
	}
}
