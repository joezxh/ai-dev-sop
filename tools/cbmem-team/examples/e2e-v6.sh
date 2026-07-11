#!/usr/bin/env bash
# examples/e2e-v6.sh — M2 验收脚本 shell 入口。
#
# 调用 `go run ./cmd/e2e-v6/`（在仓库 tools/cbmem-team/ 下）。
# 通过即返回 0。
#
# 用法：
#   ./examples/e2e-v6.sh
#   ADMIN_TOKEN=... JWT_SECRET=... ./examples/e2e-v6.sh
#
# 5+1 场景：
#   1. /v2/tools 返回 49 条 + 包含 A、B 轨道
#   2. 限流（qps=1）：5 个并发 → 至少 1 个 429
#   3. 新建 BP → 201 / version=1 / status=draft
#   4. 修订 BP（body 变化）→ version +1 + bp_versions 多一行
#   5. 关联图：≥1 tool + ≥1 hall 节点
#   6. UI: GET /ui/m2/ → 200 + 含 "cbmem-team · M2"
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "e2e-v6" "$*"; }

log "=== M2 e2e (v6) ==="
log "Building + running cmd/e2e-v6 ..."
go run ./cmd/e2e-v6/
log "✅ M2 e2e (v6) 全绿"