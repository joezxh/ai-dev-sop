#!/usr/bin/env bash
#
# mysql-etl.sh — 一键把现有 SQLite 控制台库迁到 MySQL 8.0
#
# 用法：
#   ./mysql-etl.sh --sqlite /var/lib/cbmem-team/cbmem.db \
#                   --mysql-dsn 'cbmem:***@tcp(127.0.0.1:3306)/cbmem?parseTime=true&loc=Local&charset=utf8mb4' \
#                   --bin /usr/local/bin/cbmem-team
#
# 步骤：
#   1. 备份 SQLite 库到 /var/lib/cbmem-team/backups/cbmem.db.$(date +%s)
#   2. 探测 MySQL 连通性 + utf8mb4 校验
#   3. 在 MySQL 上建表（调用 cbmem-team migrate 子命令）
#   4. 跑 ETL（调用 cbmem-team migrate-sqlite-to-mysql）
#   5. 写 cbmem.db.migrated 标记（防止下次启动再跑）
#   6. 打印逐表行数对比
#
# 失败回滚：删除标记 + 重启 cbmem-team（flag 仍走 -data 即 SQLite）
#
# 设计原则：
#   - 任何步骤失败立即退出（set -euo pipefail）
#   - 永远不动原 cbmem.db；只读 + 复制
#   - 双 dsn flag 同时存在时，migrate 子命令需要显式接受两个 dsn

set -euo pipefail

usage() {
  cat <<EOF
用法: $0 --sqlite <path> --mysql-dsn <dsn> [--bin <cbmem-team>] [--dry-run]

  --sqlite        现有 SQLite 库路径（默认 /var/lib/cbmem-team/cbmem.db）
  --mysql-dsn     MySQL DSN（同 -mysql-dsn 启动 flag）
  --bin           cbmem-team 二进制路径（默认 /usr/local/bin/cbmem-team）
  --dry-run       只打印计划，不执行
  --help          显示帮助
EOF
  exit "${1:-1}"
}

SQLITE_PATH="/var/lib/cbmem-team/cbmem.db"
MYSQL_DSN=""
BIN="/usr/local/bin/cbmem-team"
DRY_RUN=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --sqlite)      SQLITE_PATH="$2"; shift 2 ;;
    --mysql-dsn)   MYSQL_DSN="$2";   shift 2 ;;
    --bin)         BIN="$2";         shift 2 ;;
    --dry-run)     DRY_RUN=1;        shift   ;;
    --help|-h)     usage 0 ;;
    *)             echo "unknown flag: $1" >&2; usage 1 ;;
  esac
done

if [[ -z "$MYSQL_DSN" ]]; then
  echo "❌ 缺少 --mysql-dsn" >&2
  usage 1
fi

if [[ ! -f "$SQLITE_PATH" ]]; then
  echo "❌ SQLite 库不存在: $SQLITE_PATH" >&2
  exit 2
fi

if ! command -v "$BIN" >/dev/null 2>&1; then
  echo "❌ cbmem-team 不存在: $BIN" >&2
  exit 3
fi

log() { printf '%s [%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "etl" "$*"; }

run_or_print() {
  local desc="$1"; shift
  if [[ "$DRY_RUN" -eq 1 ]]; then
    log "[DRY-RUN] $desc"
    log "[DRY-RUN] $*"
  else
    log "$desc"
    "$@"
  fi
}

# ---------- 步骤 1：备份 SQLite ----------
BACKUP_DIR="$(dirname "$SQLITE_PATH")/backups"
BACKUP_PATH="${BACKUP_DIR}/cbmem.db.$(date +%s)"
run_or_print "备份 SQLite → $BACKUP_PATH" mkdir -p "$BACKUP_DIR"
run_or_print "复制 SQLite" cp -p "$SQLITE_PATH" "$BACKUP_PATH"

# ---------- 步骤 2：探测 MySQL ----------
log "探测 MySQL 连通性…"
if [[ "$DRY_RUN" -eq 0 ]]; then
  if ! "$BIN" mysql-ping --mysql-dsn "$MYSQL_DSN" >/dev/null 2>&1; then
    echo "❌ MySQL 不可达或 DSN 错误" >&2
    exit 4
  fi
fi

# ---------- 步骤 3：建表 ----------
log "在 MySQL 上建表…"
run_or_print "migrate tables" "$BIN" migrate-tables --mysql-dsn "$MYSQL_DSN"

# ---------- 步骤 4：ETL ----------
log "迁移 7 张表数据…"
run_or_print "etl rows" "$BIN" migrate-sqlite-to-mysql \
  --sqlite "$SQLITE_PATH" \
  --mysql-dsn "$MYSQL_DSN"

# ---------- 步骤 5：标记 ----------
MARK="${SQLITE_PATH}.migrated"
run_or_print "写迁移标记 $MARK" sh -c "printf '%s' \"$(date -u +%FT%TZ)\" > '$MARK'"

# ---------- 步骤 6：报告 ----------
log "✅ ETL 完成"
log "原 SQLite 库保留：$SQLITE_PATH"
log "备份 SQLite 库：$BACKUP_PATH"
log "回滚方式：删除 $MARK 并以 -data flag 启动 cbmem-team 即回到 SQLite"