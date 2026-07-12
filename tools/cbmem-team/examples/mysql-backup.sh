#!/usr/bin/env bash
#
# mysql-backup.sh — 每日 cbmem-team MySQL 库备份 + binlog 保留
#
# 用法：
#   sudo crontab -e
#   0 3 * * * /opt/cbmem-team/scripts/mysql-backup.sh
#
# 输出：
#   /var/backups/cbmem-mysql/cbmem_YYYYMMDD_HHMMSS.sql.gz
#   保留 14 天本地；7 天 binlog（mysqld 服务端配置控制）
#
# 还原：
#   gunzip -c cbmem_*.sql.gz | mysql -h<host> -u<user> -p<pw> cbmem

set -euo pipefail

BACKUP_DIR="/var/backups/cbmem-mysql"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-cbmem}"
MYSQL_PASS="${MYSQL_PASS:-}" # 推荐使用 ~/.my.cnf 配 [client] password
RETENTION_DAYS="${RETENTION_DAYS:-14}"

if ! command -v mysqldump >/dev/null 2>&1; then
  echo "❌ mysqldump not found" >&2
  exit 2
fi

mkdir -p "$BACKUP_DIR"

TS=$(date +%Y%m%d_%H%M%S)
OUT="${BACKUP_DIR}/cbmem_${TS}.sql.gz"

echo "[$(date '+%F %T')] backup → $OUT"
mysqldump \
  -h "$MYSQL_HOST" \
  -P "$MYSQL_PORT" \
  -u "$MYSQL_USER" \
  ${MYSQL_PASS:+-p"$MYSQL_PASS"} \
  --single-transaction --quick --routines --triggers --events \
  --hex-blob \
  cbmem | gzip -9 > "$OUT"

ls -lh "$OUT"

# 清理 14 天前
find "$BACKUP_DIR" -type f -name 'cbmem_*.sql.gz' -mtime +$RETENTION_DAYS -delete
echo "[$(date '+%F %T')] retention: deleted backups older than $RETENTION_DAYS days"