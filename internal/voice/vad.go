package voice

import (
	"context"
	"encoding/binary"
	"math"
	"time"
)

type VADConfig struct {
	SampleRate       int
	FrameSize        int
	EnergyThreshold  float32
	SilenceDuration  time.Duration
	SpeechDuration   time.Duration
	MinSpeechSamples int
	MinSilenceSamples int
}

type VAD struct {
	cfg         VADConfig
	isSpeaking  bool
	speechStart time.Time
	silenceStart time.Time
	energyHistory []float32
	historyIdx   int
}

type VADEvent int

const (
	VADEventSpeechStart VADEvent = iota
	VADEventSpeechEnd
	VADEventBargeIn
)

type VADResult struct {
	Event     VADEvent
	Timestamp time.Time
	Duration  time.Duration
}

func NewVAD(cfg VADConfig) *VAD {
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 16000
	}
	if cfg.FrameSize == 0 {
		cfg.FrameSize = 480
	}
	if cfg.EnergyThreshold == 0 {
		cfg.EnergyThreshold = 0.05
	}
	if cfg.SilenceDuration == 0 {
		cfg.SilenceDuration = 500 * time.Millisecond
	}
	if cfg.SpeechDuration == 0 {
		cfg.SpeechDuration = 100 * time.Millisecond
	}

	return &VAD{
		cfg:          cfg,
		energyHistory: make([]float32, 20),
	}
}

func (v *VAD) ProcessAudio(ctx context.Context, audioData []byte) (*VADResult, error) {
	samples := bytesToFloat32(audioData)
	energy := v.calculateEnergy(samples)

	v.energyHistory[v.historyIdx] = energy
	v.historyIdx = (v.historyIdx + 1) % len(v.energyHistory)

	avgEnergy := v.getAverageEnergy()

	if energy > v.cfg.EnergyThreshold && energy > avgEnergy*0.5 {
		if !v.isSpeaking {
			v.speechStart = time.Now()
			v.isSpeaking = true
			return &VADResult{
				Event:     VADEventSpeechStart,
				Timestamp: v.speechStart,
			}, nil
		}
		v.silenceStart = time.Time{}
	} else if v.isSpeaking {
		if v.silenceStart.IsZero() {
			v.silenceStart = time.Now()
		}

		silenceDuration := time.Since(v.silenceStart)
		if silenceDuration >= v.cfg.SilenceDuration {
			v.isSpeaking = false
			speechDuration := v.silenceStart.Sub(v.speechStart)
			return &VADResult{
				Event:     VADEventSpeechEnd,
				Timestamp: v.silenceStart,
				Duration:  speechDuration,
			}, nil
		}
	}

	return nil, nil
}

func (v *VAD) calculateEnergy(samples []float32) float32 {
	var sum float32
	for _, s := range samples {
		sum += s * s
	}
	return sum / float32(len(samples))
}

func (v *VAD) getAverageEnergy() float32 {
	var sum float32
	count := 0
	for _, e := range v.energyHistory {
		if e > 0 {
			sum += e
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float32(count)
}

func (v *VAD) Reset() {
	v.isSpeaking = false
	v.speechStart = time.Time{}
	v.silenceStart = time.Time{}
	v.historyIdx = 0
	for i := range v.energyHistory {
		v.energyHistory[i] = 0
	}
}

func (v *VAD) IsSpeaking() bool {
	return v.isSpeaking
}

func bytesToFloat32(data []byte) []float32 {
	samples := make([]float32, len(data)/2)
	for i := 0; i < len(samples); i++ {
		int16Val := int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		samples[i] = float32(int16Val) / 32768.0
	}
	return samples
}

func Float32ToBytes(samples []float32) []byte {
	data := make([]byte, len(samples)*2)
	for i, s := range samples {
		int16Val := int16(math.Max(-1, math.Min(1, float64(s))) * 32767)
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(int16Val))
	}
	return data
}
