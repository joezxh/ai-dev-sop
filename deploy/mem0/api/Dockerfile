FROM python:3.12-slim

WORKDIR /app

# 安装基础依赖（使用阿里云 pip 镜像加速）
COPY requirements.txt .
RUN pip install --no-cache-dir \
    -i https://mirrors.aliyun.com/pypi/simple/ \
    --trusted-host mirrors.aliyun.com \
    -r requirements.txt \
    "psycopg[binary,pool]"

# 安装图记忆 + 排序依赖
# mem0ai 2.x has no graph engine (its "graph" extra does not exist); the graph
# layer is self-built in server/graph_memory.py on top of langchain-neo4j.
RUN pip install --no-cache-dir \
    -i https://mirrors.aliyun.com/pypi/simple/ \
    --trusted-host mirrors.aliyun.com \
    rank-bm25 langchain-neo4j neo4j

# 复制应用源码
COPY . .

# supervisord 同容器运行 mem0-api (uvicorn :8000) 与 mem0-mcp (:8080)
COPY supervisord.conf /etc/supervisor/supervisord.conf

EXPOSE 8000 8080

ENV PYTHONUNBUFFERED=1

CMD ["supervisord", "-c", "/etc/supervisor/supervisord.conf"]
