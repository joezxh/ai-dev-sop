package console

import (
	"bytes"
	"errors"
	"net/http"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// v2 (M5) Memory template HTTP handlers.
// =============================================================================
//
//   GET    /api/v2/memory-templates                 list (built-ins first)
//   GET    /api/v2/memory-templates/:id             detail
//   POST   /api/v2/memory-templates/:id/render      render body_template with fields
//   POST   /api/v2/memory-templates                 create user template (admin)
//
// Render executes the body_template (a Go text/template) with the
// supplied Fields. Returns the rendered markdown. The same engine
// is used by the front-end preview so the rendered output matches.
// =============================================================================

// MemoryTemplateHandlers wraps the v2 storage layer.
type MemoryTemplateHandlers struct {
	DB *DB
}

// NewMemoryTemplateHandlers builds a MemoryTemplateHandlers.
func NewMemoryTemplateHandlers(db *DB) *MemoryTemplateHandlers {
	return &MemoryTemplateHandlers{DB: db}
}

// List returns the catalogue of templates. The UI uses this to
// render the "pick a template" dropdown on the memory create form.
func (h *MemoryTemplateHandlers) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		builtinOnly := c.Query("builtin_only") == "true"
		rows, err := h.DB.ListMemoryTemplates(ctx, builtinOnly)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000090, "list templates: "+err.Error())
			return
		}
		out := make([]MemoryTemplateDTO, 0, len(rows))
		for _, t := range rows {
			out = append(out, templateToDTO(t))
		}
		OK(c, gin.H{"list": out, "total": len(out)})
	}
}

// Get returns one template by id.
func (h *MemoryTemplateHandlers) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		t, err := h.DB.GetMemoryTemplate(ctx, id)
		if err != nil {
			if errors.Is(err, ErrTemplateMissing) {
				Fail(c, http.StatusNotFound, 4040080, "template not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000091, "read template: "+err.Error())
			return
		}
		OK(c, templateToDTO(t))
	}
}

// Render executes the template body with the supplied fields. The
// date field is auto-injected so ADR/decision-style templates don't
// have to expose it as a user-facing input.
func (h *MemoryTemplateHandlers) Render() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		t, err := h.DB.GetMemoryTemplate(ctx, id)
		if err != nil {
			if errors.Is(err, ErrTemplateMissing) {
				Fail(c, http.StatusNotFound, 4040081, "template not found")
				return
			}
			Fail(c, http.StatusInternalServerError, 5000092, "read template: "+err.Error())
			return
		}
		var req RenderTemplateReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000080, "invalid request: "+err.Error())
			return
		}
		// Inject auto fields. Callers can override by passing "date"
		// explicitly but typically don't.
		if req.Fields == nil {
			req.Fields = map[string]any{}
		}
		if _, ok := req.Fields["date"]; !ok {
			req.Fields["date"] = time.Now().UTC().Format("2006-01-02")
		}
		// Parse and execute. Missing keys render as "<no value>" so
		// the UI can flag missing required fields.
		tpl, err := template.New(t.ID).Option("missingkey=default").Parse(t.BodyTemplate)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 5000093, "parse template: "+err.Error())
			return
		}
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, req.Fields); err != nil {
			Fail(c, http.StatusBadRequest, 4000081, "render: "+err.Error())
			return
		}
		OK(c, RenderTemplateResp{Rendered: buf.String()})
	}
}

// Create inserts a user template. Admin-only because the catalogue
// is curated and the 5 built-ins already cover the common cases.
func (h *MemoryTemplateHandlers) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		if CurrentRole(c) != RoleAdmin {
			Fail(c, http.StatusForbidden, 4030080, "admin role required")
			return
		}
		var req CreateTemplateReq
		if err := c.ShouldBindJSON(&req); err != nil {
			Fail(c, http.StatusBadRequest, 4000082, "invalid request: "+err.Error())
			return
		}
		if req.Name == "" {
			Fail(c, http.StatusBadRequest, 4000083, "name required")
			return
		}
		t := &MemoryTemplate{
			ID:           "tpl_" + randomHex(4),
			Name:         req.Name,
			Description:  req.Description,
			FieldsJSON:   req.FieldsJSON,
			BodyTemplate: req.BodyTemplate,
		}
		if err := h.DB.CreateMemoryTemplate(ctx, t); err != nil {
			Fail(c, http.StatusInternalServerError, 5000094, "create template: "+err.Error())
			return
		}
		OK(c, templateToDTO(t))
	}
}

// templateToDTO flattens the storage row into the wire shape.
// Fields are parsed from FieldsJSON; parse failures degrade to nil
// (the UI just won't pre-fill the form) rather than 500-ing.
func templateToDTO(t *MemoryTemplate) MemoryTemplateDTO {
	fields := parseTemplateFields(t.FieldsJSON)
	return MemoryTemplateDTO{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Fields:      fields,
		IsBuiltin:   t.IsBuiltin,
	}
}