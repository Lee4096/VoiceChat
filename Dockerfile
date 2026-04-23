# Voice Server Dockerfile
# 多阶段构建优化镜像大小

# ============ 构建阶段 ============
FROM golang:1.23-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache \
    gcc \
    musl-dev \
    make \
    git

# 设置工作目录
WORKDIR /build

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建参数
ARG VERSION=latest
ARG BUILD_TIME=unknown

# 构建二进制文件
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="\
        -s -w \
        -X main.Version=${VERSION} \
        -X main.BuildTime=${BUILD_TIME}" \
        -o voice-server \
        ./cmd/server

# ============ 运行阶段 ============
FROM ubuntu:22.04

# 标签
LABEL maintainer="Voice Team"
LABEL description="VoiceChat Server - Real-time Voice AI Service"
LABEL version="${VERSION}"

# 安装运行时依赖
RUN apt-get update && apt-get install -y \
    --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 创建非 root 用户
RUN groupadd -r voicechat && useradd -r -g voicechat voicechat

# 创建必要的目录
RUN mkdir -p /app/logs /app/models && \
    chown -R voicechat:voicechat /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/voice-server /app/

# 复制健康检查脚本
COPY --from=builder /build/scripts/healthcheck.sh /app/scripts/

# 设置工作目录
WORKDIR /app

# 切换到非 root 用户
USER voicechat

# 暴露端口
EXPOSE 8080 8081 8082

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD /app/scripts/healthcheck.sh || exit 1

# 环境变量
ENV APP_HOME=/app \
    LOG_LEVEL=INFO \
    TZ=Asia/Shanghai

# 入口点
ENTRYPOINT ["/app/voice-server"]
