# VoiceChat 环境变量配置

复制 `.env.example` 为 `.env` 并修改配置：

```bash
cp .env.example .env
```

## 数据库配置

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `DB_PASSWORD` | postgres | PostgreSQL 密码 |
| `REDIS_PASSWORD` | redis | Redis 密码 |

## JWT 配置

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `JWT_SECRET` | change-me-in-production | JWT 签名密钥（生产环境必须修改） |

## GitHub OAuth

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `GITHUB_CLIENT_ID` | - | GitHub OAuth App Client ID |
| `GITHUB_CLIENT_SECRET` | - | GitHub OAuth App Client Secret |
| `GITHUB_CALLBACK_URL` | http://localhost/api/v1/auth/callback/github | 回调地址 |

## Google OAuth

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `GOOGLE_CLIENT_ID` | - | Google OAuth Client ID |
| `GOOGLE_CLIENT_SECRET` | - | Google OAuth Client Secret |
| `GOOGLE_CALLBACK_URL` | http://localhost/api/v1/auth/callback/google | 回调地址 |

## 日志配置

| 变量 | 默认值 | 描述 |
|------|--------|------|
| `LOG_LEVEL` | INFO | 日志级别: DEBUG, INFO, WARN, ERROR |
