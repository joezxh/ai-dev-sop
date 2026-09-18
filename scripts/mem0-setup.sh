#!/usr/bin/env bash
# mem0 一键接入（macOS / Linux / WSL）
#   ./mem0-setup.sh [-i codebuddy,cursor,...] [-u url] [-k m0sk_...] [-r repo] [-n dryrun]
set -euo pipefail
SERVICE="mem0"; REST_PORT=8888; DASH_PORT=3001
IDE=""; URL="http://127.0.0.1:8080/mcp"; APIKEY="${MEM0_API_KEY:-}"; REPO="$(pwd)"; DRYRUN=0
while getopts "i:u:k:r:nh" o; do
  case "$o" in
    i) IDE="$OPTARG";; u) URL="$OPTARG";; k) APIKEY="$OPTARG";; r) REPO="$OPTARG";;
    n) DRYRUN=1;; h) echo "用法: $0 [-i ide列表] [-u mcp-url] [-k apikey] [-r repo] [-n]"; exit 0;;
  esac
done

ok()   { printf "  \033[32m[OK]\033[0m   %s\n" "$1"; }
err()  { printf "  \033[31m[FAIL]\033[0m %s\n" "$1"; }
info() { printf "  \033[33m[i]\033[0m    %s\n" "$1"; }
head() { echo; printf "\033[36m== %s ==\033[0m\n" "$1"; }
die()  { err "$1"; exit 1; }

# name:dir:mcp:kind:rule:prefix:agent
MATRIX=(
  "codebuddy:$HOME/.codebuddy:$HOME/.codebuddy/mcp.json:json-cb:CODEBUDDY.md:cb:CodeBuddy"
  "cursor:$HOME/.cursor:$HOME/.cursor/mcp.json:json-http:.cursor/rules/mem0.mdc:cu:Cursor"
  "qoder:$HOME/.qoder:$HOME/.qoder/mcp.json:json-http:.qoder/rules/mem0.md:qd:Qoder"
  "codex:$HOME/.codex:$HOME/.codex/config.toml:toml:AGENTS.md:cx:Codex"
  "claude:$HOME/.claude:$HOME/.claude.json:json-http:CLAUDE.md:cc:Claude Code"
)

command -v python3 >/dev/null 2>&1 || die "需要 python3（用于 JSON 合并）"
TPL="$(cd "$(dirname "$0")" && pwd)/templates/mem0-rules.md"
[ -f "$TPL" ] || die "规则模板缺失: $TPL"

# 选择 IDE
SELECTED=""
if [ -n "$IDE" ]; then
  IFS=',' read -ra want <<< "$IDE"
  for x in "${want[@]}"; do
    x="$(echo "$x" | tr -d ' ')"
    found=""
    for row in "${MATRIX[@]}"; do [ "${row%%:*}" = "$x" ] && found="$row"; done
    [ -z "$found" ] && die "未知 IDE: $x（支持 codebuddy,cursor,qoder,codex,claude）"
    SELECTED="${SELECTED}${found}"$'\n'
  done
else
  for row in "${MATRIX[@]}"; do
    d="$(echo "$row" | cut -d: -f2)"
    [ -d "$d" ] && SELECTED="${SELECTED}${row}"$'\n'
  done
  [ -z "$SELECTED" ] && die "未检测到已安装 IDE；装在非默认路径时用 -i 指定"
fi

# 密钥三层回退
if [ -z "$APIKEY" ]; then
  DASHURL="$(echo "$URL" | sed -E "s|:[0-9]+/|:$DASH_PORT/|; s|/mcp\$|/dashboard/api-keys|")"
  info "未提供密钥。打开: $DASHURL 创建后粘贴"
  (open "$DASHURL" 2>/dev/null || xdg-open "$DASHURL" 2>/dev/null) || true
  read -r -p "粘贴 m0sk_ 密钥: " APIKEY
  APIKEY="$(echo "$APIKEY" | tr -d '[:space:]')"
  [[ "$APIKEY" =~ ^m0sk_.{10,} ]] || die "密钥格式不合法（应为 m0sk_ 开头）"
fi

REST_URL="$(echo "$URL" | sed -E "s|:[0-9]+/|:$REST_PORT/|; s|/mcp\$||")"
GIT_REMOTE="$(git -C "$REPO" remote get-url origin 2>/dev/null || true)"
if [ -z "$GIT_REMOTE" ]; then
  GIT_REMOTE="$(basename "$REPO" | tr ' ' '-')"
  info "git remote 不存在，git_remote 派生自目录名: $GIT_REMOTE"
fi
PROJECT_ID="$(basename "$GIT_REMOTE" .git)"
CRED="$REPO/.mem0/mem0.config.json"

head "写入配置$( [ "$DRYRUN" = 1 ] && echo ' (DryRun)')"
while IFS= read -r row; do
  [ -z "$row" ] && continue
  IFS=: read -r name dir mcp kind rule prefix agent <<< "$row"

  # --- MCP 配置 ---
  if [ "$kind" = "toml" ]; then
    SECTION="[mcp_servers.$SERVICE]
url = \"$URL\"
"
    if [ "$DRYRUN" = 1 ]; then info "[DryRun] TOML: $mcp"
    else
      mkdir -p "$(dirname "$mcp")"
      if [ -f "$mcp" ] && grep -q "^\[mcp_servers\.$SERVICE\]" "$mcp"; then
        python3 - "$mcp" "$SECTION" <<'PYEOF'
import re, sys
path, sec = sys.argv[1], sys.argv[2]
txt = open(path, encoding='utf-8').read()
txt = re.sub(r'(?ms)^\[mcp_servers\.mem0\]\n.*?(?=^\[|\Z)', sec, txt)
open(path, 'w', encoding='utf-8', newline='').write(txt)
PYEOF
      else
        printf "%s" "$SECTION" >> "$mcp"
      fi
      ok "MCP 配置(TOML): $mcp"
    fi
  else
    if [ "$DRYRUN" = 1 ]; then info "[DryRun] JSON: $mcp"
    else
      mkdir -p "$(dirname "$mcp")"
      python3 - "$mcp" "$URL" "$kind" <<'PYEOF'
import json, sys
path, url, kind = sys.argv[1], sys.argv[2], sys.argv[3]
try:
    with open(path, encoding='utf-8') as f:
        cfg = json.load(f)
except Exception:
    cfg = {}
cfg.setdefault('mcpServers', {})
if kind == 'json-cb':
    cfg['mcpServers']['mem0'] = {"url": url, "transport": "streamable-http", "disabled": False}
else:
    cfg['mcpServers']['mem0'] = {"type": "http", "url": url, "transport": "streamable-http"}
with open(path, 'w', encoding='utf-8') as f:
    json.dump(cfg, f, indent=2, ensure_ascii=False)
PYEOF
      ok "MCP 配置: $mcp"
    fi
  fi

  # --- 规则文件（模板可能带 CRLF，统一 strip \r 后再替换占位符）---
  RULEPATH="$REPO/$rule"
  BODY="$(python3 - "$TPL" "$prefix" "$agent" <<'PYEOF'
import sys
tpl, prefix, agent = sys.argv[1], sys.argv[2], sys.argv[3]
txt = open(tpl, encoding='utf-8').read().replace('\r\n', '\n').replace('\r', '\n')
for old, new in (
    ('{{CRED_FILE}}', '.mem0/mem0.config.json'),
    ('{{SESSION_PREFIX}}', prefix),
    ('{{SESSION_FILE}}', '.mem0/.session_id-' + prefix),
    ('{{AGENT}}', agent),
):
    txt = txt.replace(old, new)
sys.stdout.write(txt)
PYEOF
)"
  if [ "$DRYRUN" = 1 ]; then info "[DryRun] 规则文件: $RULEPATH"
  else
    mkdir -p "$(dirname "$RULEPATH")"
    if [ -f "$RULEPATH" ] && grep -q "mem0:rules:begin" "$RULEPATH"; then
      python3 - "$RULEPATH" "$BODY" <<'PYEOF'
import re, sys
path, body = sys.argv[1], sys.argv[2]
txt = open(path, encoding='utf-8').read()
txt = re.sub(r'(?s)<!-- mem0:rules:begin -->.*?<!-- mem0:rules:end -->', body.strip(), txt)
open(path, 'w', encoding='utf-8', newline='').write(txt)
PYEOF
      ok "规则文件(更新标记区): $RULEPATH"
    elif [ -f "$RULEPATH" ]; then
      printf "\n\n%s\n" "$BODY" >> "$RULEPATH"
      ok "规则文件(追加): $RULEPATH"
    else
      if [[ "$rule" == *.mdc ]]; then
        printf -- "---\ndescription: mem0 auto memory\nglobs:\nalwaysApply: true\n---\n\n%s\n" "$BODY" > "$RULEPATH"
      else
        printf "%s\n" "$BODY" > "$RULEPATH"
      fi
      ok "规则文件(新建): $RULEPATH"
    fi
  fi
done <<< "$SELECTED"

# --- 凭证文件 ---
if [ "$DRYRUN" = 1 ]; then info "[DryRun] 凭证文件: $CRED"
else
  mkdir -p "$REPO/.mem0"
  python3 - "$CRED" "$APIKEY" "$GIT_REMOTE" "$PROJECT_ID" <<'PYEOF'
import json, sys, os
path, key, gr, pid = sys.argv[1:5]
cfg = {}
if os.path.exists(path):
    try:
        cfg = json.load(open(path, encoding='utf-8'))
    except Exception:
        cfg = {}
cfg.update({
    "api_key": key,
    "git_remote": gr,
    "project_id": pid,
    "admin_user_id": "admin@mem0.dev",
    "metadata_defaults": {"source": "setup", "type_enum": ["fact", "decision", "preference", "note"]},
    "load_on_session_start": True,
    "save_policy": "auto-on-durable-fact",
})
with open(path, 'w', encoding='utf-8') as f:
    json.dump(cfg, f, indent=2, ensure_ascii=False)
PYEOF
  ok "凭证文件: $CRED"
fi

# --- 端到端校验 ---
head "端到端校验"
CODE="$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$URL" || true)"
[ -z "$CODE" ] && CODE="000"
if [ "$CODE" = "406" ]; then ok "MCP 端点存活 ($URL, HTTP 406)"; else err "MCP 端点异常（HTTP $CODE）→ 服务未启动？"; fi
if [ "$DRYRUN" = 1 ]; then
  info "[DryRun] 跳过 REST 写读往返（DryRun 不落盘、不写入记忆）"
else
  RESP="$(curl -s --max-time 30 -X POST "$REST_URL/memories" \
    -H "X-API-Key: $APIKEY" -H "Content-Type: application/json" \
    -d "{\"text\":\"mem0-setup validation $(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"infer\":false}" || echo '{"error":true}')"
  if echo "$RESP" | grep -q '"error"[[:space:]]*:[[:space:]]*true'; then
    err "REST 写入失败 → 密钥无效或 8888 不可达。到 Dashboard 重建密钥后重跑。"
  else
    ok "REST 写入成功 ($REST_URL/memories)"
    if curl -s --max-time 30 "$REST_URL/memories?limit=5" -H "X-API-Key: $APIKEY" | grep -q "mem0-setup validation"; then
      ok "REST 查回成功——密钥有效、链路闭环"
    else
      err "写入后未查回 → 检查 REST 端口/防火墙"
    fi
  fi
fi

head "完成"
echo "下一步：重启 IDE → MCP 面板中 mem0 应为绿色 → 让 Agent 记住一条再问回读以验证。"
