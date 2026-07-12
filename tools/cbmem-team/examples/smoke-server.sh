#!/bin/sh
H='X-Admin-Token: dev-admin-token-change-me-2026'
BASE='http://localhost:8787'

echo '--- healthz ---'
curl -fsS "$BASE/healthz"; echo

echo '--- create alice ---'
curl -fsS -X POST "$BASE/admin/users" -H "$H" -H 'Content-Type: application/json' -d '{"id":"alice","display_name":"Alice","project_paths":["/tmp/test-proj"]}'
echo

echo '--- list users ---'
curl -fsS "$BASE/admin/users" -H "$H"; echo

echo '--- mint token ---'
TOKEN=$(curl -fsS -X POST "$BASE/admin/users/alice/token?ttl=1h" -H "$H" | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
echo "TOKEN=${TOKEN:0:48}..."

echo '--- MCP initialize ---'
curl -fsS -X POST "$BASE/mcp?as=alice&project=/tmp/test-proj" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}'
echo

echo '--- MCP tools/list ---'
curl -fsS -X POST "$BASE/mcp?as=alice&project=/tmp/test-proj" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | head -c 800
echo

echo '--- stats ---'
curl -fsS "$BASE/admin/stats" -H "$H"; echo

echo OK