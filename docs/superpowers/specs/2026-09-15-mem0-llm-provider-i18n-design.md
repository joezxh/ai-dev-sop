# mem0 LLM Provider 扩展 + Dashboard i18n 设计文档

- 日期：2026-09-15
- 状态：已评审，待编写实施计划
- 仓库：mem0 子模块（`d:\projects\ai-dev-sop\mem0`，即 `mem0ai/mem0`）
- 范围：Dashboard 与 server 端配置层，不变更 `mem0/mem0/` 核心库

---

## 1. 背景与目标

mem0 自带的 Dashboard（Next.js 管理控制台）当前只支持少数 LLM provider，且 server 端通过白名单限制可选 provider，Dashboard UI 也没有 `base_url` 输入框。本设计目标：

1. **本地 OpenAI 兼容模型**（vLLM / Ollama / LM Studio）与**阿里云百炼**等第三方供应商可一键配置接入。
2. **Embedder（向量嵌入）同步支持**自定义 `base_url`，LLM 与 Embedder 配置解耦为独立三件套。
3. **Dashboard 支持中/英切换**（i18n）。

### 现状关键事实（已核实）

- `mem0/mem0/llms/openai.py` 的 `OpenAILLM` 已支持 `openai_base_url` 配置项（config 字段 `openai_base_url` / env `OPENAI_BASE_URL`）。
- `mem0/mem0/embeddings/openai.py` 的 `OpenAIEmbedding` 已支持 `openai_base_url`（L22-27）。
- server 白名单 `BUNDLED_LLM_PROVIDERS = ("openai","anthropic","gemini")`、`BUNDLED_EMBEDDER_PROVIDERS = ("openai","gemini")`（`server/main.py`）。
- server 镜像 `requirements.txt` 仅装 `anthropic` + `google-generativeai` 两个 SDK。**`ollama`/`litellm`/`lmstudio` 等专用 provider 的 Python 包不在镜像内**，因此不能简单把这些 provider 加进白名单（运行时会 `ImportError`）。所有 OpenAI 兼容端点统一走 `provider="openai"` 是唯一安全路径。
- Dashboard 当前无 i18n 基础设施（无 `next-intl`/`react-i18next`），文案全部硬编码英文，覆盖 76 个 tsx 文件。
- 配置热更新已具备：`server_state.update_config()` 重建 `Memory` 实例并持久化到 Postgres `Settings` 表。

---

## 2. 总体架构

```
Dashboard 预设下拉
   └─ POST /configure { llm:{provider:"openai", config:{openai_base_url, model, api_key}},
                        embedder:{provider:"openai", config:{openai_base_url, model, api_key, embedding_dims}} }
        ├─ _validate_bundled_providers  (白名单保持现状)
        └─ update_config → Memory.from_config → LlmFactory/EmbedderFactory → OpenAILLM/OpenAIEmbedding
   （热更新：重建 Memory + 持久化 Postgres Settings，无需新做）

Dashboard 「测试连接」按钮
   └─ POST /configure/test { llm?, embedder? }  // 临时配置，不落库，发最小请求，返回结构化结果
```

**不变更 `mem0/mem0/` 核心库**。改动集中在三层：server 配置 API、Dashboard UI、部署文档。

---

## Phase 1 — LLM / Embedder Provider 扩展（零核心库改动）

### P1-1. server 端点扩展（`mem0/server/main.py`）

**① 扩展 `GET /configure/providers`**

在现有返回 `{llm:[...], embedder:[...]}` 结构基础上**追加** `presets` 字段（向后兼容：旧 Dashboard 不读 `presets` 不报错）：

```python
PRESETS = {
    "openai":     {"label": "OpenAI", "provider": "openai",
                   "base_url": "https://api.openai.com/v1", "llm_model": "gpt-5-mini",
                   "embedder_model": "text-embedding-3-small", "embedding_dims": 1536,
                   "api_key_required": True},
    "dashscope":  {"label": "阿里云百炼", "provider": "openai",
                   "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
                   "llm_model": "qwen-plus", "embedder_model": "text-embedding-v4",
                   "embedding_dims": 1024, "api_key_required": True},
    "deepseek":   {"label": "DeepSeek", "provider": "openai",
                   "base_url": "https://api.deepseek.com/v1", "llm_model": "deepseek-chat",
                   "embedder_model": "", "embedding_dims": None, "api_key_required": True},
    "ollama":     {"label": "Ollama (本地)", "provider": "openai",
                   "base_url": "http://host.docker.internal:11434/v1", "llm_model": "qwen3:8b",
                   "embedder_model": "nomic-embed-text", "embedding_dims": 768,
                   "api_key_required": False},
    "vllm":       {"label": "vLLM (本地)", "provider": "openai",
                   "base_url": "http://host.docker.internal:8000/v1", "llm_model": "",
                   "embedder_model": "", "embedding_dims": None, "api_key_required": False},
    "lmstudio":   {"label": "LM Studio (本地)", "provider": "openai",
                   "base_url": "http://host.docker.internal:1234/v1", "llm_model": "",
                   "embedder_model": "", "embedding_dims": None, "api_key_required": False},
    "anthropic":  {"label": "Anthropic", "provider": "anthropic", "base_url": "",
                   "llm_model": "claude-sonnet-4", "embedder_model": "", "embedding_dims": None,
                   "api_key_required": True},
    "gemini":     {"label": "Google Gemini", "provider": "gemini", "base_url": "",
                   "llm_model": "gemini-2.5-flash", "embedder_model": "text-embedding-004",
                   "embedding_dims": 768, "api_key_required": True},
    "custom":     {"label": "自定义 OpenAI 兼容", "provider": "openai", "base_url": "",
                   "llm_model": "", "embedder_model": "", "embedding_dims": None,
                   "api_key_required": False},
}
```

- 所有 OpenAI 兼容预设的 `provider` 均为 `"openai"`，因此 `BUNDLED_LLM_PROVIDERS` / `BUNDLED_EMBEDDER_PROVIDERS` **保持不变**（其语义是“镜像里装了哪些 SDK”，不新增依赖则不扩）。
- `base_url` 预设值使用 `host.docker.internal`（api 容器访问宿主机本地服务）；文档说明裸跑 server（非容器）时改 `localhost`。
- `embedding_dims` 为 `None` 表示交给后端默认（不强制下发 `dimensions`，避免非 matryoshka 后端报错——参考 `OpenAIEmbedding._pass_dimensions_to_api` 逻辑）。

**② 新增 `POST /configure/test`**

- 路径：`/configure/test`；鉴权：`Depends(verify_auth)`（任意登录用户可用，仅用于验证配置正确性）。
- body：`{ llm?: {...}, embedder?: {...} }`，结构与 `/configure` 同。
- 行为：**不调用 `update_config`，不落库，不影响当前运行的 `Memory`**。现场构造一次性客户端：
  - LLM：取 `llm` 配置 → `LlmFactory.create(provider, config)` → `generate_response([{role:"user", content:"ping"}], max_tokens=16)`（用 `model` 字段决定模型；若用 openai 兼容端点，直接发 chat completion）。
  - Embedder：取 `embedder` 配置 → `EmbedderFactory.create(provider, config)` → `embed("ping")`。
- 超时上限 **15s**（用 `asyncio.wait_for` 包裹同步 SDK 调用，放到 executor）。
- 返回结构（HTTP 始终 200，错误在 body）：

```json
{ "ok": true,  "latency_ms": 812, "model": "qwen-plus" }
{ "ok": false, "error": { "type": "auth|connection|model_not_found|timeout|other",
                          "detail": "<上游错误摘要，不含 key>" } }
```

- 错误分类依据：捕获 `openai` 异常的 `status_code`/`type`（401/403→auth，404→model_not_found，连接异常→connection，超时→timeout，其余→other）。
- 日志：用现有 `logging`，记录 `provider/model/result`，**绝不打印 api_key**；`base_url` 非敏感不脱敏。

### P1-2. Dashboard 改动

**`src/utils/self-hosted-config.ts`**
- 增加与 server `presets` 字段对齐的前端兜底预设表（server 不可达时仍可渲染下拉）。
- `buildProviderConfig()` 扩展：支持 `openai_base_url` 字段；输出 LLM/Embedder 各自独立 `{provider, config:{model, api_key, openai_base_url, embedding_dims?}}`；`embedding_dims` 仅在非 `null` 时下发。

**`src/utils/api-endpoints.ts`**
- 增加 `CONFIGURE_TEST = "/configure/test"`。

**`src/app/setup/page.tsx` 与 `src/app/(root)/dashboard/configuration/page.tsx`**（两页逻辑一致）
- Provider 下拉选项改为读 `presets`（分组/扁平均可），选中即预填 `base_url` + 默认 `model`，`base_url` 可手动修改。
- API key 输入框：当 `api_key_required=false`（Ollama/vLLM/LM Studio/custom）时标注「本地部署无需密钥」，允许为空提交。
- LLM 与 Embedder 各自独立的 **base_url / model / api_key** 三件套；去掉现状「embedder 复用 LLM key」的隐式耦合（同预设时默认预填相同 key，可单独改）。
- 每个三件套旁加「测试连接」按钮 → `POST /configure/test`，展示 `✓ 延迟 xx ms (model)` 或 `✗ 错误原因`；按钮带 loading 态，失败不阻断保存。
- 配置回显：`GET /configure` 返回的 `[redacted]` api_key 保持现有脱敏逻辑，前端不回填 key（占位提示「已配置，留空则不修改」）；`base_url` 正常回显可编辑。
- Redux store 不变（现状仅 `layout` reducer，配置走本地 state）。

### P1-3. 文档

新增 `deploy/mem0/PROVIDERS.md`：
- 各预设端点、默认模型、密钥申请方式（百炼：DashScope API Key；DeepSeek：platform key；本地：无需）。
- Docker 内访问宿主机本地服务的写法（`host.docker.internal` vs 裸跑 `localhost`）。
- **已知坑 1**：若环境变量存在 `OPENROUTER_API_KEY`，`OpenAILLM` 会强制走 OpenRouter 覆盖 `base_url`（`mem0/llms/openai.py` L42-44）——文档提醒避免误设。
- **已知坑 2**：切换 `embedder` 导致 `embedding_dims` 变化时，pgvector 已存向量维度不匹配，需清库重建（文档给出清理命令）。

### P1-4. Phase 1 测试与验证

- server pytest（mock `openai` client）：
  - `/configure/test` 成功、auth/connection/model_not_found/timeout 四类失败路径。
  - `/configure/providers` 含 `presets` 字段且 key 完整。
  - 白名单回归：anthropic/gemini 仍可 POST `/configure`。
- Dashboard：`pnpm typecheck` + 现有 `prettier --check` lint。
- 端到端实测：`deploy/mem0` 的 `docker compose` 起服务，用**百炼（真实 key）** + **Ollama（本地）**各跑一遍 setup 向导 → Quick Test → 记忆增删查。
- 回归：现有 OpenAI 官方端点路径不受影响（默认 `base_url` 逻辑未改）。

---

## Phase 2 — Dashboard i18n（中/英切换）

### P2-1. 库选型与默认行为

- **选型：轻量自建 `LanguageProvider`**（不引入 `next-intl`/`react-i18next`）。理由：Dashboard 基本是 client components，无 SSR/SEO 需求；`next-intl` 需 `[locale]` 路由重构、改所有 `app/(root)` 路径，代价过大；自建方案对 `next.config`/路由零改动，包体最小。
- **默认语言行为（已确认）**：首次访问按浏览器 `navigator.language` 自动选（含 `zh` → 中文，其余 → 英文）；用户手动切换后写入 `localStorage` 持久化，刷新不丢；切换组件在导航栏与 setup 页均可见。

### P2-2. 结构与实现

```
src/i18n/
  index.tsx      // LanguageProvider + useTranslation() + 读写 localStorage + 自动探测
  en.ts          // 英文文案字典（点路径 key）
  zh.ts          // 中文文案字典（与 en.ts key 完全一致）
```

- `useTranslation()` 返回 `t(key, vars?)`，支持 `{name}` 插值；dev 环境下未命中的 key 用 `console.warn` 告警（便于发现遗漏）。
- `LanguageProvider` 挂在根 `layout.tsx`，与现有 `AuthProvider` 并列。
- 字典 key 用点路径（如 `setup.step1.title`、`common.save`、`config.testConnection`）。

### P2-3. 分阶段抽取范围

- **P2.1 基础设施**：`LanguageProvider` + `t()` + `en.ts`/`zh.ts` 加载 + 语言切换组件（放 `nav-wrapper.tsx` 顶部，setup 页角落）。
- **P2.2 优先页面（与 Phase 1 重合，故 Phase 1 先落地逻辑/字段，P2.2 再包 `t()`）**：`setup/page.tsx`、`(root)/dashboard/configuration/page.tsx`、`(auth)/login/login-form.tsx`、`nav-wrapper.tsx` 菜单、通用按钮文案（Save / Test / Test Connection / Cancel / 测试连接）。
- **P2.3（后续，不在本 spec 实现）**：memories / api-keys / entities / analytics / export / webhooks / settings 等其余页面。本 spec 仅列清单与字典规范，作为后续独立实施项。

### P2-4. 后端

无需改动（API 返回数据而非 UI 文案）。

### P2-5. Phase 2 测试

- dev 期未翻译 key 的 `console.warn` 告警可观测。
- zh/en 字典 key 一致性校验脚本（CI 或手动）：两文件 key 集合相等。
- 切换语言后两页（setup/configuration/login/nav）文案无遗漏目检；回归现有英文为默认行为。

---

## 3. 风险与未决项

- **重叠文件顺序**：`setup`/`configuration` 两页在 Phase 1 与 Phase 2.2 均改动，实施时先 Phase 1 再 Phase 2.2，避免冲突。
- **`host.docker.internal` 可解析性**：Windows/Mac Docker Desktop 默认支持；Linux 裸 Docker 需额外配置或改用宿主机 IP——文档说明。
- **embedder 维度切换**：迁移风险已在 P1-3 文档提示。
- **`OPENROUTER_API_KEY` 干扰**：已在 P1-3 文档提示。

## 4. 验收标准

- [ ] `GET /configure/providers` 返回 `presets`，含百炼/Ollama/vLLM/LM Studio/DeepSeek/custom。
- [ ] 百炼（qwen-plus + text-embedding-v4）经 Dashboard 配置后 Quick Test 通过、记忆可用。
- [ ] 本地 Ollama（qwen3:8b + nomic-embed-text）经 `host.docker.internal` 配置后可用，无需密钥。
- [ ] `测试连接` 按钮对任意预设返回延迟或错误原因，不破坏当前配置。
- [ ] LLM 与 Embedder 可配不同 base_url / key。
- [ ] 切换语言刷新后保持，导航/setup/configuration/login 文案中英文正确。
- [ ] 现有 OpenAI 官方端点路径与 anthropic/gemini 白名单回归通过。
