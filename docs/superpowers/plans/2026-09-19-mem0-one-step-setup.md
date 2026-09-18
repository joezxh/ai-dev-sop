# mem0 一键接入（mem0-setup）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 一条命令完成 mem0 接入（IDE MCP 配置 + 规则文件 + 凭证文件 + 端到端校验），Windows 用 `mem0-setup.ps1`、macOS/Linux 用 `mem0-setup.sh`；并修复 `mem0-manual.md` 的服务名/session_id/凭证路径三类一致性缺陷。

**Architecture:** MCP 服务名约定为 `mem0`（消除"两处名称对齐"错误源）；规则模板抽为共享文件 `scripts/templates/mem0-rules.md`（含占位符），两平台脚本读取同一模板按 IDE 替换后写入；所有写入幂等（JSON 合并 / TOML section 替换 / 规则文件标记区间）；校验走 REST（`:8888/memories`，X-API-Key 头）做密钥写读往返。

**Tech Stack:** PowerShell 5.1+（Windows）、bash 3.2+ + python3（macOS/Linux/WSL，JSON 合并）。

**Spec:** `docs/superpowers/specs/2026-09-19-mem0-one-step-setup-design.md`

---

## 文件结构

| 文件 | 职责 | 动作 |
|------|------|------|
| `scripts/templates/mem0-rules.md` | 规则模板唯一事实源（含 `{{占位符}}`），两脚本共用 | 新建 |
| `scripts/mem0-setup.ps1` | Windows 一键接入 | 新建 |
| `scripts/mem0-setup.sh` | macOS/Linux/WSL 一键接入 | 新建 |
| `docs/quick-ref/mem0-manual.md` | F1 服务名 / F2 session_id / F3 凭证路径 / F4 快捷方式 / F5 清单 | 修改 |
| `CODEBUDDY.md` | 凭证路径→`.mem0/`、服务名→约定名 `mem0` | 修改 |
| `.gitignore` | 追加 `.mem0/` 忽略项 | 修改 |
| `scripts/install-all.ps1`、`install-all.sh` | 末尾加 mem0-setup 指引（不改逻辑） | 修改 |
| `.codebuddy/mem0.config.json` → `.mem0/mem0.config.json` | 本仓库存量迁移 | 移动 |

---

## Task 1: 规则模板（scripts/templates/mem0-rules.md）

**Files:**
- Create: `scripts/templates/mem0-rules.md`

- [ ] **Step 1: 创建模板文件（完整内容如下）**

````markdown
<!-- mem0:rules:begin -->

## mem0 项目记忆（自动加载与留痕）

本项目使用自托管 **mem0** 服务作为长期记忆，通过当前 IDE 的 MCP 服务 `mem0` 访问
（服务名约定为 `mem0`；实际以 IDE 配置文件 `mcp.json`/`config.toml` 中的条目为准）。
凭证文件：`{{CRED_FILE}}`（git-ignored，含 api_key / git_remote / project_id）。

### 1. 加载记忆（会话开始时主动执行，不等用户开口）

1. 从 `{{CRED_FILE}}` 读取 `api_key`、`git_remote`、`project_id`。
2. `get_memories(api_key=..., git_remote=...)` 拉取项目共享池（跨用户），给用户 2–3 行召回摘要。
3. 配置缺失或 MCP 不可达 → 告知用户并停止，**不得臆造记忆**。

### 2. 保存记忆（出现持久事实时：决策 / 偏好 / 约束 / 人员）

```python
add_memory(
    text="<事实的自然语言描述>",
    api_key="<来自凭证文件>",
    git_remote="<来自凭证文件>",
    project_id="<来自凭证文件>",
    metadata={"type": "fact|decision|preference|note", "created_at": "<ISO8601>"},
)
```

- 始终传 `git_remote`（服务端解析为 project_id 并写入共享池）。
- 相关事实合并成一条；mem0 按内容哈希去重，重提安全。

### 3. 会话留痕（每轮回复完成时固定触发，寒暄也入库）

- **session_id**：格式 `{{SESSION_PREFIX}}-<YYYYMMDD>-<6位hex>`，持久化到 `{{SESSION_FILE}}`
  （git-ignored）。已存在**且日期为当天**→复用；**日期与当天不一致或用户明确开启新会话
  → 必须重新生成并覆写**。
- 每轮提交：

```python
add_memory(
    text="Q: <用户本轮原话，逐字>\n\nA: <完整执行转录>",
    api_key="<来自凭证文件>",
    git_remote="<来自凭证文件>",
    project_id="<来自凭证文件>",
    metadata={
        "type": "conversation",
        "session_id": "{{SESSION_PREFIX}}-<YYYYMMDD>-<6hex>",
        "agent": "{{AGENT}}",
        "role": "turn",
        "turn_seq": "<会话内从 1 递增>",
        "created_at": "<ISO8601>",
    },
)
```

- **A 部分五层，与 IDE 界面全选复制得到的 Markdown 逐字节一致**：
  ① 过程叙述行（界面可见的全部过渡叙述，逐字按序）
  ② 最终回复原文（完整 Markdown，含标题/表格/代码块）
  ③ Deep Thinking 全文（不得压缩改写）
  ④ Tool 调用逐条（工具名 + 关键参数 + 执行结果，失败也记录）
  ⑤ 完成结果（状态 + 交付物 + 验证结论）
  提交前自检：界面复制文本必须能在 ①② 中**原样找到**。

<!-- mem0:rules:end -->
````

- [ ] **Step 2: 校验占位符**

Run: `Select-String -Path scripts/templates/mem0-rules.md -Pattern '\{\{[^}]+\}\}' -AllMatches | ForEach-Object { $_.Matches.Value } | Sort-Object -Unique`
（必须带 `-AllMatches`：同一行出现多个占位符时不加会漏报。）
Expected: `{{AGENT}}` `{{CRED_FILE}}` `{{SESSION_FILE}}` `{{SESSION_PREFIX}}`

- [ ] **Step 3: Commit**

```bash
git add scripts/templates/mem0-rules.md
git commit -m "feat(mem0-setup): add shared mem0 rules template"
```

---

## Task 2: Windows 脚本（scripts/mem0-setup.ps1）

**Files:**
- Create: `scripts/mem0-setup.ps1`

- [ ] **Step 1: 写入完整脚本**

```powershell
#Requires -Version 5.1
<#
.SYNOPSIS  mem0 一键接入：IDE MCP 配置 + 规则文件 + 凭证文件 + 端到端校验。
.EXAMPLE  .\mem0-setup.ps1
.EXAMPLE  .\mem0-setup.ps1 -Url http://192.168.110.169:8080/mcp -ApiKey m0sk_xxx -DryRun
#>
param(
    [string]$Ide = "",
    [string]$Url = "http://127.0.0.1:8080/mcp",
    [string]$ApiKey = "",
    [string]$Repo = (Get-Location).Path,
    [switch]$DryRun,
    [switch]$Force
)
$ErrorActionPreference = "Stop"
$ServiceName = "mem0"
$RestPort = 8888
$DashPort = 3001
$Utf8 = [System.Text.UTF8Encoding]::new($false)

function Write-Ok($m)   { Write-Host "  [OK]   $m" -ForegroundColor Green }
function Write-Err($m)  { Write-Host "  [FAIL] $m" -ForegroundColor Red }
function Write-Info($m) { Write-Host "  [i]    $m" -ForegroundColor Yellow }
function Write-Head($m) { Write-Host ""; Write-Host "== $m ==" -ForegroundColor Cyan }

$Matrix = @{
    codebuddy = @{ Dir = "$env:USERPROFILE\.codebuddy"; Mcp = "$env:USERPROFILE\.codebuddy\mcp.json"; Kind = "json-cb";   Rule = "CODEBUDDY.md";           Prefix = "cb"; Agent = "CodeBuddy" }
    cursor    = @{ Dir = "$env:USERPROFILE\.cursor";    Mcp = "$env:USERPROFILE\.cursor\mcp.json";    Kind = "json-http"; Rule = ".cursor/rules/mem0.mdc"; Prefix = "cu"; Agent = "Cursor" }
    qoder     = @{ Dir = "$env:USERPROFILE\.qoder";     Mcp = "$env:USERPROFILE\.qoder\mcp.json";     Kind = "json-http"; Rule = ".qoder/rules/mem0.md";   Prefix = "qd"; Agent = "Qoder" }
    codex     = @{ Dir = "$env:USERPROFILE\.codex";     Mcp = "$env:USERPROFILE\.codex\config.toml";  Kind = "toml";      Rule = "AGENTS.md";              Prefix = "cx"; Agent = "Codex" }
    claude    = @{ Dir = "$env:USERPROFILE\.claude";    Mcp = "$env:USERPROFILE\.claude.json";        Kind = "json-http"; Rule = "CLAUDE.md";              Prefix = "cc"; Agent = "Claude Code" }
}

function Resolve-IdeList {
    if ($Ide -ne "") {
        $list = @($Ide.Split(",") | ForEach-Object { $_.Trim().ToLower() })
        foreach ($x in $list) { if (-not $Matrix.ContainsKey($x)) { Write-Err "未知 IDE: $x（支持: $($Matrix.Keys -join ',')）"; exit 1 } }
        return $list
    }
    $found = @($Matrix.Keys | Where-Object { Test-Path $Matrix[$_].Dir })
    if ($found.Count -eq 0) { Write-Err "未检测到已安装 IDE（支持: $($Matrix.Keys -join ',')）。装在非默认路径时用 -Ide 指定。"; exit 1 }
    return $found
}

function Resolve-ApiKey {
    if ($ApiKey -ne "") { return $ApiKey }
    if ($env:MEM0_API_KEY) { Write-Info "使用环境变量 MEM0_API_KEY"; return $env:MEM0_API_KEY }
    $dashUrl = (($Url -replace ":\d+/", ":$DashPort/") -replace "/mcp$", "/dashboard/api-keys")
    Write-Info "未提供密钥。打开 Dashboard API 密钥页: $dashUrl"
    try { Start-Process $dashUrl } catch { Write-Info "请手动打开上述地址创建密钥" }
    $k = (Read-Host "粘贴 m0sk_ 密钥").Trim()
    if ($k -notmatch "^m0sk_\S{10,}") { Write-Err "密钥格式不合法（应为 m0sk_ 开头）"; exit 1 }
    return $k
}

function Write-TextFile([string]$path, [string]$content, [string]$desc) {
    if ($DryRun) { Write-Info "[DryRun] 将写入 $desc -> $path"; return }
    $dir = Split-Path $path -Parent
    if ($dir -and -not (Test-Path $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
    [System.IO.File]::WriteAllText($path, $content, $Utf8)
    Write-Ok "$desc -> $path"
}

function Read-JsonFile([string]$path) {
    $cfg = @{ mcpServers = @{} }
    if (Test-Path $path) {
        try { $cfg = Get-Content $path -Raw -Encoding UTF8 | ConvertFrom-Json -AsHashtable }
        catch {
            $bak = "$path.bak-" + (Get-Date -Format "yyyyMMddHHmmss")
            Copy-Item $path $bak
            Write-Err "原配置 JSON 解析失败，已备份到 $bak。请修复后重跑。"
            exit 1
        }
        if (-not $cfg.ContainsKey("mcpServers")) { $cfg["mcpServers"] = @{} }
    }
    return $cfg
}

function Set-McpConfig([hashtable]$m) {
    Write-Head "MCP 配置（$($m.Kind)）"
    if ($m.Kind -eq "toml") {
        $section = "[mcp_servers.$ServiceName]`nurl = `"$Url`"`n"
        $content = ""
        if (Test-Path $m.Mcp) { $content = [System.IO.File]::ReadAllText($m.Mcp, $Utf8) }
        $pattern = "(?ms)^\[mcp_servers\.$ServiceName\]\r?\n.*?(?=^\[|\z)"
        if ($content -match $pattern) { $content = [regex]::Replace($content, $pattern, $section) }
        else { if ($content -and -not $content.EndsWith("`n")) { $content += "`n" }; $content += $section }
        Write-TextFile $m.Mcp $content "MCP 配置(TOML)"
        return
    }
    $cfg = Read-JsonFile $m.Mcp
    if ($m.Kind -eq "json-cb") {
        $cfg["mcpServers"][$ServiceName] = @{ url = $Url; transport = "streamable-http"; disabled = $false }
    } else {
        $cfg["mcpServers"][$ServiceName] = @{ type = "http"; url = $Url; transport = "streamable-http" }
    }
    Write-TextFile $m.Mcp ($cfg | ConvertTo-Json -Depth 10) "MCP 配置"
}

function Set-RuleFile([hashtable]$m) {
    Write-Head "规则文件（$($m.Agent)）"
    $tplPath = Join-Path $PSScriptRoot "templates\mem0-rules.md"
    if (-not (Test-Path $tplPath)) { Write-Err "模板缺失: $tplPath"; exit 1 }
    $tpl = [System.IO.File]::ReadAllText($tplPath, $Utf8)
    $body = $tpl.Replace("{{CRED_FILE}}", ".mem0/mem0.config.json").
                Replace("{{SESSION_PREFIX}}", $m.Prefix).
                Replace("{{SESSION_FILE}}", ".mem0/.session_id-$($m.Prefix)").
                Replace("{{AGENT}}", $m.Agent)
    $rulePath = Join-Path $Repo $m.Rule
    $begin = "<!-- mem0:rules:begin -->"
    $end = "<!-- mem0:rules:end -->"
    $isMdc = $m.Rule.EndsWith(".mdc")
    $content = ""
    if (Test-Path $rulePath) { $content = [System.IO.File]::ReadAllText($rulePath, $Utf8) }
    if ($content -and $content.Contains($begin) -and $content.Contains($end)) {
        if (-not $Force) {
            $pre = $content.Substring(0, $content.IndexOf($begin))
            $post = $content.Substring($content.IndexOf($end) + $end.Length)
            $new = $pre + $body.Trim() + $post
            Write-TextFile $rulePath $new "规则文件(更新标记区)"
            return
        }
        Write-Info "-Force：重建规则文件"
    }
    $prefix = ""
    if ($isMdc) { $prefix = "---`ndescription: mem0 auto memory`nglobs:`nalwaysApply: true`n---`n`n" }
    if ($isMdc) {
        Write-TextFile $rulePath ($prefix + $body.Trim() + "`n") "规则文件(.mdc 新建)"
        return
    }
    $sep = if ($content) { "`n`n" } else { "" }
    Write-TextFile $rulePath ($content + $sep + $body.Trim() + "`n") "规则文件(追加/新建)"
}

function Set-CredFile([string]$key, [string]$source) {
    Write-Head "凭证文件"
    $path = Join-Path $Repo ".mem0\mem0.config.json"
    $cfg = @{}
    if (Test-Path $path) {
        try { $cfg = Get-Content $path -Raw -Encoding UTF8 | ConvertFrom-Json -AsHashtable } catch { $cfg = @{} }
    }
    $cfg["api_key"] = $key
    if (-not $cfg.ContainsKey("git_remote")) {
        $gr = ""
        try { $gr = (git -C $Repo remote get-url origin 2>$null) } catch { $gr = "" }
        if ([string]::IsNullOrWhiteSpace($gr)) {
            $gr = (Split-Path $Repo -Leaf) -replace "\s+", "-"
            Write-Info "git remote 不存在，git_remote 派生自目录名: $gr"
        }
        $cfg["git_remote"] = $gr
    }
    if (-not $cfg.ContainsKey("project_id")) {
        $cfg["project_id"] = ((Split-Path $cfg["git_remote"] -Leaf) -replace "\.git$")
    }
    $cfg["admin_user_id"] = "admin@mem0.dev"
    $cfg["metadata_defaults"] = @{ source = $source; type_enum = @("fact", "decision", "preference", "note") }
    $cfg["load_on_session_start"] = $true
    $cfg["save_policy"] = "auto-on-durable-fact"
    Write-TextFile $path ($cfg | ConvertTo-Json -Depth 10) "凭证文件"
}

function Test-EndToEnd([string]$key) {
    Write-Head "端到端校验"
    $code = 0
    try { Invoke-WebRequest $Url -UseBasicParsing -TimeoutSec 5 | Out-Null } catch { $code = [int]$_.Exception.Response.StatusCode }
    if ($code -eq 406) { Write-Ok "MCP 端点存活 ($Url, HTTP 406 = streamable-http 正常)" }
    else { Write-Err "MCP 端点异常（HTTP $code）。服务未启动？→ deploy/mem0 或 install-all" }
    $rest = ($Url -replace ":\d+/", ":$RestPort/") -replace "/mcp$", ""
    $headers = @{ "X-API-Key" = $key; "Content-Type" = "application/json" }
    $payload = @{ text = "mem0-setup validation $(Get-Date -Format o)"; infer = $false } | ConvertTo-Json
    try {
        $r = Invoke-RestMethod -Method Post -Uri "$rest/memories" -Headers $headers -Body $payload -TimeoutSec 30
        Write-Ok "REST 写入成功 ($rest/memories)"
        $got = Invoke-RestMethod -Method Get -Uri "$rest/memories?limit=5" -Headers $headers -TimeoutSec 30
        if (($got | ConvertTo-Json -Depth 10) -match "mem0-setup validation") { Write-Ok "REST 查回成功——密钥有效、链路闭环" }
        else { Write-Err "写入后未查回，检查 REST 端口/防火墙" }
        $id = $r.results[0].id
        if ($id) { Invoke-RestMethod -Method Delete -Uri "$rest/memories/$id" -Headers $headers -TimeoutSec 30 | Out-Null; Write-Info "测试记忆已清理" }
    } catch {
        Write-Err "REST 往返失败: $($_.Exception.Message)"
        Write-Info "修复：密钥无效→Dashboard 重建；端口不通→确认 8888 可达"
    }
}

Write-Head "mem0 一键接入$(if ($DryRun) { ' (DryRun)' })"
$key = Resolve-ApiKey
$ides = Resolve-IdeList
Write-Info "目标 IDE: $($ides -join ', ')"
$primary = $Matrix[$ides[0]]
foreach ($n in $ides) { Set-McpConfig $Matrix[$n]; Set-RuleFile $Matrix[$n] }
Set-CredFile $key $primary.Agent
Test-EndToEnd $key
Write-Head "完成"
Write-Host "下一步：重启 IDE → MCP 面板中 mem0 应为绿色 → 让 Agent 记住一条再问回读以验证。"
```

- [ ] **Step 2: 语法校验**

Run: `powershell -NoProfile -Command "$e=$null; [void][System.Management.Automation.Language.Parser]::ParseFile('d:\projects\ai-dev-sop\scripts\mem0-setup.ps1',[ref]$null,[ref]$e); $e.Count"`
Expected: `0`

- [ ] **Step 3: DryRun 冒烟**

Run: `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\mem0-setup.ps1 -DryRun`
Expected: 输出目标 IDE 清单 + "将写入 …" 行 + 校验段；无文件落盘（随后 `git status` 干净）。

- [ ] **Step 4: Commit**

```bash
git add scripts/mem0-setup.ps1
git commit -m "feat(mem0-setup): one-step Windows setup script"
```

---

## Task 3: bash 脚本（scripts/mem0-setup.sh）

**Files:**
- Create: `scripts/mem0-setup.sh`

- [ ] **Step 1: 写入完整脚本**

```bash
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

  # --- 规则文件 ---
  RULEPATH="$REPO/$rule"
  BODY="$(sed -e "s|{{CRED_FILE}}|.mem0/mem0.config.json|g" \
               -e "s|{{SESSION_PREFIX}}|$prefix|g" \
               -e "s|{{SESSION_FILE}}|.mem0/.session_id-$prefix|g" \
               -e "s|{{AGENT}}|$agent|g" "$TPL")"
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
CODE="$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$URL" || echo 000)"
if [ "$CODE" = "406" ]; then ok "MCP 端点存活 ($URL, HTTP 406)"; else err "MCP 端点异常（HTTP $CODE）→ 服务未启动？"; fi
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

head "完成"
echo "下一步：重启 IDE → MCP 面板中 mem0 应为绿色 → 让 Agent 记住一条再问回读以验证。"
```

- [ ] **Step 2: 语法与可执行权限**

Run: `bash -n scripts/mem0-setup.sh && chmod +x scripts/mem0-setup.sh && echo OK`
Expected: `OK`

- [ ] **Step 3: DryRun 冒烟（WSL 或本机 bash）**

Run: `bash scripts/mem0-setup.sh -n`
Expected: 与 PowerShell 版一致的 DryRun 输出，不落盘。

- [ ] **Step 4: Commit**

```bash
git add scripts/mem0-setup.sh
git commit -m "feat(mem0-setup): one-step bash setup script"
```

---

## Task 4: 手册 F2 —— session_id 跨天重生成条款（5 模板 + 5 表格 + 故障表）

**Files:**
- Modify: `docs/quick-ref/mem0-manual.md`（约 11 处）

- [ ] **Step 1: 五份规则模板正文同步（replace_all，同一句尾）**

old: `已存在则直接复用，避免同会话产生多个 id。`
new: `已存在则直接复用，避免同会话产生多个 id；**但若 id 日期与当天不一致、或用户明确开启新会话，必须重新生成并覆写**（否则今天的内容会混入昨天的 dashboard 会话流）。`

命中位置：§4.2 CodeBuddy、§4.3 Qoder、§4.4 Cursor、§4.5 Codex、§4.6 Claude Code 模板的"会话 ID（session_id）"小节。

- [ ] **Step 2: 五处 Step 5 表格行同步（replace_all）**

old: `| 已存在则复用 | 避免同一会话产生多个 id（dashboard 按 session_id 分组会失散） |`
new: `| 已存在则复用 | 避免同一会话产生多个 id（dashboard 按 session_id 分组会失散）；**id 日期与当天不一致时须重新生成** |`

- [ ] **Step 3: 故障表补跨天情形**

old: `| 同一会话在 dashboard 被拆成多组 | `.session_id` 被删除导致每轮重新生成 | 保留该文件；规则中"已存在则复用"条款不可删 |`
new: `| 同一会话在 dashboard 被拆成多组 | `.session_id` 被删除导致每轮重新生成 | 保留该文件；规则中"已存在则复用"条款不可删 |
| 今天的内容出现在昨天的会话流里 | 跨天继续对话却复用了昨天的 `.session_id` | 重新生成当天日期的 session_id 并覆写（格式 `cb-<YYYYMMDD>-<6hex>`） |`

- [ ] **Step 4: 验证**

Run: `Select-String -Path docs\quick-ref\mem0-manual.md -Pattern '当天不一致|跨天' | Select-Object -ExpandProperty LineNumber`
Expected: 至少 11 个行号（5 模板 + 5 表格 + 1 故障表）。

- [ ] **Step 5: Commit**

```bash
git add docs/quick-ref/mem0-manual.md
git commit -m "fix(mem0-manual): add cross-day session_id regeneration rule to all 5 templates"
```

---

## Task 5: 手册 F1 + F3 —— 服务名统一 `mem0`、凭证路径统一 `.mem0/mem0.config.json`

**Files:**
- Modify: `docs/quick-ref/mem0-manual.md`

- [ ] **Step 1: 服务名（replace_all，按序执行，逐条顺序不可并行）**

1. `"mem0-local"` → `"mem0"`（JSON/TOML 示例与 mcp-add 命令中的引号形式，约 12 处）
2. `mem0-local` → `mem0`（散文形式，如"服务名 `mem0-local`"，约 10 处）
3. `mem0-remote` → `mem0`（仅服务名语境，约 4 处：头部适用范围、§2.2.1 L164、§4.2 L484、§4.2 模板 L537/539/549）
   **例外（不得替换）**：`~/mem0-remote/`（§1.3 远程部署目录名，L86）、`deploy/mem0/`（`mem0` 子目录路径）。

- [ ] **Step 2: 头部适用范围改写**

old: `> **适用范围**: 本仓库二开自托管 mem0（本地 `mem0-local` @ `127.0.0.1:8080/mcp`；远程 `mem0-remote` @ `192.168.110.169:8080/mcp`）`
new: `> **适用范围**: 本仓库二开自托管 mem0（本地 `127.0.0.1:8080/mcp`；远程 `192.168.110.169:8080/mcp`）。环境用 URL 区分，**MCP 服务名约定为 `mem0`**（以各 IDE 配置文件实际条目为准）`

- [ ] **Step 3: 凭证路径统一（replace_all，按序）**

1. `.mem0.config.json` → `.mem0/mem0.config.json`（§4.3–4.6，约 30 处）
2. `.codebuddy/mem0.config.json` → `.mem0/mem0.config.json`（§4.2、§1.3、§1.4.2、§5.3.1、§5.4，约 10 处）
3. 4 处"共用"提示句：
   old: `> 同一仓库多工具并用时，可直接共用 CodeBuddy 的 `.codebuddy/mem0.config.json`，避免同一把密钥维护多份。`
   new: `> 凭证文件位于工具无关的 `.mem0/mem0.config.json`，多工具并用时天然共享，无需各 IDE 各存一份。`
4. §5.3.1 校验清单两行：
   old: `- [ ] `.codebuddy/mem0.config.json` 存在且含 `api_key` / `git_remote` / `project_id``
   new: `- [ ] `.mem0/mem0.config.json` 存在且含 `api_key` / `git_remote` / `project_id``
   old: `- [ ] `.gitignore` 覆盖 `mem0.config.json` 与 `.session_id``
   new: `- [ ] `.gitignore` 覆盖 `.mem0/`（含 `mem0.config.json` 与 `.session_id-*`）`
5. §5.4 L2047 安全条目中的路径同步。

- [ ] **Step 4: 验证（须 0 命中）**

Run:
```powershell
$manual = 'd:\projects\ai-dev-sop\docs\quick-ref\mem0-manual.md'
"mem0-local : " + ((Select-String -Path $manual -Pattern 'mem0-local' | Measure-Object).Count)
".codebuddy/mem0.config.json : " + ((Select-String -Path $manual -Pattern [regex]::Escape('.codebuddy/mem0.config.json') | Measure-Object).Count)
".mem0.config.json(旧) : " + ((Select-String -Path $manual -Pattern [regex]::Escape('`.mem0.config.json') | Measure-Object).Count)
"mem0-remote(仅部署目录) : " + ((Select-String -Path $manual -Pattern 'mem0-remote' | Measure-Object).Count)
```
Expected: `0 / 0 / 0 / 1`（仅 §1.3 部署目录 `~/mem0-remote/` 保留）。

- [ ] **Step 5: Commit**

```bash
git add docs/quick-ref/mem0-manual.md
git commit -m "fix(mem0-manual): unify MCP service name to mem0 and credential path to .mem0/"
```

---

## Task 6: 手册 F4 + F5 —— 一键脚本快捷方式与校验清单增强

**Files:**
- Modify: `docs/quick-ref/mem0-manual.md`

- [ ] **Step 1: §4.2–4.6 每章开头插入快捷方式段**

在每章标题下方追加（IDE 名按章替换）：

```markdown
> ⚡ **一键完成**：以上 5 步可由
> `scripts/mem0-setup.ps1`（Windows）/ `scripts/mem0-setup.sh`（macOS·Linux）一键完成，
> 服务名自动约定为 `mem0` 并内置端到端校验。手工流程保留用于理解原理与特殊场景。
```

- [ ] **Step 2: §5.3.1 校验清单增加脚本项**

在清单首行前插入：
`- [ ] 已运行 `mem0-setup`（`-DryRun` 预览 → 实跑）且全部输出 ✓`

- [ ] **Step 3: 目录与 §2.1 工具矩阵（可选但建议）**

§2.1 表新增列「一键接入」：5 个 IDE 均填 `mem0-setup`。

- [ ] **Step 4: Commit**

```bash
git add docs/quick-ref/mem0-manual.md
git commit -m "docs(mem0-manual): add mem0-setup one-step shortcut and checklist item"
```

---

## Task 7: 仓库配套变更（CODEBUDDY.md / .gitignore / install-all / 自身迁移）

**Files:**
- Modify: `CODEBUDDY.md`、`.gitignore`、`scripts/install-all.ps1`、`scripts/install-all.sh`
- Move: `.codebuddy/mem0.config.json` → `.mem0/mem0.config.json`

- [ ] **Step 1: CODEBUDDY.md 凭证路径 + session_id 路径（replace_all，缺一不可）**

1. 凭证路径：old `.codebuddy/mem0.config.json` → new `.mem0/mem0.config.json`
2. session_id 文件路径：old `.codebuddy/.session_id` → new `.mem0/.session_id-cb`
   （否则 Agent 会去读 Step 4 已迁走的文件，重现"会话 ID 失效"缺陷）
3. 在 §1 加载记忆段落末尾追加兼容说明：
   `（兼容：若 `.mem0/mem0.config.json` 不存在，回退读取 `.codebuddy/mem0.config.json`。）`
4. 验证：`Select-String -Path CODEBUDDY.md -Pattern '\.codebuddy/'` 应为 0 命中。

- [ ] **Step 2: .gitignore 追加**

```gitignore
# mem0 本地凭证与会话标识（含密钥，禁止入库）
.mem0/
```

- [ ] **Step 3: 两个 install-all 末尾加指引**

`scripts/install-all.ps1` 与 `install-all.sh` 的"下一步"输出块末尾各加一行：
`  4. 接入 IDE：运行 scripts/mem0-setup（一键配置 MCP + 规则文件 + 凭证并校验）`

- [ ] **Step 4: 本仓库存量迁移（文件在 git-ignore 内）**

```powershell
New-Item -ItemType Directory -Force -Path "d:\projects\ai-dev-sop\.mem0" | Out-Null
Move-Item "d:\projects\ai-dev-sop\.codebuddy\mem0.config.json" "d:\projects\ai-dev-sop\.mem0\mem0.config.json" -Force
Move-Item "d:\projects\ai-dev-sop\.codebuddy\.session_id" "d:\projects\ai-dev-sop\.mem0\.session_id-cb" -Force -ErrorAction SilentlyContinue
Get-Content "d:\projects\ai-dev-sop\.mem0\mem0.config.json"
```
Expected: 凭证 JSON 含 `api_key`/`git_remote`/`project_id`；`.mem0/.session_id-cb` 存在（若原先无 .session_id 则跳过）。

- [ ] **Step 5: Commit**

```bash
git add CODEBUDDY.md .gitignore scripts/install-all.ps1 scripts/install-all.sh
git commit -m "chore(mem0): move credentials to .mem0/ and point install-all at mem0-setup"
```

---

## Task 8: 端到端验证（实跑 + 回归）

**Files:**
- 无新增文件；验证前序全部任务

- [ ] **Step 1: 实跑 Windows 脚本（当前仓库，CodeBuddy 已安装）**

密钥来源：已设置 `MEM0_API_KEY` 时直接运行即可；否则用 `-ApiKey m0sk_...`（明文仅用于本轮验证，验证完无需保存）。

Run: `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\mem0-setup.ps1`
Expected: `目标 IDE: codebuddy` → MCP 配置 OK → 规则文件 OK → 凭证文件 OK → 校验 `MCP 端点存活` + `REST 写入成功` + `REST 查回成功`。

- [ ] **Step 2: 幂等回归（连跑两次）**

Run: 再次执行同一命令
Expected: 规则文件输出为"(更新标记区)"，文件中标记区仅一份；`~/.codebuddy/mcp.json` 中其他 MCP 服务条目（Playwright、sqlbot）保留。

Run: `python3 -c "import json;print(sorted(json.load(open(r'C:\Users\joezxh\.codebuddy\mcp.json'))['mcpServers']))"`（或 PS 解析）
Expected: 包含 `Playwright MCP Server`、`mem0`、`sqlbot`。

- [ ] **Step 2b: 矩阵覆盖测试 T3（沙箱 HOME，不污染真实配置）**

```powershell
$tmp = "$env:TEMP\mem0-setup-test"; New-Item -ItemType Directory -Force -Path "$tmp\.cursor","$tmp\.qoder","$tmp\.codex" | Out-Null
$env:USERPROFILE = $tmp
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\mem0-setup.ps1 -Ide cursor,qoder,codex -DryRun
```
Expected: 三个 IDE 的 MCP 配置路径（`$tmp\.cursor\mcp.json` / `.qoder\mcp.json` / `.codex\config.toml`）与规则文件路径（`.cursor/rules/mem0.mdc` / `.qoder/rules/mem0.md` / `AGENTS.md`）均被打印，无落盘。
随后去掉 `-DryRun` 实跑一次并回读：

Run: `python3 -c "import json;print(json.load(open(r'$env:TEMP\mem0-setup-test\.cursor\mcp.json'))['mcpServers']['mem0'])"` 与 `Get-Content "$env:TEMP\mem0-setup-test\.codex\config.toml"`
Expected: cursor 为 `{'type':'http','url':...,'transport':'streamable-http'}`；Codex TOML 为 `[mcp_servers.mem0]` + `url = "..."`。
清理：`Remove-Item $env:TEMP\mem0-setup-test -Recurse -Force`（并恢复 USERPROFILE：重开终端即可）。

- [ ] **Step 2c: 错误路径测试 T4 / T5**

T4 无效密钥：`powershell -NoProfile -File scripts\mem0-setup.ps1 -ApiKey m0sk_0000000000 -DryRun`
Expected：密钥格式校验通过（格式合法），实跑时 REST 段输出 `REST 写入失败 → 密钥无效或 8888 不可达`，其余步骤不中断。
T5 端点不可达：`powershell -NoProfile -File scripts\mem0-setup.ps1 -Url http://127.0.0.1:9999/mcp -DryRun`
Expected：配置写入仍执行（DryRun 打印），校验段 `MCP 端点异常（HTTP 000）→ 服务未启动？`。

- [ ] **Step 3: 全工程残留扫描（须 0 命中）**

Run:
```powershell
Get-ChildItem -Path d:\projects\ai-dev-sop -Recurse -File -Include *.md,*.ps1,*.sh |
  Where-Object { ($_.FullName -split '\\') -notcontains 'node_modules' -and ($_.FullName -split '\\') -notcontains '.git' } |
  Select-String -Pattern 'mem0-local|\.codebuddy/mem0\.config\.json|mem0/config\.json（旧路径）'
```
Expected: 0 命中（`mem0-remote` 仅允许出现在 §1.3 部署目录描述）。

- [ ] **Step 4: 冒烟（需用户重启 IDE 后人工确认）**

提示用户：重启 CodeBuddy → Settings → MCP → `mem0` 绿色 → 对 Agent 说"记住我偏好 TypeScript"→ 新会话问"我的偏好是什么"能读回即通过。

- [ ] **Step 5: 最终 Commit**

```bash
git add -A
git commit -m "feat: mem0 one-step setup (ps1/sh) + manual consistency fixes"
```
