package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// capturedReq records what the fake MemPalace HTTP MCP endpoint received.
type capturedReq struct {
	sync.Mutex
	path   string
	auth   string
	body   string
	method string // HTTP method
}

// newDrawerServer stands in for the MemPalace `POST /mcp` endpoint. It
// records the last request and replies with the supplied JSON-RPC body.
func newDrawerServer(reply string, status int) (*httptest.Server, *capturedReq) {
	rec := &capturedReq{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.Lock()
		defer rec.Unlock()
		b, _ := io.ReadAll(r.Body)
		rec.path = r.URL.Path
		rec.auth = r.Header.Get("Authorization")
		rec.body = string(b)
		rec.method = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	return srv, rec
}

const okDrawerReply = `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"status\":\"filed\"}"}]}}`

func TestMemPalaceAddDrawerSendsToolCall(t *testing.T) {
	srv, rec := newDrawerServer(okDrawerReply, 200)
	defer srv.Close()

	mp := NewMemPalace(srv.URL, "")
	if err := mp.AddDrawer(nil, "wing-x", "room-y", "hall-z", "hello world"); err != nil {
		t.Fatal(err)
	}

	rec.Lock()
	defer rec.Unlock()

	if rec.path != "/mcp" {
		t.Fatalf("path = %q, want /mcp", rec.path)
	}

	// The body must be a JSON-RPC tools/call for mempalace_add_drawer with
	// hall carried in source_file (MemPalace rejects unknown arg keys).
	var got jsonRPCRequest
	if err := json.Unmarshal([]byte(rec.body), &got); err != nil {
		t.Fatalf("unmarshal request body: %v (%s)", err, rec.body)
	}
	if got.Method != "tools/call" {
		t.Fatalf("method = %q, want tools/call", got.Method)
	}
	if got.Params.Name != "mempalace_add_drawer" {
		t.Fatalf("tool name = %q, want mempalace_add_drawer", got.Params.Name)
	}
	a := got.Params.Arguments
	if a.Wing != "wing-x" || a.Room != "room-y" || a.Content != "hello world" {
		t.Fatalf("args mismatch: %+v", a)
	}
	if a.SourceFile != "hall-z" {
		t.Fatalf("hall not mapped to source_file: %+v", a)
	}
	// No token configured => no Authorization header.
	if rec.auth != "" {
		t.Fatalf("unexpected auth header: %q", rec.auth)
	}
}

func TestMemPalaceAddDrawerOnceSendsBearer(t *testing.T) {
	srv, rec := newDrawerServer(okDrawerReply, 200)
	defer srv.Close()

	mp := NewMemPalace(srv.URL, "secret-token")
	if err := mp.AddDrawerOnce(nil, "w", "r", "h", "c"); err != nil {
		t.Fatal(err)
	}

	rec.Lock()
	defer rec.Unlock()
	if rec.path != "/mcp" {
		t.Fatalf("path = %q, want /mcp", rec.path)
	}
	if rec.auth != "Bearer secret-token" {
		t.Fatalf("auth = %q, want Bearer secret-token", rec.auth)
	}
}

func TestMemPalaceAddDrawerRPCError(t *testing.T) {
	// A JSON-RPC error object (HTTP 200) must surface as a Go error — e.g.
	// MemPalace rejecting a missing/unknown parameter with -32602.
	reply := `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"Missing required parameter 'wing'"}}`
	srv, _ := newDrawerServer(reply, 200)
	defer srv.Close()

	mp := NewMemPalace(srv.URL, "")
	err := mp.AddDrawerOnce(nil, "", "r", "h", "c")
	if err == nil {
		t.Fatal("expected error on JSON-RPC error reply, got nil")
	}
	if !strings.Contains(err.Error(), "-32602") {
		t.Fatalf("error missing rpc code: %v", err)
	}
}

func TestMemPalaceAddDrawerHTTPError(t *testing.T) {
	srv, _ := newDrawerServer(`{"error":"boom"}`, 500)
	defer srv.Close()

	mp := NewMemPalace(srv.URL, "")
	if err := mp.AddDrawerOnce(nil, "w", "r", "h", "c"); err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}
}
