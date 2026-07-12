#!/bin/bash
set -e
BASE='http://localhost:8787'
H='X-Admin-Token: dev-admin-token-change-me-2026'

# Mint token -> save to file (avoid passing through bash args)
curl -sS -X POST "$BASE/admin/users/alice/token?ttl=1h" -H "$H" \
    | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])' > /tmp/alice.tok

TOKEN=$(cat /tmp/alice.tok)
echo "TOKEN length=${#TOKEN}"
echo "TOKEN preview=${TOKEN:0:48}..."
echo

# Use a header file to avoid quoting hell with Bearer
cat > /tmp/auth.h <<EOF
Authorization: Bearer $TOKEN
EOF

echo '--- MCP initialize ---'
curl -sS -i -X POST "$BASE/mcp?as=alice&project=/tmp/test-proj" \
    -H "@/tmp/auth.h" \
    -H 'Content-Type: application/json' \
    --data-binary @/tmp/req-init.json

echo
echo '--- tools/list ---'
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' > /tmp/r2.json
curl -sS -X POST "$BASE/mcp?as=alice&project=/tmp/test-proj" \
    -H "@/tmp/auth.h" \
    -H 'Content-Type: application/json' \
    --data-binary @/tmp/r2.json | head -c 1000
echo

echo '--- stats ---'
curl -sS "$BASE/admin/stats" -H "$H"
echo

echo '--- subprocess test (direct) ---'
echo "ls /var/lib/cbmem-team/users/alice/projects/"
echo tianqueshuaige | sudo -S ls /var/lib/cbmem-team/users/alice/projects/ 2>&1 | tail -3