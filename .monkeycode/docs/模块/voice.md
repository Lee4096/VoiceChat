# 语音服务模块

## 概述

语音服务模块 (`internal/voice/`) 负责音频处理的核心功能，包括语音识别 (ASR)、语音合成 (TTS) 和语音活动检测 (VAD)。

## 核心组件

### 1. 接口定义 (interfaces.go)

```go
// Recognizer 语音识别接口
type Recognizer interface {
    Recognize(samples []float32) (*RecognitionResult, error)
}

// Synthesizer 语音合成接口
type Synthesizer interface {
    Synthesize(text string, voiceID int, speed float32) ([]float32, error)
}
```

### 2. Sherpa ASR (asr.go)

Sherpa-ONNX 实时语音识别服务。

**配置**:
```go
type ASRConfig struct {
    EncoderPath string  // encoder.onnx 路径
    DecoderPath string  // decoder.onnx 路径
    TokensPath  string  // tokens.txt 路径
    NThreads    int     // CPU 线程数
}
```

**特性**:
- 流式识别
- 中英双语
- 低延迟 (16kHz 音频)

### 3. Kokoro TTS (tts.go)

Kokoro-ONNX 语音合成服务。

**配置**:
```go
type KokoroTTSConfig struct {
    ModelPath  string  // 模型路径
    VoicesPath string  // 语音文件
    TokensPath string  // tokens.txt
    DataDir    string  // espeak-ng-data 目录
    Lang       string  // 语言代码 (如 "en-us")
    Speed      float32 // 语速 (默认 1.0)
}
```

**特性**:
- 流式合成
- 多语音支持
- 可调语速

### 4. WebRTC VAD (webrtc_vad.go)

基于 WebRTC 的语音活动检测。

**功能**:
- 检测语音段和静音段
- 支持多种 aggressiveness 级别 (0-3)
- 16kHz 采样率

### 5. 流式管道 (streaming_pipeline.go)

完整的 ASR -> LLM -> TTS 流式处理管道。

**处理流程**:
1. 接收音频 chunks
2. ASR 实时识别
3. LLM 流式生成文本
4. TTS 并发合成
5. 流式输出音频

**特性**:
- 并发 LLM 和 TTS
- 句子边界检测
- 可中断处理

## 数据格式

### 输入音频格式
- 采样率: 16000 Hz
- 位深: 16-bit PCM
- 通道: 单声道
- 编码: Base64 传输

### 输出音频格式
- 采样率: 24000 Hz (TTS 输出)
- 格式: float32 数组
- 编码: Base64 传输

## 核心类型

```go
// RecognitionResult ASR 识别结果
type RecognitionResult struct {
    Text    string  // 识别文本
    Start   float64 // 开始时间
    End     float64 // 结束时间
}

// VoiceProcessor 语音处理器
type VoiceProcessor struct {
    recognizer Recognizer
    synthesizer Synthesizer
    sampleRate int
}
```

## 使用示例

```go
// 创建语音服务
svc, err := voice.NewSherpaVoiceService(voice.SherpaConfig{
    ASRConfig: voice.ASRConfig{
        EncoderPath: "./models/encoder.onnx",
        DecoderPath: "./models/decoder.onnx",
        TokensPath:  "./models/tokens.txt",
        NThreads:    4,
    },
    TTSConfig: voice.KokoroTTSConfig{
        ModelPath:  "./models/model.onnx",
        VoicesPath: "./models/voices.bin",
        TokensPath: "./models/tokens.txt",
        DataDir:    "./models/espeak-ng-data",
        Lang:       "en-us",
        Speed:      1.0,
    },
})

// 创建处理器
processor := voice.NewVoiceProcessor(svc, 16000)

// 语音识别
result, err := processor.Recognize(samples)

// 语音合成
audio, err := processor.Synthesize("Hello world")
```

## 错误处理

| 错误类型 | 说明 |
|----------|------|
| ErrASRFailed | ASR 识别失败 |
| ErrTTSFailed | TTS 合成失败 |
| ErrVADTimeout | VAD 检测超时 |
| ErrAudioInvalid | 无效的音频数据 |
