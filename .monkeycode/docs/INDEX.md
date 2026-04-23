# VoiceChat 项目文档

VoiceChat 是一个实时语音聊天应用，支持 AI 语音对话、多人语音房间和 WebRTC 通信。

## 文档索引

- [系统架构](./ARCHITECTURE.md) - 系统架构设计和核心组件
- [接口文档](./INTERFACES.md) - HTTP API 和 WebSocket 接口规范
- [开发者指南](./DEVELOPER_GUIDE.md) - 开发环境设置和开发指南
- [部署文档](./DEPLOYMENT.md) - 生产环境部署指南

## 模块文档

- [语音服务](./模块/voice.md) - ASR/TTS/VAD 服务详解
- [网关服务](./模块/gateway.md) - HTTP 和 WebSocket 网关
- [认证服务](./模块/auth.md) - JWT/OAuth2 认证实现
- [信令服务](./模块/signaling.md) - WebRTC 信令服务器

## 快速导航

### 后端 (Go)
```
cmd/server/          # 服务入口
internal/
├── gateway/         # HTTP/WebSocket 网关
├── voice/           # 语音服务 (ASR/TTS/VAD)
├── auth/            # 认证服务
├── ai/              # LLM 服务
├── signaling/       # WebRTC 信令
├── user/            # 用户服务
├── room/            # 房间服务
└── config/          # 配置管理
pkg/
├── errors/          # 错误处理
├── models/          # 数据模型
└── utils/          # 工具函数
```

### 前端 (React/TypeScript)
```
frontend/src/
├── components/      # React 组件
├── hooks/           # 自定义 Hooks
├── store/           # 状态管理
├── lib/             # 工具库
└── types/           # 类型定义
```

## 技术栈

### 后端
- **语言**: Go 1.21+
- **Web 框架**: Gin
- **数据库**: PostgreSQL 15
- **缓存**: Redis 7
- **语音识别**: Sherpa-ONNX (Paraformer)
- **语音合成**: Kokoro-ONNX TTS
- **LLM**: OpenRouter API (支持多种模型)
- **WebRTC**: 原生实现 + simple-peer
- **配置**: Viper

### 前端
- **框架**: React 18 + TypeScript
- **构建**: Vite
- **状态管理**: Zustand
- **WebRTC**: SimplePeer
- **样式**: Tailwind CSS
- **测试**: Vitest

## 版本信息

- **当前版本**: 1.0.0
- **构建时间**: 2026-04-23
