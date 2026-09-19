#!/bin/bash
set -e

# 在 mem0 主库启用 pgvector 扩展，并创建 mem0_app 应用库（用户/鉴权/API key 数据）。
# 仅当数据目录为空（首次初始化）时由 PostgreSQL 容器的 entrypoint 执行一次。
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS vector;
    SELECT 'CREATE DATABASE mem0_app'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'mem0_app')\gexec
EOSQL
