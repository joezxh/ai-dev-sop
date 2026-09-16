# Mem0 自建图记忆层设计（对齐 1.x graph_store，方案 C）

- 日期：2026-09-16
- 状态：已确认（用户批准）
- 参考基线：mem0 v1.0.11 `mem0/memory/graph_memory.py`（`MemoryGraph`）与 `mem0/graphs/configs.py`（`GraphStoreConfig`）
- 范围：`mem0/server`（后端）；Dashboard 仅 P1 的 relations 展示
- 背景：mem0ai 2.0.20 OSS 已移除图引擎（无 `graph_store` 配置、无 `mem0.graphs` 模块；Dockerfile 的 `mem0ai[graph]` 为无效安装）；Neo4j 5.26 容器（`mwb-neo4j`，端口 7687 已发布宿主机）现成可用

## 1. 已确认决策

| 决策点 | 结论 |
| --- | --- |
| 实现策略 | **方案 A：自研移植模块**——`server/graph_memory.py` 按 v1.0.11 `MemoryGraph` 语义移植 + `graph_router.py`；复用 mem0 的 `LlmFactory/EmbedderFactory`；mem0 包零改动 |
| 集成深度 | **全对齐 1.x**：add 自动建图、`/search` 融合 relations、独立 `/graph/search`、`/graph/get_all`、删除软删联动、项目池图检索（扩展 1.x 无的项目维度） |
| 降级策略 | **`GRAPH_ENABLED` 开关（默认 true）+ 失败静默**：抽取失败/超时/Neo4j 不可达只记日志；向量记忆主流程零影响；依赖缺失或 Neo4j 不可达时自动降级为禁用 |

## 2. 参考基线：1.x graph_store 行为（对齐目标）

- **配置**：`graph_store.provider=neo4j`，`config{url, username, password, database?, base_label?}`，`llm?`（graph 专用，优先级高于全局 llm），`custom_prompt?`（注入抽取规则 #4），`threshold=0.7`（节点消歧相似度阈值）
- **add 建图**：LLM 抽取实体（自我指涉映射为 `{user_id}` 实体）→ LLM 抽取关系三元组 → 实体名 embedding 相似匹配既有节点（≥threshold 复用，否则 MERGE 新节点）→ `MERGE` 节点（mentions 自增 + 时间戳）→ `MERGE` 关系（类型大写化，`valid=true`）
- **search**：查询抽实体 → embedding 匹配图节点 → 1 跳关系（出入向，作用域过滤 + `valid` 过滤）→ BM25 重排 top5 → 三元组 `{source, relationship, destination}`；`Memory.search` 返回 `{results, relations}`
- **删除**：单条删除=软删（`valid=false, invalidated_at`）；delete_all=按作用域 `DETACH DELETE`；`get_all` 返回作用域内全部三元组
- **实现依赖**：`langchain_neo4j.Neo4jGraph` + `rank-bm25` + 节点 embedding 属性（Neo4j 向量函数；5.26 支持）

## 3. 配置与开关

```
GRAPH_ENABLED=true      # 总开关；false 时零图调用
GRAPH_THRESHOLD=0.7     # 节点消歧相似度阈值
GRAPH_CUSTOM_PROMPT=    # 关系抽取自定义规则（对齐 1.x custom_prompt）
NEO4J_URI / NEO4J_USERNAME / NEO4J_PASSWORD   # 复用现有变量，组装 GraphStoreConfig.url
```
- 自动降级：模块初始化时探测（导入 `langchain_neo4j`/`rank_bm25`、Neo4j 连通性）；失败 → 记日志、`GRAPH_ENABLED` 视为 false，后续调用直接返回空
- 无新增配置文件；全部 env 注入

## 4. 数据模型（Neo4j）

- 节点：`(:__Entity__ {name, type, embedding, mentions, created, user_id, project_id})`；`base_label` 固定启用
- 关系：类型 = 抽取关系的大写化（`WORKS_AT`）；属性 `{mentions, created, updated_at, valid, invalidated_at, user_id, project_id}`
- **作用域扩展（1.x 无 → 本方案新增）**：节点键 = `(name, user_id, project_id)`；
  - 个人图：按 `user_id` 匹配；
  - 项目池图：按 `project_id` 匹配（跨用户共享，与记忆共享池同口径）；无项目时仅 user 维度
- 索引（启动时幂等创建）：`(__Entity__.user_id)`、`(__Entity__.project_id)`、`(__Entity__.name, user_id)`（单键索引，规避企业版组合索引限制）

## 5. 写入链路

`add_memory` 成功返回后（webhooks 同款 `ThreadPoolExecutor` fire-and-forget）→ 提交建图任务：
1. LLM 抽取实体（工具调用式；自我指涉 → `{user_id}`；实体名规范化小写+下划线）
2. LLM 抽取关系三元组（`custom_prompt` = `GRAPH_CUSTOM_PROMPT` + 项目级 `custom_instructions` 拼接）
3. 实体 embedding → 相似匹配既有节点（≥`GRAPH_THRESHOLD` 复用，否则 MERGE）
4. `MERGE` 关系 + `mentions` 自增 + `valid=true`
- LLM 调用超时 30s；失败仅 `logging.exception`；`GRAPH_ENABLED=false`/降级时整段跳过
- 复用当前实例的 embedder（`EmbedderFactory.create`，与向量库同维度）与 llm（`LlmFactory.create`，graph 可覆盖）

## 6. 检索与删除链路

| 端点/路径 | 行为 |
| --- | --- |
| `POST /search` | 向量结果照旧；新增 `relations` 字段（图三元组数组）；`SearchRequest.include_graph`（默认 true）；图不可用 → `relations: []` |
| `POST /graph/search` | 纯图检索（对齐 1.x `MemoryGraph.search`）：`query` + `project_id\|git_remote?` + `limit`；个人作用域强制 user_id；BM25 重排 top5 |
| `GET /graph/get_all` | 按 user/project 作用域返回全部三元组（`limit` 默认 100） |
| `DELETE /memories/{id}` | 成功后按该记忆文本重跑抽取 → **软删**对应关系（对齐 1.x） |
| Danger Zone（`DELETE /memories?project_id`） | 成功后按 `project_id` **硬删**图子图（`DETACH DELETE`） |
| `delete_entity` | 按 user 硬删该实体图数据 |

- 隔离：普通用户图检索强制 `user_id=当前用户`；项目池图跨用户可读（与记忆共享池同口径）；admin 不受限
- MCP 不加新工具（`search_memories` 经 `/search` 已带 relations）

## 7. 测试与验证

- 单测（fake LLM/驱动，变异敏感）：实体抽取 prompt 构造（自我指涉映射）、关系清洗大写化、cypher 模板与作用域谓词（user/project）、BM25 重排、payload/relations 格式、开关与降级分支
- 集成（手动，需真实 key）：`GRAPH_ENABLED=true` → add → Neo4j Browser `:7474` 验图 → search 出 relations → 软删验证；无 key 环境如实记录为运行时限制
- 回归：既有 92 个测试全部通过

## 8. 非目标（P2/后续）

- Dashboard 图谱可视化 UI（本轮仅 Memories 详情展示 relations）
- MCP 新增 graph 工具
- 图数据重试队列、导出
- Dockerfile 中无效的 `mem0ai[graph]` 安装清理（随本实现单独提交移除）

## 9. 风险与已知取舍

- LLM 抽取质量依赖模型能力；`custom_prompt` 可约束领域
- 图写入最终一致（异步）：检索可能短暂查不到刚写入的图
- 关系类型字符串直接进 cypher（对齐 1.x），依赖上游式清洗（大写+下划线）防注入
- 节点 embedding 占用 Neo4j 存储；`mentions` 计数用于后续热度裁剪（本轮不做）
