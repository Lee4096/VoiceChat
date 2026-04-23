package voice

import (
	"fmt"
	"log"
	"os"
	"sync"

	sherpa "github.com/k2-fsa/sherpa-onnx-go-linux"
)

// ASRConfig 保存 Sherpa ASR（自动语音识别）客户端的配置。
// 支持流式 paraformer 模型用于实时语音识别。
type ASRConfig struct {
	ModelPath   string // 离线模型路径（仅用于离线客户端）
	EncoderPath string // Paraformer 编码器模型路径
	DecoderPath string // Paraformer 解码器模型路径
	TokensPath  string // Token 文件路径
	NThreads    int    // 推理的 CPU 线程数
	Provider    string // 计算提供者（"cpu" 或 "cuda"）
}

// ASRClient 使用 Sherpa paraformer 模型提供流式语音识别。
// 线程安全，支持并发识别请求。
type ASRClient struct {
	cfg        ASRConfig
	recognizer *sherpa.OnlineRecognizer // Sherpa 在线识别器实例
	stream     *sherpa.OnlineStream     // 可重用的识别流
	mu         sync.Mutex
}

// ASRResult 包含语音识别请求的结果。
type ASRResult struct {
	Text    string // 识别的文本
	IsFinal bool   // 是否为最终结果（与临时结果相对）
}

// NewASRClient 使用给定配置创建新的 ASR 客户端。
// 使用 Sherpa paraformer 流式模型进行实时识别。
func NewASRClient(cfg ASRConfig) (*ASRClient, error) {
	if cfg.EncoderPath == "" || cfg.DecoderPath == "" {
		return nil, fmt.Errorf("EncoderPath and DecoderPath are required for streaming paraformer")
	}
	if cfg.TokensPath == "" {
		return nil, fmt.Errorf("TokensPath is required")
	}
	if cfg.NThreads == 0 {
		cfg.NThreads = 4
	}
	if cfg.Provider == "" {
		cfg.Provider = "cpu"
	}

	if _, err := os.Stat(cfg.EncoderPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("EncoderPath does not exist: %s", cfg.EncoderPath)
	}
	if _, err := os.Stat(cfg.DecoderPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("DecoderPath does not exist: %s", cfg.DecoderPath)
	}
	if _, err := os.Stat(cfg.TokensPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("TokensPath does not exist: %s", cfg.TokensPath)
	}

	log.Printf("[ASR] Initializing with encoder=%s, decoder=%s, tokens=%s",
		cfg.EncoderPath, cfg.DecoderPath, cfg.TokensPath)

	config := &sherpa.OnlineRecognizerConfig{
		FeatConfig: sherpa.FeatureConfig{
			SampleRate: 16000,
			FeatureDim: 80,
		},
		ModelConfig: sherpa.OnlineModelConfig{
			Paraformer: sherpa.OnlineParaformerModelConfig{
				Encoder: cfg.EncoderPath,
				Decoder: cfg.DecoderPath,
			},
			Tokens:     cfg.TokensPath,
			NumThreads: cfg.NThreads,
			Provider:   cfg.Provider,
			ModelType:  "paraformer",
		},
		DecodingMethod: "greedy_search",
	}

	recognizer := sherpa.NewOnlineRecognizer(config)
	if recognizer == nil {
		return nil, fmt.Errorf("failed to create online recognizer")
	}

	stream := sherpa.NewOnlineStream(recognizer)
	if stream == nil {
		sherpa.DeleteOnlineRecognizer(recognizer)
		return nil, fmt.Errorf("failed to create online stream")
	}

	return &ASRClient{
		cfg:        cfg,
		recognizer: recognizer,
		stream:     stream,
	}, nil
}

// Recognize 对提供的音频样本执行批量语音识别。
// 音频应为 16kHz 采样率的归一化 float32。
// 分块处理音频并返回最终识别结果。
// 识别完成后自动重置流以支持下一次识别。
func (c *ASRClient) Recognize(audioData []float32) (*ASRResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sampleRate := 16000
	for i := 0; i < len(audioData); i += sampleRate {
		end := i + sampleRate
		if end > len(audioData) {
			end = len(audioData)
		}
		c.stream.AcceptWaveform(sampleRate, audioData[i:end])
	}

	c.stream.InputFinished()

	for c.recognizer.IsReady(c.stream) {
		c.recognizer.Decode(c.stream)
	}

	result := c.recognizer.GetResult(c.stream)
	if result == nil {
		c.recognizer.Reset(c.stream)
		return &ASRResult{Text: "", IsFinal: true}, nil
	}

	text := result.Text
	c.recognizer.Reset(c.stream)

	return &ASRResult{
		Text:    text,
		IsFinal: true,
	}, nil
}

// RecognizeStream 处理流式音频块进行增量识别。
// 如果音频不足无法进行识别则返回 nil。
func (c *ASRClient) RecognizeStream(audioChunk []float32) (*ASRResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stream.AcceptWaveform(16000, audioChunk)

	if !c.recognizer.IsReady(c.stream) {
		return nil, nil
	}

	c.recognizer.Decode(c.stream)

	result := c.recognizer.GetResult(c.stream)
	if result == nil {
		return nil, nil
	}

	return &ASRResult{
		Text:    result.Text,
		IsFinal: true,
	}, nil
}

// Reset 清除 ASR 流状态，准备新的识别会话。
func (c *ASRClient) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.recognizer.Reset(c.stream)
}

// Close 释放 ASR 客户端持有的所有资源。
func (c *ASRClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stream != nil {
		sherpa.DeleteOnlineStream(c.stream)
		c.stream = nil
	}
	if c.recognizer != nil {
		sherpa.DeleteOnlineRecognizer(c.recognizer)
		c.recognizer = nil
	}
}

// OfflineASRClient 提供批量/离线语音识别。
// 使用单个模型文件（非流式 paraformer）。
type OfflineASRClient struct {
	cfg        ASRConfig
	recognizer *sherpa.OfflineRecognizer
}

// OfflineASRResult 包含离线语音识别的结果。
type OfflineASRResult struct {
	Text     string  // 识别的文本
	Duration float64 // 音频时长（秒）
}

// NewOfflineASRClient 使用单个模型文件创建新的离线 ASR 客户端。
func NewOfflineASRClient(cfg ASRConfig) (*OfflineASRClient, error) {
	if cfg.ModelPath == "" && cfg.EncoderPath == "" {
		return nil, fmt.Errorf("no model path provided")
	}
	if cfg.TokensPath == "" {
		return nil, fmt.Errorf("no tokens path provided")
	}
	if cfg.NThreads == 0 {
		cfg.NThreads = 4
	}
	if cfg.Provider == "" {
		cfg.Provider = "cpu"
	}

	config := &sherpa.OfflineRecognizerConfig{
		FeatConfig: sherpa.FeatureConfig{
			SampleRate: 16000,
			FeatureDim: 80,
		},
		ModelConfig: sherpa.OfflineModelConfig{
			Tokens:     cfg.TokensPath,
			NumThreads: cfg.NThreads,
			Provider:   cfg.Provider,
		},
		DecodingMethod: "greedy_search",
	}

	// 离线 paraformer 使用单个模型文件
	if cfg.ModelPath != "" {
		config.ModelConfig.Paraformer.Model = cfg.ModelPath
	} else {
		return nil, fmt.Errorf("offline paraformer requires ModelPath (single model file)")
	}

	recognizer := sherpa.NewOfflineRecognizer(config)
	if recognizer == nil {
		return nil, fmt.Errorf("failed to create offline recognizer")
	}

	return &OfflineASRClient{
		cfg:        cfg,
		recognizer: recognizer,
	}, nil
}

// Recognize 对提供的音频执行离线批量识别。
// 返回识别的文本和音频时长。
func (c *OfflineASRClient) Recognize(audioData []float32) (*OfflineASRResult, error) {
	sampleRate := 16000

	stream := sherpa.NewOfflineStream(c.recognizer)
	if stream == nil {
		return nil, fmt.Errorf("failed to create offline stream")
	}
	defer sherpa.DeleteOfflineStream(stream)

	stream.AcceptWaveform(sampleRate, audioData)
	c.recognizer.Decode(stream)

	result := stream.GetResult()
	if result == nil {
		return &OfflineASRResult{Text: "", Duration: float64(len(audioData)) / float64(sampleRate)}, nil
	}

	return &OfflineASRResult{
		Text:     result.Text,
		Duration: float64(len(audioData)) / float64(sampleRate),
	}, nil
}

// Close 释放离线 ASR 客户端持有的资源。
func (c *OfflineASRClient) Close() {
	if c.recognizer != nil {
		sherpa.DeleteOfflineRecognizer(c.recognizer)
		c.recognizer = nil
	}
}

// Int16ToFloat32 将 16 位 PCM 音频转换为归一化 float32。
// 输入范围 [-32768, 32767] 映射到 [-1.0, 1.0]。
func Int16ToFloat32(audioData []int16) []float32 {
	result := make([]float32, len(audioData))
	for i, v := range audioData {
		result[i] = float32(v) / 32768.0
	}
	return result
}

// Float32ToInt16 将归一化 float32 音频转换为 16 位 PCM。
// 输入范围 [-1.0, 1.0] 被限制并映射到 [-32768, 32767]。
func Float32ToInt16(audioData []float32) []int16 {
	result := make([]int16, len(audioData))
	for i, v := range audioData {
		if v > 1.0 {
			v = 1.0
		} else if v < -1.0 {
			v = -1.0
		}
		result[i] = int16(v * 32767)
	}
	return result
}
