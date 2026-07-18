# cbmem-team Windows startup script
# Usage: .\start.ps1
# Stops any existing cbmem-team process and starts a new one with correct paths.

$ErrorActionPreference = "Stop"

$cbmemExe   = "D:\projects\ai-dev-sop\tools\cbmem-team\cbmem-team.exe"
$mcpBin     = "C:\Users\joezxh\AppData\Local\Programs\codebase-memory-mcp\codebase-memory-mcp.exe"
$dataDir    = "D:\projects\ai-dev-sop\tmp\cbmem-data"
$jwtSecret  = "dev-jwt-secret-please-rotate"
$adminToken = "change-me-too"

# MemPalace HTTP MCP integration (POST /mcp). Leave $memPalaceBase empty to
# disable auto-sync. $memPalaceToken must match MemPalace's
# MEMPALACE_MCP_HTTP_TOKEN (required when MemPalace binds a non-loopback host).
$memPalaceBase  = "http://127.0.0.1:8765"
$memPalaceToken = ""

# Stop existing process
$existing = Get-Process cbmem-team -ErrorAction SilentlyContinue
if ($existing) {
    Write-Host "Stopping existing cbmem-team (PID $($existing.Id))..." -ForegroundColor Yellow
    $existing | Stop-Process -Force
    Start-Sleep -Seconds 1
}

# Assemble argument list; only append MemPalace flags when a base URL is set.
$cbmemArgs = @("-mcp-bin",$mcpBin,"-data",$dataDir,"-jwt-secret",$jwtSecret,"-admin-token",$adminToken)
if ($memPalaceBase -ne "") {
    $cbmemArgs += @("-mempalace-base",$memPalaceBase)
    if ($memPalaceToken -ne "") {
        $cbmemArgs += @("-mempalace-token",$memPalaceToken)
    }
}

# Start new process
Write-Host "Starting cbmem-team..." -ForegroundColor Green
Write-Host "  mcp-bin  : $mcpBin"
Write-Host "  data     : $dataDir"
Write-Host "  listen   : :8787"
if ($memPalaceBase -ne "") {
    Write-Host "  mempalace: $memPalaceBase"
}

Start-Process -FilePath $cbmemExe `
    -ArgumentList $cbmemArgs `
    -WorkingDirectory "D:\projects\ai-dev-sop\tools\cbmem-team"

Start-Sleep -Seconds 2

$proc = Get-Process cbmem-team -ErrorAction SilentlyContinue
if ($proc) {
    Write-Host "cbmem-team started successfully (PID $($proc.Id))" -ForegroundColor Green
} else {
    Write-Host "cbmem-team failed to start. Run manually for details:" -ForegroundColor Red
    Write-Host "  & `"$cbmemExe`" $($cbmemArgs -join ' ')"
}
