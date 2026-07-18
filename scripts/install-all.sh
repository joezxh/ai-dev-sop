#!/bin/bash
# install-all.sh - 双轨记忆系统一键安装脚本
#
# 使用方法:
#   ./install-all.sh [选项]
#
# 选项:
#   --skip-mempalace    跳过 MemPalace 安装
#   --skip-cbmem        跳过 cbmem-team 安装
#   --skip-ide          跳过 IDE 配置
#   --help              显示帮助

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
SKIP_MEMPALACE=false
SKIP_CBMEM=false
SKIP_IDE=false
SERVER_URL="http://192.168.100.83:8787"
MEMPALACE_URL="http://192.168.100.83:8089"

# 帮助信息
show_help() {
    cat << EOF
${BLUE}双轨记忆系统一键安装脚本${NC}

${YELLOW}使用方法:${NC}
    $0 [选项]

${YELLOW}选项:${NC}
    --skip-mempalace    跳过 MemPalace 安装
    --skip-cbmem        跳过 cbmem-team 安装
    --skip-ide          跳过 IDE 配置
    --server-url URL    指定 cbmem-team 服务地址 (默认: $SERVER_URL)
    --mempalace-url URL 指定 MemPalace 服务地址 (默认: $MEMPALACE_URL)
    --help              显示此帮助信息

${YELLOW}示例:${NC}
    $0                          # 完整安装
    $0 --skip-mempalace         # 仅安装 cbmem-team
    $0 --server-url http://localhost:8787  # 使用本地服务

EOF
}

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-mempalace)
            SKIP_MEMPALACE=true
            shift
            ;;
        --skip-cbmem)
            SKIP_CBMEM=true
            shift
            ;;
        --skip-ide)
            SKIP_IDE=true
            shift
            ;;
        --server-url)
            SERVER_URL="$2"
            shift 2
            ;;
        --mempalace-url)
            MEMPALACE_URL="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}未知选项: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 打印标题
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

# 打印成功
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# 打印信息
print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# 打印错误
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# 检查命令
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 未安装"
        return 1
    fi
    print_success "$1 已安装"
    return 0
}

# 检查端口是否可用
check_port() {
    local port=$1
    local service=$2
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1 || netstat -an | grep -q ":$port.*LISTEN"; then
        print_info "$service (端口 $port) 已在运行，跳过"
        return 0
    fi
    return 1
}

# ========== MemPalace 安装 ==========
install_mempalace() {
    if [ "$SKIP_MEMPALACE" = true ]; then
        print_info "跳过 MemPalace 安装"
        return
    fi

    print_header "安装 MemPalace"

    # 检查 Docker
    if ! check_command docker; then
        print_error "请先安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi

    # 检查端口
    if ! check_port 8089 "MemPalace"; then
        print_info "启动 MemPalace..."

        # 进入部署目录
        cd "$(dirname "$0")/../deploy"

        # 复制环境变量
        if [ ! -f .env ]; then
            cp mempalace.env.example .env
            print_info "已创建 .env 文件，请编辑设置 MEMPALACE_PORT=8089"
        fi

        # 启动服务
        docker compose -f docker-compose.mempalace.yml up -d

        # 等待服务就绪
        print_info "等待 MemPalace 启动..."
        sleep 5

        # 验证
        if curl -sf "$MEMPALACE_URL/healthz" > /dev/null 2>&1; then
            print_success "MemPalace 安装成功"
        else
            print_error "MemPalace 启动失败，请检查日志"
            docker compose -f docker-compose.mempalace.yml logs
        fi
    fi
}

# ========== cbmem-team 安装 ==========
install_cbmem() {
    if [ "$SKIP_CBMEM" = true ]; then
        print_info "跳过 cbmem-team 安装"
        return
    fi

    print_header "安装 cbmem-team"

    # 检查 Go
    if ! check_command go; then
        print_error "请先安装 Go: https://go.dev/dl/"
        exit 1
    fi

    # 检查端口
    if ! check_port 8787 "cbmem-team"; then
        print_info "编译 cbmem-team..."

        # 进入项目目录
        cd "$(dirname "$0")/../tools/cbmem-team"

        # 编译
        go build -o bin/cbmem-team ./cmd/cbmem-team

        # 安装
        print_info "安装 cbmem-team 到 /usr/local/bin..."
        sudo install -m 0755 bin/cbmem-team /usr/local/bin/cbmem-team

        # 检查 systemctl
        if command -v systemctl &> /dev/null; then
            print_info "创建 systemd 服务..."
            sudo cp deploy/cbmem-team.service /etc/systemd/system/
            sudo systemctl daemon-reload
            sudo systemctl enable --now cbmem-team
        fi

        # 验证
        sleep 2
        if curl -sf "$SERVER_URL/healthz" > /dev/null 2>&1; then
            print_success "cbmem-team 安装成功"
        else
            print_error "cbmem-team 启动失败，请检查配置"
        fi
    fi
}

# ========== IDE 配置 ==========
configure_ide() {
    if [ "$SKIP_IDE" = true ]; then
        print_info "跳过 IDE 配置"
        return
    fi

    print_header "配置 AI IDE MCP"

    # 检测 IDE
    local ide=""
    if [ -d "$HOME/.cursor" ]; then
        ide="cursor"
    elif [ -d "$HOME/.qoder" ]; then
        ide="qoder"
    fi

    if [ -z "$ide" ]; then
        print_info "未检测到支持的 IDE (Cursor/Qoder)，跳过"
        return
    fi

    print_info "检测到 $ide"

    # 创建配置目录
    local config_dir="$HOME/.$ide"
    mkdir -p "$config_dir"

    # 生成配置
    local config_file="$config_dir/mcp.json"
    print_info "创建 MCP 配置: $config_file"

    cat > "$config_file" << EOF
{
  "mcpServers": {
    "cbmem-team": {
      "url": "$SERVER_URL/mcp?as=\${USER_ID}&project=\${SERVER_PROJECT_PATH}",
      "headers": {
        "Authorization": "Bearer \${JWT_TOKEN}"
      }
    }
  }
}
EOF

    print_success "MCP 配置已创建"
    print_info "请编辑 $config_file 填入正确的 USER_ID, SERVER_PROJECT_PATH 和 JWT_TOKEN"
    print_info "然后重启 $ide 生效"
}

# ========== 验证安装 ==========
verify_install() {
    print_header "验证安装"

    # MemPalace
    if [ "$SKIP_MEMPALACE" = false ]; then
        print_info "检查 MemPalace..."
        if curl -sf "$MEMPALACE_URL/healthz" > /dev/null 2>&1; then
            print_success "MemPalace: $MEMPALACE_URL ✓"
        else
            print_error "MemPalace: $MEMPALACE_URL ✗"
        fi
    fi

    # cbmem-team
    if [ "$SKIP_CBMEM" = false ]; then
        print_info "检查 cbmem-team..."
        if curl -sf "$SERVER_URL/healthz" > /dev/null 2>&1; then
            print_success "cbmem-team: $SERVER_URL ✓"
        else
            print_error "cbmem-team: $SERVER_URL ✗"
        fi
    fi
}

# ========== 主流程 ==========
main() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  双轨记忆系统一键安装${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    install_mempalace
    install_cbmem
    configure_ide
    verify_install

    print_header "安装完成"
    echo -e "${GREEN}下一步:${NC}"
    echo "  1. 获取 JWT Token (联系管理员)"
    echo "  2. 编辑 ~/.cursor/mcp.json 填入 Token"
    echo "  3. 重启 Cursor/IDE"
    echo ""
    echo -e "${BLUE}详细文档: docs/sop/SOP-M1-installation.md${NC}"
}

main
