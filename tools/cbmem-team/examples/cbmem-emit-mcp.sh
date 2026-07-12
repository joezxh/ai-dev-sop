#!/usr/bin/env bash
# mint-and-emit (POSIX): mint a long-lived JWT and write ~/.cursor/mcp.json
# for the cbmem-team entry.
#
# Usage:
#   BASE=http://192.168.100.83:8787 \
#   ADMIN_TOKEN=e1c96... \
#   USER=tianque \
#   PROJECT_SERVER_PATH=/var/lib/cbmem-team/users/tianque/repos/... \
#   bash examples/cbmem-emit-mcp.sh
#
# Or pass flags:
#   bash examples/cbmem-emit-mcp.sh \
#     --base http://... --admin-token ... --user ... --project-server-path ...
#
# Environment overrides (any of these may be set; flags win):
#   BASE, ADMIN_TOKEN, USER, PROJECT_SERVER_PATH, MCP_PATH, TTL,
#   SERVER_NAME, PORT, HOST_OVERRIDE
#
set -euo pipefail

BASE="${BASE:-http://192.168.100.83:8787}"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"
USER_ID="${USER:-}"
PROJECT_SERVER_PATH="${PROJECT_SERVER_PATH:-}"
MCP_PATH="${MCP_PATH:-$HOME/.cursor/mcp.json}"
TTL="${TTL:-4320h}"
SERVER_NAME="${SERVER_NAME:-cbmem-team}"
PORT="${PORT:-8787}"
HOST_OVERRIDE="${HOST_OVERRIDE:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base) BASE="$2"; shift 2 ;;
    --admin-token) ADMIN_TOKEN="$2"; shift 2 ;;
    --user) USER_ID="$2"; shift 2 ;;
    --project-server-path) PROJECT_SERVER_PATH="$2"; shift 2 ;;
    --mcp-path) MCP_PATH="$2"; shift 2 ;;
    --ttl) TTL="$2"; shift 2 ;;
    --server-name) SERVER_NAME="$2"; shift 2 ;;
    --port) PORT="$2"; shift 2 ;;
    --host-override) HOST_OVERRIDE="$2"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$ADMIN_TOKEN" || -z "$USER_ID" || -z "$PROJECT_SERVER_PATH" ]]; then
  echo "usage: $0 --base URL --admin-token TOK --user ID --project-server-path PATH [--ttl 4320h]" >&2
  exit 2
fi

say() { printf '\033[36m==> %s\033[0m\n' "$*"; }
warn() { printf '\033[33m!!  %s\033[0m\n' "$*"; }
ok()   { printf '\033[32mOK  %s\033[0m\n' "$*"; }

# 1. mint
say "minting JWT for $USER_ID (ttl=$TTL)"
resp="$(curl -fsS -X POST -H "X-Admin-Token: $ADMIN_TOKEN" \
  "$BASE/admin/users/$USER_ID/token?ttl=$TTL")"
jwt="$(printf '%s' "$resp" | python -c 'import json,sys; print(json.load(sys.stdin)["token"])')"
expires="$(printf '%s' "$resp" | python -c 'import json,sys; print(json.load(sys.stdin)["expires"])')"
ok "token expires $expires"

# 2. resolve host + URL
host_for_url="${HOST_OVERRIDE:-$(printf '%s' "$BASE" | sed -E 's|^https?://||; s|/.*||')}"
encoded_user="$(python -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "$USER_ID")"
encoded_proj="$(python -c "import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=''))" "$PROJECT_SERVER_PATH")"
mcp_url="http://${host_for_url}:${PORT}/mcp?as=${encoded_user}&project=${encoded_proj}"

# 3. upsert mcp.json
say "writing $MCP_PATH"
python - "$MCP_PATH" "$SERVER_NAME" "$mcp_url" "$jwt" <<'PY'
import json, os, sys
from pathlib import Path
path = Path(sys.argv[1])
server_name = sys.argv[2]
url = sys.argv[3]
jwt = sys.argv[4]

if path.exists():
    cfg = json.loads(path.read_text(encoding='utf-8'))
else:
    cfg = {"mcpServers": {}}
cfg.setdefault("mcpServers", {})
cfg["mcpServers"][server_name] = {
    "type": "http",
    "url": url,
    "headers": {"Authorization": f"Bearer {jwt}"},
    "enabled": True,
}
path.parent.mkdir(parents=True, exist_ok=True)
path.write_text(json.dumps(cfg, indent=2, ensure_ascii=False) + "\n", encoding='utf-8')
print(f"OK  wrote {path.resolve()}", flush=True)
PY

# 4. summary
echo "SUMMARY server=$SERVER_NAME url=$mcp_url ttl=$TTL expires=$expires"

# 5. smoke verify
say "verification (tools/list)"
body='{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
http_code="$(curl -s -o /tmp/cbmem-verify.out -w '%{http_code}' \
  -X POST "$mcp_url" \
  -H "Authorization: Bearer $jwt" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d "$body" || echo '000')"
if [[ "$http_code" == "200" ]]; then
  n="$(python -c 'import json,sys; print(len(json.load(open("/tmp/cbmem-verify.out"))["result"]["tools"]))')"
  ok "tools/list returned $n tools"
else
  warn "verify failed (http=$http_code); config still written. body:"
  head -c 400 /tmp/cbmem-verify.out || true
  echo
fi