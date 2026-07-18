#!/bin/sh
# MemPalace 初始化脚本
# 用于创建团队 Wing 和 Room 结构
#
# 使用方法:
#   docker compose --profile cli run mempalace-cli /scripts/init-team.sh

set -e

echo "=== MemPalace Team Initialization ==="

# 初始化 Palace
echo "[1/5] 初始化 Palace..."
mempalace init /data

# 创建团队共享 Wing
echo "[2/5] 创建团队共享 Wing..."
mempalace create-wing --name "team-shared" --description "团队共享知识库"
mempalace create-wing --name "team-backend" --description "后端团队知识库"
mempalace create-wing --name "team-frontend" --description "前端团队知识库"

# 创建团队 Room
echo "[3/5] 创建团队 Room..."
# team-shared 的 Room
mempalace create-room --wing "team-shared" --name "api-standards" --description "API 设计规范"
mempalace create-room --wing "team-shared" --name "architecture" --description "架构决策"
mempalace create-room --wing "team-shared" --name "best-practices" --description "最佳实践"
mempalace create-room --wing "team-shared" --name "lessons-learned" --description "经验教训"

# team-backend 的 Room
mempalace create-room --wing "team-backend" --name "database-guidelines" --description "数据库设计规范"
mempalace create-room --wing "team-backend" --name "incident-reports" --description "事故报告"
mempalace create-room --wing "team-backend" --name "performance-tuning" --description "性能调优"

# team-frontend 的 Room
mempalace create-room --wing "team-frontend" --name "component-library" --description "组件库规范"
mempalace create-room --wing "team-frontend" --name "coding-style" --description "编码风格"

# 创建示例 Drawer
echo "[4/5] 创建示例 Drawer..."
mempalace add-drawer \
  --wing "team-shared" \
  --room "api-standards" \
  --hall "hall_advice" \
  --content "RESTful API 设计规范 v1.0

1. URL 使用名词复数: /users, /orders
2. 使用 HTTP 方法: GET(查询), POST(创建), PUT(更新), DELETE(删除)
3. 返回标准状态码: 200成功, 201创建, 400客户端错误, 500服务端错误
4. 分页使用 limit/offset: /users?limit=20&offset=0
5. 错误响应格式: {code, message, data}"

mempalace add-drawer \
  --wing "team-shared" \
  --room "architecture" \
  --hall "hall_facts" \
  --content "架构决策记录 (ADR)

ADR-001: 选择微服务架构
日期: 2024-01-15
决策者: 技术委员会
状态: 已接受

决策:
- 采用微服务架构，按业务领域划分服务
- 使用 API Gateway 统一入口
- 服务间通信使用 REST/gRPC

影响:
- 需要引入服务治理
- 增加运维复杂度"

echo "[5/5] 验证初始化..."
mempalace status

echo "=== MemPalace Team Initialization Complete ==="
