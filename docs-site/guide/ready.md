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


## 1. 记忆体安装  

<span style="color: #d32f2f; font-size: 1em; font-weight: 700;">强烈建议安装记忆，这样不同的 AI Coding 工具的记忆可以共享，善用 MemPalace 的分项目分团队功能。</span>

### 1.1 MemPalace 安装

MemPalace 是本地优先的 AI 记忆体，提供 36 个 MCP 工具、实体知识图谱、verbatim 存储 + 语义检索（LongMemEval R@5 = 96.6%），零 API 调用。cbmem-team 通过它的 HTTP MCP 端点（`POST /mcp` + `tools/call mempalace_add_drawer`）实现团队共享记忆同步。

#### 1.1.1 方式 A：Qoder Lite 提示词安装（推荐本地开发）

**环境要求**
- Python 3.9+
- 包管理器：`uv`（推荐）或 `pipx` 或 `pip`（在 venv 中）
- 约 300 MB 磁盘空间（嵌入模型）

**安装提示词**（直接粘贴给 Qoder / Cursor / Claude Code）
```
请帮我安装并配置 MemPalace，要求：
1. 用 uv 安装：uv tool install mempalace
   - 如果 uv 未安装，先执行：pip install uv 或 curl -LsSf https://astral.sh/uv/install.sh | sh
2. 在当前项目目录初始化：mempalace init .
3. 启动 HTTP MCP 服务端（供 cbmem-team 接入）：
   mempalace serve --host 127.0.0.1 --port 8765
4. 验证：curl http://127.0.0.1:8765/healthz 应返回 200
5. 将以下 MCP 配置写入 .qoder/mcp.json（以及 .cursor/mcp.json 若同时使用 Cursor）：
   {
     "mcpServers": {
       "mempalace": {
         "command": "mempalace",
         "args": ["mcp"]   // stdio 模式供 IDE 直连
       }
     }
   }
6. 最后运行 mempalace status 输出安装报告。
```

**配置说明**
- `mempalace serve --host 0.0.0.0`：绑定非 loopback 时会自动生成 bearer token 并保存到 `~/.mempalace/server/`（0600 权限），cbmem-team 需用 `-mempalace-token` 传入同一 token
- `--transport http`：由 `serve` 子命令隐式启用，无需显式传
- 默认后端 ChromaDB 零配置；可选 `--backend milvus|qdrant|pgvector` 切换

**验证安装**
```bash
mempalace --version                # 应打印版本号
mempalace status                   # 查看 palace 路径 / 后端 / drawer 数量
curl http://127.0.0.1:8765/healthz # HTTP 探针
```

#### 1.1.2 方式 B：Docker 完整部署（推荐团队共享 / 服务器）

**环境要求**
- Docker 20.10+ 或 Docker Desktop
- 至少 1 GB 可用内存（嵌入模型加载）

**docker run 一键启动**
```bash
# 构建（CPU 版，含 extract + spellcheck extras）
docker build -t mempalace tools/mempalace

# 启动 HTTP MCP 服务端，持久化到命名卷
docker run -d --name mempalace \
  -p 8765:8765 \
  -v mempalace-data:/data \
  -e MEMPALACE_MCP_HTTP_TOKEN=change-me-in-prod \
  mempalace serve --host 0.0.0.0 --port 8765
```

**docker-compose.yml**（推荐，便于维护）
```yaml
version: "3.9"
services:
  mempalace:
    build: ./tools/mempalace
    image: mempalace:latest
    container_name: mempalace
    ports:
      - "8765:8765"
    volumes:
      - mempalace-data:/data
    environment:
      # 非 loopback 绑定必须启用 token；cbmem-team 用同一 token 连接
      MEMPALACE_MCP_HTTP_TOKEN: ${MEMPALACE_TOKEN:-change-me-in-prod}
    command: serve --host 0.0.0.0 --port 8765
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 2G

volumes:
  mempalace-data:
```

启动：`docker compose up -d`；停止：`docker compose down`（数据保留在 volume 中）。

**GPU 加速嵌入**（可选）
```bash
docker build -f tools/mempalace/Dockerfile.gpu -t mempalace:gpu .
docker run -d --gpus all -p 8765:8765 -v mempalace-data:/data mempalace:gpu serve --host 0.0.0.0 --port 8765
```

**验证安装**
```bash
curl http://127.0.0.1:8765/healthz                    # 200 = 服务正常
docker exec mempalace mempalace status                # 查看 palace 状态
docker logs mempalace 2>&1 | tail                     # 查看启动日志
```

**常见问题**
| 现象 | 原因 | 解决 |
|---|---|---|
| `healthz` 返回 401 | 服务端要求 bearer token | 请求头加 `Authorization: Bearer $MEMPALACE_MCP_HTTP_TOKEN` |
| 容器启动后立即退出 | `serve` 参数错误 | `docker logs mempalace` 查看报错；确认 command 是 `serve --host 0.0.0.0 --port 8765` |
| 嵌入模型下载慢 | 首次启动需拉 ~300MB 模型 | 预先挂载缓存卷 `-v mempalace-cache:/root/.cache` |
| cbmem-team 报 `-32602 unknown argument` | 传了 mempalace_add_drawer 不支持的参数 | 只传 `wing/room/content/source_file/added_by`，hall 映射到 `source_file` |

---

### 1.2 codebase-mem-mcp 安装

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

[记忆工具使用指南 SOP-M2: 项目理解指南](./SOP-M2-understanding.md)

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
6. awesome-design-md  本地已经clone路径: d:\work\awesome-design-md

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
根据当前的后端代码模块 @ruoyi/ruoyi-module-uaa 与前端代码模块 @ruoyi-ui 的现有架构和技术栈，为整个调解平台工程项目制定并实施一套统一的开发约束和代码规范，并生成为当前开发工具cursor的Project Rules,并安装。

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
  - memplace MCP
  - codebase-mem-mcp

### 4.2 Cursor
**MCP 安装**    
  必须安装：  
  - playwright  
  - memplace MCP
  - codebase-mem-mcp

### 4.3 CodeBuddy
**MCP 安装**  
  必须安装：
  - playwright
  - memplace MCP
  - codebase-mem-mcp

### 4.4 MCP配置脚本：
使用以下json给各个AI Coding工具安装MCP  
```json
{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    },

    "mempalace": {
      "command": "mempalace",
      "args": ["mcp"]
    },
    "mempalace-http": {
      "url": "http://127.0.0.1:8765/mcp",
      "_comment": "团队共享模式：服务端用 mempalace serve --host 0.0.0.0 启动时，需带 bearer token"
    },
    "mempalace-http-auth": {
      "url": "http://YOUR_MEMPALACE_HOST:8765/mcp",
      "headers": {
        "Authorization": "Bearer <MEMPALACE_MCP_HTTP_TOKEN>"
      }
    },

    "codebase-memory-mcp": {
      "command": "codebase-memory-mcp",
      "args": []
    },
    "codebase-memory-mcp-docker": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "cbm-data:/data",
        "-v", "${workspaceFolder}:/workspace",
        "ghcr.io/deusdata/codebase-memory-mcp:latest"
      ]
    },

    "cbmem-team": {
      "url": "http://localhost:8787/mcp?as=<YOUR_USER_ID>&project=<YOUR_PROJECT_PATH>",
      "headers": {
        "Authorization": "Bearer <JWT_TOKEN>"
      }
    }
  }
}
```

**配置说明**

| 服务 | 模式 | 必填字段 | 备注 |
|---|---|---|---|
| `playwright` | stdio | command/args | 浏览器自动化，Qoder/Cursor 通用 |
| `mempalace` | stdio | command=`mempalace` | 本地 CLI 直连，零配置，palace 在 `~/.mempalace/` |
| `mempalace-http` | Streamable HTTP | url | 本机 loopback，无需 token |
| `mempalace-http-auth` | Streamable HTTP + Bearer | url + headers | 服务端绑非 loopback 时必须带 token |
| `codebase-memory-mcp` | stdio | command | 单二进制零依赖，推荐本地安装 |
| `codebase-memory-mcp-docker` | stdio via docker | command=`docker` | 隔离环境或 CI 使用，`-i` 标志必需 |
| `cbmem-team` | Streamable HTTP + JWT | url + headers | URL 中 `as` 是用户 ID，`project` 是工程路径 |

**各 IDE 安装位置**

| IDE | 配置文件路径 |
|---|---|
| Qoder | `<workspace>/.qoder/mcp.json` 或 `~/.qoder/mcp.json`（全局） |
| Cursor | `<workspace>/.cursor/mcp.json` |
| Claude Code | `claude mcp add <name> [args]`（命令行注册） |
| CodeBuddy | `<workspace>/.codebuddy/mcp.json` |

**获取 cbmem-team 的 JWT Token**

```bash
# 用 admin token 换取 JWT（默认 admin/admin123，admin token 默认 change-me-too）
curl -X POST http://localhost:8787/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
# 返回的 access_token 即 JWT，有效期 30 天，填入上方 <JWT_TOKEN>
```

**获取 MemPalace 的 bearer token**

```bash
# loopback 启动（无需 token）
mempalace serve --host 127.0.0.1 --port 8765

# 非 loopback 启动（自动生成 token 并保存到 ~/.mempalace/server/）
mempalace serve --host 0.0.0.0 --port 8765
# 查看自动生成的 token：
cat ~/.mempalace/server/token

# 或显式指定 token：
MEMPALACE_MCP_HTTP_TOKEN=my-secret mempalace serve --host 0.0.0.0 --port 8765
```


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

### 5.2 codebase-mem-mcp方式  

如果已经部署并配置 codebase-mem-mcp，使用其 tools 理解全文并生成架构图：  

**前置条件**
- `codebase-memory-mcp` 已安装并注册到 IDE 的 MCP 配置（见 [1.2 节](#12-codebase-mem-mcp-安装)）
- 推荐启用自动索引，避免每次会话手动触发：
  ```bash
  codebase-memory-mcp config set auto_index true
  codebase-memory-mcp config set auto_watch true
  ```

**提示词：codebase-mem-mcp 方式**
```
请基于 codebase-memory-mcp 的 MCP 工具，对当前工程做全文理解并输出架构图，按以下步骤执行：

1. 触发索引（首次或代码有变更时）：
   - 调用 index_repository 工具对当前工程做全量索引
   - 等待返回 nodes/edges 数量，确认索引完成

2. 获取全局架构概览：
   - 调用 get_architecture 工具
   - 它会一次性返回：语言分布、包结构、入口点、HTTP 路由、热点模块、
     边界层、分层结构、Louvain 社区检测结果

3. 探索关键调用链：
   - 对核心入口函数调用 trace_path(direction="inbound"/"outbound", depth=5)
   - 用 search_graph 按正则查找所有 Handler/Service/Repository 类节点
   - 用 Cypher 风格查询跨模块依赖，例如：
     MATCH (a)-[:CALLS]->(b) WHERE a.layer <> b.layer RETURN a, b

4. 生成 Mermaid 架构图，至少包含三张：
   - 模块/分层图（graph TB，展示 packages 与调用方向）
   - 核心调用链图（从 main/入口 controller 到关键 service）
   - 跨服务 HTTP/事件流图（如有微服务或消息队列）

5. 将以下内容写入 @docs/scene/graph-cbm.md 覆盖该文件：
   - 工程一句话定位
   - 技术栈与语言分布（来自 get_architecture）
   - 三张 Mermaid 图
   - 关键入口点清单（函数名 + 文件路径）
   - 热点/边界模块说明
   - 你发现的 3-5 条架构洞察（来自社区检测 + 调用链分析）

注意：所有图使用 Mermaid 语法，不要加 style/classDef/fill 等视觉定制，
保持节点与关系清晰即可。
```

**可选：启动 3D 图可视化 UI**

如果需要交互式浏览知识图谱（而非静态 Mermaid），使用 UI 变体：
```bash
# 安装 UI 变体（如果尚未安装）
curl -fsSL https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh | bash -s -- --ui

# 启动后访问 http://localhost:9749
codebase-memory-mcp --ui=true --port=9749
```
UI 支持 3D 旋转、节点筛选、跨模块高亮，适合团队评审时实时演示架构。

**产出物对照**

| 方式 | 产出位置 | 内容形态 |
|---|---|---|
| Skill（/gsd-map-codebase） | `.planning/codecase/*.md` | 7 份结构化 markdown |
| codebase-mem-mcp | `docs/scene/graph-cbm.md` | Mermaid 图 + 入口清单 + 架构洞察 |
| UI 变体 | `http://localhost:9749` | 交互式 3D 知识图谱 |

两种方式可以互补：Skill 产出适合归档与文档评审，codebase-mem-mcp 产出适合快速可视化与持续查询（后续任何代码问题都能直接 query 知识图谱，无需重新读源码）。

## 6.标准操作顺序：  
- 1.编写提示词：  
   在AI编程工具编写提示词，填写你的需求
- 2.提示词优化：  
  使用qoder提示词优化功能【lite模式即可，每个账号每天有次数限制】。
- 3.选择skill：  
   根据任务选择skill，无脑选着brainstorm [superpower]  
- 4.执行任务：  
   将优化的提示词，复制到你想执行的ai code工具里执行开发任务【codebase-mem-mcp 的全文理解已配置最佳，节省token】。  