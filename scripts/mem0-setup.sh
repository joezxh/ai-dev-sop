#!/usr/bin/env bash
# mem0 一键接入（macOS / Linux / WSL）
#   ./mem0-setup.sh [-i codebuddy,cursor,...] [-u url] [-k m0sk_...] [-r repo] [-S user|repo] [-n dryrun]
set -euo pipefail
SERVICE="mem0"; REST_PORT=8002; DASH_PORT=3001
IDE=""; URL="http://127.0.0.1:8080/mcp"; APIKEY="${MEM0_API_KEY:-}"; REPO="$(pwd)"; SCOPE="user"; DRYRUN=0
while getopts "i:u:s:k:r:S:nh" o; do
  case "$o" in
    i) IDE="$OPTARG";; u) URL="$OPTARG";; s) REST_URL_OPT="$OPTARG";; k) APIKEY="$OPTARG";; r) REPO="$OPTARG";; S) SCOPE="$OPTARG";;
    n) DRYRUN=1;; h) echo "用法: $0 [-i ide列表] [-u mcp-url] [-k apikey] [-r repo] [-S user|repo] [-n]"; exit 0;;
  esac
done
case "$SCOPE" in user|repo) ;; *) echo "  [FAIL] -S 仅支持 user|repo"; exit 1;; esac
# UI 型用户规则的 IDE：文件写入后可能需在设置页 Rules 手动粘贴
UI_RULE_AGENTS="codebuddy cursor qoder"

ok()   { printf "  \033[32m[OK]\033[0m   %s\n" "$1"; }
err()  { printf "  \033[31m[FAIL]\033[0m %s\n" "$1"; }
info() { printf "  \033[33m[i]\033[0m    %s\n" "$1"; }
head() { echo; printf "\033[36m== %s ==\033[0m\n" "$1"; }
die()  { err "$1"; exit 1; }

# name:dir:mcp:kind:rule:userrule:prefix:agent
MATRIX=(
  "codebuddy:$HOME/.codebuddy:$HOME/.codebuddy/mcp.json:json-cb:CODEBUDDY.md:$HOME/.codebuddy/CODEBUDDY.md:cb:CodeBuddy"
  "cursor:$HOME/.cursor:$HOME/.cursor/mcp.json:json-http:.cursor/rules/mem0.mdc:$HOME/.cursor/rules/mem0.mdc:cu:Cursor"
  "qoder:$HOME/.qoder:$HOME/.qoder/mcp.json:json-http:.qoder/rules/mem0.md:$HOME/.qoder/rules/mem0.md:qd:Qoder"
  "codex:$HOME/.codex:$HOME/.codex/config.toml:toml:AGENTS.md:$HOME/.codex/AGENTS.md:cx:Codex"
  "claude:$HOME/.claude:$HOME/.claude.json:json-http:CLAUDE.md:$HOME/.claude/CLAUDE.md:cc:Claude Code"
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

if [ -n "${REST_URL_OPT:-}" ]; then
  REST_URL="${REST_URL_OPT%/}"
else
  REST_URL="$(echo "$URL" | sed -E "s|:[0-9]+/|:$REST_PORT/|; s|/mcp\$||")"
fi
if [ "$SCOPE" = "user" ]; then
  # user 级凭证不绑定单一仓库：Agent 运行时按当前项目现取 git remote
  GIT_REMOTE=""
  PROJECT_ID=""
  CRED="$HOME/.mem0/mem0.config.json"
  info "工程级 .mem0/mem0.config.json（若存在）优先于用户级凭证文件；各项目记忆池按 git_remote 自然隔离"
else
  GIT_REMOTE="$(git -C "$REPO" remote get-url origin 2>/dev/null || true)"
  if [ -z "$GIT_REMOTE" ]; then
    GIT_REMOTE="$(basename "$REPO" | tr ' ' '-')"
    info "git remote 不存在，git_remote 派生自目录名: $GIT_REMOTE"
  fi
  PROJECT_ID="$(basename "$GIT_REMOTE" .git)"
  CRED="$REPO/.mem0/mem0.config.json"
fi

head "写入配置（$SCOPE 级）$( [ "$DRYRUN" = 1 ] && echo '(DryRun)')"
while IFS= read -r row; do
  [ -z "$row" ] && continue
  IFS=: read -r name dir mcp kind rule userrule prefix agent <<< "$row"

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
  if [ "$SCOPE" = "user" ]; then
    RULEPATH="$userrule"
    CRED_REF="$HOME/.mem0/mem0.config.json"
    SESS_REF="$HOME/.mem0/.session_id-"
  else
    RULEPATH="$REPO/$rule"
    CRED_REF="$REPO/.mem0/mem0.config.json"
    SESS_REF="$REPO/.mem0/.session_id-"
  fi
  case " $UI_RULE_AGENTS " in *" $name "*)
    info "user 级规则：若 $agent 未自动加载该文件，请在设置页 Rules 中粘贴其内容";; esac
  BODY="$(python3 - "$TPL" "$prefix" "$agent" "$CRED_REF" "$SESS_REF$prefix" <<'PYEOF'
import sys
tpl, prefix, agent, cred_ref, sess_ref = sys.argv[1:6]
txt = open(tpl, encoding='utf-8').read().replace('\r\n', '\n').replace('\r', '\n')
for old, new in (
    ('{{CRED_FILE}}', cred_ref),
    ('{{SESSION_PREFIX}}', prefix),
    ('{{SESSION_FILE}}', sess_ref),
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
    "admin_user_id": "admin@mem0.dev",
    "metadata_defaults": {"source": "setup", "type_enum": ["fact", "decision", "preference", "note"]},
    "load_on_session_start": True,
    "save_policy": "auto-on-durable-fact",
})
if gr:
    cfg["git_remote"] = gr
    cfg["project_id"] = pid
else:
    cfg["git_remote_mode"] = "runtime"
    cfg["project_id_mode"] = "runtime"
with open(path, 'w', encoding='utf-8') as f:
    json.dump(cfg, f, indent=2, ensure_ascii=False)
PYEOF
  ok "凭证文件: $CRED"
fi

# --- 端到端校验 ---
head "端到端校验"
CODE="$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$URL" || true)"
[ -z "$CODE" ] && CODE="000"
if [ "$CODE" = "406" ]; then
  ok "MCP 端点存活 ($URL, HTTP 406)"
elif [ "$CODE" = "000" ]; then
  err "MCP 端点不可达（无 HTTP 响应）→ 检查主机名/端口是否正确、服务是否启动：$URL"
else
  err "MCP 端点异常（HTTP $CODE）→ 服务未启动？"
fi
if [ "$DRYRUN" = 1 ]; then
  info "[DryRun] 跳过 REST 写读往返（DryRun 不落盘、不写入记忆）"
else
  RESPFILE="$(mktemp)"
  STATUS="$(curl -s -o "$RESPFILE" -w "%{http_code}" --max-time 30 -X POST "$REST_URL/memories" \
    -H "X-API-Key: $APIKEY" -H "Content-Type: application/json" \
    -d "{\"messages\":[{\"role\":\"user\",\"content\":\"mem0-setup validation $(date -u +%Y-%m-%dT%H:%M:%SZ)\"}],\"infer\":false,\"user_id\":\"admin@mem0.dev\"}" || echo 000)"
  case "$STATUS" in
    200|201) ;;
    *)
      err "REST 写入失败（HTTP $STATUS）→ 密钥无效或 $REST_URL 不可达。到 Dashboard 重建密钥后重跑（Docker 部署若 REST≠8002，用 -s/--RestUrl 指定实际端口）。"
      [ -s "$RESPFILE" ] && head -c 200 "$RESPFILE" && echo
      rm -f "$RESPFILE"; exit 0;;
  esac
  rm -f "$RESPFILE"
  if true; then
    ok "REST 写入成功 ($REST_URL/memories)"
    if curl -s --max-time 30 "$REST_URL/memories?limit=5" -H "X-API-Key: $APIKEY" | grep -q "mem0-setup validation"; then
      ok "REST 查回成功——密钥有效、链路闭环"
    else
      err "写入后未查回 → 检查 REST 端口/防火墙"
    fi
  fi
fi

# --- 转录直投 Hook（零 LLM 逐字留痕，仅 codebuddy）---
if printf '%s' "$SELECTED" | grep -q '^codebuddy:'; then
  head "转录直投 Hook（零 LLM 逐字留痕，仅 codebuddy）"
  POSTER_SRC="$(cd "$(dirname "$0")" && pwd)/mem0-transcript-poster.mjs"
  if [ ! -f "$POSTER_SRC" ]; then
    err "poster 脚本缺失: $POSTER_SRC"
  elif ! NODE_BIN="$(command -v node || true)" || [ -z "$NODE_BIN" ]; then
    err "未找到 node（Hook 需要 Node.js ≥ 18）——跳过 Hook 安装，可手动配置（见 scripts/README-mem0-poster.md）"
  else
    HOOK_DIR="$HOME/.codebuddy/hooks"
    DST="$HOOK_DIR/mem0-transcript-poster.mjs"
    if [ "$DRYRUN" = 1 ]; then
      info "[DryRun] 将复制 $POSTER_SRC -> $DST"
    else
      mkdir -p "$HOOK_DIR"
      cp "$POSTER_SRC" "$DST"
      ok "poster 已复制 -> $DST"
    fi
    SETTINGS_JSON="$HOME/.codebuddy/settings.json"
    if [ "$DRYRUN" = 1 ]; then
      info "[DryRun] 将合并 SessionStart hook 到 $SETTINGS_JSON"
    else
      mkdir -p "$HOME/.codebuddy"
      python3 - "$SETTINGS_JSON" "$NODE_BIN" "$DST" "$REST_URL" "$REPO" <<'PYEOF'
import json, sys, os
path, node, dst, rest, repo = sys.argv[1:6]
cfg = {}
if os.path.exists(path):
    try:
        cfg = json.load(open(path, encoding='utf-8'))
    except Exception:
        cfg = {}
hooks = cfg.setdefault('hooks', {})
cmd = f'"{node}" "{dst}" --workspace "{repo}" --rest-url "{rest}"'

def _hook_cmds(entry):
    h = entry.get('hooks') if isinstance(entry, dict) else None
    if isinstance(h, list):
        return h
    if isinstance(h, dict):
        return [h]
    return []

# 同时注册 SessionStart（flush 上一会话，兜底）与 SessionEnd（flush 当前会话）
changed = False
for ev in ('SessionStart', 'SessionEnd'):
    entries = hooks.setdefault(ev, [])
    found = False
    for e in entries:
        for h in _hook_cmds(e):
            if isinstance(h, dict) and 'mem0-transcript-poster' in h.get('command', ''):
                found = True
                break
        if found:
            break
    if not found:
        entries.append({"hooks": [{"type": "command", "command": cmd, "timeout": 60}]})
        changed = True
if changed:
    with open(path, 'w', encoding='utf-8') as f:
        json.dump(cfg, f, indent=2, ensure_ascii=False)
    print('  [OK]   settings.json (转录直投 hook: SessionStart+SessionEnd)')
else:
    print('  [i]    转录直投 hook 已存在（幂等跳过）')
PYEOF
      info "hook 依赖 IDE 在工程 cwd 下执行 SessionStart；poster 按 cwd 找 <cwd>/.mem0/mem0.config.json，找不到则静默退出"
    fi
  fi
fi

# --- 工具层捕获 Hook（零 LLM，仅 codebuddy）---
# 背景：CodeBuddy IDE 对话转录实际落盘本地
#   %LOCALAPPDATA%/CodeBuddyExtension/Data/<userId>/CodeBuddyIDE/<userId>/history/<workspaceHash>/<sessionId>/messages/<msgId>.json
# 工具层捕获是 IDE 会话的另一种逐字留痕通道（PostToolUse 原样传出），与下方的会话转录直投互为补充。
if printf '%s' "$SELECTED" | grep -q '^codebuddy:'; then
  head "工具层捕获 Hook（零 LLM，仅 codebuddy）"
  CAPTURE_SRC="$(cd "$(dirname "$0")" && pwd)/mem0-tool-capture-hook.mjs"
  if [ ! -f "$CAPTURE_SRC" ]; then
    err "capture 脚本缺失: $CAPTURE_SRC"
  elif ! NODE_BIN="$(command -v node || true)" || [ -z "$NODE_BIN" ]; then
    err "未找到 node——跳过工具捕获 Hook 安装"
  else
    DST2="$HOME/.codebuddy/hooks/mem0-tool-capture-hook.mjs"
    if [ "$DRYRUN" = 1 ]; then
      info "[DryRun] 将复制 $CAPTURE_SRC -> $DST2"
    else
      mkdir -p "$HOME/.codebuddy/hooks"
      cp "$CAPTURE_SRC" "$DST2"
      ok "capture 已复制 -> $DST2"
    fi
    if [ "$DRYRUN" = 1 ]; then
      info "[DryRun] 将合并 PostToolUse 工具捕获 hook 到 ~/.codebuddy/settings.json"
    else
      python3 - "$HOME/.codebuddy/settings.json" "$NODE_BIN" "$DST2" <<'PYEOF'
import json, sys, os
path, node, dst = sys.argv[1:4]
cfg = {}
if os.path.exists(path):
    try:
        cfg = json.load(open(path, encoding='utf-8'))
    except Exception:
        cfg = {}
hooks = cfg.setdefault('hooks', {})
ptu = hooks.setdefault('PostToolUse', [])
cmd = f'"{node}" "{dst}"'
for entry in ptu:
    for h in entry.get('hooks', []):
        if 'mem0-tool-capture-hook' in h.get('command', ''):
            print('  [i]    PostToolUse 工具捕获 hook 已存在（幂等跳过）')
            sys.exit(0)
ptu.append({"matcher": ".*", "hooks": [{"type": "command", "command": cmd, "timeout": 10}]})
with open(path, 'w', encoding='utf-8') as f:
    json.dump(cfg, f, indent=2, ensure_ascii=False)
print('  [OK]   settings.json (PostToolUse 工具捕获 hook)')
PYEOF
      info "工具事件将落盘到 <工程>/.mem0/tool-events.jsonl（零 LLM 原样记录）"
    fi
  fi
fi

# --- IDE 会话转录直投 Hook（零 LLM 逐字留痕，仅 codebuddy）---
# 数据源（落盘事实）：%LOCALAPPDATA%/CodeBuddyExtension/Data/<userId>/CodeBuddyIDE/<userId>/
#   history/<workspaceHash>/<sessionId>/messages/<msgId>.json
if printf '%s' "$SELECTED" | grep -q '^codebuddy:'; then
  head "IDE 会话转录直投 Hook（零 LLM 逐字留痕，仅 codebuddy）"
  IDE_POSTER_SRC="$(cd "$(dirname "$0")" && pwd)/mem0-ide-session-poster.mjs"
  if [ ! -f "$IDE_POSTER_SRC" ]; then
    err "poster 脚本缺失: $IDE_POSTER_SRC"
  elif ! NODE_BIN="$(command -v node || true)" || [ -z "$NODE_BIN" ]; then
    err "未找到 node（Hook 需要 Node.js ≥ 18）——跳过 Hook 安装，可手动配置（见 scripts/README-mem0-poster.md）"
  else
    IDE_DST="$HOME/.codebuddy/hooks/mem0-ide-session-poster.mjs"
    if [ "$DRYRUN" = 1 ]; then
      info "[DryRun] 将复制 $IDE_POSTER_SRC -> $IDE_DST"
    else
      mkdir -p "$HOME/.codebuddy/hooks"
      cp "$IDE_POSTER_SRC" "$IDE_DST"
      ok "IDE poster 已复制 -> $IDE_DST"
    fi
    SETTINGS_JSON="$HOME/.codebuddy/settings.json"
    if [ "$DRYRUN" = 1 ]; then
      info "[DryRun] 将合并 SessionStart hook 到 $SETTINGS_JSON"
    else
      mkdir -p "$HOME/.codebuddy"
      # poster 用 --workspace 指向本工程（读取 <REPO>/.mem0/mem0.config.json 取 api_key/git_remote），rest-url 显式指定
      python3 - "$SETTINGS_JSON" "$NODE_BIN" "$IDE_DST" "$REST_URL" "$REPO" <<'PYEOF'
import json, sys, os
path, node, dst, rest, repo = sys.argv[1:6]
cfg = {}
if os.path.exists(path):
    try:
        cfg = json.load(open(path, encoding='utf-8'))
    except Exception:
        cfg = {}
hooks = cfg.setdefault('hooks', {})
cmd = f'"{node}" "{dst}" --workspace "{repo}" --rest-url "{rest}"'

def _hook_cmds(entry):
    h = entry.get('hooks') if isinstance(entry, dict) else None
    if isinstance(h, list):
        return h
    if isinstance(h, dict):
        return [h]
    return []

# 同时注册 SessionStart（flush 上一会话，兜底）与 SessionEnd（flush 当前会话）
changed = False
for ev in ('SessionStart', 'SessionEnd'):
    entries = hooks.setdefault(ev, [])
    found = False
    for e in entries:
        for h in _hook_cmds(e):
            if isinstance(h, dict) and 'mem0-ide-session-poster' in h.get('command', ''):
                found = True
                break
        if found:
            break
    if not found:
        entries.append({"hooks": [{"type": "command", "command": cmd, "timeout": 60}]})
        changed = True
if changed:
    with open(path, 'w', encoding='utf-8') as f:
        json.dump(cfg, f, indent=2, ensure_ascii=False)
    print('  [OK]   settings.json (IDE 会话转录直投 hook: SessionStart+SessionEnd)')
else:
    print('  [i]    IDE 会话转录直投 hook 已存在（幂等跳过）')
PYEOF
      info "hook 在 SessionStart 时 flush 上一会话 IDE 本地落盘的会话转录（history/<ws>/<sessionId>/messages）到 mem0（零 LLM 逐字留痕）"
    fi
  fi
fi

head "完成"
echo "下一步：重启 IDE → MCP 面板中 mem0 应为绿色 → 让 Agent 记住一条再问回读以验证。"
