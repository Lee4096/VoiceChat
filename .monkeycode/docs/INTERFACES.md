# VoiceChat 接口文档

## 概述

VoiceChat 提供两类接口：
1. **HTTP REST API** - 用户认证、房间管理
2. **WebSocket API** - 实时消息、AI 语音聊天、WebRTC 信令

## HTTP API

基础路径: `/api/v1`

### 认证接口

#### 获取 OAuth 登录 URL
```
GET /api/v1/auth/login/:provider
```

**路径参数**:
- `provider`: `github` | `google`

**响应**:
```json
{
  "url": "https://github.com/login/oauth/authorize?..."
}
```

#### OAuth 回调
```
GET /api/v1/auth/callback/:provider?code=xxx
```

**路径参数**:
- `provider`: `github` | `google`

**响应**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "user_xxx",
    "email": "user@example.com",
    "name": "User Name",
    "avatar": ""
  }
}
```

#### 注册
```
POST /api/v1/auth/register
```

**请求体**:
```json
{
  "email": "user@example.com",
  "password": "password123",
  "name": "User Name"
}
```

**响应** (201 Created):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "user_xxx",
    "email": "user@example.com",
    "name": "User Name",
    "avatar": ""
  }
}
```

#### 密码登录
```
POST /api/v1/auth/login
```

**请求体**:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**响应**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "user_xxx",
    "email": "user@example.com",
    "name": "User Name",
    "avatar": ""
  }
}
```

#### 刷新 Token
```
POST /api/v1/auth/refresh
Authorization: Bearer <token>
```

**响应**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### 房间接口

#### 创建房间
```
POST /api/v1/rooms
Authorization: Bearer <token>
```

**请求体**:
```json
{
  "name": "My Room"
}
```

**响应** (201 Created):
```json
{
  "id": "room_xxx",
  "name": "My Room",
  "owner_id": "user_xxx",
  "created_at": "2026-04-23T10:00:00Z"
}
```

#### 房间列表
```
GET /api/v1/rooms?limit=20&offset=0
Authorization: Bearer <token>
```

**查询参数**:
- `limit`: 返回数量 (默认 20, 最大 100)
- `offset`: 偏移量 (默认 0)

**响应**:
```json
{
  "rooms": [
    {
      "id": "room_xxx",
      "name": "My Room",
      "owner_id": "user_xxx",
      "created_at": "2026-04-23T10:00:00Z"
    }
  ]
}
```

#### 获取房间详情
```
GET /api/v1/rooms/:id
Authorization: Bearer <token>
```

**响应**:
```json
{
  "room": {
    "id": "room_xxx",
    "name": "My Room",
    "owner_id": "user_xxx",
    "created_at": "2026-04-23T10:00:00Z"
  },
  "members": [
    {
      "id": "member_xxx",
      "user_id": "user_xxx",
      "room_id": "room_xxx",
      "joined_at": "2026-04-23T10:00:00Z"
    }
  ]
}
```

#### 加入房间
```
POST /api/v1/rooms/:id/join
Authorization: Bearer <token>
```

**响应**:
```json
{
  "member": {
    "id": "member_xxx",
    "user_id": "user_xxx",
    "room_id": "room_xxx",
    "joined_at": "2026-04-23T10:00:00Z"
  }
}
```

#### 离开房间
```
POST /api/v1/rooms/:id/leave
Authorization: Bearer <token>
```

**响应**:
```json
{
  "message": "已离开房间"
}
```

### 健康检查
```
GET /health
```

**响应**:
```json
{
  "status": "ok"
}
```

## WebSocket API

连接地址: `ws://host:8081/ws`

### 连接与认证

客户端连接时需在消息中携带 token:
```json
{
  "type": "auth",
  "token": "user_jwt_token"
}
```

### 消息格式

**客户端消息**:
```json
{
  "type": "message_type",
  "room_id": "room_xxx",
  "user_id": "user_xxx",
  "payload": {}
}
```

**服务器消息**:
```json
{
  "type": "message_type",
  "payload": {}
}
```

### 房间消息

#### 加入房间
```json
{
  "type": "join_room",
  "room_id": "room_xxx",
  "user_id": "user_xxx",
  "token": "user_jwt_token"
}
```

**响应**:
```json
{
  "type": "room_joined",
  "payload": {"room_id": "room_xxx"}
}
```

#### 离开房间
```json
{
  "type": "leave_room"
}
```

### AI 语音聊天

#### 发送语音
```json
{
  "type": "ai_voice_chat",
  "payload": {
    "audio": "base64_encoded_pcm_audio",
    "format": "pcm",
    "sample_rate": 16000
  }
}
```

**服务器响应流程**:

1. 思考状态 (识别中):
```json
{
  "type": "thinking",
  "payload": {"status": "recognizing"}
}
```

2. 思考状态 (生成中):
```json
{
  "type": "thinking",
  "payload": {"status": "generating"}
}
```

3. 增量文本:
```json
{
  "type": "ai_text_delta",
  "payload": {"text": "Hello"}
}
```

4. 音频响应 (流式):
```json
{
  "type": "ai_voice_response",
  "payload": {
    "audio": "base64_encoded_audio",
    "is_final": false
  }
}
```

5. 完成状态:
```json
{
  "type": "thinking",
  "payload": {"status": "done"}
}
```

### AI 文本聊天

#### 发送文本
```json
{
  "type": "ai_text_chat",
  "payload": {
    "text": "Hello, how are you?"
  }
}
```

**响应**:
```json
{
  "type": "ai_text_response",
  "payload": {
    "text": "I'm doing well, thank you!"
  }
}
```

### 打断功能

#### 发送打断
```json
{
  "type": "interrupt"
}
```

**服务器向房间内其他用户广播**:
```json
{
  "type": "stop_audio",
  "payload": {"user_id": "user_xxx"}
}
```

### WebRTC 信令

#### 发送 Offer
```json
{
  "type": "offer",
  "room_id": "room_xxx",
  "user_id": "user_xxx",
  "payload": {"sdp": "v=0..."}
}
```

#### 发送 Answer
```json
{
  "type": "answer",
  "room_id": "room_xxx",
  "user_id": "user_xxx",
  "payload": {"sdp": "v=0..."}
}
```

#### 发送 ICE Candidate
```json
{
  "type": "ice_candidate",
  "room_id": "room_xxx",
  "user_id": "user_xxx",
  "payload": {"candidate": "..."}
}
```

### 语音数据转发

#### 发送语音数据
```json
{
  "type": "voice_data",
  "payload": {
    "audio": "base64_encoded_audio",
    "format": "pcm",
    "sample_rate": 16000
  }
}
```

**服务器广播给房间内其他用户**

### 用户状态

#### 用户加入
```json
{
  "type": "user_joined",
  "payload": {"user_id": "user_xxx"}
}
```

#### 用户离开
```json
{
  "type": "user_left",
  "payload": {"user_id": "user_xxx"}
}
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 1000 | 内部错误 |
| 1001 | 认证失败 |
| 1002 | Token 无效 |
| 1003 | Token 过期 |
| 1004 | OAuth 认证失败 |
| 2001 | 房间不存在 |
| 2002 | 房间已满 |
| 3001 | 音频负载过大 |
| 3002 | 文本长度超限 |
| 3003 | LLM 请求超时 |
| 3004 | TTS 合成超时 |

## 错误响应格式

```json
{
  "error": {
    "code": 1001,
    "message": "认证失败"
  }
}
```
