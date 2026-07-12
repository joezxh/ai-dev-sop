#!/bin/bash
# End-to-end smoke test for cbmem-team
H='X-Admin-Token: dev-admin-token-change-me-2026'
BASE='http://localhost:8787'

echo '--- list users ---'
curl -fsS "$BASE/admin/users" -H "$H"
echo

echo '--- mint token ---'
TOKEN=$(curl -fsS -X POST "$BASE/admin/users/alice/token?ttl=1h" -H "$H" | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
echo "TOKEN=${TOKEN:0:48}..."
echo

echo '--- MCP initialize ---'
cat > /tmp/req1.json <<'JSON'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}
JSON
curl -fsS -i -X POST "$BASE/mcp?as=alice&project=/tmp/test-proj" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    --data-binary @/tmp/req1.json
echo

echo '--- tools/list ---'
cat > /tmp/req2.json <<'JSON'
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
JSON
curl -fsS -X POST "$BASE/mcp?as=alice&project=/tmp/test-proj" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    --data-binary @/tmp/req2.json | head -c 800
echo

echo '--- stats ---'
curl -fsS "$BASE/admin/stats" -H "$H"
echo

echo OK