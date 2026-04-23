package voice

import (
	"fmt"
	"sync"

	sherpa "github.com/k2-fsa/sherpa-onnx-go-linux"
)

// SherpaConfig 配置 Sherpa ASR/TTS 语音服务的参数。
// ASRConfig: ASR 模型路径和参数
// TTSConfig: Kokoro TTS 模型路径和参数
// EnableVAD: 是否启用语音活动检测
// SampleRate: 音频采样率（ASR 通常为 16000）
type SherpaConfig struct {
	ASRConfig   ASRConfig
	TTSConfig   KokoroTTSConfig
	EnableVAD   bool
	SampleRate  int
}

// SherpaVoiceService 封装 Sherpa ASR 和 Kokoro TTS 客户端。
// 提供线程安全的语音识别和合成访问。
type SherpaVoiceService struct {
	cfg        SherpaConfig
	asrClient  *ASRClient        // Sherpa ASR 客户端，用于语音转文字
	ttsClient  *KokoroTTSClient  // Kokoro TTS 客户端，用于文字转语音
	mu         sync.RWMutex
}

// NewSherpaVoiceService 创建支持 ASR 和/或 TTS 的 Sherpa 语音服务。
// ASR 和 TTS 都是可选的 - 只要配置了至少一个就能成功初始化。
func NewSherpaVoiceService(cfg SherpaConfig) (*SherpaVoiceService, error) {
	svc := &SherpaVoiceService{cfg: cfg}

	// 如果提供了编码器/解码器路径，则初始化 ASR 客户端
	if cfg.ASRConfig.EncoderPath != "" && cfg.ASRConfig.DecoderPath != "" {
		asr, err := NewASRClient(cfg.ASRConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create ASR client: %w", err)
		}
		svc.asrClient = asr
	}

	// 如果提供了模型/语音路径，则初始化 TTS 客户端
	if cfg.TTSConfig.ModelPath != "" && cfg.TTSConfig.VoicesPath != "" {
		tts, err := NewKokoroTTSClient(cfg.TTSConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create TTS client: %w", err)
		}
		svc.ttsClient = tts
	}

	return svc, nil
}

// Recognize 对给定的音频样本执行语音转文字。
// 音频应为归一化的 float32 值，采样率通常为 16kHz。
func (s *SherpaVoiceService) Recognize(audioData []float32) (*ASRResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.asrClient == nil {
		return nil, fmt.Errorf("ASR client not initialized")
	}
	return s.asrClient.Recognize(audioData)
}

// RecognizeStream 处理流式音频块进行语音识别。
// 用于实时语音输入场景，音频以小块形式传入。
func (s *SherpaVoiceService) RecognizeStream(audioChunk []float32) (*ASRResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.asrClient == nil {
		return nil, fmt.Errorf("ASR client not initialized")
	}
	return s.asrClient.RecognizeStream(audioChunk)
}

// Synthesize 将文本转换为语音，返回 float32 格式的音频样本。
// 使用默认说话人 (sid=0) 和正常速度 (1.0)。
func (s *SherpaVoiceService) Synthesize(text string) ([]float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ttsClient == nil {
		return nil, fmt.Errorf("TTS client not initialized")
	}
	return s.ttsClient.Synthesize(text, 0, 1.0)
}

// SynthesizeRaw 返回包含音频样本和采样率的原始 GeneratedAudio 结构。
// 当需要采样率元数据时很有用。
func (s *SherpaVoiceService) SynthesizeRaw(text string) (*sherpa.GeneratedAudio, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ttsClient == nil {
		return nil, fmt.Errorf("TTS client not initialized")
	}
	samples, err := s.ttsClient.Synthesize(text, 0, 1.0)
	if err != nil {
		return nil, err
	}
	return &sherpa.GeneratedAudio{
		Samples:    samples,
		SampleRate: s.ttsClient.SampleRate(),
	}, nil
}

// ResetASR 重置 ASR 状态，清除任何累积的音频上下文。
// 在开始新的识别会话时调用。
func (s *SherpaVoiceService) ResetASR() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.asrClient != nil {
		s.asrClient.Reset()
	}
}

// Close 释放语音服务持有的所有资源。
// 关闭服务时应调用此方法。
func (s *SherpaVoiceService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.asrClient != nil {
		s.asrClient.Close()
	}
	if s.ttsClient != nil {
		s.ttsClient.Close()
	}
}

// VoiceProcessor 提供音频缓冲和语音处理能力。
// 封装 SherpaVoiceService 并管理内部音频缓冲区，用于流式场景。
type VoiceProcessor struct {
	svc        *SherpaVoiceService
	audioBuf   []float32  // 缓冲的音频样本，等待处理
	sampleRate int        // 音频采样率（例如 16000 Hz）
	mu         sync.Mutex // 保护并发访问时的 audioBuf
}

// NewVoiceProcessor 创建具有给定语音服务和采样率的新 VoiceProcessor。
func NewVoiceProcessor(svc *SherpaVoiceService, sampleRate int) *VoiceProcessor {
	return &VoiceProcessor{
		svc:        svc,
		audioBuf:   make([]float32, 0),
		sampleRate: sampleRate,
	}
}

// AddAudio 将音频样本追加到内部缓冲区。
// 线程安全，支持并发调用。
func (p *VoiceProcessor) AddAudio(samples []float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.audioBuf = append(p.audioBuf, samples...)
}

// Process 从所有缓冲音频中识别语音，然后清除缓冲区。
// 如果缓冲区为空则返回 nil。线程安全。
func (p *VoiceProcessor) Process() (*ASRResult, error) {
	p.mu.Lock()
	if len(p.audioBuf) == 0 {
		p.mu.Unlock()
		return nil, nil
	}
	audio := make([]float32, len(p.audioBuf))
	copy(audio, p.audioBuf)
	p.audioBuf = p.audioBuf[:0]
	p.mu.Unlock()

	return p.svc.Recognize(audio)
}

// Recognize 对给定样本执行即时语音识别。
// 不使用内部缓冲区。
func (p *VoiceProcessor) Recognize(samples []float32) (*ASRResult, error) {
	return p.svc.Recognize(samples)
}

// Synthesize 使用底层语音服务将文本转换为语音。
func (p *VoiceProcessor) Synthesize(text string) ([]float32, error) {
	return p.svc.Synthesize(text)
}

// Reset 清除音频缓冲区并重置 ASR 状态。
// 在开始新的语音交互会话时调用。
func (p *VoiceProcessor) Reset() {
	p.mu.Lock()
	p.audioBuf = p.audioBuf[:0]
	p.mu.Unlock()
	p.svc.ResetASR()
}
