# cbmem-team Windows startup script
# Usage: .\start.ps1
# Stops any existing cbmem-team process and starts a new one with correct paths.

$ErrorActionPreference = "Stop"

$cbmemExe   = "D:\projects\ai-dev-sop\tools\cbmem-team\cbmem-team.exe"
$mcpBin     = "C:\Users\joezxh\AppData\Local\Programs\codebase-memory-mcp\codebase-memory-mcp.exe"
$dataDir    = "D:\projects\ai-dev-sop\tmp\cbmem-data"
$jwtSecret  = "dev-jwt-secret-please-rotate"
$adminToken = "change-me-too"

# Stop existing process
$existing = Get-Process cbmem-team -ErrorAction SilentlyContinue
if ($existing) {
    Write-Host "Stopping existing cbmem-team (PID $($existing.Id))..." -ForegroundColor Yellow
    $existing | Stop-Process -Force
    Start-Sleep -Seconds 1
}

# Start new process
Write-Host "Starting cbmem-team..." -ForegroundColor Green
Write-Host "  mcp-bin : $mcpBin"
Write-Host "  data    : $dataDir"
Write-Host "  listen  : :8787"

Start-Process -FilePath $cbmemExe `
    -ArgumentList "-mcp-bin",$mcpBin,"-data",$dataDir,"-jwt-secret",$jwtSecret,"-admin-token",$adminToken `
    -WorkingDirectory "D:\projects\ai-dev-sop\tools\cbmem-team"

Start-Sleep -Seconds 2

$proc = Get-Process cbmem-team -ErrorAction SilentlyContinue
if ($proc) {
    Write-Host "cbmem-team started successfully (PID $($proc.Id))" -ForegroundColor Green
} else {
    Write-Host "cbmem-team failed to start. Run manually for details:" -ForegroundColor Red
    Write-Host "  & `"$cbmemExe`" -mcp-bin `"$mcpBin`" -data `"$dataDir`" -jwt-secret `"$jwtSecret`" -admin-token `"$adminToken`""
}
