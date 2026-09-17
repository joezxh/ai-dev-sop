# ai-dev-sop

**AI Development SOP with Skills**

A comprehensive standard operating procedure (SOP) repository for AI-assisted development, featuring development workflows, skills, and automation tools for building full-stack applications.

## Overview

This repository contains the complete development SOP for **mediation-platform**, a full-stack Java + Vue enterprise application platform. It provides:

- Standardized development workflows from "one-sentence requirement" to "code delivery"
- Skills for AI coding tools (Cursor, Qoder, CodeBuddy, Claude Code)
- Long-term memory via self-hosted **mem0** (vector + graph memory), MCP-integrated across all AI IDEs
- Scenario-based pipelines for different development patterns
- QA automation with end-to-end browser testing
- Documentation templates and scene prompts

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        AI Development Pipeline                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Installation → Understanding → Prompt Generation → Scenario Selection   │
│        ↓                                                          ↓     │
│        └────── Testing Automation ←── Docs Automation ←─────────────┘     │
│        ↓                                                                │
│        └────────── Operations Demand Automation (FDE闭环回流)             │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. Development SOP (`docs/cn/`)

The core SOP document (`develop-sop.md`) covers:

| Section | Description |
|---------|-------------|
| §0 | Document Overview - Goals, roles, flowcharts, architecture |
| §1 | Preparation - Reading existing projects, installing Skills, IDE configuration |
| §2 | Prompt Generation - Test prompts, development prompts, scene templates |
| §3 | Testing & Development - End-to-end execution, batch processing |
| §4 | Scenario SOPs - Framework, one-sentence, upgrade, copy pipelines |
| §5 | Long-Term Memory - mem0 memory system (store/retrieve/team sharing) |
| §6 | Documentation - Product research, market research templates |

### 2. Skills (`skills/`)

- **qa-dev**: Test + development automation skill with `/qa-dev` command
- Supports batch execution, unattended mode, regression testing

### 3. Memory Service (`deploy/mem0/`)

- **mem0**: self-hosted long-term memory service (API :8888 / MCP :8080 / Dashboard :3001),
  backed by PostgreSQL (pgvector) + Neo4j, MCP-integrated with CodeBuddy / Qoder / Cursor etc.

### 4. Example Projects (`example/`)

- **ai-coding-boot**: Reference implementation using ruoyi-vue-pro architecture

## Seven Development Stages

| Stage | Description | Deliverables |
|-------|-------------|--------------|
| 1. Installation | Install Skills, toolchains, configure IDE | Skill set, Design docs |
| 2. Understanding | Map codebase with `/gsd-map-codebase` | 7 cognitive files |
| 3. Prompt Generation | Generate test/dev prompts from templates | Scene documents, PRDs |
| 4. Scenario Selection | Route to appropriate scenario pipeline | Code, commits, reports |
| 5. Testing Automation | E2E browser automation with `/qa` | Test reports, bug lists |
| 6. Documentation Automation | Auto-generate project docs | API docs, manuals |
| 7. Operations Automation | Customer demand → FDE execution |闭环 reports |

## Scenario Pipelines

| Pipeline | Use Case |
|----------|----------|
| `framework-pipeline.md` | Baseline scaffold development |
| `one-sentence-pipeline.md` | From one-sentence requirement to prototype |
| `docs-pipeline.md` | Automated project documentation |
| `copy-web-pipeline.md` | Clone existing web projects |
| `copy-app-pipeline.md` | Clone mobile apps (uniapp) |
| `java-upgrade-pipeline.md` | Technology stack migration |

## Long-Term Memory (mem0)

Provides long-term memory to all AI development tools via MCP:

| Capability | Description |
|------------|-------------|
| Long-term memory | Conversations/facts/decisions/preferences, vector retrieval (pgvector) |
| Graph memory | Entity relation graph (Neo4j), `GRAPH_ENABLED=true` |
| Team sharing | `git_remote` → `project_id` shared pool, cross-user retrieval |
| Session transcript | Per-turn Q/A verbatim archive, grouped by `session_id` |

### Deployment

```bash
# One-shot install (service + IDE config)
./scripts/install-all.sh          # Linux/macOS
./scripts/install-all.ps1         # Windows

# Or manual
cd deploy/mem0 && docker compose up -d
```

### IDE Integration

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

See [docs/quick-ref/mem0-ai-tools-config-guide.md](docs/quick-ref/mem0-ai-tools-config-guide.md) for the full guide.

## Getting Started

### 1. Install Skills

```bash
# Using npx skills
npx skills add obra/superpowers -a qoder --global
npx skills add garrytan/gstack -a qoder --global
npx get-shit-done-cc@latest --qoder --global
npm install -g @fission-ai/openspec@latest

# Clone design-md
git clone https://github.com/VoltAgent/awesome-design-md.git temp-design
cp temp-design/design-md/vercel/DESIGN.md ./DESIGN.md
rm -rf temp-design
```

### 2. Configure IDE

**Qoder:**
```json
{
  "general": {
    "defaultPermissionMode": "auto"
  }
}
```

**Cursor:** Install the `mem0` MCP server (see [docs/ide-config/ide-mcp-templates.md](docs/ide-config/ide-mcp-templates.md)).

### 3. Map Your Codebase

```bash
/gsd-map-codebase
```

This generates 7 cognitive files in `.planning/codecase/`:
- ARCHITECTURE.md
- CONCERNS.md
- CONVENTIONS.md
- INTEGRATIONS.md
- STACK.md
- STRUCTURE.md
- TESTING.md

### 4. Generate Scene Prompts

```bash
# Generate test + development prompts from existing code
/brainstorm
参照模板 @mediation-web/docs/scene/scene-template.md 的结构与风格，生成一份完整的服务端到端浏览器自动化测试开发文档...
```

### 5. Execute Development

```bash
# Full flow: test → report → fix → develop → verify
/qa-dev --file docs/scene/system-scene.md --module SYS-01

# Batch execution (unattended)
/qa-dev --batch --unattended --file docs/scene/system-scene.md
```

## Recommended AI Models

| IDE | Recommended Model |
|-----|-------------------|
| Cursor | composer-2.5 |
| Qoder | Lite / QWen3.7-MAX |
| CodeBuddy | Hy3 |

## Project Structure

```
ai-dev-sop/
├── README.md
├── README_CN.md          # Chinese version
├── README_JP.md           # Japanese version
├── docs/
│   ├── cn/               # Chinese documentation
│   │   ├── develop-sop.md
│   │   ├── scene-template.md
│   │   ├── one-sentence-pipeline.md
│   │   ├── framework-pipeline.md
│   │   ├── copy-web-pipeline.md
│   │   ├── copy-app-pipeline.md
│   │   ├── java-upgrade-pipeline.md
│   │   ├── docs-pipeline.md
│   │   ├── qa-dev-sop.md
│   │   └── benchmark/       # Evaluation datasets (historical, archived with old memory system)
│   │       ├── DS-Decision.md
│   │       ├── DS-CallPath.md
│   │       ├── DS-Cross.md
│   │       ├── DS-ADR.md
│   │       ├── DS-Customer.md
│   │       └── DS-DeadCode.md
│   └── scene/
├── skills/
│   ├── qa-dev/
│   └── mem0-longterm-memory/   # mem0 memory skill
└── deploy/
    └── mem0/               # Self-hosted mem0 (API/MCP/Dashboard)
```

## Resources

- [mem0 Documentation](https://docs.mem0.ai/)
- [mem0 MCP Integration Guide](https://docs.mem0.ai/platform/mem0-mcp)
- [Mem0 Cloud Dashboard](https://app.mem0.ai/)

## License

MIT License

## Contributing

Contributions are welcome! Please follow the SOP guidelines when contributing new scenarios, pipelines, or skills.
