# Graph Task 12 报告 — 清理 Dockerfile 中无效的 `mem0ai[graph]` 安装

**状态：✅ 完成** — commit `85421872`（chore: drop no-op mem0ai[graph] install from dev.Dockerfile）— **单独提交**（按需求"随本实现单独提交移除"）。

## 变更
- `server/dev.Dockerfile` 第 19 行：
  - 前：`RUN pip install -e .[graph]`
  - 后：`RUN pip install -e .`
- 背景：`mem0ai` 2.x 没有 graph 引擎（其 `[graph]` extra 不存在，设计文档 deploy/mem0/api/Dockerfile 注释已说明）。图谱能力由 `server/graph_memory.py` + `langchain-neo4j` 提供，而 `langchain-neo4j>=0.4,<1.0` 已在 `server/requirements.txt`（第 20 行），`pip install -e .` 经由 `-r requirements.txt` 仍会安装，依赖完整。

## 验证
- `requirements.txt` 确认含 `langchain-neo4j>=0.4,<1.0`（dev.Dockerfile 改后图依赖不缺失）。
- 该文件位于 mem0 子模块，提交进 `feat/graph-memory` 分支（与 1-3 项同为子模块内提交）；属"单独提交"要求。

## 偏差
- 无功能性偏差；纯清理。
