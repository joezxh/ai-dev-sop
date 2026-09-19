#!/bin/bash
# install-all.sh - mem0 记忆系统一键安装脚本
#
# 旧双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，
# 记忆功能统一由自托管 mem0 提供（deploy/mem0）。
#
# 使用方法:
#   ./install-all.sh [选项]
#
# 选项:
#   --skip-mem0    跳过 mem0 服务安装
#   --help         显示帮助
#
# 仅负责 mem0 服务端安装/启动；IDE 接入（MCP 配置 + 规则文件 + 凭证）
# 请运行 scripts/mem0-setup.sh。

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认配置
SKIP_MEM0=false

show_help() {
    cat << EOF
${BLUE}mem0 记忆系统一键安装脚本${NC}

${YELLOW}使用方法:${NC}
    $0 [选项]

${YELLOW}选项:${NC}
    --skip-mem0    跳过 mem0 服务安装
    --help         显示此帮助信息

${YELLOW}示例:${NC}
    $0                                # 完整安装（启动 mem0 服务）

EOF
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-mem0)  SKIP_MEM0=true; shift ;;
        --help)       show_help; exit 0 ;;
        *) echo -e "${RED}未知选项: $1${NC}"; show_help; exit 1 ;;
    esac
done

print_header()  { echo ""; echo -e "${BLUE}========================================${NC}"; echo -e "${BLUE}  $1${NC}"; echo -e "${BLUE}========================================${NC}"; echo ""; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_info()    { echo -e "${YELLOW}ℹ $1${NC}"; }
print_error()   { echo -e "${RED}✗ $1${NC}"; }

check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 未安装"; return 1
    fi
    print_success "$1 已安装"; return 0
}

check_port() {
    local port=$1 service=$2
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1 || netstat -an | grep -q ":$port.*LISTEN"; then
        print_info "$service (端口 $port) 已在运行，跳过"; return 0
    fi
    return 1
}

# ========== mem0 服务安装 ==========
install_mem0() {
    if [ "$SKIP_MEM0" = true ]; then
        print_info "跳过 mem0 服务安装"; return
    fi

    print_header "安装 mem0"

    if ! check_command docker; then
        print_error "请先安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi

    if ! check_port 8080 "mem0-api (MCP)"; then
        print_info "启动 mem0 (API:8888 / MCP:8080 / Dashboard:3001)..."
        cd "$(dirname "$0")/../deploy/mem0"

        if [ ! -f .env ]; then
            print_error "缺少 deploy/mem0/.env，请按 docker-compose.yaml 注释准备环境变量"
            exit 1
        fi

        docker compose up -d
        print_info "等待 mem0 启动..."
        sleep 8

        if curl -sf "http://localhost:8888/docs" > /dev/null 2>&1; then
            print_success "mem0 安装成功"
        else
            print_error "mem0 启动失败，请检查日志"
            docker compose logs --tail 50
        fi
        cd - > /dev/null
    fi
}

# ========== 验证 ==========
verify_install() {
    print_header "验证安装"

    if [ "$SKIP_MEM0" = false ]; then
        print_info "检查 mem0..."
        if curl -sf "http://localhost:8888/docs" > /dev/null 2>&1; then
            print_success "mem0 API: http://localhost:8888 ✓"
        else
            print_error "mem0 API: http://localhost:8888 ✗"
        fi
        if curl -sf "http://localhost:3001" > /dev/null 2>&1; then
            print_success "mem0 Dashboard: http://localhost:3001 ✓"
        fi
    fi
}

main() {
    print_header "mem0 记忆系统一键安装"

    install_mem0
    verify_install

    print_header "安装完成"
    echo -e "${GREEN}下一步:${NC}"
    echo "  1. 在 Dashboard (http://localhost:3001) 创建 API Key"
    echo "  2. 接入 IDE：运行 scripts/mem0-setup.sh（一键配置 MCP + 规则文件 + 凭证并校验）"
    echo ""
    echo -e "${BLUE}详细文档: docs/quick-ref/mem0-manual.md${NC}"
    echo ""
}

main
