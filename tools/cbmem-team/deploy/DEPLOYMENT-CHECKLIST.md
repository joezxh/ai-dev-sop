# 双轨记忆系统部署检查清单

> **版本**: v1.0
> **目标环境**: `http://192.168.100.83`
> **日期**: 2026-07-14

---

## 1. 基础设施概览

```
┌─────────────────────────────────────────────────────────────────┐
│                     部署架构                                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  MemPalace  │    │ cbmem-team │    │   MySQL    │         │
│  │   :8089     │    │   :8787    │    │   :3306    │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                                                                  │
│  服务地址:                                                        │
│  - MemPalace: http://192.168.100.83:8089                        │
│  - cbmem-team: http://192.168.100.83:8787                      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 组件检查清单

### 2.1 MemPalace 服务

| 检查项 | 状态 | 命令/说明 |
|--------|------|----------|
| MemPalace 容器运行 | ⬜ | `docker ps | grep mempalace` |
| 健康检查 | ⬜ | `curl http://192.168.100.83:8089/healthz` |
| 数据卷存在 | ⬜ | `docker volume ls | grep mempalace` |
| MCP 工具列表 | ⬜ | `echo '{"method":"tools/list"}' \| docker exec -i mempalace curl -s -X POST http://localhost:8089/mcp -H "Content-Type: application/json" -d @-` |

### 2.2 cbmem-team 服务

| 检查项 | 状态 | 命令/说明 |
|--------|------|----------|
| 服务进程运行 | ⬜ | `curl http://192.168.100.83:8787/healthz` |
| JWT Secret 配置 | ⬜ | 检查 systemd 环境变量或配置文件 |
| Admin Token 配置 | ⬜ | 检查 systemd 环境变量或配置文件 |
| MemPalace 集成 | ⬜ | 确认 `-mempalace-base http://192.168.100.83:8089` 配置 |
| MySQL 连接 | ⬜ | `curl http://192.168.100.83:8787/admin/stats -H "X-Admin-Token: xxx"` |
| Console 前端 | ⬜ | 访问 `http://192.168.100.83:8787/console/` |

### 2.3 codebase-memory-mcp

| 检查项 | 状态 | 命令/说明 |
|--------|------|----------|
| 二进制存在 | ⬜ | `ls -la /home/tianque/codebase-memory-mcp` 或 `/usr/local/bin/codebase-memory-mcp` |
| 执行权限 | ⬜ | `file /path/to/codebase-memory-mcp` |
| 版本信息 | ⬜ | `/path/to/codebase-memory-mcp --version` |
| 项目路径可访问 | ⬜ | 确认 cbmem-team 配置的 `DataDir` 路径 |

### 2.4 MySQL (可选)

| 检查项 | 状态 | 命令/说明 |
|--------|------|----------|
| MySQL 容器运行 | ⬜ | `docker ps | grep mysql` 或 `docker ps | grep cbmem-mysql` |
| 连接测试 | ⬜ | `mysql -h 127.0.0.1 -P 3306 -u cbmem -p -e "SELECT 1"` |
| 数据库存在 | ⬜ | `SHOW DATABASES LIKE 'cbmem'` |
| 表结构创建 | ⬜ | `cbmem-team migrate-tables -mysql-dsn "..."` |

---

## 3. 部署文件清单

### 3.1 cbmem-team 部署文件

```
deploy/
├── cbmem-team.env              # 环境变量配置
├── cbmem-team.service          # systemd 服务 (SQLite)
├── cbmem-team-mysql.service    # systemd 服务 (MySQL)
├── cbmem-team.service.v2       # v2 服务配置
├── docker-compose.mysql.yml     # MySQL Docker Compose
├── docker-compose.mempalace.yml # MemPalace Docker Compose (新建)
├── mempalace.env.example        # MemPalace 环境变量模板 (新建)
├── mysql.env.example            # MySQL 环境变量模板
├── dev.env.example              # 开发环境变量模板
├── Caddyfile                    # Caddy 反向代理配置
├── init-scripts/
│   └── init-team.sh             # MemPalace 初始化脚本 (新建)
└── sql/
    ├── schema.sql               # SQLite schema
    ├── schema-mysql.sql         # MySQL schema
    ├── seed.sql                 # 种子数据
    └── seed-mysql.sql           # MySQL 种子数据
```

### 3.2 配置模板

| 文件 | 用途 | 必填配置 |
|------|------|----------|
| `cbmem-team.env` | 主服务环境变量 | JWT_SECRET, ADMIN_TOKEN, MCP_BINARY |
| `mempalace.env.example` | MemPalace 环境变量 | MEMPALACE_BACKEND |
| `mysql.env.example` | MySQL 环境变量 | MYSQL_PASSWORD, CBMEM_MYSQL_DSN |

---

## 4. 部署验证流程

### 4.1 基础验证

```bash
# 1. MemPalace 健康检查
curl -s http://192.168.100.83:8089/healthz | jq

# 2. cbmem-team 健康检查
curl -s http://192.168.100.83:8787/healthz | jq

# 3. 验证服务间通信
curl -X POST http://192.168.100.83:8787/api/console/stats \
  -H "X-Admin-Token: $ADMIN_TOKEN" | jq
```

### 4.2 MCP 连接测试

```bash
# 测试 MCP 端点 (需要有效 JWT)
TOKEN="your-jwt-token"
curl -X POST "http://192.168.100.83:8787/mcp?as=testuser&project=/test" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"},"capabilities":{}}}'
```

### 4.3 Console 验证

1. 打开浏览器访问 `http://192.168.100.83:8787/console/`
2. 使用 Admin Token 登录
3. 验证以下模块可用：
   - 用户管理
   - 项目管理
   - 会话记录
   - 归纳/蒸馏

---

## 5. 常见问题排查

### 5.1 MemPalace 连接失败

```
症状: curl http://192.168.100.83:8089/healthz 超时
排查:
1. 检查容器状态: docker ps | grep mempalace
2. 检查端口占用: netstat -tlnp | grep 8089
3. 查看日志: docker logs mempalace
修复:
docker compose -f docker-compose.mempalace.yml restart
```

### 5.2 cbmem-team 401 Unauthorized

```
症状: /mcp 返回 401
排查:
1. JWT 是否过期 → 续期: POST /refresh
2. JWT secret 是否匹配 → 检查配置
3. 用户是否已注册 → 检查 POST /admin/users
修复:
curl -X POST "http://192.168.100.83:8787/refresh?ttl=720h" \
  -H "Authorization: Bearer $OLD_TOKEN"
```

### 5.3 codebase-memory-mcp 无法启动

```
症状: Process Pool 中 subprocess 启动失败
排查:
1. 二进制路径: ls -la $(which codebase-memory-mcp)
2. 执行权限: file /path/to/binary
3. 目录权限: ls -la /home/tianque/ (如在 home 目录下)
修复:
# 方案1: 移动到标准路径
sudo mv /home/tianque/codebase-memory-mcp /usr/local/bin/
sudo chmod +x /usr/local/bin/codebase-memory-mcp

# 方案2: 修复目录权限
sudo chmod o+rx /home/tianque
```

### 5.4 MySQL 连接失败

```
症状: cbmem-team 日志显示 "connect: connection refused"
排查:
1. MySQL 容器运行: docker ps | grep mysql
2. 端口监听: netstat -tlnp | grep 3306
3. DSN 配置: 检查 cbmem-team-mysql.service 中的 -mysql-dsn
修复:
# 重启 MySQL
docker compose -f docker-compose.mysql.yml restart

# 验证 DSN
cbmem-team mysql-ping -mysql-dsn "cbmem:pass@tcp(127.0.0.1:3306)/cbmem"
```

---

## 6. 配置参考

### 6.1 cbmem-team-mysql.service 关键配置

```ini
[Service]
ExecStart=/usr/local/bin/cbmem-team \
  -listen :8787 \
  -data /var/lib/cbmem-team \
  -mcp-bin /usr/local/bin/codebase-memory-mcp \
  -jwt-secret ${JWT_SECRET} \
  -admin-token ${ADMIN_TOKEN} \
  -mysql-dsn ${CBMEM_MYSQL_DSN} \
  -mempalace-base http://192.168.100.83:8089 \
  -cors-allow-origins "https://docs.fin-ai.net" \
  -log info
```

### 6.2 MemPalace Docker Compose 启动

```bash
cd tools/cbmem-team/deploy

# 复制并修改环境变量
cp mempalace.env.example .env
# 编辑 .env 设置 MEMPALACE_PORT=8089

# 启动 MemPalace
docker compose -f docker-compose.mempalace.yml up -d

# 初始化团队 Wing (可选)
docker compose -f docker-compose.mempalace.yml \
  --profile cli run mempalace-cli /scripts/init-team.sh
```

---

## 7. 签收确认

| 组件 | 检查人 | 日期 | 签名 |
|------|--------|------|------|
| MemPalace | | | |
| cbmem-team | | | |
| codebase-memory-mcp | | | |
| MySQL | | | |
| Console | | | |

---

*文档更新: 2026-07-14*
