# VoiceChat

实时语音 AI 聊天系统，使用 WebSocket 进行实时通信，Sherpa-ONNX 进行语音识别（ASR）和语音合成（TTS）。

## 技术栈

- **后端**: Go + WebSocket + PostgreSQL + Redis
- **前端**: React + Vite + TypeScript + TailwindCSS
- **语音**: Sherpa-ONNX (Paraformer ASR + Kokoro TTS)
- **AI**: DeepSeek LLM API

## 项目结构

```
.
├── cmd/server/          # Go 后端服务器
├── internal/            # 内部包
│   ├── ai/             # LLM 服务
│   ├── auth/           # 认证
│   ├── config/         # 配置
│   ├── gateway/        # 网关 (HTTP, WebSocket)
│   ├── repository/     # 数据访问层
│   ├── room/           # 房间管理
│   ├── signaling/      # 信令服务
│   ├── user/           # 用户管理
│   └── voice/          # 语音服务 (ASR/TTS)
├── pkg/                # 公共包
│   ├── models/         # 数据模型
│   └── utils/          # 工具函数
├── models/             # AI 模型
│   ├── paraformer/     # ASR 模型
│   └── kokoro/         # TTS 模型
├── frontend/           # React 前端
├── deploy/            # 部署配置
└── docs/              # 文档
```

## 系统架构

```mermaid
graph TB
    subgraph Client["客户端"]
        Browser["浏览器 React + Vite"]
        Mobile["移动端 App"]
    end

    subgraph Gateway["API 网关层"]
        Nginx["Nginx 反向代理"]
        HTTP["HTTP API Gin"]
        WS["WebSocket Gorilla"]
        Signal["信令服务 WebRTC"]
        Auth["JWT 认证 OAuth2"]
    end

    subgraph Core["核心服务层"]
        Room["房间服务"]
        User["用户服务"]
        Voice["语音服务"]
        AI["AI 服务"]
    end

    subgraph Data["数据层"]
        PG["PostgreSQL"]
        Redis["Redis 缓存"]
    end

    subgraph AIModels["AI 模型"]
        ASR["Paraformer ASR"]
        TTS["Kokoro TTS"]
    end

    subgraph External["外部服务"]
        LLM["LLM API DeepSeek"]
        OAuth["OAuth2 GitHub Google"]
        STUN["STUN/TURN"]
    end

    Browser --> Nginx
    Mobile --> Nginx
    Nginx --> HTTP
    Nginx --> WS
    HTTP --> Auth
    WS --> Signal
    Signal --> Room
    Room --> PG
    Room --> Redis
    User --> PG
    Voice --> ASR
    Voice --> TTS
    AI --> LLM
    Auth --> OAuth
    Signal --- STUN
```

### 组件说明

| 组件 | 技术 | 说明 |
|------|------|------|
| Nginx | 反向代理 | 负载均衡、SSL 终结 |
| HTTP API | Gin | RESTful API |
| WebSocket | Gorilla | 实时通信 |
| 信令服务 | WebRTC | SDP offer/answer、ICE candidate 转发 |
| ASR | Paraformer | 语音识别 |
| TTS | Kokoro | 语音合成 |
| LLM | DeepSeek | 大语言模型 |

## 核心流程

### 1. 用户认证流程

```mermaid
sequenceDiagram
    participant U as 用户
    participant F as 前端
    participant A as OAuth2
    participant B as 后端
    participant DB as 数据库

    U->>F: 访问登录页面
    F->>U: 显示 OAuth 登录选项
    U->>A: 点击 GitHub/Google 登录
    A->>U: 重定向到 OAuth 授权页面
    U->>A: 授权确认
    A->>F: 回调 with code
    F->>B: 发送 code
    B->>A: 交换 access_token
    A-->>B: 返回用户信息
    B->>DB: 创建/更新用户
    B->>F: 返回 JWT Token
    F->>U: 登录成功
```

### 2. 语音聊天流程

```mermaid
sequenceDiagram
    participant U as 用户
    participant F as 前端
    participant WS as WebSocket
    participant VAD as VAD
    participant ASR as ASR
    participant LLM as LLM
    participant TTS as TTS
    participant RTC as WebRTC

    U->>F: 按住说话按钮
    F->>RTC: 开始采集音频
    RTC->>F: 音频流
    F->>WS: 发送音频数据
    WS->>VAD: 语音活动检测
    VAD-->>WS: 语音结束
    WS->>ASR: 发送音频数据
    ASR-->>WS: 返回识别文本
    WS->>LLM: 发送用户文本
    LLM-->>WS: 返回 AI 响应
    WS->>TTS: 发送响应文本
    TTS-->>WS: 返回音频流
    WS->>F: 转发 AI 音频
    F->>RTC: 播放 AI 语音
    U->>F: 松开按钮
```

### 3. WebRTC 信令流程

```mermaid
sequenceDiagram
    participant U1 as 用户1
    participant S as 信令服务器
    participant U2 as 用户2

    U1->>S: 加入房间 (JWT)
    S->>S: 验证 Token
    S->>U1: 房间已创建/加入成功

    U1->>S: 发送 SDP Offer
    S->>U2: 转发 SDP Offer

    U2->>S: 发送 SDP Answer
    S->>U1: 转发 SDP Answer

    U1->>S: ICE Candidate
    S->>U2: 转发 ICE Candidate

    U2->>S: ICE Candidate
    S->>U1: 转发 ICE Candidate

    Note over U1,U2: P2P 直连建立
```

## 快速开始

### 1. 下载 AI 模型

模型文件较大（3GB），需要单独下载：

```bash
# 创建模型目录
mkdir -p models/paraformer models/kokoro

# 下载 Paraformer ASR 模型 (607MB + 218MB + 158MB + 69MB)
cd models/paraformer
wget https://github.com/snakers4/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-paraformer-bilingual-zh-en.tar.bz2
tar -xjf sherpa-onnx-streaming-paraformer-bilingual-zh-en.tar.bz2
mv sherpa-onnx-streaming-paraformer-bilingual-zh-en/* .
rmdir sherpa-onnx-streaming-paraformer-bilingual-zh-en

# 下载 Kokoro TTS 模型 (330MB)
cd ../kokoro
wget https://github.com/remsky/Kokoro-ONNX/releases/download/v0.19.0/kokoro-en-v0_19.tar.bz2
tar -xjf kokoro-en-v0_19.tar.bz2

# 返回项目根目录
cd ../..
```

或使用脚本自动下载：

```bash
bash deploy/install.sh
```

### 2. 启动后端

```bash
go mod download
go run cmd/server/main.go
```

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

### 3. 配置

环境变量或配置文件参考 `internal/config/config.go`。

## 功能

- 实时语音聊天
- AI 语音响应
- 文本聊天
- WebRTC 音视频通话
- 用户认证
- 房间管理

## 端口

- HTTP API: 8080
- WebSocket: 8081
- Signaling: 8082
- Frontend: 3000
