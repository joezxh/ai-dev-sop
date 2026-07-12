#!/bin/bash
BASE='http://localhost:8787'
H='X-Admin-Token: dev-admin-token-change-me-2026'

curl -sS -X POST "$BASE/admin/users/alice/token?ttl=1h" -H "$H" > /tmp/tok.json
python3 -c 'import json;print(json.load(open("/tmp/tok.json"))["token"])' > /tmp/tok
TOKEN=$(cat /tmp/tok)
echo "Authorization: Bearer $TOKEN" > /tmp/auth.h
echo "auth file content:"
cat /tmp/auth.h

echo
echo '--- search_code call ---'
curl -sS -X POST "$BASE/mcp?as=alice&project=/tmp/test-proj" \
    -H @/tmp/auth.h \
    -H 'Content-Type: application/json' \
    --data-binary @/tmp/rs.json | head -c 800
echo

echo '--- pool stats ---'
curl -sS "$BASE/admin/stats" -H "$H"
echo

echo '--- write a real python file to index for next test ---'
mkdir -p /tmp/test-proj
cat > /tmp/test-proj/sample.py <<'PY'
def greet(name):
    return f"Hello, {name}"

class Calculator:
    def add(self, a, b):
        return a + b
    def multiply(self, a, b):
        return a * b
PY
ls /tmp/test-proj/