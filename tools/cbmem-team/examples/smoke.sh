#!/usr/bin/env bash
# Quick smoke test for cbmem-team without needing a real Cursor client.
# Requires: curl, jq
set -euo pipefail

BASE=${BASE:-http://localhost:8787}
ADMIN=${ADMIN_TOKEN:?ADMIN_TOKEN env var required}

echo "==> health"
curl -fsS "$BASE/healthz" | jq .

echo "==> create user alice"
curl -fsS -X POST "$BASE/admin/users" \
    -H "X-Admin-Token: $ADMIN" \
    -H "Content-Type: application/json" \
    -d '{"id":"alice","display_name":"Alice","project_paths":["/code/foo"]}' | jq .

echo "==> mint token"
TOKEN=$(curl -fsS -X POST "$BASE/admin/users/alice/token?ttl=1h" \
    -H "X-Admin-Token: $ADMIN" | jq -r .token)
echo "token: ${TOKEN:0:24}..."

echo "==> initialize MCP session"
curl -fsS -X POST "$BASE/mcp?as=alice&project=/code/foo" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}' | jq .

echo "==> list tools"
curl -fsS -X POST "$BASE/mcp?as=alice&project=/code/foo" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | jq .

echo "==> stats"
curl -fsS "$BASE/admin/stats" -H "X-Admin-Token: $ADMIN" | jq .

echo "OK"