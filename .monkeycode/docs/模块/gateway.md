# 网关服务模块

## 概述

网关服务模块 (`internal/gateway/`) 是系统的入口层，处理所有外部请求，包括 HTTP API 和 WebSocket 实时通信。

## HTTP 网关 (http/)

### 路由结构 (router.go)

```
/api/v1/
├── auth/
│   ├── login/:provider          # OAuth 登录
│   ├── callback/:provider       # OAuth 回调
│   ├── register                # 用户注册
│   ├── login                   # 密码登录
│   └── refresh                 # Token 刷新
├── rooms/
│   ├── POST   /                # 创建房间
│   ├── GET    /                # 房间列表
│   ├── GET    /:id             # 房间详情
│   ├── POST   /:id/join        # 加入房间
│   └── POST   /:id/leave      # 离开房间
├── users/
│   ├── GET    /:id             # 用户详情
│   └── PUT    /:id             # 更新用户
└── health                        # 健康检查
```

### 中间件 (middleware/)

1. **认证中间件** - JWT Token 验证
2. **CORS 中间件** - 跨域资源共享
3. **日志中间件** - 请求日志记录
4. **限流中间件** - 速率限制

### 处理器 (handler/)

#### AuthHandler
- `Register` - 用户注册
- `PasswordLogin` - 密码登录
- `Login` - OAuth 登录
- `GetLoginURL` - 获取 OAuth URL
- `RefreshToken` - 刷新 Token

#### RoomHandler
- `Create` - 创建房间
- `List` - 房间列表
- `Get` - 房间详情
- `Join` - 加入房间
- `Leave` - 离开房间

#### UserHandler
- `Get` - 获取用户信息
- `Update` - 更新用户信息

#### HealthHandler
- `Check` - 健康检查

## WebSocket 网关 (websocket/)

### Server 结构

```go
type Server struct {
    cfg           Config
    logger        Logger
    signaling     *signaling.Server
    clients       map[string]*Client
    voiceService  *voice.SherpaVoiceService
    llm           *ai.LLMService
    voiceProcessor *voice.VoiceProcessor
}
```

### 消息类型

| 消息类型 | 说明 | 方向 |
|----------|------|------|
| auth | 认证 | C -> S |
| join_room | 加入房间 | C -> S |
| leave_room | 离开房间 | C -> S |
| room_joined | 加入成功 | S -> C |
| user_joined | 用户加入 | S -> C |
| user_left | 用户离开 | S -> C |
| offer | WebRTC Offer | C -> S -> C |
| answer | WebRTC Answer | C -> S -> C |
| ice_candidate | ICE 候选 | C -> S -> C |
| voice_data | 语音数据 | C -> S -> C |
| ai_voice_chat | AI 语音聊天 | C -> S |
| ai_text_chat | AI 文本聊天 | C -> S |
| ai_voice_response | AI 语音响应 | S -> C |
| ai_text_response | AI 文本响应 | S -> C |
| ai_text_delta | AI 文本增量 | S -> C |
| thinking | 思考状态 | S -> C |
| interrupt | 打断请求 | C -> S |
| stop_audio | 停止音频 | S -> C |

### AI 语音聊天流程

```go
// handleAIVoiceChat 处理流程
func (s *Server) handleAIVoiceChat(client *Client, msg *ClientMessage) {
    // 1. 解析音频数据
    // 2. ASR 识别
    // 3. 发送识别结果
    // 4. LLM 流式生成
    // 5. TTS 并发合成
    // 6. 流式发送音频
}
```

### 并发处理

AI 语音聊天采用并发管道：
- LLM 流式输出文本
- TTS 并发合成音频
- 非阻塞发送响应

### 错误处理

- 音频负载限制: 512KB
- 文本长度限制: 2000 字符
- LLM 超时: 30s
- TTS 超时: 60s

## 配置

```go
type Config struct {
    Port           int    // WebSocket 端口
    ReadTimeout    int    // 读取超时 (秒)
    WriteTimeout   int    // 写入超时 (秒)
    ASREncoderPath string // ASR 编码器路径
    ASRDecoderPath string // ASR 解码器路径
    TTSModelPath   string // TTS 模型路径
    LLMEndpoint    string // LLM API 端点
    LLMApiKey      string // LLM API 密钥
}
```
