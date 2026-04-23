package voice

import (
	"context"
	"sync"
	"time"
)

type WebRTCVADConfig struct {
	SampleRate        int
	FrameSize        int
	Aggressiveness   int
	MinSpeechSamples int
	MinSilenceSamples int
	SpeechTimeout    time.Duration
}

type WebRTCVAD struct {
	cfg              WebRTCVADConfig
	isSpeaking       bool
	speechStart     time.Time
	silenceStart    time.Time
	lastEnergy      float32
	energyHistory   []float32
	historyIdx      int
	mu              sync.Mutex
	frameCount      int
	speechFrames    int
	silenceFrames   int
}

type WebRTCVADEvent int

const (
	WebRTCVADEventNone WebRTCVADEvent = iota
	WebRTCVADEventSpeechStart
	WebRTCVADEventSpeechEnd
	WebRTCVADEventBargeIn
)

type WebRTCVADResult struct {
	Event       WebRTCVADEvent
	Timestamp   time.Time
	Duration    time.Duration
	IsSpeaking  bool
	Energy      float32
}

func NewWebRTCVAD(cfg WebRTCVADConfig) *WebRTCVAD {
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 16000
	}
	if cfg.FrameSize == 0 {
		cfg.FrameSize = 480
	}
	if cfg.Aggressiveness == 0 {
		cfg.Aggressiveness = 2
	}
	if cfg.MinSpeechSamples == 0 {
		cfg.MinSpeechSamples = cfg.SampleRate / 50
	}
	if cfg.MinSilenceSamples == 0 {
		cfg.MinSilenceSamples = cfg.SampleRate / 20
	}
	if cfg.SpeechTimeout == 0 {
		cfg.SpeechTimeout = 10 * time.Second
	}

	return &WebRTCVAD{
		cfg:            cfg,
		energyHistory:  make([]float32, 32),
		speechFrames:   0,
		silenceFrames:  0,
	}
}

func (v *WebRTCVAD) ProcessAudio(ctx context.Context, audioData []byte) (*WebRTCVADResult, error) {
	samples := bytesToFloat32(audioData)
	energy := v.calcEnergy(samples)

	v.mu.Lock()
	defer v.mu.Unlock()

	v.updateEnergyHistory(energy)

	isSpeech := v.detectSpeech(energy)

	result := &WebRTCVADResult{
		Timestamp:  time.Now(),
		Energy:     energy,
		IsSpeaking: v.isSpeaking,
	}

	if isSpeech && !v.isSpeaking {
		v.speechFrames++
		v.silenceFrames = 0

		if v.speechFrames >= v.getMinSpeechFrames() {
			v.isSpeaking = true
			v.speechStart = time.Now()
			result.Event = WebRTCVADEventSpeechStart
		}
	} else if !isSpeech && v.isSpeaking {
		v.silenceFrames++

		if v.silenceFrames >= v.minSilenceFrames() {
			v.isSpeaking = false
			speechDuration := time.Since(v.speechStart)
			result.Event = WebRTCVADEventSpeechEnd
			result.Duration = speechDuration
			v.speechFrames = 0
		}
	} else if isSpeech {
		v.speechFrames++
		v.silenceFrames = 0
	} else {
		v.silenceFrames++
		v.speechFrames = 0
	}

	v.lastEnergy = energy
	v.frameCount++

	return result, nil
}

func (v *WebRTCVAD) detectSpeech(energy float32) bool {
	threshold := v.calcThreshold()

	avgEnergy := v.getAverageEnergy()

	if v.cfg.Aggressiveness == 0 {
		return energy > threshold && energy > avgEnergy*0.3
	} else if v.cfg.Aggressiveness == 1 {
		return energy > threshold && energy > avgEnergy*0.5
	} else if v.cfg.Aggressiveness == 2 {
		return energy > threshold && energy > avgEnergy*0.7
	}
	return energy > threshold*1.5
}

func (v *WebRTCVAD) calcThreshold() float32 {
	baseThreshold := float32(0.05)

	switch v.cfg.Aggressiveness {
	case 0:
		return baseThreshold * 0.5
	case 1:
		return baseThreshold
	case 2:
		return baseThreshold * 1.5
	case 3:
		return baseThreshold * 2.0
	}
	return baseThreshold
}

func (v *WebRTCVAD) calcEnergy(samples []float32) float32 {
	var sum float32
	for _, s := range samples {
		sum += s * s
	}
	return sum / float32(len(samples))
}

func (v *WebRTCVAD) updateEnergyHistory(energy float32) {
	v.energyHistory[v.historyIdx] = energy
	v.historyIdx = (v.historyIdx + 1) % len(v.energyHistory)
}

func (v *WebRTCVAD) getAverageEnergy() float32 {
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

func (v *WebRTCVAD) getMinSpeechFrames() int {
	return v.cfg.MinSpeechSamples / v.cfg.FrameSize
}

func (v *WebRTCVAD) minSilenceFrames() int {
	return v.cfg.MinSilenceSamples / v.cfg.FrameSize
}

func (v *WebRTCVAD) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.isSpeaking = false
	v.speechStart = time.Time{}
	v.silenceStart = time.Time{}
	v.lastEnergy = 0
	v.frameCount = 0
	v.speechFrames = 0
	v.silenceFrames = 0

	for i := range v.energyHistory {
		v.energyHistory[i] = 0
	}
	v.historyIdx = 0
}

func (v *WebRTCVAD) IsSpeaking() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.isSpeaking
}

func (v *WebRTCVAD) GetSpeechDuration() time.Duration {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.speechStart.IsZero() {
		return 0
	}
	return time.Since(v.speechStart)
}

type VADProcessor struct {
	vad       *WebRTCVAD
	listener  VADEventListener
	stopChan  chan struct{}
}

type VADEventListener interface {
	OnSpeechStart()
	OnSpeechEnd(duration time.Duration)
	OnBargeIn()
}

func NewVADProcessor(cfg WebRTCVADConfig, listener VADEventListener) *VADProcessor {
	return &VADProcessor{
		vad:       NewWebRTCVAD(cfg),
		listener:  listener,
		stopChan:  make(chan struct{}),
	}
}

func (p *VADProcessor) Process(ctx context.Context, audioData []byte) error {
	result, err := p.vad.ProcessAudio(ctx, audioData)
	if err != nil {
		return err
	}

	switch result.Event {
	case WebRTCVADEventSpeechStart:
		if p.listener != nil {
			p.listener.OnSpeechStart()
		}
	case WebRTCVADEventSpeechEnd:
		if p.listener != nil {
			p.listener.OnSpeechEnd(result.Duration)
		}
	case WebRTCVADEventBargeIn:
		if p.listener != nil {
			p.listener.OnBargeIn()
		}
	}

	return nil
}

func (p *VADProcessor) Reset() {
	p.vad.Reset()
}

func (p *VADProcessor) Stop() {
	close(p.stopChan)
}

func (p *VADProcessor) IsSpeaking() bool {
	return p.vad.IsSpeaking()
}
