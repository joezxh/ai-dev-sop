# Cursor MCP 配置指南

> **目标服务**: `http://192.168.100.83:8787`
> **协议**: HTTP + JWT Bearer Token
> **文档版本**: v1.0

---

## 1. 工作原理

```
┌─────────────────┐         HTTP + JWT          ┌─────────────────┐
│                 │  ─────────────────────────▶  │                 │
│   Cursor IDE    │                              │  cbmem-team     │
│   (MCP Client)  │  ◀─────────────────────────  │  (HTTP Server)  │
│                 │         JSON-RPC             │                 │
└─────────────────┘                              └────────┬────────┘
                                                           │
                                                           ▼
                                                 ┌─────────────────┐
                                                 │ Per-User Pool   │
                                                 │ codebase-memory │
                                                 │ -mcp subprocess │
                                                 └─────────────────┘
```

---

## 2. 前置条件

### 2.1 服务端已部署

确认服务端运行中：

```bash
curl http://192.168.100.83:8787/healthz
# 期望返回: {"status":"ok","users":N}
```

### 2.2 获取 JWT Token

需要联系管理员获取用户 JWT Token，或使用现有 Token 续期。

**方式 A: 管理员分配**

联系管理员执行：

```bash
# 管理员侧
curl -X POST "http://192.168.100.83:8787/admin/users/<your-user-id>/token?ttl=720h" \
  -H "X-Admin-Token: <admin-token>"
```

**方式 B: Token 续期（已有有效 Token）**

```bash
curl -X POST "http://192.168.100.83:8787/refresh?ttl=4320h" \
  -H "Authorization: Bearer <your-existing-jwt>"
```

---

## 3. Cursor 配置步骤

### 3.1 定位 MCP 配置文件

Windows 系统：

```
%USERPROFILE%\.cursor\mcp.json
```

即：`C:\Users\<YourUsername>\.cursor\mcp.json`

### 3.2 编辑配置

打开或创建 `mcp.json`，添加 `cbmem-team` 配置：

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://192.168.100.83:8787/mcp?as=<your-user-id>&project=<server-project-path>",
      "headers": {
        "Authorization": "Bearer <your-jwt-token>"
      }
    }
  }
}
```

### 3.3 参数说明

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `as` | 是 | 你的用户 ID，决定路由到哪个 per-user 子进程 | `alice` |
| `project` | 是 | **服务端文件系统路径**（不是本地路径） | `/var/lib/cbmem-team/users/alice/repos/myproject` |
| `Authorization` | 是 | JWT Bearer Token | `eyJhbGciOiJIUzI1NiIs...` |

### 3.4 配置示例

假设：
- 用户 ID：`developer01`
- JWT Token：`eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJkZXYiLCJleHAiOjE3NTAwMDAwMDB9.xxx`
- 服务端项目路径：`/var/lib/cbmem-team/users/developer01/projects/myapp`

完整配置：

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://192.168.100.83:8787/mcp?as=developer01&project=/var/lib/cbmem-team/users/developer01/projects/myapp",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJkZXYiLCJleHAiOjE3NTAwMDAwMDB9.xxx"
      }
    }
  }
}
```

---

## 4. 多项目配置

如果需要连接多个项目，添加多个配置项：

```json
{
  "mcpServers": {
    "cbmem-team-web": {
      "url": "http://192.168.100.83:8787/mcp?as=developer01&project=/var/lib/cbmem-team/users/developer01/projects/web-frontend",
      "headers": {
        "Authorization": "Bearer <jwt-token>"
      }
    },
    "cbmem-team-api": {
      "url": "http://192.168.100.83:8787/mcp?as=developer01&project=/var/lib/cbmem-team/users/developer01/projects/backend-api",
      "headers": {
        "Authorization": "Bearer <jwt-token>"
      }
    }
  }
}
```

---

## 5. 重启 Cursor 生效

1. **完全关闭 Cursor**（不仅是当前窗口）
2. **重新启动 Cursor**
3. 检查 Cursor 左下角 MCP 状态图标

---

## 6. 验证连接

### 6.1 服务端健康检查

```bash
curl http://192.168.100.83:8787/healthz
# 期望: {"status":"ok","users":1}
```

### 6.2 MCP 握手测试

```bash
curl -X POST "http://192.168.100.83:8787/mcp?as=developer01&project=/var/lib/cbmem-team/users/developer01/projects/myapp" \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"cursor","version":"1.0"},"capabilities":{}}}'
```

期望返回包含 `serverInfo` 和 `capabilities`。

### 6.3 Cursor 内验证

在 Cursor 中打开项目，使用 MCP 工具（如 `codebase_search`）测试：

```
Q: 搜索项目中包含 "config" 的函数
```

---

## 7. 常见问题

### Q1: 返回 401 Unauthorized

**原因**：
1. JWT Token 过期
2. JWT Secret 不匹配
3. 用户 ID 不存在

**解决**：
```bash
# 续期 Token
curl -X POST "http://192.168.100.83:8787/refresh?ttl=720h" \
  -H "Authorization: Bearer <current-token>"
```

### Q2: 返回 403 Forbidden (project not in allow-list)

**原因**：项目路径不在用户的白名单中

**解决**：联系管理员将项目路径添加到用户配置

### Q3: MCP 服务器显示离线

**排查步骤**：
1. 确认服务运行：`curl http://192.168.100.83:8787/healthz`
2. 检查 JWT Token 格式正确
3. 检查 `project` 参数使用服务端路径
4. 重启 Cursor

### Q4: Token 过期频率太高

**解决**：续期时使用更长 TTL
```bash
curl -X POST "http://192.168.100.83:8787/refresh?ttl=8640h" \
  -H "Authorization: Bearer <current-token>"
# 8640h = 360天
```

---

## 8. 安全建议

### 8.1 Token 安全

- **不要**将 Token 提交到 Git
- Token 定期轮换（建议 30 天）
- 使用环境变量存储 Token

### 8.2 生产环境

- 在 cbmem-team 前配置 Nginx/Caddy 实现 HTTPS
- 限制 CORS 源（配置 `-cors-allow-origins`）
- 使用强 JWT Secret：`openssl rand -hex 32`

### 8.3 Token 刷新机制

建议在 IDE 启动脚本中自动刷新 Token：

```bash
# 获取新 Token
NEW_TOKEN=$(curl -sS -X POST "http://192.168.100.83:8787/refresh?ttl=720h" \
  -H "Authorization: Bearer $CURRENT_TOKEN" | jq -r '.token')

# 更新配置（需要重启 IDE 生效）
```

---

## 9. 参考文档

- [cbmem-team 用户手册 (English)](./manual.en.md)
- [cbmem-team README](./README.md)
- [cbmem-team PRD](./cbmem-team-prd.md)

---

*文档更新时间: 2026-07-14*
