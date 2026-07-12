// Package console · M3 ticket auto-open from high-risk invocations.
//
// One table — `tickets` — captures reviewer-owned follow-ups. When the
// high-risk dashboard is queried, the route handler in tickets_handler.go
// also runs autoOpenTicket which scans tool_invocation_logs for the
// latest "un-ticketed" high-risk row and creates a draft ticket so the
// reviewer has something to triage. Manual create + list + resolve
// endpoints complete the loop.
//
// Tickets are intentionally minimal — id / source / severity / status /
// resolution_note. The M3 spec calls them "工单化：自动开单 + 复审动作"
// and the dedicated UI lives in M3's third-party review dashboard.
package console

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Ticket severity / status values.
const (
	TicketSevCritical = "critical"
	TicketSevHigh     = "high"
	TicketSevMedium   = "medium"
	TicketSevLow      = "low"

	TicketStatusOpen     = "open"
	TicketStatusResolved = "resolved"
	TicketStatusWontFix  = "wont_fix"
)

// TicketRecord is one row of `tickets`.
type TicketRecord struct {
	ID             string    `json:"id"`        // "tk-<uuid>"
	Source         string    `json:"source"`    // "high_risk_invocation" | "manual"
	InvocationID   string    `json:"invocation_id,omitempty"`
	ToolID         string    `json:"tool_id"`
	UserID         string    `json:"user_id"`
	Severity       string    `json:"severity"`
	Status         string    `json:"status"`
	Title          string    `json:"title"`
	DetailJSON     string    `json:"detail_json,omitempty"`
	ResolutionNote string    `json:"resolution_note,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// randomTicketHex returns a random hex suffix for ticket ids. Same
// pattern as randomBPHex / randomWorkflowHex so all three entity types
// follow one house style.
func randomTicketHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("randomTicketHex: %v", err))
	}
	return hex.EncodeToString(b)
}

// severityFor picks a severity bucket from the high-risk invocation row.
// Fatal error codes are bumped to critical; everything else is high.
func severityFor(errCode string, affectedHallsJSON, affectedADRsJSON string) string {
	switch errCode {
	case "UPSTREAM_ERROR", "FORBIDDEN", "TIMEOUT":
		return TicketSevCritical
	}
	if affectedHallsJSON != "" && affectedHallsJSON != "{}" && affectedHallsJSON != "[]" {
		return TicketSevCritical
	}
	if affectedADRsJSON != "" && affectedADRsJSON != "{}" && affectedADRsJSON != "[]" {
		return TicketSevHigh
	}
	return TicketSevMedium
}

// titleForInvocation builds a human-friendly title for the ticket.
func titleForInvocation(toolID, errCode string) string {
	if errCode != "" {
		return fmt.Sprintf("%s 触发 %s", toolID, errCode)
	}
	return fmt.Sprintf("%s 高风险调用", toolID)
}

// autoOpenTicket finds the latest un-ticketed high-risk invocation in
// the last 7 days and creates a draft ticket. Returns the new ticket id
// or empty string when nothing was opened. The function is idempotent
// within a run via the (source, invocation_id) unique-key emulation
// (we probe with a SELECT before INSERT).
func autoOpenTicket(ctx context.Context, db *DB) (string, error) {
	row := db.QueryRowContext(ctx, highRiskLatestSQL(db.driver))
	var invID, toolID, userID, errCode string
	if err := row.Scan(&invID, &toolID, &userID, &errCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("scan latest high-risk: %w", err)
	}

	// Skip if a ticket already references this invocation.
	var existing int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM tickets WHERE source = 'high_risk_invocation' AND invocation_id = ?`,
		invID,
	).Scan(&existing); err != nil {
		return "", fmt.Errorf("count existing tickets: %w", err)
	}
	if existing > 0 {
		return "", nil
	}

	// Probe affected_halls/adrs so severity reflects impact, not just
	// the error_code string.
	var hallsJSON, adrsJSON string
	_ = db.QueryRowContext(ctx,
		`SELECT IFNULL(affected_halls_json,''), IFNULL(affected_adrs_json,'')
           FROM tool_invocation_logs WHERE invocation_id = ?`, invID,
	).Scan(&hallsJSON, &adrsJSON)

	detail := map[string]any{
		"tool_id":           toolID,
		"user_id":           userID,
		"error_code":        errCode,
		"affected_halls":    hallsJSON,
		"affected_adrs":     adrsJSON,
		"invocation_id":     invID,
	}
	detailJSON, _ := json.Marshal(detail)

	t := &TicketRecord{
		ID:           "tk-" + randomTicketHex(6),
		Source:       "high_risk_invocation",
		InvocationID: invID,
		ToolID:       toolID,
		UserID:       userID,
		Severity:     severityFor(errCode, hallsJSON, adrsJSON),
		Status:       TicketStatusOpen,
		Title:        titleForInvocation(toolID, errCode),
		DetailJSON:   string(detailJSON),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO tickets
           (ticket_id, source, invocation_id, tool_id, user_id,
            severity, status, title, detail_json, created_at, updated_at)
         VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.Source, t.InvocationID, t.ToolID, t.UserID,
		t.Severity, t.Status, t.Title, t.DetailJSON, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return "", fmt.Errorf("insert ticket: %w", err)
	}
	return t.ID, nil
}

// highRiskLatestSQL returns the most recent high-risk invocation for
// the auto-open path. Driver-aware.
func highRiskLatestSQL(driver string) string {
	if driver == "mysql" {
		return `SELECT invocation_id, tool_id, user_id, IFNULL(error_code,'')
                FROM v_high_risk_invocations_7d
                ORDER BY started_at DESC LIMIT 1`
	}
	return `SELECT invocation_id, tool_id, user_id, IFNULL(error_code,'')
            FROM tool_invocation_logs
            WHERE started_at >= datetime('now','-7 days') AND (
                JSON_LENGTH(affected_halls_json) > 0
                OR JSON_LENGTH(affected_adrs_json) > 0
                OR JSON_LENGTH(blast_radius_json) > 0
                OR error_code IN ('UPSTREAM_ERROR','FORBIDDEN','TIMEOUT')
            )
            ORDER BY started_at DESC LIMIT 1`
}

// fetchTicket loads one ticket by id; sql.ErrNoRows when missing.
func fetchTicket(ctx context.Context, db *DB, id string) (*TicketRecord, error) {
	row := db.QueryRowContext(ctx,
		`SELECT ticket_id, source, IFNULL(invocation_id,''), tool_id, user_id,
                severity, status, title, IFNULL(detail_json,''), IFNULL(resolution_note,''),
                created_at, updated_at
           FROM tickets WHERE ticket_id = ?`, id)
	var t TicketRecord
	if err := row.Scan(&t.ID, &t.Source, &t.InvocationID, &t.ToolID, &t.UserID,
		&t.Severity, &t.Status, &t.Title, &t.DetailJSON, &t.ResolutionNote,
		&t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

// listTickets returns tickets in the given status ("" = all), newest first.
func listTickets(ctx context.Context, db *DB, status string, limit int) ([]TicketRecord, error) {
	q := `SELECT ticket_id, source, IFNULL(invocation_id,''), tool_id, user_id,
                 severity, status, title, IFNULL(detail_json,''), IFNULL(resolution_note,''),
                 created_at, updated_at
          FROM tickets`
	args := []any{}
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TicketRecord{}
	for rows.Next() {
		var t TicketRecord
		if err := rows.Scan(&t.ID, &t.Source, &t.InvocationID, &t.ToolID, &t.UserID,
			&t.Severity, &t.Status, &t.Title, &t.DetailJSON, &t.ResolutionNote,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// updateTicketStatus flips a ticket to resolved/wont_fix with an
// optional resolution note.
func updateTicketStatus(ctx context.Context, db *DB, id, status, note string) error {
	res, err := db.ExecContext(ctx,
		`UPDATE tickets SET status = ?, resolution_note = ?, updated_at = ?
         WHERE ticket_id = ?`,
		status, note, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}