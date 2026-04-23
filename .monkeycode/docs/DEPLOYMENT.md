# VoiceChat 部署文档

## 概述

VoiceChat 支持多种部署方式：
1. **Docker Compose** - 开发和小规模生产
2. **Kubernetes** - 大规模生产环境

## Docker Compose 部署

### 环境要求
- Docker 20.10+
- Docker Compose 2.0+
- 至少 4GB RAM

### 快速部署

1. **配置环境变量**
```bash
# 创建 .env 文件
cat > .env << EOF
DB_PASSWORD=your_secure_db_password
REDIS_PASSWORD=your_secure_redis_password
LLM_API_KEY=your_openrouter_api_key
EOF
```

2. **启动服务**
```bash
# 开发环境
docker-compose up -d

# 生产环境
make prod
```

3. **验证部署**
```bash
# 检查服务状态
docker-compose ps

# 检查健康状态
curl http://localhost:8080/health
curl http://localhost:8081/health
```

### 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| HTTP API | 8080 | REST API |
| WebSocket | 8081 | 实时通信 |
| Signaling | 8082 | WebRTC 信令 |
| Nginx | 80/443 | 反向代理 (生产) |
| PostgreSQL | 5432 | 数据库 |
| Redis | 6379 | 缓存 |

## Kubernetes 部署

### 前置要求
- Kubernetes 1.24+
- kubectl 配置完成
- Ingress Controller (如 nginx-ingress)

### 部署步骤

1. **创建命名空间**
```bash
kubectl create namespace voicechat
```

2. **配置 Secret**
```bash
kubectl create secret generic voicechat-secrets \
  --from-literal=DB_PASSWORD=xxx \
  --from-literal=REDIS_PASSWORD=xxx \
  --from-literal=LLM_API_KEY=xxx \
  -n voicechat
```

3. **部署数据库和缓存**
```bash
kubectl apply -f deploy/k8s/database.yaml -n voicechat
```

4. **部署应用**
```bash
kubectl apply -f deploy/k8s/deployment.yaml -n voicechat
```

5. **配置 Ingress**
```bash
kubectl apply -f deploy/k8s/ingress.yaml -n voicechat
```

### Kubernetes 配置说明

#### Deployment 配置
- 副本数: 2-3
- 资源限制: 2 CPU, 2GB Memory
- 健康检查: HTTP GET /health
- 滚动更新策略

#### Persistence
- PostgreSQL: PersistentVolumeClaim
- Redis: PersistentVolumeClaim
- 模型文件: ConfigMap 或 PersistentVolumeClaim (只读)

## 生产环境配置

### Nginx 配置
参考 `deploy/nginx/nginx.conf`:
- SSL/TLS 终止
- WebSocket 代理
- 静态文件缓存
- Gzip 压缩

### 数据库优化
- 连接池大小: 20-50
- 缓存: 256MB
- 备份策略: 每日全量 + 增量

### Redis 配置
- 内存限制: 256MB-1GB
- 持久化: AOF
- 密码认证

### 安全配置

1. **防火墙规则**
```bash
# 仅允许必要端口
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 8080/tcp  # 仅内网
```

2. **Docker 安全**
- 以非 root 用户运行
- 只读根文件系统
- 限制容器能力

3. **环境变量**
- 使用 Secret 管理敏感信息
- 定期轮换密钥

## 监控与日志

### 日志
```bash
# Docker Compose 日志
docker-compose logs -f

# Kubernetes 日志
kubectl logs -f deployment/voicechat-server -n voicechat
```

### 健康检查
```bash
# HTTP 健康检查
curl http://localhost:8080/health

# WebSocket 健康检查
curl http://localhost:8081/health
```

### 资源监控
- Docker stats
- Kubernetes Dashboard
- Prometheus + Grafana (可选)

## 备份与恢复

### 数据库备份
```bash
# 备份
pg_dump -h localhost -U postgres voicechat > backup.sql

# 恢复
psql -h localhost -U postgres voicechat < backup.sql
```

### Redis 备份
```bash
# 保存快照
redis-cli SAVE

# 复制备份文件
cp /var/lib/redis/dump.rdb backup/
```

## 故障排查

### 服务无法启动
1. 检查日志: `docker-compose logs voicechat-server`
2. 检查端口占用: `netstat -tulpn | grep 8080`
3. 验证环境变量

### 数据库连接失败
1. 检查 PostgreSQL 状态
2. 验证数据库凭据
3. 检查网络连接

### AI 模型加载失败
1. 确认模型文件存在
2. 检查挂载路径
3. 验证文件权限

## 扩展

### 水平扩展
- 增加 WebSocket Server 副本
- 使用负载均衡
- Redis Session 共享

### 性能优化
- 启用 Redis 缓存
- 数据库索引优化
- CDN 静态资源
