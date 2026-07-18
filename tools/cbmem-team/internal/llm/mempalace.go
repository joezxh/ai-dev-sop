package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MemPalace struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewMemPalace(baseURL, token string) *MemPalace {
	return &MemPalace{BaseURL: baseURL, Token: token, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// The MemPalace server speaks MCP JSON-RPC over POST /mcp — there is no REST
// /api/drawers endpoint. A drawer is filed by calling the mempalace_add_drawer
// tool, whose schema requires wing/room/content and accepts an optional
// source_file. Unknown argument keys are rejected with JSON-RPC -32602, so
// cbmem-team's "hall" concept is carried in source_file rather than its own key.
type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  toolCallParam `json:"params"`
}

type toolCallParam struct {
	Name      string        `json:"name"`
	Arguments addDrawerArgs `json:"arguments"`
}

type addDrawerArgs struct {
	Wing       string `json:"wing"`
	Room       string `json:"room"`
	Content    string `json:"content"`
	SourceFile string `json:"source_file,omitempty"`
	AddedBy    string `json:"added_by,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCResponse struct {
	Error *jsonRPCError `json:"error"`
}

// buildAddDrawerRequest builds a POST /mcp request carrying a tools/call for
// mempalace_add_drawer. hall is mapped onto source_file (see type comment).
func (m *MemPalace) buildAddDrawerRequest(ctx context.Context, wing, room, hall, content string) (*http.Request, error) {
	body, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: toolCallParam{
			Name: "mempalace_add_drawer",
			Arguments: addDrawerArgs{
				Wing:       wing,
				Room:       room,
				Content:    content,
				SourceFile: hall,
				AddedBy:    "cbmem-team",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal drawer: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", m.BaseURL+"/mcp", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.Token != "" {
		req.Header.Set("Authorization", "Bearer "+m.Token)
	}
	return req, nil
}

// parseAddDrawerResponse maps the JSON-RPC reply to success/failure. A non-2xx
// status or a JSON-RPC error object fails; any other reply (including a
// "duplicate" result payload) is treated as success.
func parseAddDrawerResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("mempalace status=%d body=%s", resp.StatusCode, string(body))
	}
	var rpc jsonRPCResponse
	if err := json.Unmarshal(body, &rpc); err != nil {
		return fmt.Errorf("mempalace decode response: %w body=%s", err, string(body))
	}
	if rpc.Error != nil {
		return fmt.Errorf("mempalace rpc error code=%d msg=%s", rpc.Error.Code, rpc.Error.Message)
	}
	return nil
}

func (m *MemPalace) AddDrawer(ctx context.Context, wing, room, hall, content string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		// Rebuild per attempt: the request body reader is single-use.
		req, err := m.buildAddDrawerRequest(ctx, wing, room, hall, content)
		if err != nil {
			return err
		}
		resp, err := m.HTTP.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		err = parseAddDrawerResponse(resp)
		_ = resp.Body.Close()
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return lastErr
}

// AddDrawerOnce performs a single POST /mcp tools/call with no internal
// retry. Callers that need retry semantics (exponential backoff, jitter,
// ctx-aware cancellation) wrap this in their own loop. The capture-time
// auto-sync uses this so the surrounding pushDrawerWithRetry loop is the
// *only* retry layer, otherwise the internal retries here would stack on top
// of its attempts and a MemPalace outage would take minutes to surface.
func (m *MemPalace) AddDrawerOnce(ctx context.Context, wing, room, hall, content string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := m.buildAddDrawerRequest(ctx, wing, room, hall, content)
	if err != nil {
		return err
	}
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return parseAddDrawerResponse(resp)
}

func (m *MemPalace) Status(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.BaseURL+"/healthz", nil)
	if err != nil {
		return "", err
	}
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "down", nil
	}
	return "up", nil
}
