#!/bin/bash
BASE='http://localhost:8787'
H='X-Admin-Token: dev-admin-token-change-me-2026'

# Mint token
curl -sS -X POST "$BASE/admin/users/alice/token?ttl=1h" -H "$H" > /tmp/tok.json
python3 -c 'import json;print(json.load(open("/tmp/tok.json"))["token"])' > /tmp/tok
TOKEN=$(cat /tmp/tok)
echo "Authorization: Bearer $TOKEN" > /tmp/auth.h
echo "Token preview: ${TOKEN:0:48}..."

# MCP per-line is one frame in/out; loop
{
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    echo ">>> $line"
    curl -sS -X POST "$BASE/mcp?as=alice&project=/tmp/test-proj" \
        -H @/tmp/auth.h \
        -H 'Content-Type: application/json' \
        --data-binary "$line" | head -c 500
    echo
  done < /tmp/r-full.jsonl
}

echo '--- stats ---'
curl -sS "$BASE/admin/stats" -H "$H"
echo