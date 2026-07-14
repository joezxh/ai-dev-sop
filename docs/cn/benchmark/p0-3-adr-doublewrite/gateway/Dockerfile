FROM python:3.11-slim

# 基础依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Python 依赖
COPY gateway/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# 应用代码
COPY gateway/ /app/gateway/
COPY scripts/ /app/scripts/

# 数据卷
RUN mkdir -p /data/memplace/index
VOLUME ["/data/memplace"]

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD curl -fsS http://localhost:8888/health || exit 1

# 暴露端口
EXPOSE 8888

# 启动命令
CMD ["uvicorn", "gateway.server:app", "--host", "0.0.0.0", "--port", "8888", "--workers", "2"]
