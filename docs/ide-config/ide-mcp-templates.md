# AI IDE MCP 配置模板（mem0）

> 旧双轨记忆系统（cbmem-team / mempalace）已于 2026-09 移除，统一接入自托管 mem0。
> 云端 / 多工具 / 进阶选项详见 [quick-ref/mem0-manual.md](../quick-ref/mem0-manual.md)。

---

## 1. 通用前提

```bash
# 启动本地 mem0（API:8888 / MCP:8080 / Dashboard:3001）
cd deploy/mem0 && docker compose up -d
```

- MCP 端点：`http://127.0.0.1:8080/mcp`（Streamable HTTP）
- API Key：Dashboard `http://localhost:3001` 登录后创建，或读 `.codebuddy/mem0.config.json`
- 环境变量：`MEM0_API_KEY=m0-...`

---

## 2. Cursor 配置

**路径**：`~/.cursor/mcp.json`

```json
{
  "mcpServers": {
    "mem0": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": {
        "Authorization": "Bearer m0-你的密钥"
      }
    }
  }
}
```

## 3. Qoder 配置

**入口**：个人设置 → MCP 服务 → 配置文件添加（插件 ≥ v2.5.0，智能体模式）

```json
{
  "mcpServers": {
    "mem0": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": {
        "Authorization": "Bearer m0-你的密钥"
      }
    }
  }
}
```

## 4. CodeBuddy 配置

**入口**：Settings → MCP → Add MCP

```json
{
  "mcpServers": {
    "mem0-local": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "transport": "streamable-http"
    }
  }
}
```

> CodeBuddy 下工具调用通过 `api_key` 参数鉴权（每工具必填），凭证读 `.codebuddy/mem0.config.json`，行为规则见根 `CODEBUDDY.md`。

## 5. Codex 配置

**路径**：`~/.codex/config.toml`（`codex mcp add` 仅支持 stdio，HTTP 须手写）

```toml
[mcp_servers.mem0]
url = "http://127.0.0.1:8080/mcp"
bearer_token_env_var = "MEM0_API_KEY"
```

## 6. Claude Desktop 配置

**路径**：`%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "mem0": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": {
        "Authorization": "Bearer m0-你的密钥"
      }
    }
  }
}
```

---

## 7. 配置生成脚本

### 7.1 PowerShell (Windows)

```powershell
param([string]$ApiKey = "")
$config = @{
    mcpServers = @{
        mem0 = @{ type = "http"; url = "http://127.0.0.1:8080/mcp"
                  headers = @{ Authorization = "Bearer $ApiKey" } }
    }
}
$config | ConvertTo-Json -Depth 10 |
    Set-Content "$env:USERPROFILE\.cursor\mcp.json" -Encoding UTF8
Write-Host "Restart Cursor to apply."
```

### 7.2 Bash (Linux/macOS)

```bash
#!/bin/bash
cat > "${HOME}/.cursor/mcp.json" << EOF
{
  "mcpServers": {
    "mem0": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": { "Authorization": "Bearer \${MEM0_API_KEY}" }
    }
  }
}
EOF
echo "Config written. Restart your IDE."
```

---

## 8. 多项目 / 云端

- **云端**：把 URL 换成 `https://mcp.mem0.ai/mcp`（OAuth 或 Bearer Key）
- **多项目**：不同 Project 各自创建 API Key，一条 Key 一个项目；跨项目共享见配置手册 §8
- **stdio 替代**：`uvx mem0-mcp-server` + `MEM0_API_KEY` 环境变量（社区版已归档，可用但不再维护）

## 9. 故障排查

| 现象 | 解决 |
|------|------|
| 连接拒绝 | `docker compose -f deploy/mem0/docker-compose.yaml ps` 确认 mem0-api 运行 |
| 401 | 重新创建 Key；确认 Bearer 前缀与空格 |
| 工具列表空 | 重启 IDE；确认 URL 以 `/mcp` 结尾 |
| 记忆检索为空 | 确认 Key 与 project_id 同属一个项目；user_id 一致 |

---

*文档更新: 2026-09-18*
