#!/usr/bin/env bash
# cbmem-team Linux startup script
# Usage: ./start.sh
# Stops any existing cbmem-team process and starts a new one with correct paths.
#
# Every setting can be overridden via environment variables, e.g.:
#   MEMPALACE_BASE=http://10.0.0.5:8765 MEMPALACE_TOKEN=secret ./start.sh
set -euo pipefail

# Resolve the directory this script lives in so relative defaults work
# regardless of the caller's CWD.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

CBMEM_BIN="${CBMEM_BIN:-$SCRIPT_DIR/cbmem-team}"
MCP_BIN="${MCP_BIN:-/usr/local/bin/codebase-memory-mcp}"
DATA_DIR="${DATA_DIR:-$SCRIPT_DIR/../../tmp/cbmem-data}"
JWT_SECRET="${JWT_SECRET:-dev-jwt-secret-please-rotate}"
ADMIN_TOKEN="${ADMIN_TOKEN:-change-me-too}"

# MemPalace HTTP MCP integration (POST /mcp). Leave MEMPALACE_BASE empty to
# disable auto-sync. MEMPALACE_TOKEN must match MemPalace's
# MEMPALACE_MCP_HTTP_TOKEN (required when MemPalace binds a non-loopback host).
MEMPALACE_BASE="${MEMPALACE_BASE:-http://127.0.0.1:8765}"
MEMPALACE_TOKEN="${MEMPALACE_TOKEN:-}"

# Stop existing process
if pgrep -f "$CBMEM_BIN" >/dev/null 2>&1; then
    echo "Stopping existing cbmem-team..."
    pkill -f "$CBMEM_BIN" || true
    sleep 1
fi

# Assemble argument list; only append MemPalace flags when a base URL is set.
args=(-mcp-bin "$MCP_BIN" -data "$DATA_DIR" -jwt-secret "$JWT_SECRET" -admin-token "$ADMIN_TOKEN")
if [ -n "$MEMPALACE_BASE" ]; then
    args+=(-mempalace-base "$MEMPALACE_BASE")
    if [ -n "$MEMPALACE_TOKEN" ]; then
        args+=(-mempalace-token "$MEMPALACE_TOKEN")
    fi
fi

# Start new process
echo "Starting cbmem-team..."
echo "  mcp-bin  : $MCP_BIN"
echo "  data     : $DATA_DIR"
echo "  listen   : :8787"
if [ -n "$MEMPALACE_BASE" ]; then
    echo "  mempalace: $MEMPALACE_BASE"
fi

nohup "$CBMEM_BIN" "${args[@]}" >"$SCRIPT_DIR/stdout.log" 2>"$SCRIPT_DIR/stderr.log" &
CBMEM_PID=$!

sleep 2

if kill -0 "$CBMEM_PID" >/dev/null 2>&1; then
    echo "cbmem-team started successfully (PID $CBMEM_PID)"
else
    echo "cbmem-team failed to start. Run manually for details:"
    echo "  \"$CBMEM_BIN\" ${args[*]}"
    echo "See $SCRIPT_DIR/stderr.log"
    exit 1
fi
