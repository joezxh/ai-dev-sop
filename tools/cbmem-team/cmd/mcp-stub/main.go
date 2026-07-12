// mcp-stub — minimal MCP stdio responder used by e2e-v6 (and any
// future e2e that needs a deterministic backend without depending on
// /usr/local/bin/codebase-memory-mcp existing).
//
// Protocol: MCP JSON-RPC over newline-delimited stdio, one frame per line.
// Behaviour:
//   - Initialize handshake: returns server info + capability list.
//   - tools/list: returns one synthetic tool named "stub-echo" so callers
//     that probe /v2/tools still see ≥1 entry.
//   - tools/call: returns a small {"content":[{...}]} result with a 200ms
//     sleep. The 200ms is long enough that 5 concurrent requests on the
//     same (user,tool) bucket overlap and the rate limiter at qps=1 has a
//     real chance to reject the late arrivals with 429. It is also short
//     enough that the e2e suite finishes in a few seconds.
//   - everything else: returns a benign empty result so the test never
//     trips on an unknown method.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "mcp-stub: read err: %v\n", err)
			}
			return
		}
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		var resp []byte
		switch req.Method {
		case "initialize":
			resp = mustJSON(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"serverInfo":      map[string]any{"name": "mcp-stub", "version": "0.0.1"},
					"capabilities":    map[string]any{},
				},
			})
		case "tools/list":
			resp = mustJSON(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"tools": []map[string]any{
						{"name": "stub-echo", "description": "always responds ok"},
					},
				},
			})
		case "tools/call":
			// Simulate real work long enough for the rate limiter to see
			// concurrent arrivals on the same (user, tool_id) bucket.
			time.Sleep(200 * time.Millisecond)
			resp = mustJSON(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": "ok"},
					},
				},
			})
		case "ping":
			resp = mustJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
		default:
			// Unknown: reply with empty result so the upstream proxy never
			// sees an error and breaks the test for an unrelated reason.
			resp = mustJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
		}
		os.Stdout.Write(resp)
		os.Stdout.Write([]byte("\n"))
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		// Should never happen for the literal shapes we build above.
		fmt.Fprintf(os.Stderr, "mcp-stub: marshal err: %v\n", err)
		return []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"internal"}}` + "\n")
	}
	return b
}