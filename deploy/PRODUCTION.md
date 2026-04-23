# Voice Server 部署指南

## 概述

Voice Server 是一个高性能的实时语音 AI 服务，支持语音识别（ASR）、语音合成（TTS）和大语言模型（LLM）集成。

## 快速开始

### 1. 环境要求

- Go 1.23+
- Docker & Docker Compose
- PostgreSQL 15+ (可选，使用 Docker)
- Redis 7+ (可选，使用 Docker)

### 2. 本地开发

```bash
# 克隆项目
git clone <repository-url>
cd voice

# 安装依赖并构建
make setup

# 启动服务（使用 Docker）
make docker-up

# 或直接运行
make run
```

### 3. 环境配置

复制环境变量配置模板：

```bash
cp .env.example .env
# 编辑 .env 填入实际配置
```

## 部署方式

### Docker Compose (开发/测试环境)

```bash
# 构建并启动
make docker-build
make docker-up

# 查看状态
docker-compose ps

# 查看日志
make logs

# 停止服务
make docker-down
```

### Docker Compose (生产环境)

```bash
# 使用生产配置启动
make prod

# 查看生产日志
make prod-logs

# 停止生产环境
make prod-down
```

### Kubernetes

```bash
# 应用所有配置
kubectl apply -f deploy/k8s/

# 检查部署状态
kubectl get pods -n voice

# 查看日志
kubectl logs -n voice -l app=voicechat-server

# 扩缩容
kubectl scale deployment voicechat-server -n voice --replicas=5
```

## 架构

```
                    ┌─────────────────────────────────────────┐
                    │              Nginx (可选)               │
                    │         反向代理 / SSL 终结             │
                    └──────────────────┬──────────────────────┘
                                       │
        ┌──────────────────────────────┼──────────────────────────────┐
        │                              │                              │
        ▼                              ▼                              ▼
┌───────────────┐              ┌───────────────┐              ┌───────────────┐
│  HTTP API     │              │  WebSocket    │              │  Signaling    │
│  :8080        │              │  :8081        │              │  :8082        │
└───────┬───────┘              └───────┬───────┘              └───────┬───────┘
        │                              │                              │
        └──────────────────────────────┼──────────────────────────────┘
                                       │
                    ┌──────────────────┴──────────────────┐
                    │         Voice Server               │
                    │  ┌─────────┐  ┌─────────┐        │
                    │  │   ASR   │  │   TTS   │        │
                    │  └────┬────┘  └────┬────┘        │
                    │       └──────┬──────┘             │
                    │          ┌───┴───┐               │
                    │          │  LLM  │               │
                    │          └───────┘               │
                    └─────────────────┬─────────────────┘
                                      │
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
      ┌───────────────┐      ┌───────────────┐      ┌───────────────┐
      │  PostgreSQL   │      │    Redis      │      │   Models      │
      │   :5432       │      │    :6379      │      │  (本地存储)   │
      └───────────────┘      └───────────────┘      └───────────────┘
```

## 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| HTTP API | 8080 | REST API |
| WebSocket | 8081 | 实时语音通信 |
| Signaling | 8082 | WebRTC 信令 |
| PostgreSQL | 5432 | 数据库 (Docker) |
| Redis | 6379 | 缓存 (Docker) |
| Nginx | 80/443 | 反向代理 (可选) |

## 健康检查

```bash
# 本地检查
./scripts/healthcheck.sh

# Docker 检查
curl http://localhost:8080/health
curl http://localhost:8081/health
```

## 日志管理

```bash
# 本地日志
tail -f logs/voice.log

# Docker 日志
docker-compose logs -f voicechat-server

# Kubernetes 日志
kubectl logs -n voice -f deployment/voicechat-server
```

## 构建发布

```bash
# 构建 Linux amd64
make build

# 构建所有平台
make build-linux
make build-darwin
make build-arm

# 打包发布
make package
```

## 配置参考

### 必需环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_HOST` | localhost | PostgreSQL 地址 |
| `DB_PASSWORD` | - | 数据库密码 |
| `REDIS_HOST` | localhost | Redis 地址 |
| `JWT_SECRET` | - | JWT 密钥 (使用 `openssl rand -base64 32` 生成) |
| `LLM_API_KEY` | - | LLM API 密钥 |

### 可选配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `LOG_LEVEL` | INFO | 日志级别 |
| `LLM_MODEL` | minimax/minimax-m2.5:free | LLM 模型 |
| `VOICE_SAMPLE_RATE` | 16000 | 音频采样率 |

## 故障排除

### 服务启动失败

1. 检查端口占用：`lsof -i :8080`
2. 检查日志：`tail -f logs/voice.log`
3. 验证环境变量：确保 `.env` 文件存在且配置正确

### 数据库连接失败

1. 检查 PostgreSQL 是否运行：`docker-compose ps postgres`
2. 验证连接：`psql -h localhost -U postgres -d voice`

### 模型文件缺失

1. 下载 ASR 模型到 `models/paraformer/`
2. 下载 TTS 模型到 `models/kokoro/`

## 安全建议

1. 生产环境务必修改所有密钥
2. 使用 HTTPS 加密通信
3. 配置防火墙规则
4. 定期更新依赖
5. 启用数据库 SSL 连接
