# VoiceChat 开发者指南

## 环境要求

### 后端
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker (可选)

### 前端
- Node.js 18+
- pnpm (推荐) 或 npm

### AI 模型
- Sherpa-ONNX Paraformer 模型 (~80MB)
- Kokoro TTS 模型 (~70MB)

## 开发环境设置

### 1. 克隆代码
```bash
git clone https://github.com/Lee4096/VoiceChat.git
cd VoiceChat
```

### 2. 下载 AI 模型
```bash
# Sherpa ASR 模型
mkdir -p models/paraformer
curl -L -o models/paraformer/sherpa-onnx-streaming-paraformer-bilingual-zh-en.tar.gz \
  https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-streaming-paraformer-bilingual-zh-en.tar.gz
tar -xzf models/paraformer/sherpa-onnx-streaming-paraformer-bilingual-zh-en.tar.gz -C models/paraformer/

# Kokoro TTS 模型
mkdir -p models/kokoro
curl -L -o models/kokoro/kokoro-v0_19.onnx https://...
```

### 3. 配置环境变量
```bash
cp .env.example .env
# 编辑 .env 填入必要配置
```

### 4. 启动基础设施
```bash
# 使用 Docker 启动 PostgreSQL 和 Redis
docker-compose up -d postgres redis

# 或直接使用本地服务
```

### 5. 配置 config.yaml
编辑 `config/config.yaml`:
```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "voicechat"

redis:
  host: "localhost"
  port: 6379

jwt:
  secret: "your-secret-key-change-in-production"
  expiration: 86400

llm:
  base_url: "https://openrouter.ai/api/v1"
  api_key: "your-api-key"
  model: "minimax/minimax-m2.5:free"

voice:
  asr_encoder_path: "./models/paraformer/sherpa-onnx-streaming-paraformer-bilingual-zh-en/encoder.onnx"
  asr_decoder_path: "./models/paraformer/sherpa-onnx-streaming-paraformer-bilingual-zh-en/decoder.onnx"
  asr_tokens_path: "./models/paraformer/sherpa-onnx-streaming-paraformer-bilingual-zh-en/tokens.txt"
  tts_model_path: "./models/kokoro/kokoro-v0_19/model.onnx"
  tts_voices_path: "./models/kokoro/kokoro-v0_19/voices.bin"
  tts_tokens_path: "./models/kokoro/kokoro-v0_19/tokens.txt"
  tts_data_dir: "./models/kokoro/kokoro-v0_19/espeak-ng-data"
```

### 6. 启动后端
```bash
# 安装依赖
make deps

# 构建
make build

# 运行
make run
```

### 7. 启动前端
```bash
cd frontend

# 安装依赖
pnpm install

# 开发模式
pnpm dev
```

## 项目结构

### 后端目录结构
```
internal/
├── gateway/
│   ├── http/           # HTTP API
│   │   ├── handler/   # 请求处理器
│   │   │   ├── auth.go
│   │   │   ├── room.go
│   │   │   ├── user.go
│   │   │   └── health.go
│   │   ├── middleware/ # 中间件
│   │   └── router.go
│   └── websocket/
│       └── server.go  # WebSocket 服务器
├── voice/             # 语音服务
│   ├── asr.go        # ASR 客户端
│   ├── tts.go        # TTS 客户端
│   ├── vad.go        # VAD
│   ├── webrtc_vad.go # WebRTC VAD
│   ├── interfaces.go # 接口定义
│   └── streaming_pipeline.go
├── auth/              # 认证服务
├── ai/                # LLM 服务
├── signaling/         # WebRTC 信令
├── user/              # 用户服务
├── room/              # 房间服务
└── config/            # 配置管理
```

### 前端目录结构
```
frontend/src/
├── components/        # React 组件
│   ├── ChatRoom.tsx
│   ├── RoomList.tsx
│   └── LoginPage.tsx
├── hooks/            # 自定义 Hooks
│   ├── useAudioPlayer.ts    # 音频播放队列
│   ├── useAudioRecorder.ts  # 音频录制
│   ├── useConversationState.ts # 对话状态机
│   ├── useWakeLock.ts       # 屏幕常亮
│   └── useWebRTC.ts         # WebRTC 处理
├── store/            # 状态管理 (Zustand)
├── lib/              # 工具库
│   ├── api.ts        # HTTP API 客户端
│   └── websocket.ts  # WebSocket 客户端
└── types/            # TypeScript 类型
```

## 开发命令

### 后端
```bash
make build        # 构建
make run          # 运行
make test         # 运行测试
make lint         # 代码检查
make check        # 全部检查 (fmt, vet, lint, test)
make docker-build # 构建 Docker 镜像
```

### 前端
```bash
pnpm dev          # 开发服务器
pnpm build        # 生产构建
pnpm preview      # 预览构建
pnpm test         # 运行测试
pnpm test:ui      # UI 模式测试
```

## 代码规范

### Go 代码规范
- 使用 `gofmt` 格式化代码
- 遵循 Go 官方命名规范
- 公共函数和类型要有文档注释
- 错误处理要明确

### TypeScript 代码规范
- 使用 TypeScript 严格模式
- 组件使用函数式写法 + Hooks
- 自定义 Hooks 以 `use` 开头
- 类型定义放在 `types/index.ts`

### 提交规范
```
feat: 新功能
fix: 修复 bug
docs: 文档更新
test: 测试更新
refactor: 重构
chore: 工具/构建更新
```

## 测试

### 后端测试
```bash
# 运行所有测试
make test

# 生成覆盖率报告
make test-coverage
```

### 前端测试
```bash
cd frontend
pnpm test          # 运行测试
pnpm test:watch   # 监听模式
pnpm test:ui      # Vitest UI
```

## 调试

### 后端调试
```bash
# 启用 debug 日志
VOICECHAT_LOG_LEVEL=DEBUG make run

# 查看日志
tail -f logs/voicechat.log
```

### 前端调试
- 使用浏览器开发者工具
- React DevTools 扩展
- 网络面板查看 API 请求

## 常见问题

### 1. WebSocket 连接失败
- 检查防火墙设置
- 确认端口 8081 开放
- 查看后端日志

### 2. AI 模型加载失败
- 确认模型文件存在
- 检查模型路径配置
- 验证文件权限

### 3. 数据库连接失败
- 检查 PostgreSQL 是否运行
- 验证数据库凭据
- 确认数据库存在

### 4. 音频播放无声音
- 检查浏览器音频权限
- 确认 AudioContext 状态
- 查看控制台错误信息
