#!/bin/bash
# VoiceChat Server 启动脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查环境
check_env() {
    log_info "检查运行环境..."

    # 检查 .env 文件
    if [ ! -f .env ]; then
        log_warn ".env 文件不存在，创建默认配置..."
        cp .env.example .env
        log_warn "请编辑 .env 文件填入实际配置"
    fi

    # 检查二进制文件
    if [ ! -f ./bin/voicechat-server ]; then
        log_warn "二进制文件不存在，正在构建..."
        make build
    fi

    # 检查模型文件
    if [ ! -d models ]; then
        log_warn "models 目录不存在，创建中..."
        mkdir -p models
    fi

    log_info "环境检查完成"
}

# 启动服务
start_service() {
    log_info "启动 VoiceChat Server..."

    # 加载环境变量
    export $(grep -v '^#' .env | xargs)

    # 创建日志目录
    mkdir -p logs

    # 后台启动
    nohup ./bin/voicechat-server > logs/voice.log 2>&1 &

    # 保存 PID
    echo $! > .voicechat-server.pid

    # 等待启动
    sleep 2

    # 检查是否启动成功
    if kill -0 $(cat .voicechat-server.pid) 2>/dev/null; then
        log_info "VoiceChat Server 启动成功 (PID: $(cat .voicechat-server.pid))"
    else
        log_error "VoiceChat Server 启动失败，请检查日志"
        exit 1
    fi
}

# 显示状态
show_status() {
    if [ -f .voicechat-server.pid ]; then
        PID=$(cat .voicechat-server.pid)
        if kill -0 $PID 2>/dev/null; then
            log_info "VoiceChat Server 运行中 (PID: $PID)"
        else
            log_warn "VoiceChat Server 未运行 (PID 文件过期)"
        fi
    else
        log_warn "VoiceChat Server 未运行"
    fi
}

# 主函数
main() {
    case "${1:-start}" in
        start)
            check_env
            start_service
            ;;
        stop)
            if [ -f .voicechat-server.pid ]; then
                log_info "停止 VoiceChat Server..."
                kill $(cat .voicechat-server.pid) 2>/dev/null || true
                rm -f .voicechat-server.pid
                log_info "VoiceChat Server 已停止"
            else
                log_warn "VoiceChat Server 未运行"
            fi
            ;;
        restart)
            $0 stop
            sleep 1
            $0 start
            ;;
        status)
            show_status
            ;;
        logs)
            tail -f logs/voice.log
            ;;
        *)
            echo "用法: $0 {start|stop|restart|status|logs}"
            exit 1
            ;;
    esac
}

main "$@"
