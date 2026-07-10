#!/usr/bin/env bash
# examples/e2e-console.sh —— cbmem-team console headless smoke test
set -e
PORT=${PORT:-18888}
ADMIN_TOKEN=${ADMIN_TOKEN:-e2e-admin-token}

DATA=$(mktemp -d)
./cbmem-team \
  -listen ":$PORT" \
  -data "$DATA" \
  -mcp-bin /bin/true \
  -jwt-secret e2e-jwt-secret-1234567890 \
  -admin-token "$ADMIN_TOKEN" \
  -llm-provider fake \
  -log info \
  > "$DATA/server.log" 2>&1 &
SERVER_PID=$!
trap "kill $SERVER_PID 2>/dev/null || true" EXIT

for i in {1..30}; do
  curl -fsS "http://127.0.0.1:$PORT/healthz" >/dev/null 2>&1 && break
  sleep 0.2
done

BASE="http://127.0.0.1:$PORT/api/console"
J() { curl -sS "$1" -H 'Content-Type: application/json' "${@:2}"; }

LOGIN=$(J -X POST "$BASE/login" -H "X-Admin-Token: $ADMIN_TOKEN")
CSRF=$(echo "$LOGIN" | jq -r .data.csrf_token)
SESSION=$(echo "$LOGIN" | jq -r .data.session_id)
test "$CSRF" != "null" -a "$CSRF" != "" || { echo "login failed: $LOGIN"; exit 1; }

H=(-H "X-CSRF-Token: $CSRF" -b "cbmem_console=$SESSION; cbmem_csrf=$CSRF")

# 1. me
test "$(J -X GET "$BASE/me" "${H[@]}" | jq -r .code)" = "0" || { echo "me failed"; exit 1; }

# 2. users create
J -X POST "$BASE/users" "${H[@]}" -d '{"id":"alice"}' >/dev/null

# 3. users list
TOTAL=$(J -X GET "$BASE/users" "${H[@]}" | jq -r .data.total)
test "$TOTAL" -ge 1 || { echo "users list total=$TOTAL"; exit 1; }

# 4. projects create
J -X POST "$BASE/projects" "${H[@]}" -d '{"name":"p","path":"/tmp/p-test"}' >/dev/null

# 5. projects list
PTOTAL=$(J -X GET "$BASE/projects" "${H[@]}" | jq -r .data.total)
test "$PTOTAL" -ge 1 || { echo "projects list total=$PTOTAL"; exit 1; }

# 6. sessions-stats
J -X GET "$BASE/sessions-stats" "${H[@]}" >/dev/null

# 7. logout
J -X POST "$BASE/logout" "${H[@]}" >/dev/null

echo "[e2e-console] OK"
