# AI IDE MCP 配置模板

> 包含 Cursor、Qoder、Claude Desktop 的 MCP 配置模板

---

## 1. Cursor 配置

### 1.1 配置文件

**路径**: `~/.cursor/mcp.json` (Windows: `C:\Users\<用户名>\.cursor\mcp.json`)

### 1.2 完整配置模板

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://192.168.100.83:8787/mcp?as={{USER_ID}}&project={{SERVER_PROJECT_PATH}}",
      "headers": {
        "Authorization": "Bearer {{JWT_TOKEN}}"
      }
    },
    "mempalace": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "ghcr.io/mempalace/mempalace:latest"]
    }
  }
}
```

### 1.3 本地 MemPalace 配置

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://192.168.100.83:8787/mcp?as={{USER_ID}}&project={{SERVER_PROJECT_PATH}}",
      "headers": {
        "Authorization": "Bearer {{JWT_TOKEN}}"
      }
    },
    "mempalace": {
      "command": "mempalace",
      "args": ["mcp", "run"]
    }
  }
}
```

### 1.4 配置说明

| 占位符 | 说明 | 示例 |
|--------|------|------|
| `{{USER_ID}}` | 你的用户 ID | `alice` |
| `{{SERVER_PROJECT_PATH}}` | 服务端项目路径 | `/var/lib/cbmem-team/users/alice/projects/myapp` |
| `{{JWT_TOKEN}}` | JWT Token | `eyJhbGciOiJIUzI1NiIs...` |

---

## 2. Qoder 配置

### 2.1 配置文件

**路径**: `~/.qoder/mcp.json` (Windows: `C:\Users\<用户名>\.qoder\mcp.json`)

### 2.2 完整配置模板

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://192.168.100.83:8787/mcp?as={{USER_ID}}&project={{SERVER_PROJECT_PATH}}",
      "headers": {
        "Authorization": "Bearer {{JWT_TOKEN}}"
      }
    },
    "mempalace": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "ghcr.io/mempalace/mempalace:latest"]
    }
  }
}
```

### 2.3 Qoder 设置

在 Qoder 中启用 MCP:

1. 打开设置 (`Ctrl + ,`)
2. 导航至「扩展」或「Plugins」
3. 确保 MCP Client 插件已启用
4. 重启 Qoder

---

## 3. Claude Desktop 配置

### 3.1 配置文件

**路径**:
- macOS: `~/.config/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

### 3.2 完整配置模板

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://192.168.100.83:8787/mcp?as={{USER_ID}}&project={{SERVER_PROJECT_PATH}}",
      "headers": {
        "Authorization": "Bearer {{JWT_TOKEN}}"
      }
    },
    "mempalace": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "ghcr.io/mempalace/mempalace:latest"]
    }
  }
}
```

---

## 4. 配置生成脚本

### 4.1 PowerShell 脚本 (Windows)

```powershell
# Generate-CursorMcpConfig.ps1
param(
    [string]$UserId = "alice",
    [string]$ServerUrl = "http://192.168.100.83:8787",
    [string]$ServerProjectPath = "/var/lib/cbmem-team/users/alice/projects/myapp",
    [string]$JwtToken = ""
)

$config = @{
    mcpServers = @{
        "cbmem-team" = @{
            url = "$ServerUrl/mcp?as=$UserId&project=$ServerProjectPath"
            headers = @{
                "Authorization" = "Bearer $JwtToken"
            }
        }
    }
}

$configPath = "$env:USERPROFILE\.cursor\mcp.json"
$config | ConvertTo-Json -Depth 10 | Set-Content $configPath -Encoding UTF8

Write-Host "Config written to: $configPath"
Write-Host "Please restart Cursor to apply changes."
```

**使用**:

```powershell
# 生成配置 (手动填入 JWT Token)
notepad $env:USERPROFILE\.cursor\mcp.json

# 或使用脚本 (需要预先获取 Token)
.\Generate-CursorMcpConfig.ps1 -UserId "alice" -JwtToken "your-token"
```

### 4.2 Bash 脚本 (Linux/macOS)

```bash
#!/bin/bash
# generate-mcp-config.sh

USER_ID="${1:-alice}"
SERVER_URL="${2:-http://192.168.100.83:8787}"
SERVER_PROJECT_PATH="${3:-/var/lib/cbmem-team/users/alice/projects/myapp}"
JWT_TOKEN="${4:-}"

CONFIG_DIR="${HOME}/.cursor"
CONFIG_FILE="${CONFIG_DIR}/mcp.json"

mkdir -p "$CONFIG_DIR"

cat > "$CONFIG_FILE" << EOF
{
  "mcpServers": {
    "cbmem-team": {
      "url": "${SERVER_URL}/mcp?as=${USER_ID}&project=${SERVER_PROJECT_PATH}",
      "headers": {
        "Authorization": "Bearer ${JWT_TOKEN}"
      }
    }
  }
}
EOF

echo "Config written to: $CONFIG_FILE"
echo "Please restart Cursor to apply changes."
```

**使用**:

```bash
# 基础使用
./generate-mcp-config.sh alice

# 完整参数
./generate-mcp-config.sh alice http://192.168.100.83:8787 /path/to/project "eyJhbGci..."

# 或手动编辑
nano ~/.cursor/mcp.json
```

---

## 5. 多项目配置

### 5.1 同时连接多个项目

```json
{
  "mcpServers": {
    "cbmem-team-web": {
      "url": "http://192.168.100.83:8787/mcp?as=alice&project=/var/lib/cbmem-team/users/alice/projects/web-frontend",
      "headers": {
        "Authorization": "Bearer {{JWT_TOKEN}}"
      }
    },
    "cbmem-team-api": {
      "url": "http://192.168.100.83:8787/mcp?as=alice&project=/var/lib/cbmem-team/users/alice/projects/backend-api",
      "headers": {
        "Authorization": "Bearer {{JWT_TOKEN}}"
      }
    },
    "mempalace": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "ghcr.io/mempalace/mempalace:latest"]
    }
  }
}
```

### 5.2 项目别名说明

| 别名 | 服务端路径 | 用途 |
|------|-----------|------|
| `cbmem-team-web` | `/projects/web-frontend` | 前端项目 |
| `cbmem-team-api` | `/projects/backend-api` | 后端 API 项目 |

---

## 6. 故障排查

### 6.1 MCP 服务器离线

```
排查步骤:
1. 确认服务端运行: curl http://192.168.100.83:8787/healthz
2. 检查 JWT Token 是否过期
3. 确认 URL 参数格式正确
4. 重启 Cursor IDE
```

### 6.2 Token 刷新

```bash
# 刷新 Token (30 天有效期)
curl -X POST "http://192.168.100.83:8787/refresh?ttl=720h" \
  -H "Authorization: Bearer $OLD_TOKEN"

# 更新 mcp.json 中的 Authorization header
# 重启 Cursor
```

---

*文档更新: 2026-07-14*
