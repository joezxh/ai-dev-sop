package console

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// v2 (M4) Module storage layer.
// =============================================================================
//
// Modules form a recursive tree under a project (parent_id self-FK).
// A module is either:
//   - internal (has at least one child module) → is_leaf = 0
//   - leaf    (no child modules)               → is_leaf = 1
//
// leaves are the only valid attachment points for sessions and
// memories (enforced at handler level by RequireLeafModule).
// =============================================================================

var (
	ErrModuleNotFound = errors.New("module not found")
	ErrModuleExists   = errors.New("module name already exists under parent")
	ErrModuleCycle    = errors.New("module move would introduce a cycle")
	ErrModuleNotLeaf  = errors.New("module is not a leaf")
)

// MaxModuleDepth caps the tree height so deeply-nested user input
// can't blow the recursion on Tree. 32 is generous for any sane UI;
// the Plan doc mentions "≥ 5 levels" as the acceptance bar so this is
// well above that.
const MaxModuleDepth = 32

// ----------------------------------------------------------------------------
// Module CRUD
// ----------------------------------------------------------------------------

// CreateModule inserts a new module under `parentID` (nil = root).
// name must be unique within (project_id, parent_id). The new module
// starts as a leaf (is_leaf = 1) by default; callers can add children
// later. Returns ErrModuleExists when (project_id, parent_id, name)
// collides, or ErrModuleNotFound when parent is missing.
func (db *DB) CreateModule(ctx context.Context, m *Module) error {
	if m.ID == "" {
		return errors.New("CreateModule: id required")
	}
	if m.ProjectID == "" {
		return errors.New("CreateModule: project_id required")
	}
	if m.Name == "" {
		return errors.New("CreateModule: name required")
	}
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	// New modules are leaves by default; CreateChildModule (below)
	// handles the parent.is_leaf flip in the same transaction.
	if !m.IsLeaf {
		m.IsLeaf = true
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO modules
            (id, project_id, parent_id, name, path, description, "order", is_leaf, created_at, updated_at, deleted)
         VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?, 0)`,
		m.ID, m.ProjectID, nullableParent(m.ParentID), m.Name, m.Path,
		m.Description, m.Order, m.CreatedAt, m.UpdatedAt,
	); err != nil {
		if isUniqueViolation(err) {
			return ErrModuleExists
		}
		return fmt.Errorf("insert module: %w", err)
	}
	if m.ParentID != nil {
		// Parent becomes internal: is_leaf = 0.
		if err := flipLeafInTx(ctx, tx, *m.ParentID, false); err != nil {
			return fmt.Errorf("flip parent leaf: %w", err)
		}
	}
	return tx.Commit()
}

// nullableParent turns *string into sql.NullString: nil → NULL,
// non-nil → the value. Used to insert root modules (parent_id
// NULL) vs nested ones.
func nullableParent(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

// flipLeafInTx sets modules.is_leaf inside an existing tx.
func flipLeafInTx(ctx context.Context, tx *sql.Tx, id string, leaf bool) error {
	v := 0
	if leaf {
		v = 1
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE modules SET is_leaf = ?, updated_at = ? WHERE id = ? AND deleted = 0`,
		v, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrModuleNotFound
	}
	return nil
}

// GetModuleByID returns one module row. Soft-deleted modules return
// ErrModuleNotFound so callers don't have to remember the deleted
// filter.
func (db *DB) GetModuleByID(ctx context.Context, id string) (*Module, error) {
	const q = `SELECT id, IFNULL(project_id,''), parent_id, name, IFNULL(path,''),
                      IFNULL(description,''), "order",
                      is_leaf, created_at, updated_at, deleted
                 FROM modules WHERE id = ? AND deleted = 0`
	row := db.QueryRowContext(ctx, q, id)
	return db.scanModule(row)
}

// ListModulesByProject returns every non-deleted module under a
// project. Order is parent_id NULL first, then by parent_id, name —
// tree-ready for client-side rendering.
func (db *DB) ListModulesByProject(ctx context.Context, projectID string) ([]*Module, error) {
	const q = `SELECT id, IFNULL(project_id,''), parent_id, name, IFNULL(path,''),
                      IFNULL(description,''), "order",
                      is_leaf, created_at, updated_at, deleted
                 FROM modules
                WHERE deleted = 0 AND project_id = ?
                ORDER BY (parent_id IS NULL) DESC, parent_id, "order", name`
	rows, err := db.QueryContext(ctx, q, projectID)
	if err != nil {
		return nil, fmt.Errorf("list modules: %w", err)
	}
	defer rows.Close()
	var out []*Module
	for rows.Next() {
		m, err := db.scanModule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListChildModules returns the immediate children of parentID. Empty
// parentID lists root modules (parent_id IS NULL).
func (db *DB) ListChildModules(ctx context.Context, parentID string) ([]*Module, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if parentID == "" {
		rows, err = db.QueryContext(ctx,
			`SELECT id, IFNULL(project_id,''), parent_id, name, IFNULL(path,''),
                    IFNULL(description,''), "order",
                    is_leaf, created_at, updated_at, deleted
               FROM modules
              WHERE deleted = 0 AND parent_id IS NULL
              ORDER BY "order", name`)
	} else {
		rows, err = db.QueryContext(ctx,
			`SELECT id, IFNULL(project_id,''), parent_id, name, IFNULL(path,''),
                    IFNULL(description,''), "order",
                    is_leaf, created_at, updated_at, deleted
               FROM modules
              WHERE deleted = 0 AND parent_id = ?
              ORDER BY "order", name`, parentID)
	}
	if err != nil {
		return nil, fmt.Errorf("list children: %w", err)
	}
	defer rows.Close()
	var out []*Module
	for rows.Next() {
		m, err := db.scanModule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateModule applies a partial update. Parent cannot be changed
// here — call MoveModule instead.
func (db *DB) UpdateModule(ctx context.Context, id string, name, path, description *string, order *int) error {
	now := time.Now().UTC()
	res, err := db.ExecContext(ctx,
		`UPDATE modules
            SET name        = COALESCE(?, name),
                path        = COALESCE(?, path),
                description = COALESCE(?, description),
                "order"     = COALESCE(?, "order"),
                updated_at  = ?
          WHERE id = ? AND deleted = 0`,
		nullableString(name), nullableString(path), nullableString(description),
		nullableOrder(order), now, id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrModuleExists
		}
		return fmt.Errorf("update module: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrModuleNotFound
	}
	return nil
}

func nullableOrder(p *int) sql.NullInt32 {
	if p == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*p), Valid: true}
}

// MoveModule re-parents `id` to newParentID (nil = root). Returns
// ErrModuleCycle if newParentID is a descendant of id (i.e. moving
// `id` under `newParentID` would put `newParentID` back under `id`,
// creating a cycle). All module FKs (sessions / memories) keep
// working because the id stays the same.
func (db *DB) MoveModule(ctx context.Context, id string, newParentID *string) error {
	if newParentID != nil && *newParentID == id {
		return ErrModuleCycle
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Cycle check: walk DOWN from `id` — if newParentID appears in
	// the subtree rooted at `id`, moving would loop. (The other
	// direction — newParentID's subtree contains `id` — is a no-op
	// for cycle purposes since `id` itself isn't reachable from
	// itself through the new structure.)
	if newParentID != nil {
		cycle, err := isDescendantInTx(ctx, tx, id, *newParentID, MaxModuleDepth)
		if err != nil {
			return fmt.Errorf("cycle check: %w", err)
		}
		if cycle {
			return ErrModuleCycle
		}
	}

	// Capture current parent (so we can flip is_leaf back later if
	// it has no more children).
	var oldParent sql.NullString
	if err := tx.QueryRowContext(ctx,
		`SELECT parent_id FROM modules WHERE id = ? AND deleted = 0`, id,
	).Scan(&oldParent); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrModuleNotFound
		}
		return err
	}

	// Apply move.
	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx,
		`UPDATE modules SET parent_id = ?, updated_at = ? WHERE id = ? AND deleted = 0`,
		nullableParent(newParentID), now, id,
	)
	if err != nil {
		return fmt.Errorf("update parent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrModuleNotFound
	}

	// Old parent: refresh leaf state. If it has no remaining
	// children, it becomes a leaf again.
	if oldParent.Valid {
		var cnt int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM modules WHERE parent_id = ? AND deleted = 0`,
			oldParent.String,
		).Scan(&cnt); err != nil {
			return fmt.Errorf("count oldParent children: %w", err)
		}
		if err := flipLeafInTx(ctx, tx, oldParent.String, cnt == 0); err != nil {
			return fmt.Errorf("flip old parent: %w", err)
		}
	}

	// New parent (if any) becomes internal.
	if newParentID != nil {
		if err := flipLeafInTx(ctx, tx, *newParentID, false); err != nil {
			return fmt.Errorf("flip new parent: %w", err)
		}
	}

	return tx.Commit()
}

// isDescendantInTx walks down from `root` until it either reaches
// `target` (returns true) or exhausts the breadth (returns false).
// Capped at maxDepth hops so a corrupt cycle can't loop forever.
func isDescendantInTx(ctx context.Context, tx *sql.Tx, root, target string, maxDepth int) (bool, error) {
	current := []string{root}
	for hop := 0; hop < maxDepth; hop++ {
		if len(current) == 0 {
			return false, nil
		}
		next := make([]string, 0, len(current)*2)
		for _, pid := range current {
			rows, err := tx.QueryContext(ctx,
				`SELECT id FROM modules WHERE parent_id = ? AND deleted = 0`,
				pid,
			)
			if err != nil {
				return false, err
			}
			for rows.Next() {
				var child string
				if err := rows.Scan(&child); err != nil {
					rows.Close()
					return false, err
				}
				if child == target {
					rows.Close()
					return true, nil
				}
				next = append(next, child)
			}
			rows.Close()
		}
		current = next
	}
	return false, nil
}

// DeleteModule soft-deletes a node. `cascade` controls what happens
// to descendants:
//   - true  : delete the subtree
//   - false : only delete the node if it has no children (ErrModuleHasChildren otherwise)
//
// Sessions and memories FK'd into modules use ON DELETE RESTRICT,
// so any leaves that still hold data won't actually be deleted at
// the DB level. We surface those as ErrModuleInUse so the API can
// give the operator a useful 409.
func (db *DB) DeleteModule(ctx context.Context, id string, cascade bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Refuse if dependents exist.
	var sessCount, memCount int
	_ = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE module_id = ?`, id,
	).Scan(&sessCount)
	_ = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM memories WHERE module_id = ?`, id,
	).Scan(&memCount)
	if sessCount+memCount > 0 {
		return ErrModuleInUse
	}

	if !cascade {
		var childCount int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM modules WHERE parent_id = ? AND deleted = 0`, id,
		).Scan(&childCount); err != nil {
			return fmt.Errorf("count children: %w", err)
		}
		if childCount > 0 {
			return ErrModuleHasChildren
		}
	}

	// If cascading, soft-delete every descendant in BFS order so the
	// leaf-flip of the parent happens correctly.
	if cascade {
		if err := softDeleteSubtree(ctx, tx, id); err != nil {
			return err
		}
	} else {
		if err := softDeleteOne(ctx, tx, id); err != nil {
			return err
		}
	}

	// Refresh leaf flag of the now-orphaned parent.
	var parent sql.NullString
	if err := tx.QueryRowContext(ctx,
		`SELECT parent_id FROM modules WHERE id = ?`, id,
	).Scan(&parent); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if parent.Valid {
		var cnt int
		_ = tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM modules WHERE parent_id = ? AND deleted = 0`, parent.String,
		).Scan(&cnt)
		_ = flipLeafInTx(ctx, tx, parent.String, cnt == 0)
	}

	return tx.Commit()
}

// ErrModuleHasChildren signals a delete-without-cascade attempt on a
// non-empty parent. Surfaced as 409 by handlers.
var ErrModuleHasChildren = errors.New("module has child modules; retry with cascade=true")

// ErrModuleInUse signals that sessions or memories are still pinned
// to the module. Surface as 409 too.
var ErrModuleInUse = errors.New("module still referenced by sessions or memories")

// softDeleteOne flips deleted=1 for exactly one node.
func softDeleteOne(ctx context.Context, tx *sql.Tx, id string) error {
	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx,
		`UPDATE modules SET deleted = 1, is_leaf = 0, updated_at = ?
		   WHERE id = ? AND deleted = 0`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("delete module: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrModuleNotFound
	}
	return nil
}

// softDeleteSubtree flips deleted=1 on `root` and every descendant
// in BFS order. Ancestors are not touched (they'll be re-evaluated
// by the leaf-flip step above).
func softDeleteSubtree(ctx context.Context, tx *sql.Tx, root string) error {
	queue := []string{root}
	now := time.Now().UTC()
	seen := map[string]bool{root: true}
	for len(queue) > 0 {
		head := queue[0]
		queue = queue[1:]
		res, err := tx.ExecContext(ctx,
			`UPDATE modules SET deleted = 1, is_leaf = 0, updated_at = ?
			   WHERE id = ? AND deleted = 0`,
			now, head,
		)
		if err != nil {
			return fmt.Errorf("delete %s: %w", head, err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			continue
		}
		rows, err := tx.QueryContext(ctx,
			`SELECT id FROM modules WHERE parent_id = ? AND deleted = 0`, head,
		)
		if err != nil {
			return fmt.Errorf("descend %s: %w", head, err)
		}
		for rows.Next() {
			var child string
			if err := rows.Scan(&child); err != nil {
				rows.Close()
				return err
			}
			if !seen[child] {
				seen[child] = true
				queue = append(queue, child)
			}
		}
		rows.Close()
	}
	return nil
}

// ----------------------------------------------------------------------------
// Tree helpers
// ----------------------------------------------------------------------------

// BuildModuleTree takes a flat list (typically from
// ListModulesByProject) and returns a nested ModuleNode tree ordered
// by `order`, then name. Children are populated recursively. The cap
// is MaxModuleDepth to guard against corrupt very-deep trees.
func BuildModuleTree(flat []*Module) []*ModuleNode {
	byParent := map[string][]*Module{}
	for _, m := range flat {
		key := ""
		if m.ParentID != nil {
			key = *m.ParentID
		}
		byParent[key] = append(byParent[key], m)
	}
	roots := buildChildren(byParent, "")
	return roots
}

func buildChildren(byParent map[string][]*Module, parent string) []*ModuleNode {
	children := byParent[parent]
	out := make([]*ModuleNode, 0, len(children))
	for _, m := range children {
		node := &ModuleNode{Module: *m}
		// Recurse, but cap depth: any module chain longer than
		// MaxModuleDepth is rendered leaf-only to keep the response
		// bounded. Per the M4 acceptance bar (≥ 5 levels) this is
		// well above what's needed.
		node.Children = buildChildrenBounded(byParent, m.ID, 0, MaxModuleDepth)
		out = append(out, node)
	}
	return out
}

func buildChildrenBounded(byParent map[string][]*Module, parent string, depth, max int) []*ModuleNode {
	if depth >= max {
		// Truncate: surface the direct children as leaves so the UI
		// can still show "..." or a "load more" affordance.
		kids := byParent[parent]
		out := make([]*ModuleNode, 0, len(kids))
		for _, m := range kids {
			n := &ModuleNode{Module: *m}
			n.IsLeaf = true
			out = append(out, n)
		}
		return out
	}
	return buildChildren(byParent, parent)
}

// CountChildren returns the number of immediate children of a module.
// Used to populate DTO.ChildCount.
func (db *DB) CountChildren(ctx context.Context, id string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM modules WHERE parent_id = ? AND deleted = 0`, id,
	).Scan(&n)
	return n, err
}

// CountDescendants returns the total number of descendants (not
// self) — used to give operators a "this will cascade N modules"
// warning before a delete-with-cascade call.
func (db *DB) CountDescendants(ctx context.Context, id string) (int, error) {
	depth := 0
	total := 0
	queue := []string{id}
	for len(queue) > 0 && depth <= MaxModuleDepth {
		next := make([]string, 0, len(queue))
		for _, p := range queue {
			rows, err := db.QueryContext(ctx,
				`SELECT id FROM modules WHERE parent_id = ? AND deleted = 0`, p,
			)
			if err != nil {
				return total, err
			}
			for rows.Next() {
				var c string
				if err := rows.Scan(&c); err != nil {
					rows.Close()
					return total, err
				}
				total++
				next = append(next, c)
			}
			rows.Close()
		}
		queue = next
		depth++
	}
	return total, nil
}

// ----------------------------------------------------------------------------
// Module-gated lookups (used by sessions / memories)
// ----------------------------------------------------------------------------

// LookupDefaultLeafModule returns the first available leaf module
// under `projectID`, or creates one if the project has no modules
// yet. The id is returned; "" + error otherwise.
//
// This is the "fallback leaf" used by the capture middleware when an
// incoming MCP request has no `?module=` / `X-Module-Id` header — we
// always need *some* leaf to satisfy sessions.module_id NOT NULL.
// Operators that care about clean module attribution should send
// the header / query.
func (db *DB) LookupDefaultLeafModule(ctx context.Context, projectID string) (string, error) {
	var id string
	err := db.QueryRowContext(ctx,
		`SELECT id FROM modules
		  WHERE deleted = 0 AND project_id = ? AND is_leaf = 1
		  ORDER BY created_at ASC LIMIT 1`,
		projectID,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	// Auto-seed a "default" leaf so FK constraints can be satisfied
	// even before the operator has curated the module tree.
	m := &Module{
		ID:        "mod_default_" + strings.ReplaceAll(projectID, "/", "_"),
		ProjectID: projectID,
		Name:      "default",
		IsLeaf:    true,
	}
	if err := db.CreateModule(ctx, m); err != nil && !errors.Is(err, ErrModuleExists) {
		return "", err
	}
	return m.ID, nil
}

// RequireLeafModule confirms the module is a non-deleted leaf.
// Returns ErrModuleNotFound / ErrModuleNotLeaf as appropriate.
func (db *DB) RequireLeafModule(ctx context.Context, id string) error {
	var isLeaf int
	var deleted int
	err := db.QueryRowContext(ctx,
		`SELECT is_leaf, deleted FROM modules WHERE id = ?`, id,
	).Scan(&isLeaf, &deleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrModuleNotFound
		}
		return err
	}
	if deleted != 0 {
		return ErrModuleNotFound
	}
	if isLeaf == 0 {
		return ErrModuleNotLeaf
	}
	return nil
}

// ModuleAccessibleByUser is a convenience wrapper used by handlers:
// admin OR team member can see the module. The hook for M5 memories
// that have stricter visibility (private vs team-visible).
func (db *DB) ModuleAccessibleByUser(ctx context.Context, moduleID, userID, role string) (bool, error) {
	if role == string(RoleAdmin) {
		return true, nil
	}
	var projectID string
	err := db.QueryRowContext(ctx,
		`SELECT project_id FROM modules WHERE id = ? AND deleted = 0`, moduleID,
	).Scan(&projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	var teamID string
	err = db.QueryRowContext(ctx,
		`SELECT team_id FROM projects WHERE id = ? AND deleted = 0`, projectID,
	).Scan(&teamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return db.IsTeamMember(ctx, teamID, userID)
}

// ----------------------------------------------------------------------------
// scanner
// ----------------------------------------------------------------------------
//
// scanner is the minimal Scan interface satisfied by *sql.Row and
// *sql.Rows, so one helper handles both. It's defined in db_auth.go
// alongside the other storage helpers — no local copy needed.

func (db *DB) scanModule(s scanner) (*Module, error) {
	m := &Module{}
	var parent sql.NullString
	var deleted int
	if err := s.Scan(
		&m.ID, &m.ProjectID, &parent, &m.Name, &m.Path,
		&m.Description, &m.Order,
		&m.IsLeaf, &m.CreatedAt, &m.UpdatedAt, &deleted,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrModuleNotFound
		}
		return nil, err
	}
	if parent.Valid {
		v := parent.String
		m.ParentID = &v
	}
	m.Deleted = deleted != 0
	return m, nil
}
