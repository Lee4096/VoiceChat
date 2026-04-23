#!/bin/bash
# VoiceChat Server 健康检查脚本

# 健康检查端点
HTTP_HEALTH_URL="http://localhost:8080/health"
WS_HEALTH_URL="http://localhost:8081/health"

# 超时时间（秒）
TIMEOUT=5

# 检查 HTTP API 健康状态
check_http_health() {
    if curl -sf --max-time $TIMEOUT "$HTTP_HEALTH_URL" > /dev/null 2>&1; then
        echo "HTTP API: OK"
        return 0
    else
        echo "HTTP API: FAIL"
        return 1
    fi
}

# 检查 WebSocket 健康状态
check_ws_health() {
    if curl -sf --max-time $TIMEOUT "$WS_HEALTH_URL" > /dev/null 2>&1; then
        echo "WebSocket: OK"
        return 0
    else
        echo "WebSocket: FAIL"
        return 1
    fi
}

# 主检查逻辑
main() {
    echo "=== VoiceChat Server 健康检查 ==="
    echo "时间: $(date '+%Y-%m-%d %H:%M:%S')"

    HTTP_STATUS=0
    WS_STATUS=0

    check_http_health || HTTP_STATUS=1
    check_ws_health || WS_STATUS=1

    echo "================================"

    if [ $HTTP_STATUS -eq 0 ] && [ $WS_STATUS -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

main "$@"
