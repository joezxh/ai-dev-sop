# install-all.ps1 - 双轨记忆系统一键安装脚本 (Windows)
#
# 使用方法:
#   .\install-all.ps1 [选项]
#
# 选项:
#   -SkipMemPalace    跳过 MemPalace 安装
#   -SkipCbmem        跳过 cbmem-team 安装
#   -SkipIDE          跳过 IDE 配置
#   -ServerUrl URL    指定 cbmem-team 服务地址
#   -Help             显示帮助

param(
    [switch]$SkipMemPalace,
    [switch]$SkipCbmem,
    [switch]$SkipIDE,
    [string]$ServerUrl = "http://192.168.100.83:8787",
    [string]$MemPalaceUrl = "http://192.168.100.83:8089",
    [switch]$Help
)

# 帮助信息
if ($Help) {
    Write-Host @"
双轨记忆系统一键安装脚本

使用方法:
    .\install-all.ps1 [选项]

选项:
    -SkipMemPalace    跳过 MemPalace 安装
    -SkipCbmem        跳过 cbmem-team 安装
    -SkipIDE          跳过 IDE 配置
    -ServerUrl URL    指定 cbmem-team 服务地址 (默认: $ServerUrl)
    -MemPalaceUrl URL 指定 MemPalace 服务地址 (默认: $MemPalaceUrl)
    -Help             显示此帮助

示例:
    .\install-all.ps1                          # 完整安装
    .\install-all.ps1 -SkipMemPalace          # 仅安装 cbmem-team
    .\install-all.ps1 -ServerUrl "http://localhost:8787"  # 使用本地服务

"@
    exit 0
}

# 颜色定义
function Write-Success { param($msg) Write-Host "✓ $msg" -ForegroundColor Green }
function Write-Info { param($msg) Write-Host "ℹ $msg" -ForegroundColor Yellow }
function Write-Error { param($msg) Write-Host "✗ $msg" -ForegroundColor Red }
function Write-Title { param($msg) Write-Host ""; Write-Host "========================================"; Write-Host "  $msg"; Write-Host "========================================"; Write-Host "" }

# 检查命令
function Test-Command {
    param($cmd)
    $null = Get-Command $cmd -ErrorAction SilentlyContinue
    return $null -ne $null
}

# 检查端口
function Test-Port {
    param($port, $service)
    $connection = Test-NetConnection -ComputerName localhost -Port $port -WarningAction SilentlyContinue
    if ($connection.TcpTestSucceeded) {
        Write-Info "$service (端口 $port) 已在运行，跳过"
        return $true
    }
    return $false
}

# ========== MemPalace 安装 ==========
function Install-MemPalace {
    if ($SkipMemPalace) {
        Write-Info "跳过 MemPalace 安装"
        return
    }

    Write-Title "安装 MemPalace"

    # 检查 Docker
    if (-not (Test-Command "docker")) {
        Write-Error "Docker 未安装"
        Write-Host "请先安装 Docker: https://docs.docker.com/desktop/install/windows-install/"
        return
    }
    Write-Success "Docker 已安装"

    # 检查端口
    if (-not (Test-Port 8089 "MemPalace")) {
        Write-Info "启动 MemPalace..."

        $deployDir = Join-Path $PSScriptRoot "tools\cbmem-team\deploy"

        # 复制环境变量
        $envFile = Join-Path $deployDir ".env"
        $envExample = Join-Path $deployDir "mempalace.env.example"
        if (-not (Test-Path $envFile)) {
            Copy-Item $envExample $envFile
            Write-Info "已创建 .env 文件，请编辑设置 MEMPALACE_PORT=8089"
        }

        # 启动服务
        Push-Location $deployDir
        try {
            docker compose -f docker-compose.mempalace.yml up -d
            Write-Info "等待 MemPalace 启动..."
            Start-Sleep -Seconds 5

            # 验证
            try {
                $response = Invoke-WebRequest -Uri "$MemPalaceUrl/healthz" -UseBasicParsing -TimeoutSec 5 -ErrorAction SilentlyContinue
                if ($response.StatusCode -eq 200) {
                    Write-Success "MemPalace 安装成功"
                }
            } catch {
                Write-Error "MemPalace 启动失败，请检查日志"
                docker compose -f docker-compose.mempalace.yml logs
            }
        } finally {
            Pop-Location
        }
    }
}

# ========== cbmem-team 安装 ==========
function Install-CbmemTeam {
    if ($SkipCbmem) {
        Write-Info "跳过 cbmem-team 安装"
        return
    }

    Write-Title "安装 cbmem-team"

    # 检查 Go
    if (-not (Test-Command "go")) {
        Write-Error "Go 未安装"
        Write-Host "请先安装 Go: https://go.dev/dl/"
        return
    }
    Write-Success "Go 已安装"

    # 检查端口
    if (-not (Test-Port 8787 "cbmem-team")) {
        Write-Info "编译 cbmem-team..."

        $projectDir = Join-Path $PSScriptRoot "tools\cbmem-team"

        # 编译
        Push-Location $projectDir
        try {
            go build -o bin/cbmem-team ./cmd/cbmem-team
            Write-Success "编译成功"
        } finally {
            Pop-Location
        }

        # 复制到 PATH
        $targetPath = "$env:LOCALAPPDATA\Programs\cbmem-team.exe"
        Copy-Item (Join-Path $projectDir "bin\cbmem-team.exe") $targetPath -Force
        Write-Info "已安装到: $targetPath"

        # 提示配置
        Write-Host ""
        Write-Host "注意: 请手动配置并启动 cbmem-team 服务"
        Write-Host "  - 配置: tools\cbmem-team\deploy\cbmem-team.env"
        Write-Host "  - 服务: tools\cbmem-team\deploy\cbmem-team.service"
    }
}

# ========== IDE 配置 ==========
function Configure-IDE {
    if ($SkipIDE) {
        Write-Info "跳过 IDE 配置"
        return
    }

    Write-Title "配置 AI IDE MCP"

    $ideDetected = $false

    # 检测 Cursor
    $cursorPath = "$env:USERPROFILE\.cursor"
    if (Test-Path $cursorPath) {
        $ideDetected = $true
        Write-Info "检测到 Cursor"

        $configDir = $cursorPath
        $configFile = Join-Path $configDir "mcp.json"

        $config = @{
            mcpServers = @{
                "cbmem-team" = @{
                    url = "$ServerUrl/mcp?as=`$USER_ID&project=`$SERVER_PROJECT_PATH"
                    headers = @{
                        "Authorization" = "Bearer `$JWT_TOKEN"
                    }
                }
            }
        }

        $config | ConvertTo-Json -Depth 10 | Set-Content $configFile -Encoding UTF8
        Write-Success "MCP 配置已创建: $configFile"
    }

    # 检测 Qoder
    $qoderPath = "$env:USERPROFILE\.qoder"
    if (Test-Path $qoderPath) {
        $ideDetected = $true
        Write-Info "检测到 Qoder"

        $configFile = Join-Path $qoderPath "mcp.json"

        $config = @{
            mcpServers = @{
                "cbmem-team" = @{
                    url = "$ServerUrl/mcp?as=`$USER_ID&project=`$SERVER_PROJECT_PATH"
                    headers = @{
                        "Authorization" = "Bearer `$JWT_TOKEN"
                    }
                }
            }
        }

        $config | ConvertTo-Json -Depth 10 | Set-Content $configFile -Encoding UTF8
        Write-Success "MCP 配置已创建: $configFile"
    }

    if (-not $ideDetected) {
        Write-Info "未检测到支持的 IDE (Cursor/Qoder)，跳过"
        return
    }

    Write-Host ""
    Write-Info "请编辑配置文件填入正确的 USER_ID, SERVER_PROJECT_PATH 和 JWT_TOKEN"
    Write-Info "然后重启 IDE 生效"
}

# ========== 验证安装 ==========
function Verify-Install {
    Write-Title "验证安装"

    if (-not $SkipMemPalace) {
        Write-Info "检查 MemPalace..."
        try {
            $response = Invoke-WebRequest -Uri "$MemPalaceUrl/healthz" -UseBasicParsing -TimeoutSec 5 -ErrorAction SilentlyContinue
            if ($response.StatusCode -eq 200) {
                Write-Success "MemPalace: $MemPalaceUrl"
            }
        } catch {
            Write-Error "MemPalace: $MemPalaceUrl"
        }
    }

    if (-not $SkipCbmem) {
        Write-Info "检查 cbmem-team..."
        try {
            $response = Invoke-WebRequest -Uri "$ServerUrl/healthz" -UseBasicParsing -TimeoutSec 5 -ErrorAction SilentlyContinue
            if ($response.StatusCode -eq 200) {
                Write-Success "cbmem-team: $ServerUrl"
            }
        } catch {
            Write-Error "cbmem-team: $ServerUrl"
        }
    }
}

# ========== 主流程 ==========
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  双轨记忆系统一键安装" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Install-MemPalace
Install-CbmemTeam
Configure-IDE
Verify-Install

Write-Title "安装完成"
Write-Host "下一步:" -ForegroundColor Green
Write-Host "  1. 获取 JWT Token (联系管理员)"
Write-Host "  2. 编辑 ~/.cursor/mcp.json 填入 Token"
Write-Host "  3. 重启 Cursor/IDE"
Write-Host ""
Write-Host "详细文档: docs/sop/SOP-M1-installation.md" -ForegroundColor Cyan
