#!/bin/bash
#
# e2e-v5.sh — M0.5 MySQL 迁移验收脚本
#
# 跑通：
#   1. 起 SQLite cbmem-team（默认）
#   2. 健康检查 /admin/users 等基础 e2e
#   3. 停 SQLite 进程
#   4. 启动 MySQL cbmem-team（带 -mysql-dsn）
#   5. 健康检查（确认 MySQL 路径工作）
#   6. 调用 migrate 子命令幂等验证（再跑一次无副作用）
#
# 用法：
#   ./e2e-v5.sh [--sqlite-only] [--mysql-only] [--skip-serve]
#
# 默认全跑；任一阶段失败立刻 exit 1。

set -euo pipefail

SQLITE_ONLY=0
MYSQL_ONLY=0
SKIP_SERVE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --sqlite-only) SQLITE_ONLY=1; shift ;;
    --mysql-only)  MYSQL_ONLY=1;  shift ;;
    --skip-serve)  SKIP_SERVE=1;  shift ;;
    *) echo "unknown flag: $1" >&2; exit 2 ;;
  esac
done

BASE='http://localhost:8787'
H='X-Admin-Token: dev-admin-token-change-me-2026'
BIN="${BIN:-/usr/local/bin/cbmem-team}"
DATA_DIR="${DATA_DIR:-/var/lib/cbmem-team}"
MYSQL_DSN="${MYSQL_DSN:-cbmem:cbmem-dev-password@tcp(127.0.0.1:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4}"

log() { printf '%s [%s] %s\n' "$(date '+%H:%M:%S')" "e2e-v5" "$*"; }

# ---------- helpers ----------
expect_status() {
  local url="$1" want="$2" desc="$3"
  local got
  got=$(curl -sS -o /dev/null -w '%{http_code}' "$url" -H "$H")
  if [[ "$got" != "$want" ]]; then
    log "❌ $desc: $url expected $want got $got"
    exit 1
  fi
  log "✓ $desc ($got)"
}

run_serve() {
  local extra="$1"
  log "starting cbmem-team $extra"
  if [[ "$SKIP_SERVE" -eq 1 ]]; then
    log "[skip-serve] not starting, healthcheck only"
    return
  fi
  "$BIN" \
    -listen "$BASE" \
    -data "$DATA_DIR" \
    -admin-token dev-admin-token-change-me-2026 \
    -jwt-secret dev-secret-change-me-2026-cbmem-team-7f8a \
    $extra > /tmp/cbmem-e2e.log 2>&1 &
  echo $! > /tmp/cbmem-e2e.pid
  sleep 2
}

stop_serve() {
  if [[ "$SKIP_SERVE" -eq 1 ]]; then
    return
  fi
  if [[ -f /tmp/cbmem-e2e.pid ]]; then
    kill "$(cat /tmp/cbmem-e2e.pid)" 2>/dev/null || true
    rm -f /tmp/cbmem-e2e.pid
  fi
}

trap stop_serve EXIT

# ---------- 阶段 1：SQLite 路径 ----------
if [[ "$MYSQL_ONLY" -eq 0 ]]; then
  log "=== 阶段 1: SQLite 路径回归 ==="
  run_serve ""
  expect_status "$BASE/healthz" 200 "healthz (sqlite)"
  expect_status "$BASE/admin/users" 200 "admin users list (sqlite)"
  stop_serve
fi

# ---------- 阶段 2：MySQL 路径 ----------
if [[ "$SQLITE_ONLY" -eq 0 ]]; then
  log "=== 阶段 2: MySQL 路径 + migrate 验证 ==="

  # 2.1 先 ping
  log "mysql-ping..."
  "$BIN" mysql-ping -mysql-dsn "$MYSQL_DSN" || {
    log "❌ mysql-ping 失败：MySQL 不可达或 DSN 错误"
    exit 3
  }

  # 2.2 migrate tables（幂等：跑两次）
  log "migrate-tables 第一次..."
  "$BIN" migrate-tables -mysql-dsn "$MYSQL_DSN"
  log "migrate-tables 第二次（验证幂等）..."
  "$BIN" migrate-tables -mysql-dsn "$MYSQL_DSN"

  # 2.3 启动 + 健康检查
  run_serve "-mysql-dsn $MYSQL_DSN"
  expect_status "$BASE/healthz" 200 "healthz (mysql)"
  expect_status "$BASE/admin/users" 200 "admin users list (mysql)"

  log "=== 阶段 2 完成 ✓ ==="
  stop_serve
fi

log "✅ e2e-v5 全绿"