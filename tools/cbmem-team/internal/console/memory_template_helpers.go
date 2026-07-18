package console

import "encoding/json"

// parseTemplateFields decodes the fields_json column into a typed
// slice. parse failures return nil — the UI treats that as "no
// schema" and renders a freeform form.
func parseTemplateFields(raw string) []TemplateField {
	if raw == "" {
		return nil
	}
	var out []TemplateField
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}