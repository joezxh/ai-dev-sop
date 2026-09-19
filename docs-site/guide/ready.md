# 准备工作
<span style="color: #d32f2f; font-size: 1.25em; font-weight: 700;">
AI开发原则： AI-First </span>

**顶层通用原则**  
- **任务拆解原则**
大需求拆成**最小独立子任务**，分模块、分步骤、分函数，逐个实现再整合。
-  **明确约束原则**
限定语言、框架、版本、运行环境、输入输出格式、性能要求。
-  **先架构后代码**
先定结构、流程、接口、数据结构，再写业务逻辑。
-  **最小可用优先**
先跑通最简版本，再迭代加功能，不一步到位写复杂逻辑。
-  **语义清晰原则**
命名规范、注释精简、逻辑直白，AI易理解、人易维护。

**AI专属编码协作原则**  
-  **精准提示原则**
写清**功能+入参+出参+异常+场景**，拒绝模糊需求。
-  **分层编写原则**
底层工具类→通用方法→业务逻辑→入口调用，自上而下编写。
-  **模块化解耦**
功能独立、低耦合、高内聚，便于修改、替换、复用。
-  **容错健壮原则**
加参数校验、异常捕获、边界判断、空值处理。
-  **可调试可注释**
关键节点留日志、断点、说明，方便排错。

**效率与规范原则**  
-  **复用优先**
优先调用通用工具函数、开源方法，不重复造轮子。
-  **标准化格式**
统一缩进、换行、代码风格，符合行业规范。
-  **版本可控**
区分测试版、正式版、精简版，按需输出。
-  **安全合规**
脱敏隐私数据、规避漏洞、禁止危险执行代码。
-  **结果闭环**
写完自测逻辑、给出调用示例、使用说明。

```
请严格按照以下规则编写代码：
1. 任务拆解：将整体需求拆分为独立功能模块，分步实现，逻辑分层清晰
2. 技术约束：指定编程语言、框架、版本、运行环境、依赖库
3. 架构优先：先设计整体流程、数据结构、函数接口，再编写业务逻辑
4. 编写规范：遵循行业编码规范，命名易懂，代码整洁，缩进统一
5. 健壮性：加入参数校验、空值判断、异常捕获、边界条件处理
6. 模块化：高内聚低耦合，功能独立，便于复用与修改
7. 注释要求：核心逻辑加精简注释，复杂步骤写明思路
8. 实现顺序：先完成基础可用版本，再优化功能与性能
9. 输出要求：完整可直接运行代码 + 调用示例 + 使用说明 + 注意事项

需求内容：
【在这里粘贴你的具体开发需求】
```


## 1. 记忆体安装（mem0）

<span style="color: #d32f2f; font-size: 1em; font-weight: 700;">强烈建议安装记忆，这样不同的 AI Coding 工具的记忆可以共享，善用 mem0 的项目共享池与团队协作功能。</span>

mem0 提供向量记忆（PostgreSQL + pgvector）与图记忆（Neo4j），通过 MCP 协议
（`http://127.0.0.1:8080/mcp`）接入 CodeBuddy / Qoder / Cursor 等全部 AI IDE，
Dashboard（`http://localhost:3001`）负责管理与回放。

### 1.1 一键安装（推荐）

```bash
# Linux/macOS
./scripts/install-all.sh

# Windows
./scripts/install-all.ps1
```

### 1.2 手动部署

```bash
cd deploy/mem0
# 首次部署先准备 .env（POSTGRES_PASSWORD / NEO4J_PASSWORD / JWT_SECRET 等）
docker compose up -d

# 验证
curl -s http://localhost:8888/docs > /dev/null && echo "API OK"
curl -s http://localhost:3001 > /dev/null && echo "Dashboard OK"
```

### 1.3 API Key 与项目凭证

```text
1. Dashboard (http://localhost:3001) 登录 → Settings → API Keys → Create API Key
2. 密钥仅创建时完整可见（格式 m0-...），立即保存
3. CodeBuddy 用户写入项目级 .codebuddy/mem0.config.json（git-ignored）：
   api_key / admin_user_id / project_id / git_remote / roster
4. 其他 IDE 通过环境变量 MEM0_API_KEY 或 mcp.json headers 传入
```

### 1.4 IDE 接入

> ⚡ **推荐：一键配置**（`scripts/mem0-setup.ps1` / `mem0-setup.sh`）
> 下面的手工 MCP 配置、规则文件、凭证文件、端到端校验，均可由一条命令自动完成，
> 服务名自动约定为 `mem0`（环境只用 URL 区分本地/远程）。手工配置保留用于理解原理与特殊场景。

```powershell
# Windows（PowerShell）
cd <仓库根目录>
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\mem0-setup.ps1
```

```bash
# macOS / Linux / WSL
cd <仓库根目录>
bash scripts/mem0-setup.sh
```

**常用参数**：

| 作用 | PowerShell | bash | 说明 |
|------|-----------|------|------|
| 指定 IDE | `-Ide codebuddy,cursor,qoder,codex,claude` | `-i ...` | 缺省自动探测已安装 IDE |
| MCP 端点 | `-Url http://host:8080/mcp` | `-u ...` | 本地/远程只用 URL 区分 |
| REST 地址 | `-RestUrl http://host:8888` | `-s ...` | 校验用；缺省由 MCP URL 换端口 8888 |
| API 密钥 | `-ApiKey m0sk_xxx` | `-k m0sk_xxx` | 缺省读 `MEM0_API_KEY`，再缺省交互粘贴 |
| 目标仓库 | `-Repo <path>` | `-r <path>` | 缺省当前目录 |
| 预演 | `-DryRun` | `-n` | 只打印将写入的内容，不落盘 |

**脚本做了什么（对应手工 5 步）**：
1. 探测并**合并写**各 IDE 的 MCP 配置（保留已有其他 MCP 服务，含 `transport: streamable-http`）
2. 按 IDE 写规则文件（`CODEBUDDY.md` / `.cursor/rules/mem0.mdc` / `.qoder/rules/mem0.md` / `AGENTS.md` / `CLAUDE.md`），标记区间幂等更新
3. 生成凭证文件 `<repo>/.mem0/mem0.config.json`（自动取 `git remote get-url origin` 得 `git_remote`/`project_id`）
4. **端到端校验**：MCP 端点存活（GET `/mcp` 返回 406 即正常）→ REST 密钥写读往返 → 测试记忆自动清理
5. 输出逐项 ✓/✗ 报告与失败修复指引

**校验输出解读**：`HTTP 406` = 端点正常；`REST 写入/查回成功` = 密钥与链路闭环；`401` = 密钥无效需重建；`无法连接 8888` = 用 `-RestUrl` 指定实际 REST 地址。配完重启 IDE 即生效。

> ⚡ 一键安装同时会**自动接线转录直投 hook**（仅 codebuddy）：复制
> `mem0-transcript-poster.mjs` 到 `~/.codebuddy/hooks/` 并合并 `settings.json` 的
> `SessionStart`——会话结束后 IDE 落盘的转录 JSONL 会被纯脚本零 LLM 逐字入库
> （`metadata.source=transcript-poster`，Dashboard 详情显示绿色徽标）。详见
> `scripts/README-mem0-poster.md` 与 mem0-manual §4.11。

---

**手工配置**（`~/.codebuddy/mcp.json` 或对应 IDE 的 MCP 配置）：

```json
{
  "mcpServers": {
    "mem0-local": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": { "Authorization": "Bearer m0-你的密钥" }
    }
  }
}
```

各 IDE 详细配置见 [ide-mcp-templates（仓库 docs/ide-config/）](https://github.com/joezxh/ai-dev-sop/blob/main/docs/ide-config/ide-mcp-templates.md)
与 [mem0 AI 工具配置手册](https://github.com/joezxh/ai-dev-sop/blob/main/docs/quick-ref/mem0-manual.md)。

### 1.5 验证安装

```bash
curl http://127.0.0.1:8080/mcp        # MCP 探针
docker compose -f deploy/mem0/docker-compose.yaml ps   # 全部 healthy
# IDE 中：让 Agent "记住我偏好 TypeScript" 后再 "我的偏好是什么？" 能正确读回
```

### 1.6 常见问题

| 现象 | 原因 | 解决 |
|---|---|---|
| 连接拒绝 | mem0-api 未启动 | `docker compose up -d`；确认 pgvector/Neo4j 容器 healthy |
| 401 | Key 无效/未传 | 重新创建 Key；检查 Bearer 头 |
| 工具列表空 | 客户端未重启 | 重启 IDE；确认 URL 以 `/mcp` 结尾 |
| 检索为空 | Key/Project 不匹配 | 统一 `.codebuddy/mem0.config.json` 中的作用域配置 |

### 1.7 codebase-mem-mcp 安装（Codebase Memory MCP，代码结构智能）

> 与 mem0 互补：mem0 负责**团队记忆**（事实/决策/会话流），
> codebase-mem-mcp 负责**代码结构智能**（AST 图谱/语义检索/调用链）。
> 两者可同时安装、同时使用。

codebase-memory-mcp 是纯 C 实现的代码智能引擎：单静态二进制、158 种语言 tree-sitter AST + Hybrid LSP 语义解析、15 个 MCP 工具、零依赖。全索引 Linux 内核（28M LOC）仅需 3 分钟，查询 < 1ms。

#### 方式 A：Qoder Lite 提示词安装（推荐本地开发）

**环境要求**
- 无（单二进制，零运行时依赖）
- 支持 macOS / Linux / Windows（amd64 + arm64）

**安装提示词**（直接粘贴给 Qoder / Cursor / Claude Code）
```
请帮我安装 codebase-memory-mcp：
1. Linux/macOS 一行命令：
   curl -fsSL https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh | bash
   - 需要带图可视化 UI：在末尾加 --ui
   - 自定义安装目录：加 --dir=/usr/local/bin
2. Windows PowerShell：
   Invoke-WebRequest -Uri https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.ps1 -OutFile install.ps1
   Unblock-File .\install.ps1
   .\install.ps1
3. 安装脚本会自动：
   - 下载对应平台的最新 release 二进制
   - 写入 ~/.local/bin/codebase-memory-mcp（Linux/macOS）或 %LOCALAPPDATA%\Programs\codebase-memory-mcp\（Windows）
   - 检测已安装的 AI IDE（Qoder/Cursor/Claude Code 等）并自动写入 MCP 配置
4. 验证：codebase-memory-mcp --version
5. 在当前项目说 "Index this project" 触发首次索引。
```

**配置说明**
- 安装器自动在 `.qoder/mcp.json` 写入 stdio 配置，无需手动编辑
- 手动配置示例：
  ```json
  {
    "mcpServers": {
      "codebase-memory-mcp": {
        "command": "/home/you/.local/bin/codebase-memory-mcp",
        "args": []
      }
    }
  }
  ```
- 图可视化 UI（可选）：`codebase-memory-mcp --ui=true --port=9749`，访问 `http://localhost:9749`

**验证安装**
```bash
codebase-memory-mcp --version       # 应打印版本号
codebase-memory-mcp --help          # 查看所有可用 MCP 工具
# 在 IDE 中说 "Index this project" 或 "list all functions in this repo"
```

#### 方式 B：Docker 完整部署（推荐 CI / 隔离环境）

**环境要求**
- Docker 20.10+
- 项目源码需挂载进容器

**docker run**
```bash
docker run -i --rm \
  -v /path/to/project:/workspace \
  -v cbm-data:/data \
  ghcr.io/deusdata/codebase-memory-mcp:latest
```
注意 `-i` 标志是 MCP stdio 协议必需的。

**docker-compose.yml**（作为 stdio MCP 服务）
```yaml
version: "3.9"
services:
  cbm:
    image: ghcr.io/deusdata/codebase-memory-mcp:latest
    stdin_open: true     # 等价 docker run -i，MCP stdio 必需
    tty: false
    volumes:
      - ./my-project:/workspace:ro
      - cbm-data:/data
    working_dir: /workspace

volumes:
  cbm-data:
```

**Qoder / Cursor MCP 配置**（让 IDE 通过 docker 启动）
```json
{
  "mcpServers": {
    "codebase-memory-mcp": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "mempalace-data:/data",
        "-v", "/path/to/project:/workspace",
        "ghcr.io/deusdata/codebase-memory-mcp:latest"
      ]
    }
  }
}
```

**验证安装**
```bash
docker run -i --rm ghcr.io/deusdata/codebase-memory-mcp:latest --version
# 在 IDE 中测试："list all functions in this repo"
```

**常见问题**
| 现象 | 原因 | 解决 |
|---|---|---|
| `exec format error` | 二进制架构与平台不匹配 | 确认下载的是 amd64 还是 arm64 版本；Apple Silicon Mac 用 arm64 |
| Windows 报 "Mark-of-the-Web" 拦截 | 浏览器给下载文件加了安全标记 | `Unblock-File .\install.ps1` 后再执行 |
| 索引大仓库时 OOM | 仓库过大（> 30M LOC）且容器内存受限 | 调高 `deploy.resources.limits.memory` 或使用宿主机安装版 |
| IDE 看不到 MCP 工具 | 配置未写入或路径错误 | 重启 IDE；检查 `.qoder/mcp.json` 中 command 路径是否可执行 |

工具速查见 [mem-tools.md §4 Codebase 工具](./mem-tools.md)；使用指南见 [SOP-M2: 项目理解指南](./SOP-M2-understanding.md)。

## 2. Skill 安装

**通用提示词**  
```
# 角色与任务
你现在是一个资深的 DevOps 工程师和全栈自动化专家。你的任务是帮助我自动化安装和配置以下5个工具/Skill包，并确保它们在我的当前工作环境中可用。

# 待安装清单
1. superpower  本地已经clone路径: d:\work\superpower
2. gstack  本地已经clone路径: d:\work\gstack
3. get-shit-done  本地已经clone路径: d:\work\get-shit-done
4. OpenSpec  本地已经clone路径: d:\work\openspec
5. Agent-skills  本地已经clone路径: d:\work\agent-skills
6. everything-claude-code  本地已经clone路径: d:\work\everything-claude-code

# 执行步骤与规范（请严格按顺序执行）：

**第一步：环境与来源侦测**
在执行任何安装命令之前，请先利用你的联网搜索能力或内置知识，确认这5个包的最佳安装方式（例如：它们是 NPM 包、Python pip 包、Homebrew 工具，还是需要 git clone 的开源项目？）。
- 分析我当前的操作系统和项目环境。
- 向我输出一个简短的清单，说明你打算用什么命令安装每一个包。

**第二步：静默与安全的安装**
在获得我的（即使是默认的）允许后，请直接在终端（Terminal）中执行安装命令。
- 如果遇到需要 sudo 权限的全局安装，请先询问我，或者尝试安装在局部环境（如 `npm install -D` 或用户目录下）。
- 如果某个包（如 awesome-design-md）是一个知识库或文档库，请帮我在当前项目根目录下创建一个 `skills` 或 `docs` 文件夹，并将其 clone 进去。

**第三步：错误处理**
如果安装过程中终端报错（如网络超时、依赖冲突、包名拼写错误），请自动分析终端输出的错误日志，并尝试更换国内镜像源（如 npm 的 taobao 源，或 pip 的 tsinghua 源）或修复依赖后重新执行，最多重试2次。

**第四步：验证与总结**
全部安装完成后，请在终端中运行相应的验证命令（如 `[工具名] --version`），并向我输出一份最终的**安装报告**，说明哪些成功了，哪些失败了以及如何调用它们。

现在，请开始执行第一步。
```
注意： 因为盛世导致github不稳定，各个skills建议本地先git clone，再把仓库clone的路径加入以上提示词，不在盛世区生活的可以直接安装。


## 3. Project Rules 安装

### 3.1 根据现有框架设置Rule  

```
根据当前的后端代码模块 @ruoyi/ruoyi-module-uaa 与前端代码模块 @ruoyi-ui 的现有架构和技术栈，为整个业务平台工程项目制定并实施一套统一的开发约束和代码规范，并生成为当前开发工具cursor的Project Rules,并安装。

具体要求：
1. 分析现有代码结构，识别当前使用的编码规范、技术栈和最佳实践
2. 制定涵盖前后端的统一代码规范，包括但不限于：
   - Java 后端规范（命名约定、注释标准、异常处理、日志记录等）
   - TypeScript/Vue 前端规范（命名约定、组件结构、API 调用模式、类型定义等）
   - 数据库设计规范
   - API 设计规范
   - Git 提交规范
3. 为所有开发任务配置相应的代码质量工具：
   - ESLint、Prettier 配置
   - Checkstyle、SonarQube 等静态分析工具
   - 提交钩子（husky/lint-staged）配置
4. 确保规范对所有现有和未来的开发任务生效，包括：
   - 现有模块的逐步规范化改造
   - 新模块开发必须遵循规范
   - CI/CD 流程中集成代码质量检查
5. 提供详细的规范文档和实施指南，确保团队成员能够理解和遵循
6. 验证规范实施效果，确保不影响现有功能的正常运行
```
提示：请根据上述要求，生成一份完整的 Project Rules 文档，并安装相应的代码质量工具。  
规范文件列表：
- api-design.md
- code-quality.md
- database.md
- frontend.md
- general.md
- git-commit.md
- java-backend.md

安装 [可以使用提示词自动安装]：
- Qoder: 拷贝到当前工程的根目录： .qoder/rules/ 下;
- Cursor: 拷贝到当前工程的根目录： .cursor/rules/ 下;
- CodeBuddy: 拷贝到当前工程的根目录： .codebuddy/rules/ 下;

### 3.2 根据design.md 配置rule作用当前前端工程：  
```
根据  @frontend/design.md 文件中定义的设计规范，为当前前端工程项目配置一套完整的开发规则(rules)，并完成部署设置。具体要求如下：

1. 仔细分析 ​frontend/design.md​ 文件中的全部规范内容，包括：
   - 视觉主题与设计理念 (Visual Theme & Atmosphere)
   - 配色体系 (Color Palette) - 包括CSS变量、色值和使用场景
   - 排版规范 (Typography) - 字体、字号、行高、字重等
   - 核心组件样式 (Component Stylings) - 按钮、卡片、输入框等及其各种状态
   - 布局原则 (Layout Principles) - 间距体系、网格、留白等
   - 层级与阴影 (Depth & Elevation) - 阴影系统、z-index规范
   - 设计准则 (Do's and Don'ts) - 允许和禁止的行为
   - 响应式规则 (Responsive Behavior) - 断点设置
   - AI编码指引 (Agent Prompt Guide)

2. 基于这些规范创建前端开发规则配置，确保：
   - 所有颜色使用CSS变量而非硬编码十六进制值
   - 全局圆角统一使用2px/3px/4px/8px四档
   - 卡片必须包含3px高的#1677ff顶部装饰条
   - 页面背景保持#ffffff，搭配rgba(0, 90, 200, 0.02)的20px×20px网格纹理
   - 过渡时间使用0.2s/0.25s/0.3s三档
   - 响应式仅使用1400px和768px两个断点
   - 字体族统一使用'PingFang SC', 'Microsoft YaHei', 'Helvetica Neue'

3. 配置开发工具规则，包括但不限于：
   - ESLint规则配置
   - Prettier格式化规则
   - TypeScript类型检查规则
   - CSS/SCSS/Less样式规范检查

4. 确保规则在开发环境中正确部署并生效，验证新创建的组件和页面能够遵循 ​ @frontend/design.md  中定义的规范。
```

## 4 AI IDE配置：

### 4.1 Qoder
**关闭Agent 确认提示：**  
  Integrations Browser Agent等都配置为 Auto-run  
  .qoder/settings.json 配置为：
```json
{
  "general": {
    "defaultPermissionMode": "auto"
  }
}
```
**MCP 安装**  
  必须安装：  
  - playwright
  - mem0 MCP（见 §4.4）

### 4.2 Cursor
**MCP 安装**    
  必须安装：  
  - playwright  
  - mem0 MCP（见 §4.4）

### 4.3 CodeBuddy
**MCP 安装**  
  必须安装：
  - playwright
  - mem0 MCP（见 §4.4）

### 4.4 MCP 配置（mem0）

**统一端点**: `http://127.0.0.1:8080/mcp`（自托管 mem0，先完成 §1 安装）

**Cursor**（`~/.cursor/mcp.json`）:
```json
{
  "mcpServers": {
    "mem0": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": { "Authorization": "Bearer m0-你的密钥" }
    }
  }
}
```

**Qoder**（个人设置 → MCP 服务 → 配置文件）:
```json
{
  "mcpServers": {
    "mem0": {
      "type": "http",
      "url": "http://127.0.0.1:8080/mcp",
      "headers": { "Authorization": "Bearer m0-你的密钥" }
    }
  }
}
```

**CodeBuddy**（Settings → MCP → Add MCP）:
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
> CodeBuddy 下工具调用通过 `api_key` 参数鉴权（凭证读 `.codebuddy/mem0.config.json`）。

**Codex**（`~/.codex/config.toml`）:
```toml
[mcp_servers.mem0]
url = "http://127.0.0.1:8080/mcp"
bearer_token_env_var = "MEM0_API_KEY"
```

**云端替代**: 将 URL 换为 `https://mcp.mem0.ai/mcp`（OAuth 或 Bearer Key）。

**服务安装脚本**（仓库 `scripts/` 提供 `install-all.sh` / `install-all.ps1`，
自动启动 mem0 服务；IDE 接入用 `mem0-setup.ps1` / `mem0-setup.sh`）。

| Server | 传输 | 配置要点 | 用途 |
|---|---|---|---|
| `mem0-local` | Streamable HTTP | url | 自托管记忆，本机部署 |
| `mem0`（云端） | Streamable HTTP + Bearer | url + headers | Mem0 Cloud，零部署 |
## 5. 全文理解工程：  

每个工程开始前，做一下全文理解工程，没有配置Mem记忆，则使用skill理解，后续调用。  

### 5.1 提示词：Skill方式

```
/gsd-map-codebase 检索当前工程的代码并理解, 输出结果使用简体中文，将结果写入  @docs/scene/graph-graphiti.md  ，覆盖该文件
```
在当前工作区的 .planning/codecase 下会产生7个当前工程信息的文件:

| 文件名             | 说明   |
|-----------------|------|
| ARCHITECTURE.md | 架构设计 |
| CONCERNS.md     | 关注点  |
| CONVENTIONS.md  | 规范   |
| INTEGRATIONS.md | 集成   |
| STACK.md        | 技术栈  |
| STRUCTURE.md    | 结构   |
| TESTING.md      | 测试   |

### 5.2 mem0 记忆检索方式

配合 mem0 长期记忆做全文理解（先记忆后代码）：

```text
1. 会话首轮 Agent 自动拉取项目共享池（CODEBUDDY.md 约定）：
   get_memories(api_key=..., project_id="ai-dev-sop")

2. 按主题精准检索：
   search_memories(query="架构 技术栈", top_k=5)
   search_memories(query="ADR 决策", top_k=5)

3. 历史经验借鉴：
   search_memories(query="<问题关键词> 历史解决方案")

4. 代码结构分析使用 IDE 原生能力（LSP 语义导航 / 文本搜索），
   与 mem0 团队记忆互补（详见 SOP-M2 项目理解）。
```

## 6.标准操作顺序：  
- 1.编写提示词：  
   在AI编程工具编写提示词，填写你的需求
- 2.提示词优化：  
  使用qoder提示词优化功能【lite模式即可，每个账号每天有次数限制】。
- 3.选择skill：  
   根据任务选择skill，无脑选着brainstorm [superpower]  
- 4.执行任务：  
   将优化的提示词，复制到你想执行的ai code工具里执行开发任务【mem0 记忆已自动加载，节省token】。  
