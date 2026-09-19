# =============================================================
# Mem0 远程部署脚本（Windows / PowerShell）
#
# 完整流程：
#   1) 本地用 deploy/mem0/api/Dockerfile 构建 mem0-api 镜像
#   2) docker save 把镜像打成 tar
#   3) pscp 通过 SSH 把 tar 传到服务器 /tmp
#   4) plink 远程执行 docker load + docker compose up -d
#
# 前置条件：
#   - 本机已安装 Docker（docker / docker compose）
#   - 服务器已安装 Docker，且当前用户(默认 deploy)在 docker 组或有免密 sudo
#   - d:\tools\plink.exe 与 d:\tools\pscp.exe 可用
#
# 用法：
#   .\deploy.ps1                       # 仅部署 API+MCP+Postgres
#   .\deploy.ps1 -Dashboard            # 同时构建并部署 Dashboard
#   .\deploy.ps1 -SkipBuild            # 跳过本地构建，仅传输/运行（镜像已存在时）
#   .\deploy.ps1 -Sudo                 # 远程 docker 命令前加 sudo
#
# 注意：本脚本把服务器密码明文写在下面变量里（按你提供的信息）。
#       该文件含密钥，请勿提交到 git；生产环境建议改用 SSH 密钥登录。
# =============================================================

[CmdletBinding()]
param(
    [switch]$Dashboard,   # 是否一并构建/部署 Dashboard 镜像
    [switch]$SkipBuild,   # 跳过本地 docker build（镜像已在本地存在）
    [switch]$Sudo         # 远程命令前加 sudo（服务器 docker 需 sudo 时）
)

$ErrorActionPreference = "Stop"

# ---------------- 连接与路径配置 ----------------
$Server   = "192.168.110.169"
$User     = "deploy"
$Password = "$env:MEM0_DEPLOY_PASSWORD"                   # 部署密码，从环境变量 MEM0_DEPLOY_PASSWORD 读取
$Plink    = "d:\tools\plink.exe"
$Pscp     = "d:\tools\pscp.exe"

# 服务器 SSH 主机密钥指纹（首次连接时由 ssh-keyscan / plink 获取）。
# 钉死后可彻底避免 plink 弹出的 host key (y/n) 交互提示，脚本无需人工确认。
$HostKey  = "SHA256:QTzyynU1QFiWlkjD3mzaasBiwEIBMG43bh5OovOBJvM"

$RepoRoot        = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot "..\.."))
$Mem0ServerDir   = Join-Path $RepoRoot "mem0\server"
$ApiDockerfile   = Join-Path $RepoRoot "deploy\mem0\api\Dockerfile"
$DashDockerfile  = Join-Path $RepoRoot "deploy\mem0\dashboard\Dockerfile"
$RemoteDirLocal  = Join-Path $PSScriptRoot "remote"
$RemoteDir       = "/home/deploy/mem0-remote"
$TarLocal        = Join-Path $env:TEMP "mem0-images.tar"
$TarRemote       = "/tmp/mem0-images.tar"

$ApiImage  = "mem0-remote-mem0-api:latest"
$DashImage = "mem0-remote-mem0-dashboard:latest"
# 远程使用外部 PostgreSQL/Neo4j，无需打包 pgvector 镜像。

# ---------------- 辅助函数 ----------------
function Write-Step($n, $msg) {
    Write-Host "`n[$n] $msg" -ForegroundColor Cyan
}

# 远程执行命令（-batch 禁交互，-hostkey 钉死指纹，避免 host key (y/n) 卡死）
function Invoke-Remote($Command) {
    $dockerCmd = if ($Sudo) { "sudo docker" } else { "docker" }
    # 用变量替换内部 docker 关键字，避免远程命令里出现需要转义的字符
    $cmd = $Command -replace '\bdocker\b', $dockerCmd
    Write-Host "  -> plink: $cmd" -ForegroundColor Gray
    & $Plink -batch -hostkey $HostKey -pw $Password "$User@$Server" $cmd
    if ($LASTEXITCODE -ne 0) { throw "远程命令执行失败 (exit $LASTEXITCODE): $cmd" }
}

# 传输文件/目录到服务器
function Send-Remote($Local, $Remote, [switch]$Recurse) {
    $args = @("-batch", "-hostkey", $HostKey, "-pw", $Password)
    if ($Recurse) { $args += "-r" }
    $args += @($Local, "$User@$Server`:$Remote")
    Write-Host "  -> pscp $Local -> $User@$Server`:$Remote" -ForegroundColor Gray
    & $Pscp @args
    if ($LASTEXITCODE -ne 0) { throw "文件传输失败 (exit $LASTEXITCODE)" }
}

# ---------------- 0. 前置检查 ----------------
Write-Step "0/6" "前置检查"
if (-not (Test-Path $Plink)) { throw "找不到 plink: $Plink" }
if (-not (Test-Path $Pscp)) { throw "找不到 pscp: $Pscp" }
if (-not (Test-Path $ApiDockerfile)) { throw "找不到 API Dockerfile: $ApiDockerfile" }
$dockerOk = Get-Command docker -ErrorAction SilentlyContinue
if (-not $dockerOk) { throw "本机未安装 Docker，请先安装。" }

# 远程 .env 必须存在（含真实密钥），否则拒绝部署
$EnvLocal = Join-Path $RemoteDirLocal ".env"
if (-not (Test-Path $EnvLocal)) {
    throw "远程 .env 不存在：$EnvLocal`n请先复制 .env.example 为 .env 并填写 POSTGRES_PASSWORD / OPENAI_API_KEY 等。"
}

# ---------------- 1. 本地构建镜像 ----------------
if (-not $SkipBuild) {
    Write-Step "1/6" "本地构建镜像"
    Write-Host "  -> 构建 $ApiImage (上下文: $Mem0ServerDir)" -ForegroundColor Yellow
    docker build -f $ApiDockerfile -t $ApiImage $Mem0ServerDir
    if ($LASTEXITCODE -ne 0) { throw "API 镜像构建失败" }

    if ($Dashboard) {
        Write-Host "  -> 构建 $DashImage (上下文: $($Mem0ServerDir)\dashboard)" -ForegroundColor Yellow
        docker build -f $DashDockerfile -t $DashImage (Join-Path $Mem0ServerDir "dashboard")
        if ($LASTEXITCODE -ne 0) { throw "Dashboard 镜像构建失败" }
    }
} else {
    Write-Step "1/6" "跳过本地构建（SkipBuild）"
}

# 远程使用外部数据库，无需打包 pgvector 镜像。

# ---------------- 2. 打包镜像为 tar ----------------
Write-Step "2/6" "docker save 打包镜像"
$images = @($ApiImage)
if ($Dashboard) { $images += $DashImage }
docker save -o $TarLocal $images
if ($LASTEXITCODE -ne 0) { throw "docker save 失败" }
$tarSize = [math]::Round((Get-Item $TarLocal).Length / 1MB, 1)
Write-Host "  打包完成：$TarLocal ($tarSize MB)" -ForegroundColor Green

# ---------------- 3. 传输 tar 到服务器 ----------------
Write-Step "3/6" "pscp 传输镜像到服务器"
Send-Remote $TarLocal $TarRemote

# ---------------- 4. 传输 compose / .env / init-db.sh ----------------
Write-Step "4/6" "pscp 传输部署文件"
Invoke-Remote "mkdir -p $RemoteDir"
# 复制 remote 目录下所有文件（含 .env / docker-compose.yaml / init-db.sh / .env.example）
Send-Remote "$RemoteDirLocal\*" "$RemoteDir/"

# ---------------- 5. 远程加载镜像 ----------------
Write-Step "5/6" "远程 docker load"
Invoke-Remote "docker load -i $TarRemote"

# ---------------- 6. 远程启动容器 ----------------
Write-Step "6/6" "远程 docker compose up -d"
$profiles = ""
if ($Dashboard) { $profiles += " --profile dash" }
Invoke-Remote "cd $RemoteDir && docker compose up -d$profiles"
# 清理服务器上的 tar，避免占盘
Invoke-Remote "rm -f $TarRemote"
# 等待并查看状态
Start-Sleep -Seconds 5
Invoke-Remote "cd $RemoteDir && docker compose ps"

# ---------------- 完成 ----------------
# 清理本地 tar
Remove-Item $TarLocal -Force -ErrorAction SilentlyContinue

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  Mem0 远程部署完成！" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  REST API : http://${Server}:8888/docs" -ForegroundColor White
Write-Host "  MCP      : http://${Server}:8080/mcp" -ForegroundColor White
if ($Dashboard) { Write-Host "  Dashboard : http://${Server}:3001" -ForegroundColor White }
Write-Host ""
Write-Host "  本地 CodeBuddy 连接远程 MCP：把 ~/.codebuddy/mcp.json 中" -ForegroundColor Gray
Write-Host "  mem0-local 的 url 改为 http://${Server}:8080/mcp 即可。" -ForegroundColor Gray
Write-Host ""
Write-Host "  查看日志：ssh $User@$Server 'cd $RemoteDir && docker compose logs -f'" -ForegroundColor Gray
