#Requires -Version 5.1
<#
.SYNOPSIS  mem0 一键接入：IDE MCP 配置 + 规则文件 + 凭证文件 + 端到端校验。
.EXAMPLE  .\mem0-setup.ps1
.EXAMPLE  .\mem0-setup.ps1 -Url http://192.168.110.169:8080/mcp -ApiKey m0sk_xxx -DryRun
#>
param(
    [string]$Ide = "",
    [string]$Url = "http://127.0.0.1:8080/mcp",
    [string]$RestUrl = "",
    [string]$ApiKey = "",
    [string]$Repo = (Get-Location).Path,
    [ValidateSet("user", "repo")]
    [string]$Scope = "user",
    [switch]$DryRun,
    [switch]$Force
)
$ErrorActionPreference = "Stop"
$ServiceName = "mem0"
$RestPort = 8002
$DashPort = 3001
$Utf8 = [System.Text.UTF8Encoding]::new($false)

function Write-Ok($m)   { Write-Host "  [OK]   $m" -ForegroundColor Green }
function Write-Err($m)  { Write-Host "  [FAIL] $m" -ForegroundColor Red }
function Write-Info($m) { Write-Host "  [i]    $m" -ForegroundColor Yellow }
function Write-Head($m) { Write-Host ""; Write-Host "== $m ==" -ForegroundColor Cyan }

$Matrix = @{
    codebuddy = @{ Dir = "$env:USERPROFILE\.codebuddy"; Mcp = "$env:USERPROFILE\.codebuddy\mcp.json"; Kind = "json-cb";   Rule = "CODEBUDDY.md";           UserRule = "$env:USERPROFILE\.codebuddy\CODEBUDDY.md"; Prefix = "cb"; Agent = "CodeBuddy" }
    cursor    = @{ Dir = "$env:USERPROFILE\.cursor";    Mcp = "$env:USERPROFILE\.cursor\mcp.json";    Kind = "json-http"; Rule = ".cursor/rules/mem0.mdc"; UserRule = "$env:USERPROFILE\.cursor\rules\mem0.mdc";  Prefix = "cu"; Agent = "Cursor" }
    qoder     = @{ Dir = "$env:USERPROFILE\.qoder";     Mcp = "$env:USERPROFILE\.qoder\mcp.json";     Kind = "json-http"; Rule = ".qoder/rules/mem0.md";   UserRule = "$env:USERPROFILE\.qoder\rules\mem0.md";    Prefix = "qd"; Agent = "Qoder" }
    codex     = @{ Dir = "$env:USERPROFILE\.codex";     Mcp = "$env:USERPROFILE\.codex\config.toml";  Kind = "toml";      Rule = "AGENTS.md";              UserRule = "$env:USERPROFILE\.codex\AGENTS.md";        Prefix = "cx"; Agent = "Codex" }
    claude    = @{ Dir = "$env:USERPROFILE\.claude";    Mcp = "$env:USERPROFILE\.claude.json";        Kind = "json-http"; Rule = "CLAUDE.md";              UserRule = "$env:USERPROFILE\.claude\CLAUDE.md";       Prefix = "cc"; Agent = "Claude Code" }
}
# UI 型用户规则的 IDE：文件写入后可能需在设置页 Rules 手动粘贴
$UiRuleAgents = @("CodeBuddy", "Cursor", "Qoder")

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

function ConvertTo-Hashtable($obj) {
    if ($null -eq $obj) { return $null }
    if ($obj -is [System.Array]) { return @($obj | ForEach-Object { ConvertTo-Hashtable $_ }) }
    if ($obj -is [System.Management.Automation.PSCustomObject]) {
        $h = @{}
        foreach ($p in $obj.PSObject.Properties) { $h[$p.Name] = ConvertTo-Hashtable $p.Value }
        return $h
    }
    return $obj
}

function Read-JsonFile([string]$path) {
    $cfg = @{ mcpServers = @{} }
    if (Test-Path $path) {
        try { $cfg = ConvertTo-Hashtable ([System.IO.File]::ReadAllText($path, $Utf8) | ConvertFrom-Json) }
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
    Write-Head "规则文件（$($m.Agent)，$Scope 级）"
    $tplPath = Join-Path $PSScriptRoot "templates\mem0-rules.md"
    if (-not (Test-Path $tplPath)) { Write-Err "模板缺失: $tplPath"; exit 1 }
    $tpl = [System.IO.File]::ReadAllText($tplPath, $Utf8)
    $isUser = $Scope -eq "user"
    $credRef = if ($isUser) { "$env:USERPROFILE\.mem0\mem0.config.json" } else { ".mem0/mem0.config.json" }
    $sessRef = if ($isUser) { "$env:USERPROFILE\.mem0\.session_id-$($m.Prefix)" } else { ".mem0/.session_id-$($m.Prefix)" }
    $body = $tpl.Replace("{{CRED_FILE}}", ($credRef -replace "\\", "/")).
                Replace("{{SESSION_PREFIX}}", $m.Prefix).
                Replace("{{SESSION_FILE}}", ($sessRef -replace "\\", "/")).
                Replace("{{AGENT}}", $m.Agent)
    $rulePath = if ($isUser) { $m.UserRule } else { Join-Path $Repo $m.Rule }
    if ($isUser -and $UiRuleAgents -contains $m.Agent) {
        Write-Info "user 级规则：若 $($m.Agent) 未自动加载该文件，请在设置页 Rules 中粘贴其内容"
    }
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
    Write-Head "凭证文件（$Scope 级）"
    $path = if ($Scope -eq "user") { Join-Path $env:USERPROFILE ".mem0\mem0.config.json" } else { Join-Path $Repo ".mem0\mem0.config.json" }
    $cfg = @{}
    if (Test-Path $path) {
        try { $cfg = ConvertTo-Hashtable ([System.IO.File]::ReadAllText($path, $Utf8) | ConvertFrom-Json) } catch { $cfg = @{} }
    }
    $cfg["api_key"] = $key
    if (-not $cfg.ContainsKey("git_remote")) {
        if ($Scope -eq "user") {
            # user 级凭证不绑定单一仓库：Agent 运行时按当前项目现取 git remote
            $cfg["git_remote_mode"] = "runtime"
            Write-Info "user 级凭证不写 git_remote——Agent 每次用当前项目 'git remote get-url origin' 现取"
            Write-Info "工程级 .mem0\mem0.config.json（若存在）优先于本文件；各项目记忆池按 git_remote 自然隔离"
        } else {
            $gr = ""
            try { $gr = (git -C $Repo remote get-url origin 2>$null) } catch { $gr = "" }
            if ([string]::IsNullOrWhiteSpace($gr)) {
                $gr = (Split-Path $Repo -Leaf) -replace "\s+", "-"
                Write-Info "git remote 不存在，git_remote 派生自目录名: $gr"
            }
            $cfg["git_remote"] = $gr
        }
    }
    if (-not $cfg.ContainsKey("project_id")) {
        if ($Scope -eq "repo") { $cfg["project_id"] = ((Split-Path $cfg["git_remote"] -Leaf) -replace "\.git$") }
        else { $cfg["project_id_mode"] = "runtime" }
    }
    $cfg["admin_user_id"] = "admin@mem0.dev"
    $cfg["metadata_defaults"] = @{ source = $source; type_enum = @("fact", "decision", "preference", "note") }
    $cfg["load_on_session_start"] = $true
    $cfg["save_policy"] = "auto-on-durable-fact"
    Write-TextFile $path ($cfg | ConvertTo-Json -Depth 10) "凭证文件"
}

function Set-TranscriptHook {
    # 转录直投 Hook：SessionStart 时 flush 上一 CLI 会话的 JSONL 转录（~/.codebuddy/projects/...）到 mem0（零 LLM 逐字留痕）。
    Write-Head "转录直投 Hook（零 LLM 逐字留痕，仅 codebuddy）"
    $posterSrc = Join-Path $PSScriptRoot "mem0-transcript-poster.mjs"
    if (-not (Test-Path $posterSrc)) { Write-Err "poster 脚本缺失: $posterSrc"; return }
    $node = (Get-Command node -ErrorAction SilentlyContinue).Source
    if (-not $node) { Write-Err "未找到 node（Hook 需要 Node.js ≥ 18）——跳过 Hook 安装，可手动配置（见 scripts/README-mem0-poster.md）"; return }
    $hookDir = Join-Path $env:USERPROFILE ".codebuddy\hooks"
    $dst = Join-Path $hookDir "mem0-transcript-poster.mjs"
    if ($DryRun) {
        Write-Info "[DryRun] 将复制 $posterSrc -> $dst"
    } else {
        if (-not (Test-Path $hookDir)) { New-Item -ItemType Directory -Force -Path $hookDir | Out-Null }
        Copy-Item $posterSrc $dst -Force
        Write-Ok "poster 已复制 -> $dst"
    }
    $rest = if ($RestUrl -ne "") { $RestUrl.TrimEnd("/") } else { ($Url -replace ":\d+/", ":$RestPort/") -replace "/mcp$", "" }
    # poster 用 --workspace 指向本工程（读取 <Repo>\.mem0\mem0.config.json 取 api_key/git_remote），rest-url 显式指定
    $hookCmd = """$node"" ""$dst"" --workspace ""$Repo"" --rest-url ""$rest"""
    $settingsPath = Join-Path $env:USERPROFILE ".codebuddy\settings.json"
    if ($DryRun) { Write-Info "[DryRun] 将合并 SessionStart hook 到 $settingsPath"; return }
    $cfg = @{}
    if (Test-Path $settingsPath) {
        try { $cfg = ConvertTo-Hashtable ([System.IO.File]::ReadAllText($settingsPath, $Utf8) | ConvertFrom-Json) }
        catch {
            $bak = "$settingsPath.bak-" + (Get-Date -Format "yyyyMMddHHmmss")
            Copy-Item $settingsPath $bak
            Write-Err "settings.json 解析失败，已备份到 $bak。请手动合并 SessionStart hook（见 scripts/README-mem0-poster.md）"
            return
        }
    }
    if (-not $cfg.ContainsKey("hooks")) { $cfg["hooks"] = @{} }
    # 同时注册 SessionStart（flush 上一会话，兜底）与 SessionEnd（flush 当前会话，解决“最后一个会话在下次 SessionStart 前不入库”的缺口）
    $changed = $false
    foreach ($ev in @("SessionStart", "SessionEnd")) {
        if (-not $cfg["hooks"].ContainsKey($ev)) { $cfg["hooks"][$ev] = @() }
        $exists = $false
        foreach ($entry in @($cfg["hooks"][$ev])) {
            foreach ($h in @($entry["hooks"])) {
                if (([string]$h["command"]) -match "mem0-transcript-poster") { $exists = $true; break }
            }
            if ($exists) { break }
        }
        if (-not $exists) {
            $cfg["hooks"][$ev] = @($cfg["hooks"][$ev]) + @{ hooks = @(@{ type = "command"; command = $hookCmd; timeout = 60 }) }
            $changed = $true
        }
    }
    if ($changed) {
        Write-TextFile $settingsPath ($cfg | ConvertTo-Json -Depth 20) "settings.json (转录直投 hook: SessionStart+SessionEnd)"
    } else {
        Write-Info "转录直投 hook 已存在（幂等跳过）"
    }
    Write-Info "转录直投 hook：SessionStart flush 上一会话，SessionEnd flush 当前会话（均幂等，零 LLM 逐字留痕）"
}

function Set-ToolCaptureHook {
    # 工具层捕获 hook：PostToolUse 把 IDE 原样传出的工具事件落盘（零 LLM）。
    # 背景：IDE 会话转录已落盘本地（CodeBuddyExtension\Data\...\history\...\messages\*.json），
    # 工具层 PostToolUse 事件另经本 hook 原样落盘 <repo>/.mem0/tool-events.jsonl，与转录通道互补。
    Write-Head "工具层捕获 Hook（零 LLM，仅 codebuddy）"
    $captureSrc = Join-Path $PSScriptRoot "mem0-tool-capture-hook.mjs"
    if (-not (Test-Path $captureSrc)) { Write-Err "capture 脚本缺失: $captureSrc"; return }
    $node = (Get-Command node -ErrorAction SilentlyContinue).Source
    if (-not $node) { Write-Err "未找到 node——跳过工具捕获 Hook 安装"; return }
    $hookDir = Join-Path $env:USERPROFILE ".codebuddy\hooks"
    $dst = Join-Path $hookDir "mem0-tool-capture-hook.mjs"
    if ($DryRun) {
        Write-Info "[DryRun] 将复制 $captureSrc -> $dst"
    } else {
        if (-not (Test-Path $hookDir)) { New-Item -ItemType Directory -Force -Path $hookDir | Out-Null }
        Copy-Item $captureSrc $dst -Force
        Write-Ok "capture 已复制 -> $dst"
    }
    $hookCmd = """$node"" ""$dst"""
    $settingsPath = Join-Path $env:USERPROFILE ".codebuddy\settings.json"
    if ($DryRun) { Write-Info "[DryRun] 将合并 PostToolUse hook 到 $settingsPath"; return }
    $cfg = @{}
    if (Test-Path $settingsPath) {
        try { $cfg = ConvertTo-Hashtable ([System.IO.File]::ReadAllText($settingsPath, $Utf8) | ConvertFrom-Json) }
        catch { Write-Err "settings.json 解析失败，请手动合并 PostToolUse 工具捕获 hook"; return }
    }
    if (-not $cfg.ContainsKey("hooks")) { $cfg["hooks"] = @{} }
    if (-not $cfg["hooks"].ContainsKey("PostToolUse")) { $cfg["hooks"]["PostToolUse"] = @() }
    foreach ($entry in @($cfg["hooks"]["PostToolUse"])) {
        foreach ($h in @($entry["hooks"])) {
            if (([string]$h["command"]) -match "mem0-tool-capture-hook") {
                Write-Info "PostToolUse 工具捕获 hook 已存在（幂等跳过）"
                return
            }
        }
    }
    $cfg["hooks"]["PostToolUse"] = @($cfg["hooks"]["PostToolUse"]) + @{
        matcher = ".*"
        hooks = @(@{ type = "command"; command = $hookCmd; timeout = 10 })
    }
    Write-TextFile $settingsPath ($cfg | ConvertTo-Json -Depth 20) "settings.json (PostToolUse 工具捕获 hook)"
    Write-Info "工具事件将落盘到 <工程>\.mem0\tool-events.jsonl（零 LLM 原样记录）"
}

function Set-IdeSessionHook {
    # IDE 会话转录直投 Hook：SessionStart 时 flush 上一会话在 IDE 本地落盘的会话转录到 mem0（零 LLM 逐字留痕）。
    # 数据源（落盘事实）：%LOCALAPPDATA%/CodeBuddyExtension/Data/<userId>/CodeBuddyIDE/<userId>/history/<workspaceHash>/<sessionId>/messages/<msgId>.json
    Write-Head "IDE 会话转录直投 Hook（零 LLM 逐字留痕，仅 codebuddy）"
    $posterSrc = Join-Path $PSScriptRoot "mem0-ide-session-poster.mjs"
    if (-not (Test-Path $posterSrc)) { Write-Err "poster 脚本缺失: $posterSrc"; return }
    $node = (Get-Command node -ErrorAction SilentlyContinue).Source
    if (-not $node) { Write-Err "未找到 node（Hook 需要 Node.js ≥ 18）——跳过 Hook 安装，可手动配置（见 scripts/README-mem0-poster.md）"; return }
    $hookDir = Join-Path $env:USERPROFILE ".codebuddy\hooks"
    $dst = Join-Path $hookDir "mem0-ide-session-poster.mjs"
    if ($DryRun) {
        Write-Info "[DryRun] 将复制 $posterSrc -> $dst"
    } else {
        if (-not (Test-Path $hookDir)) { New-Item -ItemType Directory -Force -Path $hookDir | Out-Null }
        Copy-Item $posterSrc $dst -Force
        Write-Ok "IDE poster 已复制 -> $dst"
    }
    $rest = if ($RestUrl -ne "") { $RestUrl.TrimEnd("/") } else { ($Url -replace ":\d+/", ":$RestPort/") -replace "/mcp$", "" }
    # poster 用 --workspace 指向本工程（读取 <Repo>\.mem0\mem0.config.json 取 api_key/git_remote），rest-url 显式指定
    $hookCmd = """$node"" ""$dst"" --workspace ""$Repo"" --rest-url ""$rest"""
    $settingsPath = Join-Path $env:USERPROFILE ".codebuddy\settings.json"
    if ($DryRun) { Write-Info "[DryRun] 将合并 SessionStart hook 到 $settingsPath"; return }
    $cfg = @{}
    if (Test-Path $settingsPath) {
        try { $cfg = ConvertTo-Hashtable ([System.IO.File]::ReadAllText($settingsPath, $Utf8) | ConvertFrom-Json) }
        catch {
            $bak = "$settingsPath.bak-" + (Get-Date -Format "yyyyMMddHHmmss")
            Copy-Item $settingsPath $bak
            Write-Err "settings.json 解析失败，已备份到 $bak。请手动合并 SessionStart hook（见 scripts/README-mem0-poster.md）"
            return
        }
    }
    if (-not $cfg.ContainsKey("hooks")) { $cfg["hooks"] = @{} }
    # 同时注册 SessionStart（flush 上一会话，兜底）与 SessionEnd（flush 当前会话，解决“最后一个会话在下次 SessionStart 前不入库”的缺口）
    $changed = $false
    foreach ($ev in @("SessionStart", "SessionEnd")) {
        if (-not $cfg["hooks"].ContainsKey($ev)) { $cfg["hooks"][$ev] = @() }
        $exists = $false
        foreach ($entry in @($cfg["hooks"][$ev])) {
            foreach ($h in @($entry["hooks"])) {
                if (([string]$h["command"]) -match "mem0-ide-session-poster") { $exists = $true; break }
            }
            if ($exists) { break }
        }
        if (-not $exists) {
            $cfg["hooks"][$ev] = @($cfg["hooks"][$ev]) + @{ hooks = @(@{ type = "command"; command = $hookCmd; timeout = 60 }) }
            $changed = $true
        }
    }
    if ($changed) {
        Write-TextFile $settingsPath ($cfg | ConvertTo-Json -Depth 20) "settings.json (IDE 会话转录直投 hook: SessionStart+SessionEnd)"
    } else {
        Write-Info "IDE 会话转录直投 hook 已存在（幂等跳过）"
    }
    Write-Info "IDE 会话转录直投 hook：SessionStart flush 上一会话，SessionEnd flush 当前会话（均幂等，零 LLM 逐字留痕）"
}

function Test-EndToEnd([string]$key) {
    Write-Head "端到端校验"
    $code = 0
    $mcpHost = ""
    try { $mcpHost = ([uri]$Url).Host } catch { }
    if ($mcpHost -match "^(host|localhost\.local|xxx|example|<[^>]*>|\{\})$") {
        Write-Err "MCP URL 主机名 '$mcpHost' 是占位符，不是真实地址。本机部署请用 -Url http://127.0.0.1:8080/mcp，远程部署用实际 IP。"
        return
    }
    try { Invoke-WebRequest $Url -UseBasicParsing -TimeoutSec 5 | Out-Null } catch { if ($_.Exception.Response) { $code = [int]$_.Exception.Response.StatusCode } }
    if ($code -eq 406) { Write-Ok "MCP 端点存活 ($Url, HTTP 406 = streamable-http 正常)" }
    elseif ($code -eq 502) {
        Write-Err "MCP 端点返回 HTTP 502 → 通常不是服务未启动，而是主机名 '$mcpHost' 无法解析、请求被系统代理拦截。当前目标: $Url"
        Write-Info "修复：确认地址真实可达（本机: http://127.0.0.1:8080/mcp）；必要时为该主机配置代理豁免后重跑"
    }
    else { Write-Err "MCP 端点异常（HTTP $code）。服务未启动？→ deploy/mem0 或 install-all" }
    if ($DryRun) { Write-Info "[DryRun] 跳过 REST 写读往返（DryRun 不落盘、不写入记忆）"; return }
    $rest = if ($RestUrl -ne "") { $RestUrl.TrimEnd("/") } else { ($Url -replace ":\d+/", ":$RestPort/") -replace "/mcp$", "" }
    $headers = @{ "X-API-Key" = $key; "Content-Type" = "application/json" }
    # REST 契约：POST /memories 需要 messages 数组（text 是 MCP 层的包装参数）
    $payload = @{
        messages = @(@{ role = "user"; content = "mem0-setup validation $(Get-Date -Format o)" })
        infer = $false
        user_id = "admin@mem0.dev"
    } | ConvertTo-Json
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
        Write-Info "修复：密钥无效→Dashboard 重建；端口不通→用 -RestUrl 指定实际 REST 地址（本机如 http://127.0.0.1:8002）"
    }
}

$scopeDesc = if ($Scope -eq "user") { "user 级（本机所有项目）" } else { "repo 级（仅当前项目 $Repo）" }
Write-Head "mem0 一键接入$(if ($DryRun) { ' (DryRun)' })【作用范围: $scopeDesc】"
$key = Resolve-ApiKey
$ides = Resolve-IdeList
Write-Info "目标 IDE: $($ides -join ', ')"
$primary = $Matrix[$ides[0]]
foreach ($n in $ides) { Set-McpConfig $Matrix[$n]; Set-RuleFile $Matrix[$n] }
Set-CredFile $key $primary.Agent
if ($ides -contains "codebuddy") { Set-TranscriptHook; Set-IdeSessionHook; Set-ToolCaptureHook }
Test-EndToEnd $key
Write-Head "完成"
Write-Host "下一步：重启 IDE → MCP 面板中 mem0 应为绿色 → 让 Agent 记住一条再问回读以验证。"
