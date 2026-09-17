# 启动 mem0-api（本机原生部署，非 Docker）
#
# 之前手工执行 `python -m uvicorn main:app --port 8002` 时未指定 --host，
# uvicorn 默认只绑 127.0.0.1，导致只有本机可访问。这里显式绑定 0.0.0.0，
# 使局域网/远端可以访问。
#
# 用法：
#   .\start-api.ps1                                       # 0.0.0.0:8002
#   $env:MEM0_API_HOST = '127.0.0.1'; .\start-api.ps1     # 仅本机（安全）
#   $env:MEM0_API_PORT = '8002';      .\start-api.ps1
#
# 安全提示：AUTH_DISABLED=true 时接口无鉴权，绑定 0.0.0.0 会导致任何人可
# 读写全部记忆，且 GET /configure 会泄露数据库密码与 LLM Key。共享网络下
# 请开启鉴权（AUTH_DISABLED=false）并用防火墙限制来源，或改用 SSH 隧道。

$apiHost = if ($env:MEM0_API_HOST) { $env:MEM0_API_HOST } else { '0.0.0.0' }
$apiPort = if ($env:MEM0_API_PORT) { $env:MEM0_API_PORT } else { '8002' }

$python = if ($env:MEM0_PYTHON) {
    $env:MEM0_PYTHON
} else {
    'D:\projects\ai-dev-sop\mem0\.venv\Scripts\python.exe'
}

$serverDir = Resolve-Path (Join-Path $PSScriptRoot '..\..\mem0\server')

# Load mem0/server/.env into the process environment so uvicorn sees
# AUTH_DISABLED/JWT_SECRET/POSTGRES_* etc. (main.py's load_dotenv can miss these
# under a detached Start-Process launch). Only fills vars that aren't already set.
$envFile = Join-Path $serverDir '.env'
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$') {
            $k = $Matches[1]; $v = $Matches[2]
            if (-not (Test-Path "env:$k")) { Set-Item -Path "env:$k" -Value $v }
        }
    }
}

# Ensure the in-repo mem0 library (submodule) shadows the venv's pinned copy so
# main.py's newer config keys (e.g. embedder send_dimensions) are supported.
$env:PYTHONPATH = (Resolve-Path (Join-Path $PSScriptRoot '..\..\mem0')).Path

$authDisabled = $env:AUTH_DISABLED -and (@('1', 'true', 'yes', 'on') -contains $env:AUTH_DISABLED.ToLower())
if ($authDisabled -and $apiHost -ne '127.0.0.1') {
    Write-Warning "AUTH_DISABLED 已开启，且正在绑定到 $apiHost：接口将无鉴权暴露到网络（GET /configure 会泄露数据库密码与 LLM Key）。请仅在可信内网使用，或先设 AUTH_DISABLED=false。"
}

Write-Host "Starting mem0-api -> ${apiHost}:${apiPort}  (cwd: $serverDir)"
Push-Location $serverDir
try {
    & $python -m uvicorn main:app --host $apiHost --port $apiPort --log-level debug
}
finally {
    Pop-Location
}
