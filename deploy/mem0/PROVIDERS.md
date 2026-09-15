# Mem0 LLM / Embedder 供应商配置

所有 OpenAI 兼容端点统一用 `provider: openai` + `openai_base_url` 接入。server 镜像只预装了
`openai` / `anthropic` / `gemini` 三个 SDK，因此本地模型（Ollama / vLLM / LM Studio）和阿里云百炼等
都走 `openai` provider 的兼容通道，无需新增 Python 依赖，也不会触发 `POST /configure` 的白名单校验。

## 预设

| 名称 | provider | base_url | 默认模型 | 嵌入模型 | 密钥 |
|------|----------|----------|----------|----------|------|
| OpenAI | openai | https://api.openai.com/v1 | gpt-5-mini | text-embedding-3-small | 必填 |
| 阿里云百炼 | openai | https://dashscope.aliyuncs.com/compatible-mode/v1 | qwen-plus | text-embedding-v4 | DashScope API Key（必填） |
| DeepSeek | openai | https://api.deepseek.com/v1 | deepseek-chat | — | 平台 Key（必填） |
| Ollama | openai | http://host.docker.internal:11434/v1 | qwen3:8b | nomic-embed-text | 无需 |
| vLLM | openai | http://host.docker.internal:8000/v1 | 部署决定 | 部署决定 | 无需 |
| LM Studio | openai | http://host.docker.internal:1234/v1 | 部署决定 | 部署决定 | 无需 |
| Anthropic | anthropic | （默认） | claude-sonnet-4 | — | 必填 |
| Gemini | gemini | （默认） | gemini-2.5-flash | text-embedding-004 | 必填 |
| 自定义 OpenAI 兼容 | openai | 你填写 | 你填写 | 你填写 | 可选 |

在 Dashboard 的初始化向导（Setup）与配置页（Configuration）中，选择预设会自动预填 `base_url` 与
默认模型；`base_url` 可手动修改，API Key 在本地预设下允许留空。

## Docker 内访问宿主机本地服务

server 以容器运行（`mem0-api` 服务），访问宿主机上的 Ollama / vLLM / LM Studio 时使用
`host.docker.internal`。若是**裸跑** server（非容器），把 `host.docker.internal` 改成 `localhost`。

- Linux 裸 Docker 若不支持 `host.docker.internal`，在 `docker-compose.yaml` 的 `mem0-api` 服务下加：
  `extra_hosts: ["host.docker.internal:host-gateway"]`。
- Dashboard 容器（`mem0-dashboard`）通过 `NEXT_PUBLIC_API_URL` 指向 api 容器，本地模型地址仍需
  用 `host.docker.internal` 让 api 容器去访问宿主机，而不是 Dashboard 容器。

## 已知坑

1. **不要设 `OPENROUTER_API_KEY` 环境变量。** `mem0/mem0/llms/openai.py` 在检测到该变量时会强制
   走 OpenRouter，覆盖你配置的 `openai_base_url`。

2. **切换 embedder 导致维度变化时需清库重建。** pgvector 表的向量维度在创建后固定，若
   `embedding_dims` 变化（例如从 OpenAI 1536 切到百炼 1024 或 Ollama 768），已存向量会维度不匹配。
   清理命令（按实际容器名调整）：
   ```bash
   docker exec -it mwb-postgres-pgvector psql -U postgres -d mem0 -c "DROP TABLE IF EXISTS vectors;"
   ```
   之后重启 `mem0-api` 让其按新维度重建表。

3. **维度对照**：百炼 `text-embedding-v4` 默认 1024；Ollama `nomic-embed-text` 为 768；OpenAI
   `text-embedding-3-small` 为 1536。务必与 pgvector 表维度一致。
