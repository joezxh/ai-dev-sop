# ai-dev-sop

**AI 开发 SOP 与 Skills**

一套完整的 AI 辅助开发标准操作流程（SOP）知识库，包含开发工作流、技能工具、自动化测试和 mem0 长期记忆系统。

## 概述

本仓库是 **business-platform**（业务平台）的完整开发 SOP，面向全栈开发者、测试工程师和产品经理。提供从"一句话需求"到"代码交付"的完整研发链路标准操作指南。

## 核心特性

- **七大阶段流水线**：安装 → 理解 → 提示词生成 → 场景选择 → 测试自动化 → 文档自动化 → 运营需求自动化
- **Skills 技能集群**：superpower、gstack、get-shit-done、OpenSpec 等 AI 开发技能
- **mem0 长期记忆**：自托管 mem0（向量 + 图记忆），MCP 接入全部 AI IDE
- **场景化 Pipeline**：脚手架、一句话需求、升级迁移、复制、Web/App 克隆
- **QA 自动化**：`/qa-dev` 技能支持端到端浏览器自动化测试
- **多 IDE 支持**：Cursor、Qoder、CodeBuddy、Claude Code

## 架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        AI 开发流水线                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  安装  →  理解  →  提示词生成  →  场景选择                              │
│        ↓                                                     ↓          │
│        └───────  测试自动化  ←───  文档自动化  ←────────────────┘        │
│        ↓                                                                │
│        └──────────────  运营需求自动化（FDE 闭环回流）                     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## 主要组件

### 1. 开发 SOP（`docs/cn/`）

核心文档 `develop-sop.md` 包含：

| 章节 | 内容说明 |
|------|----------|
| §0 | 文档概述 - 目标、角色、流程图、架构图 |
| §1 | 准备阶段 - 读取工程、安装 Skills、配置 IDE |
| §2 | 提示词生成 - 测试提示词、开发提示词、场景模板 |
| §3 | 测试与开发 - 端到端执行、批量处理 |
| §4 | 场景 SOP - 脚手架、一句话需求、升级、复制 Pipeline |
| §5 | 长期记忆 - mem0 记忆系统（存储/检索/团队共享） |
| §6 | 文档编写 - 产品研究、市场调研模板 |

### 2. Skills（`skills/`）

- **qa-dev**: 测试+开发自动化技能，支持 `/qa-dev` 命令
- 支持批量执行、无人值守模式、回归测试

### 3. 记忆服务（`deploy/mem0/`）

- **mem0**: 自托管长期记忆服务（API :8888 / MCP :8080 / Dashboard :3001），
  基于 PostgreSQL(pgvector) + Neo4j，通过 MCP 接入 CodeBuddy / Qoder / Cursor 等全部 AI IDE

### 4. 示例项目（`example/`）

- **ai-coding-boot**: 基于 ruoyi-vue-pro 架构的参考实现

## 七大开发阶段

| 阶段 | 描述 | 产出物 |
|------|------|--------|
| 1. 安装 | 安装 Skills、工具链、配置 IDE | 技能集、设计规范 |
| 2. 理解 | 使用 `/gsd-map-codebase` 映射代码库 | 7 份认知文件 |
| 3. 提示词生成 | 从模板生成测试/开发提示词 | 场景文档、PRD |
| 4. 场景选择 | 路由到对应场景 Pipeline | 代码、提交记录、报告 |
| 5. 测试自动化 | 端到端浏览器自动化测试 | 测试报告、问题清单 |
| 6. 文档自动化 | 自动生成项目文档 | API 文档、运维手册 |
| 7. 运营自动化 | 客户需求 → FDE 执行 | 闭环记录 |

## 场景 Pipeline

| Pipeline | 适用场景 |
|----------|----------|
| `framework-pipeline.md` | 基线脚手架开发 |
| `one-sentence-pipeline.md` | 一句话需求到前端原型 |
| `docs-pipeline.md` | 项目文档自动化 |
| `copy-web-pipeline.md` | 复制已有 Web 项目 |
| `copy-app-pipeline.md` | 复制移动 App（uniapp） |
| `java-upgrade-pipeline.md` | 技术栈迁移升级 |

## mem0 长期记忆系统

通过 MCP 协议为所有 AI 开发工具提供长期记忆：

| 能力 | 说明 |
|------|------|
| 长期记忆 | 对话/事实/决策/偏好，向量检索（pgvector） |
| 图记忆 | 实体关系图谱（Neo4j），`GRAPH_ENABLED=true` |
| 团队共享 | `git_remote` → `project_id` 项目共享池，跨用户读取 |
| 会话留痕 | 每轮 Q/A 原文入库，按 `session_id` 分组回放 |

### 部署

```bash
# 一键安装（mem0 服务）
./scripts/install-all.sh          # Linux/macOS
./scripts/install-all.ps1         # Windows

# 或手动启动
cd deploy/mem0 && docker compose up -d
```

### IDE 接入

```json
{
  "mcpServers": {
    "mem0-local": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

完整配置见 [docs/quick-ref/mem0-manual.md](docs/quick-ref/mem0-manual.md)。

## 快速开始

### 1. 安装 Skills

```bash
# 使用 npx skills
npx skills add obra/superpowers -a qoder --global
npx skills add garrytan/gstack -a qoder --global
npx get-shit-done-cc@latest --qoder --global
npm install -g @fission-ai/openspec@latest

# 克隆 design-md
git clone https://github.com/VoltAgent/awesome-design-md.git temp-design
cp temp-design/design-md/vercel/DESIGN.md ./DESIGN.md
rm -rf temp-design
```

### 2. 配置 IDE

**Qoder:**
```json
{
  "general": {
    "defaultPermissionMode": "auto"
  }
}
```

**Cursor:** 安装 `mem0` MCP 服务器（配置见 [docs/ide-config/ide-mcp-templates.md](docs/ide-config/ide-mcp-templates.md)）。

### 3. 映射代码库

```bash
/gsd-map-codebase
```

生成 7 份认知文件到 `.planning/codecase/`：
- ARCHITECTURE.md
- CONCERNS.md
- CONVENTIONS.md
- INTEGRATIONS.md
- STACK.md
- STRUCTURE.md
- TESTING.md

### 4. 生成场景提示词

```bash
# 从现有代码生成测试+开发提示词
/brainstorm
参照模板 @business-web/docs/scene/scene-template.md 的结构与风格，生成一份完整的服务端到端浏览器自动化测试开发文档...
```

### 5. 执行开发

```bash
# 完整流程：测试 → 报告 → 修复 → 开发 → 验证
/qa-dev --file docs/scene/system-scene.md --module SYS-01

# 批量执行（无人值守）
/qa-dev --batch --unattended --file docs/scene/system-scene.md
```

## 推荐 AI 模型

| IDE | 推荐模型 |
|-----|----------|
| Cursor | composer-2.5 |
| Qoder | Lite / QWen3.7-MAX |
| CodeBuddy | Hy3 |

## 项目结构

```
ai-dev-sop/
├── README.md           # English version
├── README_CN.md        # 中文版
├── README_JP.md         # 日语版
├── docs/
│   ├── cn/               # 中文文档
│   │   ├── develop-sop.md
│   │   ├── scene-template.md
│   │   ├── one-sentence-pipeline.md
│   │   ├── framework-pipeline.md
│   │   ├── copy-web-pipeline.md
│   │   ├── copy-app-pipeline.md
│   │   ├── java-upgrade-pipeline.md
│   │   ├── docs-pipeline.md
│   │   ├── qa-dev-sop.md
│   │   └── benchmark/       # 评测数据集（历史，已随旧记忆系统归档）
│   │       ├── DS-Decision.md
│   │       ├── DS-CallPath.md
│   │       ├── DS-Cross.md
│   │       ├── DS-ADR.md
│   │       ├── DS-Customer.md
│   │       └── DS-DeadCode.md
│   └── scene/
├── skills/
│   ├── qa-dev/
│   └── mem0-longterm-memory/   # mem0 记忆技能
└── deploy/
    └── mem0/               # 自托管 mem0（API/MCP/Dashboard）
```

## 资源链接

- [mem0 官方文档](https://docs.mem0.ai/)
- [mem0 MCP 接入指南](https://docs.mem0.ai/platform/mem0-mcp)
- [Mem0 Cloud Dashboard](https://app.mem0.ai/)

## 许可证

MIT License

## 贡献指南

欢迎贡献新的场景、Pipeline 或 Skills。请遵循 SOP 指南进行贡献。
