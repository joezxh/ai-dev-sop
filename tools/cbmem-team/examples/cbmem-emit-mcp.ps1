#!/usr/bin/env pwsh
# mint-and-emit: mint a long-lived JWT for a cbmem-team user and (re)write the
# Cursor mcp.json entry for that server.
#
# Why this exists:
#   We added GET /admin/users/:id/token?ttl=... so admins can mint tokens
#   without restarting the server, and POST /refresh so an end user can swap
#   a still-valid JWT for a fresh one. But every refresh still requires a
#   curl call. This script automates that dance so you can re-run it any
#   time your token expires (or every N days as a cron) without thinking.
#
# Usage:
#   pwsh examples/cbmem-emit-mcp.ps1 \
#     -Base http://192.168.100.83:8787 \
#     -AdminToken e1c96... \
#     -User tianque \
#     -ProjectServerPath /var/lib/cbmem-team/users/tianque/repos/github.com__you_ai-dev-sop \
#     -McpPath "$HOME/.cursor/mcp.json" \
#     -Ttl 4320h
#
# What it does:
#   1. POST /admin/users/<user>/token?ttl=<ttl> with X-Admin-Token
#   2. Read or create ~/.cursor/mcp.json
#   3. Upsert the "cbmem-team" entry with the fresh token + URL
#   4. Print a one-line summary so you can pipe into other tooling
#
# Required:
#   pwsh 7+ (Windows ships 5.1; install 7 from microsoft.com or via `winget`)
#
[CmdletBinding()]
param(
  [Parameter(Mandatory)][string]$Base,
  [Parameter(Mandatory)][string]$AdminToken,
  [Parameter(Mandatory)][string]$User,
  [Parameter(Mandatory)][string]$ProjectServerPath,
  [string]$McpPath = (Join-Path $HOME '.cursor/mcp.json'),
  [string]$Ttl = '4320h',           # 180 days
  [string]$ServerName = 'cbmem-team',
  [int]$Port = 8787,
  [string]$HostOverride = ''         # set if Base host != what you want Cursor to use
)

$ErrorActionPreference = 'Stop'

function Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Warn($msg)  { Write-Host "!!  $msg" -ForegroundColor Yellow }
function Ok($msg)    { Write-Host "OK  $msg" -ForegroundColor Green }

# 1. mint
Step "minting JWT for $User (ttl=$Ttl)"
$mintUrl = "$Base/admin/users/$User/token?ttl=$Ttl"
$resp = Invoke-RestMethod -Uri $mintUrl -Headers @{ 'X-Admin-Token' = $AdminToken } -Method POST -TimeoutSec 15
if (-not $resp.token) { throw "mint failed: $(ConvertTo-Json $resp)" }
$jwt = $resp.token
$expires = $resp.expires
Ok "token expires $expires"

# 2. resolve mcp.json
Step "loading $McpPath"
if (Test-Path -LiteralPath $McpPath) {
  $cfg = Get-Content -LiteralPath $McpPath -Raw -Encoding UTF8 | ConvertFrom-Json
} else {
  $cfg = [pscustomobject]@{ mcpServers = [pscustomobject]@{} }
  $parent = Split-Path -Parent $McpPath
  if ($parent -and -not (Test-Path -LiteralPath $parent)) {
    New-Item -ItemType Directory -Path $parent -Force | Out-Null
  }
}

# 3. build URL (preserve query/escaping)
$hostForUrl = if ($HostOverride) { $HostOverride } else { ([uri]$Base).Host }
$mcpUrl = "http://${hostForUrl}:${Port}/mcp?as=$([uri]::EscapeDataString($User))&project=$([uri]::EscapeDataString($ProjectServerPath))"

# 4. upsert
$entry = [pscustomobject]@{
  type    = 'http'
  url     = $mcpUrl
  headers = [pscustomobject]@{ Authorization = "Bearer $jwt" }
  enabled = $true
}

if ($cfg.mcpServers.PSObject.Properties.Name -notcontains $ServerName) {
  $cfg.mcpServers | Add-Member -NotePropertyName $ServerName -NotePropertyValue $entry -Force
} else {
  $cfg.mcpServers.$ServerName = $entry
}

Step "writing $McpPath"
$json = $cfg | ConvertTo-Json -Depth 8
[System.IO.File]::WriteAllText($McpPath, $json, [System.Text.UTF8Encoding]::new($false))
Ok "wrote $([System.IO.Path]::GetFullPath($McpPath))"

# 5. emit one-line summary
Write-Output "SUMMARY server=$ServerName url=$mcpUrl ttl=$Ttl expires=$expires"

Step "verification (tools/list, expect 200 with N tools)"
$body = '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
try {
  $r = Invoke-WebRequest -Uri $mcpUrl `
    -Headers @{ Authorization = "Bearer $jwt"; 'Content-Type' = 'application/json'; 'Accept' = 'application/json, text/event-stream' } `
    -Method POST -Body $body -TimeoutSec 30 -UseBasicParsing
  $tools = ($r.Content | ConvertFrom-Json).result.tools.Count
  Ok "tools/list returned $tools tools"
} catch {
  Warn "verify failed (config still written): $($_.Exception.Message)"
  if ($_.Exception.Response) {
    $reader = [System.IO.StreamReader]::new($_.Exception.Response.GetResponseStream())
    Warn "  body: $($reader.ReadToEnd())"
  }
}