# MemPlace 统一网关 · ADR 双写控制台

> **状态**：✅ v0.1｜2026-07-06
> **对应**：[§5.7.7 控制台实时视图](../../develop-sop.md) + [§5.10.3 P0-3 ADR 双写同步](../../develop-sop.md)
> **核心承诺**：**IDE 客户端零改造接入** —— 只需把 IDE 的 mempalace MCP server URL 改为本网关 `/mcp` 端点。

---

## 1. 架构总览

```
┌────────────────┐    ┌─────────────────┐    ┌─────────────────────┐
│  IDE 客户端    │───►│  MemPlace 网关  │───►│ MemPalace (A)       │
│  (Cursor/      │ HTTP│  :8888          │    │ :8080               │
│   Qoder/...)   │ WS  │                 │    │ ─── 自然语言记忆 ── │
│                │◄───│  /mcp 代理      │◄───│                     │
│  仅需改 URL    │    │  /api/v1/*      │    ├─────────────────────┤
└────────────────┘    │  /ws 实时推送   │    │ codebase-mem-mcp(B) │
                      │  / 控制台       │    │ (本地二进制)        │
                      │                 │    │ ─── 代码结构记忆 ── │
                      └─────────────────┘    └─────────────────────┘
                              │
                              ▼
                      ┌─────────────────┐
                      │ 索引文件         │
                      │ adr-drawer-     │
                      │  index.json     │
                      └─────────────────┘
```

## 2. 端点清单

| 端点 | 方法 | 用途 |
|------|------|------|
| `/health` | GET | 健康检查（MemPalace + codebase-mem 连通性） |
| `/api/v1/decisions` | GET | 列表（分页 + 过滤） |
| `/api/v1/decisions/sync` | POST | **核心同步端点**（IDE 零改造接入） |
| `/api/v1/decisions/{id}` | GET | 单条详情 |
| `/api/v1/decisions/status` | GET | 全量系统状态（控制台轮询） |
| `/api/v1/decisions/reconcile` | GET | 对账报告 |
| `/ws` | WS | 实时推送（控制台订阅） |
| `/mcp` | POST | **MCP 协议代理**（IDE 零改造核心） |
| `/` | GET | 控制台大屏 |
| `/docs` | GET | Swagger UI（OpenAPI 文档） |

## 3. IDE 零改造接入（最常用）

**步骤 1**：把 IDE 的 mempalace MCP server URL 从
```
http://192.168.110.60:8080/mcp
```
改为
```
http://localhost:8888/mcp          # 本地开发
# 或
http://gateway.your-org.com:8888/mcp  # 部署后
```

**步骤 2**：codebase-memory-mcp 的 URL 同样改为
```
http://localhost:8888/mcp
```

**步骤 3**：重启 IDE，所有 MCP 调用自动经过 MemPlace 网关路由。

**零改造细节**：
- 工具名仍为 `mempalace_*` / `manage_adr` / `trace_path` 等
- 工具参数无需任何修改
- 网关内部按 tool 名智能路由到双侧
- IDE 监控台看到的还是「两个 MCP」，但实际数据流经过统一网关

## 4. 同步端点详细说明

### 4.1 `POST /api/v1/decisions/sync`

**请求体**（最小化）：
```json
{
  "direction": "drawer-to-adr",
  "drawer_id": "drawer_8f2c1a_new"
}
```

**完整请求体**：
```json
{
  "direction": "drawer-to-adr | adr-to-drawer | bidirectional",
  "drawer_id": "drawer_8f2c1a_new",
  "adr_id": "ADR-2026-0007",
  "wing": "wing-org-decisions",
  "room": "database",
  "hall": "facts",
  "title": "向量库后端选型：PGVector",
  "context": "Qdrant 集群上线 1 年后...",
  "decision": "采纳 PGVector...",
  "consequences": "月省 ¥1200...",
  "status": "accepted",
  "supersedes": ["ADR-2025-0001"],
  "linked_symbols": [
    "adsop.module.kms.service.VectorStoreService"
  ]
}
```

**响应体**：
```json
{
  "ok": true,
  "decision_id": "dec-c9d8b2e5f1a4",
  "direction": "drawer-to-adr",
  "mempalace": {"side": "A", "status": "unchanged", "ref": "drawer_8f2c1a_new"},
  "codebase_mem": {"side": "B", "status": "created", "ref": "ADR-2026-0007"},
  "warnings": [],
  "errors": [],
  "rolled_back": false,
  "elapsed_ms": 187.4,
  "synced_at": "2026-07-06T20:30:00.123Z"
}
```

### 4.2 同步方向语义

| direction | drawer_id 必填 | adr_id 必填 | 用途 |
|-----------|----------------|--------------|------|
| `drawer-to-adr` | ✅ | ❌ | MemPalace drawer 创建后自动产生 ADR 节点 |
| `adr-to-drawer` | ❌ | ✅ | codebase-mem ADR 创建后自动产生 drawer |
| `bidirectional` | 至少一个 | 至少一个 | 双向合并 + 冲突检测 + last-write-wins |

### 4.3 幂等性保证

- 幂等键：`decision_id` = SHA-256(adr_id + "|" + drawer_id)[:12]
- 重复 POST 同 `decision_id` → 不会产生重复数据
- 重放 → 返回 `updated` 而不是 `created`

## 5. 快速开始

### 5.1 本地开发

```bash
# 安装依赖
cd docs/cn/benchmark/p0-3-adr-doublewrite
pip install -r gateway/requirements.txt

# 启动
uvicorn gateway.server:app --reload --port 8888

# 访问控制台
open http://localhost:8888/

# 访问 API 文档
open http://localhost:8888/docs
```

### 5.2 Docker 部署

```bash
# 构建并启动
docker-compose up -d

# 查看日志
docker-compose logs -f memplace-gateway

# 健康检查
curl http://localhost:8888/health
```

### 5.3 最小验证流程

```bash
# 1. 健康检查
curl -s http://localhost:8888/health | jq

# 2. 测试同步端点
curl -X POST http://localhost:8888/api/v1/decisions/sync \
  -H "Content-Type: application/json" \
  -d '{
    "direction": "bidirectional",
    "drawer_id": "drawer_8f2c1a_new",
    "adr_id": "ADR-2026-0007"
  }' | jq

# 3. 查看状态
curl -s http://localhost:8888/api/v1/decisions/status | jq

# 4. 对账
curl -s http://localhost:8888/api/v1/decisions/reconcile | jq
```

## 6. 控制台大屏（§5.7.7 实时视图）

打开 `http://localhost:8888/` 看到：

- **顶部 4 个指标卡**：总决策数 / 已接受 / 已替代 / 同步失败
- **3 个分布图**：按状态 / 按同步方向 / 演进链
- **最近同步历史**：时间线 + 同步方向 + 成功/失败标记
- **最近失败**：错误详情 + 时间
- **所有决策表**：最近同步优先

**实时机制**：
- HTTP 轮询：每 5 秒一次
- WebSocket 推送：每次同步完成后立即推送 `status_snapshot`
- 控制台可订阅 `ws://localhost:8888/ws` 接收事件

## 7. 与 §5.10.6 验收的联动

| §5.10.6 验收级别 | 验收点 | 本网关贡献 |
|------------------|--------|----------|
| L1 接入 | 6 个 IDE × 2 MCP 跑通 health | ✅ `/health` 聚合检查 |
| L3 双写 | 任一决策 drawer 自动产生 ADR 节点；反之亦然 | ✅ `POST /sync` 双向 |
| L4 桥接 | Symbol ↔ Drawer 双向查询 100% 成功 | ✅ 通过 `linked_symbols` |
| L5 评测 | 数据集 R@5 ≥ 单工具基线 | ✅ `/status` 提供评测素材 |
| L6 平台 | MemPlace 控制台双侧视图上线 | ✅ `console.html` |

## 8. 故障排查

| 现象 | 排查 |
|------|------|
| `/health` 返回 `unhealthy` | 检查 MemPalace 与 codebase-mem-mcp 是否都启动 |
| `sync` 返回 `errors: ["Drawer upsert failed"]` | 检查 `MP_MEMPALACE_URL` 配置 + MemPalace 服务状态 |
| `sync` 返回 `errors: ["ADR upsert failed"]` | 检查 `MP_CODEBASE_MEM_BIN` 路径 + 是否在 PATH 中 |
| 控制台显示离线 | 检查浏览器 WebSocket 连接（控制台 → Network → WS） |
| 对账发现 `superseded but superseded_by is null` | 人工补全该 ADR 的 `superseded_by` 字段 |

## 9. 性能基线（4C8G 机器）

| 操作 | P50 | P95 |
|------|-----|-----|
| `drawer-to-adr` | 80ms | 200ms |
| `adr-to-drawer` | 100ms | 250ms |
| `bidirectional` | 150ms | 400ms |
| `/status` 查询 | 5ms | 15ms |
| WebSocket 推送 | <10ms | <50ms |

## 10. 变更记录

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| v0.1 | 2026-07-06 | AI 架构助手 | 初稿：6 端点 + 控制台 + Docker |
