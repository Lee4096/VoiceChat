package voice

import (
	"fmt"

	sherpa "github.com/k2-fsa/sherpa-onnx-go-linux"
)

type GeneratedAudio = sherpa.GeneratedAudio

type TTSProvider interface {
	Synthesize(text string, sid int, speed float32) ([]float32, error)
	SampleRate() int
	NumSpeakers() int
	Close()
}

type TTSConfig struct {
	ModelPath   string
	LexiconPath string
	TokensPath  string
	DataDir     string
	Voice       string
	NumThreads  int
	Provider    string
	Speed       float32
}

type TTSClient struct {
	cfg        *sherpa.OfflineTts
	sampleRate int
	speakerNum int
}

type TTSResult struct {
	Audio      []float32
	SampleRate int
	Duration   float64
}

func NewTTSClient(cfg TTSConfig) (*TTSClient, error) {
	if cfg.ModelPath == "" {
		cfg.ModelPath = "./models/vits.onnx"
	}
	if cfg.TokensPath == "" {
		cfg.TokensPath = "./models/tokens.txt"
	}
	if cfg.NumThreads == 0 {
		cfg.NumThreads = 4
	}
	if cfg.Provider == "" {
		cfg.Provider = "cpu"
	}
	if cfg.Speed == 0 {
		cfg.Speed = 1.0
	}

	config := sherpa.OfflineTtsConfig{
		Model: sherpa.OfflineTtsModelConfig{
			Vits: sherpa.OfflineTtsVitsModelConfig{
				Model:   cfg.ModelPath,
				Lexicon: cfg.LexiconPath,
				Tokens:  cfg.TokensPath,
				DataDir: cfg.DataDir,
			},
			NumThreads: cfg.NumThreads,
			Provider:   cfg.Provider,
		},
	}

	tts := sherpa.NewOfflineTts(&config)
	if tts == nil {
		return nil, fmt.Errorf("failed to create offline TTS")
	}

	return &TTSClient{
		cfg:        tts,
		sampleRate: tts.SampleRate(),
		speakerNum: tts.NumSpeakers(),
	}, nil
}

func (c *TTSClient) Synthesize(text string, sid int, speed float32) ([]float32, error) {
	if speed == 0 {
		speed = 1.0
	}

	audio := c.cfg.Generate(text, sid, speed)
	if audio == nil {
		return nil, fmt.Errorf("failed to generate audio")
	}

	return audio.Samples, nil
}

func (c *TTSClient) SampleRate() int {
	return c.sampleRate
}

func (c *TTSClient) NumSpeakers() int {
	return c.speakerNum
}

func (c *TTSClient) Close() {
	if c.cfg != nil {
		sherpa.DeleteOfflineTts(c.cfg)
		c.cfg = nil
	}
}

type KokoroTTSConfig struct {
	ModelPath   string
	VoicesPath  string
	TokensPath  string
	DataDir     string
	NumThreads  int
	Provider    string
	Speed       float32
	Lexicon     string
	Lang        string
}

type KokoroTTSClient struct {
	cfg        *sherpa.OfflineTts
	sampleRate int
	speakerNum int
}

func NewKokoroTTSClient(cfg KokoroTTSConfig) (*KokoroTTSClient, error) {
	if cfg.ModelPath == "" {
		cfg.ModelPath = "./models/kokoro/kokoro-v1.0.onnx"
	}
	if cfg.VoicesPath == "" {
		cfg.VoicesPath = "./models/kokoro/voices-v1.0.bin"
	}
	if cfg.NumThreads == 0 {
		cfg.NumThreads = 4
	}
	if cfg.Provider == "" {
		cfg.Provider = "cpu"
	}
	if cfg.Speed == 0 {
		cfg.Speed = 1.0
	}

	config := sherpa.OfflineTtsConfig{
		Model: sherpa.OfflineTtsModelConfig{
			Kokoro: sherpa.OfflineTtsKokoroModelConfig{
				Model:   cfg.ModelPath,
				Voices:  cfg.VoicesPath,
				Tokens:  cfg.TokensPath,
				DataDir: cfg.DataDir,
				Lexicon: cfg.Lexicon,
				Lang:    cfg.Lang,
			},
			NumThreads: cfg.NumThreads,
			Provider:   cfg.Provider,
		},
	}

	tts := sherpa.NewOfflineTts(&config)
	if tts == nil {
		return nil, fmt.Errorf("failed to create Kokoro TTS")
	}

	return &KokoroTTSClient{
		cfg:        tts,
		sampleRate: tts.SampleRate(),
		speakerNum: tts.NumSpeakers(),
	}, nil
}

func (c *KokoroTTSClient) Synthesize(text string, sid int, speed float32) ([]float32, error) {
	if speed == 0 {
		speed = 1.0
	}

	cfg := &sherpa.GenerationConfig{
		SilenceScale: 0.2,
		Speed:        speed,
		Sid:          sid,
	}

	var allSamples []float32
	generated := c.cfg.GenerateWithConfig(text, cfg, func(samples []float32, progress float32) bool {
		allSamples = append(allSamples, samples...)
		return true
	})

	if generated == nil {
		return nil, fmt.Errorf("failed to generate audio")
	}

	if len(allSamples) > 0 {
		generated.Samples = allSamples
	}

	return generated.Samples, nil
}

func (c *KokoroTTSClient) SampleRate() int {
	return c.sampleRate
}

func (c *KokoroTTSClient) NumSpeakers() int {
	return c.speakerNum
}

func (c *KokoroTTSClient) Close() {
	if c.cfg != nil {
		sherpa.DeleteOfflineTts(c.cfg)
		c.cfg = nil
	}
}
