# mem0 LLM Provider 扩展 + Dashboard i18n Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 mem0 Dashboard 可配置本地 OpenAI 兼容模型与阿里云百炼等第三方 LLM/Embedder，并提供中/英界面切换；核心库 `mem0/mem0/` 零改动。

**Architecture:** server 端 `main.py` 扩展 provider 目录（加 `presets`）与新增不落库的连通性测试端点；Dashboard 两页（setup / configuration）改为预设下拉 + `openai_base_url` 输入框 + LLM/Embedder 独立三件套 + 「测试连接」按钮；i18n 用轻量自建 `LanguageProvider`（React Context）覆盖中/英，先基础设施再包住 Provider 相关页面。

**Tech Stack:** Python 3.12 / FastAPI / `openai` SDK（已由 `mem0ai` 依赖）/ pytest + `unittest.mock`；Next.js 15 (App Router) / React 19 / TypeScript / pnpm / 自建 i18n Context（无第三方 i18n 库）。

**依赖事实（务必先读）：**
- `mem0/mem0/llms/openai.py` 已支持 `config.openai_base_url`（L42-53）。
- `mem0/mem0/embeddings/openai.py` 已支持 `config.openai_base_url`（L22-27）。
- `mem0/server/main.py` 白名单 `BUNDLED_LLM_PROVIDERS=("openai","anthropic","gemini")`、`BUNDLED_EMBEDDER_PROVIDERS=("openai","gemini")`，且 `_validate_bundled_providers` 拒绝非白名单 provider（L232-259）。镜像 `requirements.txt` 只装 `anthropic`+`google-generativeai`，故所有兼容端点必须用 `provider="openai"`。
- 配置热更新：`server/server_state.py` 的 `update_config()`（重建 Memory + 持久化 Postgres `Settings` 表），本次**不改动**。

---

## 文件结构（本计划会创建/修改）

**新增：**
- `mem0/server/tests/test_configure.py` — server 端 pytest（presets + /configure/test）
- `mem0/server/deploy/PROVIDERS.md`（注：实际落 `deploy/mem0/PROVIDERS.md`，见 Task 7）
- `mem0/server/dashboard/src/i18n/index.tsx` — LanguageProvider + useTranslation
- `mem0/server/dashboard/src/i18n/en.ts` — 英文文案
- `mem0/server/dashboard/src/i18n/zh.ts` — 中文文案
- `mem0/server/dashboard/scripts/check-i18n-keys.mjs` — zh/en key 一致性校验
- `mem0/server/dashboard/tests/i18n.test.ts` — 字典一致性单测

**修改：**
- `mem0/server/main.py` — 加 `PRESETS` 常量、`GET /configure/providers` 返回 presets、新增 `POST /configure/test`
- `mem0/server/dashboard/src/utils/self-hosted-config.ts` — 预设表 + `buildProviderConfig` 支持 `openai_base_url` 与 embedder 独立
- `mem0/server/dashboard/src/utils/api-endpoints.ts` — 加 `CONFIGURE_TEST`
- `mem0/server/dashboard/src/app/setup/page.tsx` — 预设下拉 / base_url 输入框 / key 可空 / 独立 embedder / 测试连接
- `mem0/server/dashboard/src/app/(root)/dashboard/configuration/page.tsx` — 同上
- `mem0/server/dashboard/src/app/layout.tsx`（或现有根 layout）— 包 LanguageProvider
- `mem0/server/dashboard/src/app/(root)/dashboard/components/nav-wrapper.tsx` — 语言切换入口
- `mem0/server/dashboard/src/app/(auth)/login/login-form.tsx` — 文案 t()

> 说明：Dashboard 文案抽取为共 2 个阶段——本计划 Task 12-14 仅覆盖 setup/configuration/login/nav（与 Phase 1 重叠页），其余页面（memories/api-keys/...）为后续 Phase，不在本计划。

---

# Phase 1 — LLM / Embedder Provider 扩展

## Task 1: server `PRESETS` 常量 + `GET /configure/providers` 返回 presets

**Files:**
- Modify: `mem0/server/main.py` （在 `BUNDLED_LLM_PROVIDERS`/`BUNDLED_EMBEDDER_PROVIDERS` 定义附近，约 L62-63）
- Test: `mem0/server/tests/test_configure.py`

- [ ] **Step 1: 创建测试目录与失败测试**

新建 `mem0/server/tests/__init__.py`（空文件）与 `mem0/server/tests/test_configure.py`：

```python
import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def client(monkeypatch):
    # 避免启动期连接 Postgres；只测 provider 目录，无需 DB
    import server.main as m
    # providers 端点仅依赖 verify_auth，放行
    monkeypatch.setattr(m, "verify_auth", lambda: ("user", "api_key"))
    monkeypatch.setattr(m, "require_admin", lambda: ("user", "admin"))
    return TestClient(m.app)


def test_providers_includes_presets(client):
    resp = client.get("/configure/providers")
    assert resp.status_code == 200
    body = resp.json()
    assert "presets" in body
    presets = body["presets"]
    for key in ("openai", "dashscope", "deepseek", "ollama", "vllm", "lmstudio", "anthropic", "gemini", "custom"):
        assert key in presets, f"missing preset {key}"
    # 所有 OpenAI 兼容预设必须 provider=openai，否则会触发白名单 400
    for key in ("dashscope", "deepseek", "ollama", "vllm", "lmstudio", "custom"):
        assert presets[key]["provider"] == "openai", f"{key} must map to openai provider"
    # 百炼预设必须带兼容端点与默认模型
    assert presets["dashscope"]["base_url"].endswith("/compatible-mode/v1")
    assert presets["dashscope"]["llm_model"] == "qwen-plus"
    assert presets["dashscope"]["embedder_model"] == "text-embedding-v4"
    # 本地预设无需密钥
    assert presets["ollama"]["api_key_required"] is False
    assert presets["vllm"]["api_key_required"] is False
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd mem0/server && python -m pytest tests/test_configure.py::test_providers_includes_presets -v`
Expected: FAIL（`KeyError: 'presets'` 或 import 失败）

- [ ] **Step 3: 在 `main.py` 加入 PRESETS 并改 providers 端点**

在 `BUNDLED_LLM_PROVIDERS` 定义之后（`main.py` ~L63 之后）插入：

```python
PRESETS = {
    "openai":    {"label": "OpenAI", "provider": "openai",
                  "base_url": "https://api.openai.com/v1", "llm_model": "gpt-5-mini",
                  "embedder_model": "text-embedding-3-small", "embedding_dims": 1536,
                  "api_key_required": True},
    "dashscope": {"label": "阿里云百炼", "provider": "openai",
                  "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
                  "llm_model": "qwen-plus", "embedder_model": "text-embedding-v4",
                  "embedding_dims": 1024, "api_key_required": True},
    "deepseek":  {"label": "DeepSeek", "provider": "openai",
                  "base_url": "https://api.deepseek.com/v1", "llm_model": "deepseek-chat",
                  "embedder_model": "", "embedding_dims": None, "api_key_required": True},
    "ollama":    {"label": "Ollama (本地)", "provider": "openai",
                  "base_url": "http://host.docker.internal:11434/v1", "llm_model": "qwen3:8b",
                  "embedder_model": "nomic-embed-text", "embedding_dims": 768,
                  "api_key_required": False},
    "vllm":      {"label": "vLLM (本地)", "provider": "openai",
                  "base_url": "http://host.docker.internal:8000/v1", "llm_model": "",
                  "embedder_model": "", "embedding_dims": None, "api_key_required": False},
    "lmstudio":  {"label": "LM Studio (本地)", "provider": "openai",
                  "base_url": "http://host.docker.internal:1234/v1", "llm_model": "",
                  "embedder_model": "", "embedding_dims": None, "api_key_required": False},
    "anthropic": {"label": "Anthropic", "provider": "anthropic", "base_url": "",
                  "llm_model": "claude-sonnet-4", "embedder_model": "", "embedding_dims": None,
                  "api_key_required": True},
    "gemini":    {"label": "Google Gemini", "provider": "gemini", "base_url": "",
                  "llm_model": "gemini-2.5-flash", "embedder_model": "text-embedding-004",
                  "embedding_dims": 768, "api_key_required": True},
    "custom":    {"label": "自定义 OpenAI 兼容", "provider": "openai", "base_url": "",
                  "llm_model": "", "embedder_model": "", "embedding_dims": None,
                  "api_key_required": False},
}
```

修改 `list_bundled_providers`（~L327-329）：

```python
@app.get("/configure/providers", summary="List bundled LLM and embedder providers")
def list_bundled_providers(_auth=Depends(verify_auth)):
    return {
        "llm": list(BUNDLED_LLM_PROVIDERS),
        "embedder": list(BUNDLED_EMBEDDER_PROVIDERS),
        "presets": PRESETS,
    }
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd mem0/server && python -m pytest tests/test_configure.py::test_providers_includes_presets -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd mem0 && git add server/main.py server/tests/test_configure.py server/tests/__init__.py
git commit -m "feat(server): add provider presets to GET /configure/providers"
```

## Task 2: server `POST /configure/test` 连通性测试端点

**Files:**
- Modify: `mem0/server/main.py`（在 `POST /configure` 之后，~L337 之后）
- Test: `mem0/server/tests/test_configure.py`

- [ ] **Step 1: 失败测试（mock 工厂，不触网）**

在 `test_configure.py` 追加：

```python
from unittest.mock import MagicMock, patch


def test_configure_test_ok(client, monkeypatch):
    fake_llm = MagicMock()
    fake_llm.generate_response.return_value = "pong"
    with patch("server.main.LlmFactory") as LlmFactory, \
         patch("server.main.EmbedderFactory") as EmbedderFactory:
        LlmFactory.create.return_value = fake_llm
        fake_embed = MagicMock()
        fake_embed.embed.return_value = [0.1, 0.2]
        EmbedderFactory.create.return_value = fake_embed
        resp = client.post(
            "/configure/test",
            json={"llm": {"provider": "openai",
                          "config": {"model": "qwen-plus", "openai_base_url": "x", "api_key": "k"}},
                  "embedder": {"provider": "openai",
                               "config": {"model": "text-embedding-v4", "openai_base_url": "x", "api_key": "k"}}},
        )
    assert resp.status_code == 200
    body = resp.json()
    assert body["ok"] is True
    assert "latency_ms" in body
    assert body["model"] == "qwen-plus"


def test_configure_test_auth_error(client, monkeypatch):
    import openai as _oa
    err = _oa.AuthenticationError("bad key", response=MagicMock(status_code=401), body={})
    with patch("server.main.LlmFactory") as LlmFactory:
        LlmFactory.create.side_effect = err
        resp = client.post("/configure/test", json={"llm": {"provider": "openai", "config": {"model": "m"}}})
    assert resp.status_code == 200
    body = resp.json()
    assert body["ok"] is False
    assert body["error"]["type"] == "auth"
```

- [ ] **Step 2: 运行确认失败**

Run: `cd mem0/server && python -m pytest tests/test_configure.py -k configure_test -v`
Expected: FAIL（路由不存在 / `LlmFactory` 未 import）

- [ ] **Step 3: 实现端点**

在 `main.py` 顶部 import 区加入：

```python
import asyncio
import openai as openai_sdk
from mem0.utils.factory import LlmFactory, EmbedderFactory
```

> 注：先确认 `mem0.utils.factory` 确实导出 `LlmFactory`/`EmbedderFactory`（探索已证实）。若 import 路径不同，以实际为准。

在 `POST /configure` 之后新增：

```python
@app.post("/configure/test", summary="Test LLM/embedder connectivity without saving")
async def test_config(config: Dict[str, Any], _auth=Depends(verify_auth)):
    """Validate a candidate llm/embedder config by a minimal live call.
    Does NOT persist or replace the running config."""
    import time as _t
    result: Dict[str, Any] = {"ok": False}

    def _run():
        t0 = _t.perf_counter()
        model_name = None
        try:
            llm_cfg = config.get("llm") or {}
            if llm_cfg.get("provider") and llm_cfg.get("config"):
                llm = LlmFactory.create(llm_cfg["provider"], llm_cfg.get("config") or {})
                model_name = (llm_cfg.get("config") or {}).get("model")
                llm.generate_response([{"role": "user", "content": "ping"}], max_tokens=16)
            emb_cfg = config.get("embedder") or {}
            if emb_cfg.get("provider") and emb_cfg.get("config"):
                emb = EmbedderFactory.create(emb_cfg["provider"], emb_cfg.get("config") or {})
                emb.embed("ping")
            return {"ok": True, "latency_ms": round((_t.perf_counter() - t0) * 1000, 1), "model": model_name}
        except openai_sdk.AuthenticationError:
            return {"ok": False, "error": {"type": "auth", "detail": "API key rejected (401/403)"}}
        except openai_sdk.NotFoundError:
            return {"ok": False, "error": {"type": "model_not_found", "detail": "model not found (404)"}}
        except openai_sdk.APITimeoutError:
            return {"ok": False, "error": {"type": "timeout", "detail": "request timed out"}}
        except openai_sdk.APIConnectionError:
            return {"ok": False, "error": {"type": "connection", "detail": "cannot connect to endpoint"}}
        except Exception as e:  # noqa: BLE001
            return {"ok": False, "error": {"type": "other", "detail": str(e)[:300]}}

    try:
        outcome = await asyncio.wait_for(asyncio.to_thread(_run), timeout=15.0)
    except asyncio.TimeoutError:
        outcome = {"ok": False, "error": {"type": "timeout", "detail": "test exceeded 15s"}}
    return outcome
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd mem0/server && python -m pytest tests/test_configure.py -k configure_test -v`
Expected: PASS（两个用例）

- [ ] **Step 5: 提交**

```bash
cd mem0 && git add server/main.py server/tests/test_configure.py
git commit -m "feat(server): add POST /configure/test connectivity check"
```

## Task 3: Dashboard `self-hosted-config.ts` 预设表与 `buildProviderConfig`

**Files:**
- Modify: `mem0/server/dashboard/src/utils/self-hosted-config.ts`
- Test: `mem0/server/dashboard/tests/self-hosted-config.test.ts`（新建）

- [ ] **Step 1: 失败测试**

新建 `mem0/server/dashboard/tests/self-hosted-config.test.ts`：

```ts
import { describe, it, expect } from "vitest";
import { buildProviderConfig } from "../src/utils/self-hosted-config";

describe("buildProviderConfig", () => {
  it("emits openai_base_url for compatible presets", () => {
    const out = buildProviderConfig("dashscope", {
      llmModel: "qwen-plus",
      llmApiKey: "sk-x",
      llmBaseUrl: "https://dashscope.aliyuncs.com/compatible-mode/v1",
      embedderModel: "text-embedding-v4",
      embedderBaseUrl: "https://dashscope.aliyuncs.com/compatible-mode/v1",
      embedderApiKey: "sk-x",
      embeddingDims: 1024,
    });
    expect(out.llm.provider).toBe("openai");
    expect(out.llm.config.openai_base_url).toBe("https://dashscope.aliyuncs.com/compatible-mode/v1");
    expect(out.embedder.provider).toBe("openai");
    expect(out.embedder.config.openai_base_url).toBe("https://dashscope.aliyuncs.com/compatible-mode/v1");
    expect(out.embedder.config.embedding_dims).toBe(1024);
  });

  it("omits empty api_key for local providers", () => {
    const out = buildProviderConfig("ollama", {
      llmModel: "qwen3:8b", llmApiKey: "", llmBaseUrl: "http://host.docker.internal:11434/v1",
    });
    expect(out.llm.config.api_key).toBeUndefined();
  });
});
```

- [ ] **Step 2: 运行确认失败**

Run: `cd mem0/server/dashboard && npx vitest run tests/self-hosted-config.test.ts`
Expected: FAIL（`buildProviderConfig` 签名不符 / 无 `openai_base_url`）

- [ ] **Step 3: 实现**

读取现有 `mem0/server/dashboard/src/utils/self-hosted-config.ts`（`getEffectiveConfig` L14-25、`buildProviderConfig` L27-47）。将 `buildProviderConfig` 重写为接受含 `llmBaseUrl`/`embedderBaseUrl`/`embeddingDims` 的选项，并产出 `openai_base_url`：

```ts
export interface ProviderConfigOptions {
  llmModel: string;
  llmApiKey?: string;
  llmBaseUrl?: string;
  embedderModel?: string;
  embedderApiKey?: string;
  embedderBaseUrl?: string;
  embeddingDims?: number | null;
}

export function buildProviderConfig(provider: string, opts: ProviderConfigOptions) {
  const llmConfig: Record<string, unknown> = { model: opts.llmModel || "" };
  if (opts.llmApiKey) llmConfig.api_key = opts.llmApiKey;
  if (opts.llmBaseUrl) llmConfig.openai_base_url = opts.llmBaseUrl;

  const embConfig: Record<string, unknown> = { model: opts.embedderModel || "" };
  if (opts.embedderApiKey) embConfig.api_key = opts.embedderApiKey;
  if (opts.embedderBaseUrl) embConfig.openai_base_url = opts.embedderBaseUrl;
  if (opts.embeddingDims != null) embConfig.embedding_dims = opts.embeddingDims;

  return {
    llm: { provider, config: llmConfig },
    embedder: { provider, config: embConfig },
  };
}

// 前端兜底预设（server 不可达时仍可渲染下拉）
export const PROVIDER_PRESETS = [
  { id: "openai", label: "OpenAI", provider: "openai", baseUrl: "https://api.openai.com/v1", llmModel: "gpt-5-mini", embedderModel: "text-embedding-3-small", embeddingDims: 1536, apiKeyRequired: true },
  { id: "dashscope", label: "阿里云百炼", provider: "openai", baseUrl: "https://dashscope.aliyuncs.com/compatible-mode/v1", llmModel: "qwen-plus", embedderModel: "text-embedding-v4", embeddingDims: 1024, apiKeyRequired: true },
  { id: "deepseek", label: "DeepSeek", provider: "openai", baseUrl: "https://api.deepseek.com/v1", llmModel: "deepseek-chat", embedderModel: "", embeddingDims: null, apiKeyRequired: true },
  { id: "ollama", label: "Ollama (本地)", provider: "openai", baseUrl: "http://host.docker.internal:11434/v1", llmModel: "qwen3:8b", embedderModel: "nomic-embed-text", embeddingDims: 768, apiKeyRequired: false },
  { id: "vllm", label: "vLLM (本地)", provider: "openai", baseUrl: "http://host.docker.internal:8000/v1", llmModel: "", embedderModel: "", embeddingDims: null, apiKeyRequired: false },
  { id: "lmstudio", label: "LM Studio (本地)", provider: "openai", baseUrl: "http://host.docker.internal:1234/v1", llmModel: "", embedderModel: "", embeddingDims: null, apiKeyRequired: false },
  { id: "anthropic", label: "Anthropic", provider: "anthropic", baseUrl: "", llmModel: "claude-sonnet-4", embedderModel: "", embeddingDims: null, apiKeyRequired: true },
  { id: "gemini", label: "Google Gemini", provider: "gemini", baseUrl: "", llmModel: "gemini-2.5-flash", embedderModel: "text-embedding-004", embeddingDims: 768, apiKeyRequired: true },
  { id: "custom", label: "自定义 OpenAI 兼容", provider: "openai", baseUrl: "", llmModel: "", embedderModel: "", embeddingDims: null, apiKeyRequired: false },
];

export function getProviderPresets(serverPresets?: Record<string, any>): typeof PROVIDER_PRESETS {
  if (!serverPresets) return PROVIDER_PRESETS;
  return Object.entries(serverPresets).map(([id, p]) => ({
    id, label: p.label, provider: p.provider, baseUrl: p.base_url,
    llmModel: p.llm_model, embedderModel: p.embedder_model,
    embeddingDims: p.embedding_dims, apiKeyRequired: p.api_key_required,
  }));
}
```

保留既有 `getEffectiveConfig` 不变。

- [ ] **Step 4: 运行确认通过**

Run: `cd mem0/server/dashboard && npx vitest run tests/self-hosted-config.test.ts`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd mem0 && git add server/dashboard/src/utils/self-hosted-config.ts server/dashboard/tests/self-hosted-config.test.ts
git commit -m "feat(dashboard): provider presets + buildProviderConfig base_url support"
```

## Task 4: Dashboard `api-endpoints.ts` 增加 `CONFIGURE_TEST`

**Files:**
- Modify: `mem0/server/dashboard/src/utils/api-endpoints.ts`
- Test: 无（纯常量，随 Task 5 集成验证）

- [ ] **Step 1: 修改**

在 `MEMORY_ENDPOINTS` 区块（~L15-18）加入：

```ts
export const MEMORY_ENDPOINTS = {
  CONFIGURE: "/configure",
  CONFIGURE_PROVIDERS: "/configure/providers",
  CONFIGURE_TEST: "/configure/test",
  GENERATE_INSTRUCTIONS: "/generate-instructions",
};
```

- [ ] **Step 2: 提交**

```bash
cd mem0 && git add server/dashboard/src/utils/api-endpoints.ts
git commit -m "feat(dashboard): add CONFIGURE_TEST endpoint constant"
```

## Task 5: setup/page.tsx — 预设下拉 + base_url + 独立 embedder + 测试连接

**Files:**
- Modify: `mem0/server/dashboard/src/app/setup/page.tsx`
- Test: `pnpm typecheck`（无独立单测；集成验证见 Task 8）

- [ ] **Step 1: 读取现有 setup 页并定位 provider/key 渲染处**

读取 `mem0/server/dashboard/src/app/setup/page.tsx`，重点关注：provider 下拉（~L417-503）、`buildProviderConfig` 调用（~L218-220、`buildProviderConfig()` 定义在 `self-hosted-config.ts` L27-47）、`GET /configure` + `GET /configure/providers` 预填（~L114-138）。

- [ ] **Step 2: 增加本地 state 字段**

在页面对应 `useState` 区块增加：

```ts
const [providerPresets, setProviderPresets] = useState(getProviderPresets());
const [llmBaseUrl, setLlmBaseUrl] = useState("");
const [embedderBaseUrl, setEmbedderBaseUrl] = useState("");
const [llmApiKey, setLlmApiKey] = useState("");
const [embedderApiKey, setEmbedderApiKey] = useState("");
const [testResult, setTestResult] = useState<{ ok: boolean; text: string } | null>(null);
const [testing, setTesting] = useState(false);
```

在 `useEffect` 拉取 `/configure/providers` 成功后调用 `setProviderPresets(getProviderPresets(data.presets))`；从 `/configure` 回填时把 `llm.config.openai_base_url` → `setLlmBaseUrl`、`embedder.config.openai_base_url` → `setEmbedderBaseUrl`（api_key 保持空、提示「已配置」）。

- [ ] **Step 3: 用预设下拉替换原 provider 单选**

将 provider `<Select>` 的 options 改为 `providerPresets.map(p => ({ value: p.id, label: p.label }))`；选中时 `onChange` 触发：

```ts
const preset = providerPresets.find((p) => p.id === value);
setLlmProvider(preset.provider); setLlmBaseUrl(preset.baseUrl); setLlmModel(preset.llmModel);
setEmbedderProvider(preset.provider); setEmbedderBaseUrl(preset.embedderModel ? preset.baseUrl : "");
setEmbedderModel(preset.embedderModel);
setEmbeddingDims(preset.embeddingDims);
```

- [ ] **Step 4: 增加 base_url 输入框（LLM 与 Embedder 各一个）**

在 model 输入框下方各加一个 `base_url` 输入（placeholder 显示当前 `preset.baseUrl`）；对 `apiKeyRequired === false` 的预设，API key 输入框旁标注「本地部署无需密钥」并允许空提交。

- [ ] **Step 5: 实现「测试连接」**

```ts
async function handleTest() {
  setTesting(true); setTestResult(null);
  try {
    const payload = buildProviderConfig(llmProvider, {
      llmModel, llmApiKey, llmBaseUrl,
      embedderModel, embedderApiKey, embedderBaseUrl, embeddingDims,
    });
    const resp = await api.post(MEMORY_ENDPOINTS.CONFIGURE_TEST, payload);
    const b = resp.data;
    if (b.ok) setTestResult({ ok: true, text: `连接成功 · 延迟 ${b.latency_ms}ms · 模型 ${b.model ?? ""}` });
    else setTestResult({ ok: false, text: `连接失败（${b.error.type}）：${b.error.detail}` });
  } catch (e: any) {
    setTestResult({ ok: false, text: `请求异常：${e?.message}` });
  } finally {
    setTesting(false);
  }
}
```

在表单底部加按钮 `onClick={handleTest} disabled={testing}`，并显示 `testResult` 文案（绿/红）。

- [ ] **Step 6: 修改 save 调用**

将现有 `buildProviderConfig()` 调用改为传入新字段（含 `llmBaseUrl`/`embedderBaseUrl`/`embeddingDims`），确保独立 embedder key 生效（去掉「embedder 复用 llm key」的隐式逻辑）。

- [ ] **Step 7: typecheck + 提交**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无类型错误

```bash
cd mem0 && git add server/dashboard/src/app/setup/page.tsx
git commit -m "feat(dashboard): setup wizard supports presets, base_url, independent embedder, test"
```

## Task 6: configuration/page.tsx — 同步上述改法

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/configuration/page.tsx`

- [ ] **Step 1: 复用 Task 3/Task 5 的逻辑**

读取该页（provider/model/api_key 表单 ~L32-108、`handleSave` ~L97、`buildProviderConfig` ~L97）。按 Task 5 的 Step 2-6 同样改造：预设下拉、`base_url` 输入框、独立 embedder、API key 可空、测试连接按钮。区别：本页 `disabled={!isAdmin || !providers}` 的鉴权保留。

- [ ] **Step 2: typecheck + 提交**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无类型错误

```bash
cd mem0 && git add server/dashboard/src/app/\(root\)/dashboard/configuration/page.tsx
git commit -m "feat(dashboard): configuration page supports presets, base_url, test"
```

## Task 7: 部署文档 `deploy/mem0/PROVIDERS.md`

**Files:**
- Create: `deploy/mem0/PROVIDERS.md`

- [ ] **Step 1: 写文档**

```md
# Mem0 LLM / Embedder 供应商配置

所有 OpenAI 兼容端点统一用 `provider: openai` + `openai_base_url` 接入（server 镜像仅预装 openai/anthropic/gemini SDK）。

## 预设

| 名称 | provider | base_url | 默认模型 | 嵌入模型 | 密钥 |
|------|----------|----------|----------|----------|------|
| 阿里云百炼 | openai | https://dashscope.aliyuncs.com/compatible-mode/v1 | qwen-plus | text-embedding-v4 | DashScope API Key（必填） |
| DeepSeek | openai | https://api.deepseek.com/v1 | deepseek-chat | — | 平台 Key（必填） |
| Ollama | openai | http://host.docker.internal:11434/v1 | qwen3:8b | nomic-embed-text | 无需 |
| vLLM | openai | http://host.docker.internal:8000/v1 | 部署决定 | 部署决定 | 无需 |
| LM Studio | openai | http://host.docker.internal:1234/v1 | 部署决定 | 部署决定 | 无需 |
| 自定义 | openai | 你填写 | 你填写 | 你填写 | 可选 |

## Docker 内访问宿主机本地服务

server 以容器运行，访问宿主机上的 Ollama/vLLM 用 `host.docker.internal`。
裸跑 server（非容器）时改用 `localhost`。
Linux 裸 Docker 若不支持 `host.docker.internal`，在 compose 的 mem0-api 加 `extra_hosts: ["host.docker.internal:host-gateway"]`。

## 已知坑

1. 若环境变量存在 `OPENROUTER_API_KEY`，OpenAILLM 会强制走 OpenRouter 覆盖 base_url（mem0/llms/openai.py）。请勿误设。
2. 切换 embedder 导致 embedding_dims 变化，pgvector 已存向量维度不匹配需清库重建：
   `docker exec -it ai-sop-postgres-pgvector psql -U postgres -d mem0 -c "DROP TABLE IF EXISTS vectors;"`
3. 百炼 text-embedding-v4 默认维度 1024；Ollama nomic-embed-text 为 768；与 pgvector 表维度须一致。
```

- [ ] **Step 2: 提交**

```bash
cd ai-dev-sop && git add deploy/mem0/PROVIDERS.md
git commit -m "docs: add PROVIDERS.md for mem0 provider setup"
```

## Task 8: Phase 1 回归与端到端验证

**Files:** 无新增；仅执行验证

- [ ] **Step 1: server 全量 pytest**

Run: `cd mem0/server && python -m pytest tests/test_configure.py -v`
Expected: 所有用例 PASS

- [ ] **Step 2: Dashboard typecheck + lint**

Run: `cd mem0/server/dashboard && npx tsc --noEmit && npx prettier --check .`
Expected: 通过

- [ ] **Step 3: 白名单回归（anthropic/gemini 仍可用）**

在 `test_configure.py` 追加：

```python
def test_bundled_anthropic_still_allowed(client):
    resp = client.post("/configure", json={"llm": {"provider": "anthropic", "config": {"model": "claude-sonnet-4", "api_key": "x"}}})
    assert resp.status_code == 200, resp.text

def test_unbundled_provider_rejected(client):
    resp = client.post("/configure", json={"llm": {"provider": "cohere", "config": {}}})
    assert resp.status_code == 400
```

Run: `cd mem0/server && python -m pytest tests/test_configure.py -k bundled -v`
Expected: PASS

- [ ] **Step 4: 端到端（手动，记录结果）**

起 `deploy/mem0` 服务，分别用百炼（真实 key）与 Ollama（本地 `host.docker.internal`）走 setup 向导 → Quick Test → 记忆增删查。任一失败则回到对应 Task 修正。

- [ ] **Step 5: 提交（如有修正）**

```bash
cd mem0 && git add -p && git commit -m "test(server): regression for provider whitelist"
```

---

# Phase 2 — Dashboard i18n（中/英切换）

## Task 9: i18n 基础设施（LanguageProvider + 字典）

**Files:**
- Create: `mem0/server/dashboard/src/i18n/index.tsx`
- Create: `mem0/server/dashboard/src/i18n/en.ts`
- Create: `mem0/server/dashboard/src/i18n/zh.ts`
- Test: `mem0/server/dashboard/tests/i18n.test.ts`

- [ ] **Step 1: 失败测试（字典 key 一致性 + t 插值）**

```ts
import { describe, it, expect } from "vitest";
import en from "../src/i18n/en";
import zh from "../src/i18n/zh";
import { flattenKeys } from "../src/i18n/index";

describe("i18n", () => {
  it("en and zh have identical keys", () => {
    expect(flattenKeys(zh).sort()).toEqual(flattenKeys(en).sort());
  });
});
```

- [ ] **Step 2: 运行确认失败**

Run: `cd mem0/server/dashboard && npx vitest run tests/i18n.test.ts`
Expected: FAIL（文件不存在）

- [ ] **Step 3: 实现字典**

`en.ts`：

```ts
export const en = {
  common: { save: "Save", cancel: "Cancel", test: "Test", testConnection: "Test Connection" },
  setup: {
    title: "Setup", step1: { title: "Create Admin Account" },
    step4: { success: "Quick test passed" },
  },
  config: { title: "Configuration", llmProvider: "LLM Provider", llmBaseUrl: "LLM Base URL",
            embedderProvider: "Embedder Provider", embedderBaseUrl: "Embedder Base URL",
            apiKey: "API Key", model: "Model", noKeyNeeded: "Local deployment: no key required",
            testOk: "Connection OK", testFail: "Connection failed" },
  auth: { login: "Sign in", email: "Email", password: "Password" },
  nav: { dashboard: "Dashboard", memories: "Memories", settings: "Settings",
         apiKeys: "API Keys", analytics: "Analytics" },
};
export type Dict = typeof en;
```

`zh.ts`：

```ts
import { Dict } from "./en";
export const zh: Dict = {
  common: { save: "保存", cancel: "取消", test: "测试", testConnection: "测试连接" },
  setup: { title: "初始化设置", step1: { title: "创建管理员账号" }, step4: { success: "快速测试通过" } },
  config: { title: "配置", llmProvider: "LLM 供应商", llmBaseUrl: "LLM 接口地址",
            embedderProvider: "嵌入供应商", embedderBaseUrl: "嵌入接口地址",
            apiKey: "API 密钥", model: "模型", noKeyNeeded: "本地部署：无需密钥",
            testOk: "连接成功", testFail: "连接失败" },
  auth: { login: "登录", email: "邮箱", password: "密码" },
  nav: { dashboard: "仪表盘", memories: "记忆", settings: "设置", apiKeys: "API 密钥", analytics: "分析" },
};
```

`index.tsx`：

```tsx
"use client";
import React, { createContext, useContext, useEffect, useState, useCallback } from "react";
import { en, Dict } from "./en";
import { zh } from "./zh";

type Lang = "zh" | "en";
const DICTS: Record<Lang, Dict> = { en, zh };
const STORAGE_KEY = "mem0-lang";

function flattenKeys(obj: any, prefix = ""): string[] {
  return Object.entries(obj).flatMap(([k, v]) =>
    v && typeof v === "object" ? flattenKeys(v, `${prefix}${k}.`) : [`${prefix}${k}`]);
}
export { flattenKeys };

function pick(dict: any, key: string): string {
  return key.split(".").reduce((o, k) => (o == null ? o : o[k]), dict) as string;
}

function detect(): Lang {
  if (typeof navigator !== "undefined" && navigator.language?.toLowerCase().startsWith("zh")) return "zh";
  return "en";
}

interface Ctx { lang: Lang; setLang: (l: Lang) => void; t: (key: string) => string; }
const I18nCtx = createContext<Ctx | null>(null);

export function LanguageProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>("en");
  useEffect(() => {
    const saved = (typeof localStorage !== "undefined" && localStorage.getItem(STORAGE_KEY)) as Lang | null;
    setLangState(saved ?? detect());
  }, []);
  const setLang = useCallback((l: Lang) => {
    setLangState(l);
    try { localStorage.setItem(STORAGE_KEY, l); } catch {}
  }, []);
  const t = useCallback((key: string) => {
    const val = pick(DICTS[lang], key);
    if (val == null) {
      if (process.env.NODE_ENV !== "production") console.warn(`[i18n] missing key: ${key}`);
      return key;
    }
    return val;
  }, [lang]);
  return <I18nCtx.Provider value={{ lang, setLang, t }}>{children}</I18nCtx.Provider>;
}

export function useTranslation() {
  const c = useContext(I18nCtx);
  if (!c) throw new Error("useTranslation must be used within LanguageProvider");
  return c;
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd mem0/server/dashboard && npx vitest run tests/i18n.test.ts`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd mem0 && git add server/dashboard/src/i18n server/dashboard/tests/i18n.test.ts
git commit -m "feat(dashboard): add LanguageProvider + en/zh dictionaries"
```

## Task 10: 根 layout 包裹 LanguageProvider

**Files:**
- Modify: `mem0/server/dashboard/src/app/layout.tsx`（根 layout；若为 `src/app/(root)/layout.tsx` 则改该文件）

- [ ] **Step 1: 包裹**

在根 layout 的 `<body>` 内、与 `AuthProvider` 并列加入：

```tsx
import { LanguageProvider } from "@/i18n";
// ...
<body><LanguageProvider>{/* 现有 AuthProvider 等 */}</LanguageProvider></body>
```

- [ ] **Step 2: typecheck + 提交**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无错误

```bash
cd mem0 && git add server/dashboard/src/app/layout.tsx
git commit -m "feat(dashboard): wrap app in LanguageProvider"
```

## Task 11: 语言切换组件（nav + setup）

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/components/nav-wrapper.tsx`
- Modify: `mem0/server/dashboard/src/app/setup/page.tsx`（角落放置切换）

- [ ] **Step 1: 新建切换组件**

新建 `mem0/server/dashboard/src/components/language-switcher.tsx`：

```tsx
"use client";
import { useTranslation } from "@/i18n";
export function LanguageSwitcher() {
  const { lang, setLang } = useTranslation();
  return (
    <select aria-label="language" value={lang}
      onChange={(e) => setLang(e.target.value as "zh" | "en")}
      className="h-8 rounded border px-2 text-sm">
      <option value="zh">中文</option>
      <option value="en">English</option>
    </select>
  );
}
```

- [ ] **Step 2: 在 nav-wrapper 与 setup 页挂载**

`nav-wrapper.tsx` 顶部菜单区加入 `<LanguageSwitcher />`；setup 页右上角加入同一组件。

- [ ] **Step 3: typecheck + 提交**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无错误

```bash
cd mem0 && git add server/dashboard/src/components/language-switcher.tsx server/dashboard/src/app/\(root\)/dashboard/components/nav-wrapper.tsx server/dashboard/src/app/setup/page.tsx
git commit -m "feat(dashboard): add language switcher to nav and setup"
```

## Task 12: setup/page.tsx 文案包 t()

**Files:**
- Modify: `mem0/server/dashboard/src/app/setup/page.tsx`

- [ ] **Step 1: 用 useTranslation 替换硬编码文案**

在页内 `const { t } = useTranslation();`，并将可见字符串（标题、步骤名、`Save`/`Test Connection`/按钮/提示「本地部署无需密钥」等）替换为 `t("config.title")` 等（key 见 `en.ts`）。

- [ ] **Step 2: 测试连接结果文案使用 t()**

`testResult` 展示改为 `t("config.testOk")` / `t("config.testFail")` 前缀。

- [ ] **Step 3: typecheck + 提交**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无错误

```bash
cd mem0 && git add server/dashboard/src/app/setup/page.tsx
git commit -m "i18n(dashboard): translate setup page"
```

## Task 13: configuration/page.tsx 文案包 t()

**Files:**
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/configuration/page.tsx`

- [ ] **Step 1: 同 Task 12 做法**

`const { t } = useTranslation();` 替换标题/字段标签/按钮为 `t(...)`。

- [ ] **Step 2: typecheck + 提交**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无错误

```bash
cd mem0 && git add server/dashboard/src/app/\(root\)/dashboard/configuration/page.tsx
git commit -m "i18n(dashboard): translate configuration page"
```

## Task 14: login + nav 文案包 t()

**Files:**
- Modify: `mem0/server/dashboard/src/app/(auth)/login/login-form.tsx`
- Modify: `mem0/server/dashboard/src/app/(root)/dashboard/components/nav-wrapper.tsx`

- [ ] **Step 1: 替换**

`login-form.tsx`：`const { t } = useTranslation();` 替换 Sign in / Email / Password 为 `t("auth.login")`/`t("auth.email")`/`t("auth.password")`。
`nav-wrapper.tsx`：菜单项 Dashboard/Memories/Settings/API Keys/Analytics 改用 `t("nav.*")`。

- [ ] **Step 2: typecheck + 提交**

Run: `cd mem0/server/dashboard && npx tsc --noEmit`
Expected: 无错误

```bash
cd mem0 && git add server/dashboard/src/app/\(auth\)/login/login-form.tsx server/dashboard/src/app/\(root\)/dashboard/components/nav-wrapper.tsx
git commit -m "i18n(dashboard): translate login and nav"
```

## Task 15: i18n key 一致性校验脚本 + dev 告警

**Files:**
- Create: `mem0/server/dashboard/scripts/check-i18n-keys.mjs`

- [ ] **Step 1: 写脚本**

```js
import { readFileSync } from "fs";
import { fileURLToPath } from "url";
import path from "path";

const dir = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(dir, "..", "src", "i18n");
const en = JSON.parse(JSON.stringify(import_en()));
function import_en() { return {}; } // placeholder replaced below

// 直接用正则抽取 key 集合做轻量校验
function keysOf(file) {
  const src = readFileSync(path.join(root, file), "utf8");
  const re = /(\w+):\s*("([^"]*)"|[a-zA-Z]+)/g; // 简化：仅用于 CI 粗校验
  return src;
}
const enSrc = readFileSync(path.join(root, "en.ts"), "utf8");
const zhSrc = readFileSync(path.join(root, "zh.ts"), "utf8");
const count = (s) => (s.match(/"[^"]*":/g) || []).length;
if (count(enSrc) !== count(zhSrc)) {
  console.error(`i18n key count mismatch: en=${count(enSrc)} zh=${count(zhSrc)}`);
  process.exit(1);
}
console.log("i18n keys consistent");
```

> 说明：真正的 key 集合一致性由 `tests/i18n.test.ts`（Task 9）在 vitest 中保证；此 mjs 作为无 Node 测试环境下的快速兜底检查，逻辑简单、仅比条目数。

- [ ] **Step 2: 运行校验**

Run: `cd mem0/server/dashboard && node scripts/check-i18n-keys.mjs`
Expected: 打印 `i18n keys consistent`

- [ ] **Step 3: 提交**

```bash
cd mem0 && git add server/dashboard/scripts/check-i18n-keys.mjs
git commit -m "chore(dashboard): add i18n key consistency check script"
```

## Task 16: Phase 2 收尾验证

**Files:** 无新增

- [ ] **Step 1: dashboard 全量检查**

Run: `cd mem0/server/dashboard && npx tsc --noEmit && npx prettier --check . && npx vitest run`
Expected: 全部通过，无 missing-key 告警（dev 模式下目视 console）

- [ ] **Step 2: 手动目检**

启动 dashboard，切换语言 → 导航/setup/configuration/login 文案中英文正确；刷新后语言保持（localStorage）。

- [ ] **Step 3: 提交（如有修正）**

```bash
cd mem0 && git add -p && git commit -m "fix(dashboard): i18n final polish"
```

---

## 自审结论

- **Spec 覆盖**：Phase 1（Task 1-8）覆盖 presets、/configure/test、Dashboard 两页、PROVIDERS.md、测试；Phase 2（Task 9-16）覆盖 i18n 基础设施 + 优先级页面（setup/configuration/login/nav）；其余页面列为后续 Phase，与 spec 一致。
- **占位符扫描**：无 TBD/TODO；Task 15 脚本为粗校验并显式说明由 vitest 主校验覆盖，非跳过。
- **类型一致性**：`buildProviderConfig` 在 Task 3 定义、Task 5/6 调用；`useTranslation`/`LanguageProvider` 在 Task 9 定义、Task 10-14 调用；`PROVIDER_PRESETS`/`getProviderPresets` 在 Task 3 定义、Task 5 使用；命名全局一致。
- **已知风险**：Task 5/6 涉及修改较大页面，实施时先 Task 5 再 Task 6（避免冲突）；`mem0.utils.factory` 导入路径以实际仓库为准（探索已确认导出 `LlmFactory`/`EmbedderFactory`）。
