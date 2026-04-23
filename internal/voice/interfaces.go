package voice

import "context"

// Recognizer 接口定义语音识别服务的通用接口。
// 允许实现不同的 ASR 后端（Sherpa、DeepSpeech、Whisper 等）。
type Recognizer interface {
	// Recognize 对给定音频执行语音识别，返回识别文本。
	Recognize(audioData []float32) (*ASRResult, error)

	// RecognizeStream 处理流式音频块进行增量识别。
	RecognizeStream(audioChunk []float32) (*ASRResult, error)

	// Reset 重置识别器状态，准备新的识别会话。
	Reset()

	// Close 释放识别器持有的所有资源。
	Close()
}

// RecognizerWithVAD 接口扩展 Recognizer，添加 VAD 支持。
type RecognizerWithVAD interface {
	Recognizer

	// ProcessWithVAD 处理音频并返回 VAD 事件。
	ProcessWithVAD(ctx context.Context, audioData []byte) (*WebRTCVADResult, error)

	// IsSpeaking 返回当前是否检测到语音。
	IsSpeaking() bool
}

// Synthesizer 接口定义语音合成服务的通用接口。
// 允许实现不同的 TTS 后端（Kokoro、Coqui、MaryTTS 等）。
type Synthesizer interface {
	// Synthesize 将文本转换为语音，返回音频样本。
	Synthesize(text string) ([]float32, error)

	// SynthesizeWithSpeaker 使用指定说话人合成语音。
	SynthesizeWithSpeaker(text string, speakerID int) ([]float32, error)

	// Close 释放合成器持有的所有资源。
	Close()
}

// AudioProcessor 接口定义音频处理服务的通用接口。
// 提供语音识别和合成的统一访问。
type AudioProcessor interface {
	Recognizer
	Synthesizer

	// Process 执行完整的语音管道：识别 + 合成。
	Process(ctx context.Context, inputAudio []float32, text string) ([]float32, error)
}
