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
