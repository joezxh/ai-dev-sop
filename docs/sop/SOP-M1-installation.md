# SOP-M1: 双轨记忆系统安装配置指南

> **版本**: v1.0
> **适用阶段**: 开发流程 M1 - 安装配置
> **目标读者**: 开发者、DevOps

---

## 1. 概述

### 1.1 目标

完成双轨记忆系统的完整安装配置，使团队成员能够在 30 分钟内完成环境搭建。

### 1.2 系统组件

```
┌─────────────────────────────────────────────────────────────┐
│                    双轨记忆系统                                │
├──────────────────────────┬──────────────────────────────────┤
│    轨道 A: MemPalace      │     轨道 B: codebase-mem-mcp     │
│    (人的记忆)             │     (机器的记忆)                  │
├──────────────────────────┼──────────────────────────────────┤
│  • 原文存储               │  • 代码 AST 索引                 │
│  • 语义搜索               │  • 调用链路追踪                   │
│  • 团队知识共享           │  • 架构分析                      │
└──────────────────────────┴──────────────────────────────────┘
```

### 1.3 前置条件

| 依赖 | 最低版本 | 说明 |
|------|---------|------|
| Docker | 20.10+ | 容器运行时 |
| Docker Compose | 2.0+ | 容器编排 |
| Git | 2.30+ | 代码管理 |
| Web Browser | Chrome/Firefox/Safari | Console 访问 |

---

## 2. 服务端安装

### 2.1 架构说明

```
                    ┌──────────────────────────────────────────┐
                    │              内部网络                      │
                    │                                          │
    ┌───────────────┴───────────────┐                         │
    │       192.168.100.83          │                         │
    │                               │                         │
    │  ┌─────────────────────────┐  │  ┌───────────────────┐  │
    │  │   MemPalace :8089      │  │  │  cbmem-team     │  │
    │  │   (HTTP MCP Server)    │  │  │    :8787        │  │
    │  └─────────────────────────┘  │  │  (HTTP Wrapper)  │  │
    │                                │  └─────────┬─────────┘  │
    │  ┌─────────────────────────┐  │            │            │
    │  │   MySQL :3306          │  │            │            │
    │  │   (可选: SQLite)       │  │            │            │
    │  └─────────────────────────┘  │            │            │
    │                               │            ▼            │
    │                               │  ┌───────────────────┐  │
    │                               │  │ codebase-memory   │  │
    │                               │  │ -mcp subprocess   │  │
    │                               │  └───────────────────┘  │
    └───────────────────────────────┘                         │
                                                          │
                                                          ▼
                                            ┌───────────────────────┐
                                            │      开发者 PC        │
                                            │  Cursor / Qoder IDE    │
                                            │  + MCP Client         │
                                            └───────────────────────┘
```

### 2.2 MemPalace 安装

#### 2.2.1 Docker 部署 (推荐)

```bash
# 1. 进入部署目录
cd tools/cbmem-team/deploy

# 2. 复制并配置环境变量
cp mempalace.env.example .env
# 编辑 .env 设置 MEMPALACE_PORT=8089

# 3. 启动 MemPalace
docker compose -f docker-compose.mempalace.yml up -d

# 4. 验证服务
curl http://192.168.100.83:8089/healthz
# 期望: {"status":"ok"}
```

#### 2.2.2 初始化团队 Wing

```bash
# 方式 A: 使用 CLI 初始化
docker compose -f docker-compose.mempalace.yml \
  --profile cli run mempalace-cli \
  /scripts/init-team.sh

# 方式 B: 手动创建
docker exec -it mempalace mempalace init /data

# 创建团队共享 Wing
docker exec -it mempalace mempalace create-wing \
  --name "team-shared" \
  --description "团队共享知识库"
```

#### 2.2.3 MemPalace 存储后端

| 后端 | 配置 | 适用场景 |
|------|------|----------|
| ChromaDB (默认) | 无需配置 | 开发/小团队 |
| SQLite Exact | `--backend sqlite_exact` | 正确性验证 |
| Qdrant | `MEMPALACE_QDRANT_URL` | 生产/向量搜索 |
| pgvector | `MEMPALACE_PGVECTOR_DSN` | 生产/PostgreSQL |

### 2.3 cbmem-team 安装

#### 2.3.1 编译二进制

```bash
# 1. 克隆或进入项目目录
cd tools/cbmem-team

# 2. 编译
go build -o bin/cbmem-team ./cmd/cbmem-team

# 3. 安装到系统路径
sudo install -m 0755 bin/cbmem-team /usr/local/bin/cbmem-team
```

#### 2.3.2 创建配置

```bash
# 1. 创建目录
sudo mkdir -p /etc/cbmem-team /var/lib/cbmem-team

# 2. 复制环境变量模板
sudo cp deploy/cbmem-team.env /etc/cbmem-team/cbmem-team.env

# 3. 编辑配置
sudo nano /etc/cbmem-team/cbmem-team.env
```

**配置参数说明**:

```ini
# 必须修改
JWT_SECRET=change-me-to-a-long-random-string  # 生成: openssl rand -hex 32
ADMIN_TOKEN=change-me-too                      # 生成: openssl rand -hex 32

# 根据环境调整
LISTEN=:8787
DATA_DIR=/var/lib/cbmem-team
MCP_BINARY=/usr/local/bin/codebase-memory-mcp  # 或 /home/tianque/codebase-memory-mcp

# MemPalace 集成
MEMPALACE_BASE=http://192.168.100.83:8089

# 可选: MySQL 配置
# MYSQL_DSN=  # 不设置则使用 SQLite
```

#### 2.3.3 systemd 服务安装

```bash
# 方式 A: SQLite 后端
sudo cp deploy/cbmem-team.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cbmem-team

# 方式 B: MySQL 后端 (生产推荐)
sudo cp deploy/cbmem-team-mysql.service /etc/systemd/system/
# 编辑服务文件确认 MySQL DSN 配置
sudo systemctl daemon-reload
sudo systemctl enable --now cbmem-team-mysql
```

#### 2.3.4 验证安装

```bash
# 1. 健康检查
curl http://192.168.100.83:8787/healthz
# 期望: {"status":"ok","users":0}

# 2. 查看日志
sudo journalctl -u cbmem-team -f --no-pager
```

### 2.4 MySQL 安装 (可选)

```bash
# 1. 启动 MySQL 容器
cd tools/cbmem-team/deploy
docker compose -f docker-compose.mysql.yml up -d

# 2. 配置环境变量
cp mysql.env.example /etc/cbmem-team/mysql.env
# 编辑密码

# 3. 创建数据库表
cbmem-team migrate-tables \
  -mysql-dsn "cbmem:cbmem-dev-password@tcp(127.0.0.1:3306)/cbmem"

# 4. 验证
mysql -h 127.0.0.1 -u cbmem -pcbmem-dev-password cbmem \
  -e "SHOW TABLES;"
```

### 2.5 Caddy 反向代理 (生产环境)

```bash
# 1. 安装 Caddy
sudo apt install -y caddy

# 2. 配置
sudo cp deploy/Caddyfile /etc/caddy/Caddyfile
sudo nano /etc/caddy/Caddyfile

# 3. 重载
sudo systemctl reload caddy
```

**Caddyfile 配置**:

```caddy
# API 端点
api.fin-ai.net {
    reverse_proxy 127.0.0.1:8787
    header {
        Strict-Transport-Security "max-age=31536000"
    }
}

# 静态文件 (可选)
docs.fin-ai.net {
    root * /var/www/docs.fin-ai.net
    file_server
}
```

---

## 3. 开发者客户端安装

### 3.1 codebase-memory-mcp 安装

#### 3.1.1 下载二进制

```bash
# 方式 A: 从 releases 下载
# 访问 https://github.com/DeusData/codebase-memory-mcp/releases

# 方式 B: 编译
git clone https://github.com/DeusData/codebase-memory-mcp.git
cd codebase-memory-mcp
go build -o codebase-memory-mcp .

# 方式 C: 使用 cbmem-team 内置
# cbmem-team 会在首次使用时自动管理
```

#### 3.1.2 安装到标准路径

```bash
# Linux/macOS
sudo install -m 0755 codebase-memory-mcp /usr/local/bin/codebase-memory-mcp

# Windows
# 将二进制复制到系统 PATH 中的目录
```

### 3.2 AI IDE MCP 配置

#### 3.2.1 Cursor 配置

**配置文件位置**:
- Windows: `%USERPROFILE%\.cursor\mcp.json`
- macOS/Linux: `~/.cursor/mcp.json`

**配置内容**:

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

**参数说明**:

| 参数 | 说明 | 示例 |
|------|------|------|
| `as` | 你的用户 ID | `alice` |
| `project` | 服务端文件系统路径 | `/var/lib/cbmem-team/users/alice/projects/myapp` |
| `Authorization` | JWT Bearer Token | `eyJhbGciOiJIUzI1NiIs...` |

#### 3.2.2 获取 JWT Token

```bash
# 方式 A: 联系管理员获取

# 方式 B: Token 续期 (已有 Token)
curl -X POST "http://192.168.100.83:8787/refresh?ttl=4320h" \
  -H "Authorization: Bearer $OLD_TOKEN"
```

#### 3.2.3 Qoder 配置

**配置文件位置**:
- Windows: `C:\Users\<用户名>\.qoder\mcp.json`
- macOS/Linux: `~/.qoder/mcp.json`

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

#### 3.2.4 Claude Desktop 配置

**配置文件位置**:
- macOS: `~/.config/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://192.168.100.83:8787/mcp?as=<your-user-id>&project=/path/to/project",
      "headers": {
        "Authorization": "Bearer <your-jwt-token>"
      }
    }
  }
}
```

### 3.3 MemPalace MCP 配置

#### 3.3.1 通过 Docker

```json
{
  "mcpServers": {
    "mempalace": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "-v", "mempalace-data:/data", "mempalace"]
    }
  }
}
```

#### 3.3.2 本地 Python 安装

```bash
# 安装
uv tool install mempalace
# 或
pip install mempalace

# 配置
{
  "mcpServers": {
    "mempalace": {
      "command": "mempalace",
      "args": ["mcp", "run"]
    }
  }
}
```

---

## 4. 验证检查清单

### 4.1 服务端验证

| 检查项 | 命令 | 期望结果 |
|--------|------|----------|
| MemPalace 健康 | `curl http://192.168.100.83:8089/healthz` | `{"status":"ok"}` |
| cbmem-team 健康 | `curl http://192.168.100.83:8787/healthz` | `{"status":"ok"}` |
| Console 访问 | 浏览器打开 `http://192.168.100.83:8787/console/` | 正常显示登录页 |
| MCP 端点 | `curl .../mcp?...initialize` | 返回 serverInfo |

### 4.2 客户端验证

| 检查项 | 操作 | 期望结果 |
|--------|------|----------|
| MCP 连接 | Cursor 重启后检查 MCP 状态 | cbmem-team 显示在线 |
| 工具列表 | 在 Cursor 中调用 `tools/list` | 显示可用工具 |
| 语义搜索 | 尝试 `mempalace_search` | 返回搜索结果 |

### 4.3 首次使用验证

```bash
# 1. MCP 初始化测试
curl -X POST "http://192.168.100.83:8787/mcp?as=alice&project=/test" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}'

# 2. 列出工具
curl -X POST "http://192.168.100.83:8787/mcp?as=alice&project=/test" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
```

---

## 5. 常见问题

### 5.1 服务无法启动

```bash
# 检查端口占用
netstat -tlnp | grep 8089
netstat -tlnp | grep 8787

# 检查日志
sudo journalctl -u cbmem-team -n 50 --no-pager
docker logs mempalace
```

### 5.2 JWT 认证失败

```
症状: /mcp 返回 401 Unauthorized
解决:
1. Token 过期 → 续期
2. JWT Secret 不匹配 → 检查服务端配置
3. 用户不存在 → 联系管理员创建用户
```

### 5.3 MCP 连接超时

```
症状: Cursor 显示 MCP 服务器离线
解决:
1. 确认服务端运行: curl http://192.168.100.83:8787/healthz
2. 检查网络连通性: ping 192.168.100.83
3. 确认端口开放: telnet 192.168.100.83 8787
```

### 5.4 MemPalace 搜索无结果

```
原因: 尚未挖掘数据
解决:
docker exec -it mempalace mempalace mine /data
# 或
mempalace mine ~/projects/myproject
```

---

## 6. 快速参考卡

### 6.1 服务地址

| 服务 | 地址 | 用途 |
|------|------|------|
| MemPalace | `http://192.168.100.83:8089` | 团队记忆存储 |
| cbmem-team | `http://192.168.100.83:8787` | Codebase 索引 |
| Console | `http://192.168.100.83:8787/console/` | Web 管理界面 |

### 6.2 常用命令

```bash
# 服务管理
systemctl status cbmem-team
systemctl restart cbmem-team

# 日志查看
journalctl -u cbmem-team -f

# MemPalace CLI
docker exec -it mempalace mempalace status
docker exec -it mempalace mempalace search "关键词"
```

### 6.3 配置路径

| 配置 | 路径 |
|------|------|
| cbmem-team 环境变量 | `/etc/cbmem-team/cbmem-team.env` |
| cbmem-team 数据目录 | `/var/lib/cbmem-team` |
| Cursor MCP 配置 | `~/.cursor/mcp.json` |
| MemPalace 数据 | Docker volume `mempalace-data` |

---

*文档更新: 2026-07-14*
*下一步: [SOP-M2: 项目理解](./SOP-M2-understanding.md)*
