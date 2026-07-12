// Package console · M3 ticket HTTP handlers.
//
// Routes mounted by router.go under /api/console/v2/tickets:
//
//   GET    /api/console/v2/tickets                       list (filter: ?status, ?limit)
//   POST   /api/console/v2/tickets                       manual open
//   GET    /api/console/v2/tickets/:id                   fetch single
//   POST   /api/console/v2/tickets/:id/resolve           mark resolved
//   POST   /api/console/v2/tickets/:id/wont-fix          mark wont_fix
//   POST   /api/console/v2/tickets/auto-open             triggers autoOpenTicket (idempotent)
//
// autoOpenTicket is invoked by the high-risk dashboard route handler
// (dashboard_high_risk.go) so the reviewer always sees the latest
// un-ticketed risk.
package console

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ListTicketsHandler returns tickets (newest first). Default limit 50.
func ListTicketsHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		limit := atoiOrDefault(c.Query("limit"), 50)
		if limit < 1 || limit > 500 {
			limit = 50
		}
		tickets, err := listTickets(c.Request.Context(), db, status, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tickets": tickets, "count": len(tickets)})
	}
}

// CreateTicketHandler opens a manual ticket.
func CreateTicketHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var t TicketRecord
		if err := c.ShouldBindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if t.ToolID == "" || t.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tool_id and title are required"})
			return
		}
		if t.ID == "" {
			t.ID = "tk-" + randomTicketHex(6)
		}
		if t.Source == "" {
			t.Source = "manual"
		}
		if t.Severity == "" {
			t.Severity = TicketSevMedium
		}
		t.Status = TicketStatusOpen
		now := time.Now().UTC()
		t.CreatedAt = now
		t.UpdatedAt = now
		_, err := db.ExecContext(c.Request.Context(),
			`INSERT INTO tickets
               (ticket_id, source, invocation_id, tool_id, user_id,
                severity, status, title, detail_json, created_at, updated_at)
             VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			t.ID, t.Source, t.InvocationID, t.ToolID, t.UserID,
			t.Severity, t.Status, t.Title, t.DetailJSON, t.CreatedAt, t.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, t)
	}
}

// GetTicketHandler fetches one ticket by id.
func GetTicketHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		t, err := fetchTicket(c.Request.Context(), db, c.Param("id"))
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, t)
	}
}

// resolveTicket is shared by /resolve and /wont-fix.
func resolveTicket(db *DB, newStatus string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Note string `json:"note"`
		}
		_ = c.ShouldBindJSON(&body)
		err := updateTicketStatus(c.Request.Context(), db, c.Param("id"), newStatus, body.Note)
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": newStatus})
	}
}

// ResolveTicketHandler marks a ticket resolved.
func ResolveTicketHandler(db *DB) gin.HandlerFunc { return resolveTicket(db, TicketStatusResolved) }

// WontFixTicketHandler marks a ticket wont_fix.
func WontFixTicketHandler(db *DB) gin.HandlerFunc { return resolveTicket(db, TicketStatusWontFix) }

// AutoOpenTicketHandler triggers the auto-open routine. Returns 200 +
// {"opened": "<ticket_id>"} or {"opened": ""} when nothing was opened.
func AutoOpenTicketHandler(db *DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := autoOpenTicket(c.Request.Context(), db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"opened": id})
	}
}