# install-all.ps1 - mem0 记忆系统一键安装脚本 (Windows)
#
# 旧双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，
# 记忆功能统一由自托管 mem0 提供（deploy/mem0）。
#
# 使用方法:
#   .\install-all.ps1 [-SkipMem0] [-SkipIde] [-McpUrl "http://127.0.0.1:8080/mcp"]

param(
    [switch]$SkipMem0,
    [switch]$SkipIde,
    [string]$McpUrl = "http://127.0.0.1:8080/mcp"
)

$ErrorActionPreference = "Stop"

function Write-Header([string]$t) {
    Write-Host ""; Write-Host "========================================" -ForegroundColor Blue
    Write-Host "  $t" -ForegroundColor Blue
    Write-Host "========================================" -ForegroundColor Blue; Write-Host ""
}
function Write-Info([string]$m)    { Write-Host "i $m" -ForegroundColor Yellow }
function Write-Ok([string]$m)      { Write-Host "`u{2713} $m" -ForegroundColor Green }
function Write-Fail([string]$m)    { Write-Host "x $m" -ForegroundColor Red }

# ========== mem0 服务安装 ==========
function Install-Mem0 {
    if ($SkipMem0) { Write-Info "跳过 mem0 服务安装"; return }

    Write-Header "安装 mem0"

    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Fail "请先安装 Docker: https://docs.docker.com/get-docker/"; exit 1
    }

    $listening = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction SilentlyContinue
    if (-not $listening) {
        Write-Info "启动 mem0 (API:8888 / MCP:8080 / Dashboard:3001)..."
        $deployDir = Join-Path $PSScriptRoot "..\deploy\mem0"
        if (-not (Test-Path (Join-Path $deployDir ".env"))) {
            Write-Fail "缺少 deploy/mem0/.env，请按 docker-compose.yaml 注释准备环境变量"; exit 1
        }
        Push-Location $deployDir
        docker compose up -d
        Write-Info "等待 mem0 启动..."
        Start-Sleep -Seconds 8
        try {
            Invoke-WebRequest "http://localhost:8888/docs" -UseBasicParsing -TimeoutSec 10 | Out-Null
            Write-Ok "mem0 安装成功"
        } catch {
            Write-Fail "mem0 启动失败，请检查日志: docker compose logs --tail 50"
        }
        Pop-Location
    } else {
        Write-Info "mem0-api (端口 8080) 已在运行，跳过"
    }
}

# ========== IDE MCP 配置 ==========
function Set-IdeConfig {
    if ($SkipIde) { Write-Info "跳过 IDE 配置"; return }

    Write-Header "配置 AI IDE MCP"

    $targets = @()
    if (Test-Path "$env:USERPROFILE\.cursor") { $targets += ".cursor" }
    if (Test-Path "$env:USERPROFILE\.qoder")  { $targets += ".qoder" }

    if ($targets.Count -eq 0) {
        Write-Info "未检测到支持的 IDE (Cursor/Qoder)，跳过"; return
    }

    foreach ($dir in $targets) {
        $configFile = "$env:USERPROFILE\$dir\mcp.json"
        $config = @{
            mcpServers = @{
                mem0 = @{
                    type    = "http"
                    url     = $McpUrl
                    headers = @{ Authorization = "Bearer `${MEM0_API_KEY}" }
                }
            }
        }
        New-Item -ItemType Directory -Force -Path (Split-Path $configFile) | Out-Null
        $config | ConvertTo-Json -Depth 10 | Set-Content $configFile -Encoding UTF8
        Write-Ok "MCP 配置已创建: $configFile"
    }
    Write-Info "请确保环境变量 MEM0_API_KEY 已设置（或手动替换占位符），然后重启 IDE"
}

# ========== 验证 ==========
function Test-Install {
    Write-Header "验证安装"
    if (-not $SkipMem0) {
        try {
            Invoke-WebRequest "http://localhost:8888/docs" -UseBasicParsing -TimeoutSec 5 | Out-Null
            Write-Ok "mem0 API: http://localhost:8888"
        } catch { Write-Fail "mem0 API: http://localhost:8888 未响应" }
        try {
            Invoke-WebRequest "http://localhost:3001" -UseBasicParsing -TimeoutSec 5 | Out-Null
            Write-Ok "mem0 Dashboard: http://localhost:3001"
        } catch { Write-Info "mem0 Dashboard: http://localhost:3001 未响应（可选）" }
    }
}

# ========== 主流程 ==========
Write-Header "mem0 记忆系统一键安装"
Install-Mem0
Set-IdeConfig
Test-Install

Write-Header "安装完成"
Write-Host "下一步:"
Write-Host "  1. 在 Dashboard (http://localhost:3001) 创建 API Key"
Write-Host "  2. 设置环境变量 MEM0_API_KEY 或编辑 IDE mcp.json"
Write-Host "  3. 重启 IDE，验证 MCP 工具可用"
Write-Host ""
Write-Host "详细文档: docs/quick-ref/mem0-manual.md"
Write-Host "  4. 接入 IDE：运行 scripts\mem0-setup.ps1（一键配置 MCP + 规则文件 + 凭证并校验）"
