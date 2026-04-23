package voice

import (
	"context"
	"strings"
	"sync"

	"voicechat/internal/ai"
)

type StreamingPipelineConfig struct {
	EnableVAD       bool
	SentenceBoundary []string
	BufferSize     int
}

type StreamingPipeline struct {
	cfg    StreamingPipelineConfig
	asr    *ASRClient
	tts    *KokoroTTSClient
	llm    LLMStreamer

	asrTextChan chan string
	llmTextChan chan string
	audioChan   chan []float32
	doneChan    chan struct{}

	mu           sync.Mutex
	textBuffer   strings.Builder
	stopped      bool
}

type LLMStreamer interface {
	ChatStreamText(ctx context.Context, messages []ai.ChatMessage) (<-chan string, error)
}

func NewStreamingPipeline(
	asr *ASRClient,
	tts *KokoroTTSClient,
	llm LLMStreamer,
	cfg StreamingPipelineConfig,
) *StreamingPipeline {
	if cfg.SentenceBoundary == nil {
		cfg.SentenceBoundary = []string{"。", "！", "？", ".", "!", "?"}
	}
	if cfg.BufferSize == 0 {
		cfg.BufferSize = 512
	}

	return &StreamingPipeline{
		cfg:          cfg,
		asr:          asr,
		tts:          tts,
		llm:          llm,
		asrTextChan:  make(chan string, cfg.BufferSize),
		llmTextChan:  make(chan string, cfg.BufferSize),
		audioChan:    make(chan []float32, cfg.BufferSize),
		doneChan:     make(chan struct{}),
	}
}

func (p *StreamingPipeline) ProcessAudioStream(
	ctx context.Context,
	audioChunks <-chan []float32,
) (<-chan []float32, <-chan error) {
	errChan := make(chan error, 1)
	audioOut := make(chan []float32, p.cfg.BufferSize)

	go func() {
		defer close(audioOut)
		defer close(errChan)

		asrCtx, cancelASR := context.WithCancel(ctx)
		defer cancelASR()

		go p.runASR(asrCtx)

		llmCtx, cancelLLM := context.WithCancel(ctx)
		defer cancelLLM()

		go p.runLLM(llmCtx)

		ttsCtx, cancelTTS := context.WithCancel(ctx)
		defer cancelTTS()

		go p.runTTS(ttsCtx, audioOut)

		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-audioChunks:
				if !ok {
					p.flushTextBuffer()
					p.doneChan <- struct{}{}
					return
				}
				p.asrTextChan <- string(make([]byte, 0))
				p.processASRChunk(chunk)
			case text := <-p.asrTextChan:
				if text != "" {
					p.bufferAndSendText(text)
				}
			}
		}
	}()

	return audioOut, errChan
}

func (p *StreamingPipeline) processASRChunk(chunk []float32) {
	if p.asr == nil {
		return
	}

	result, err := p.asr.RecognizeStream(chunk)
	if err != nil || result == nil {
		return
	}

	if result.Text != "" {
		select {
		case p.asrTextChan <- result.Text:
		case <-p.doneChan:
		}
	}
}

func (p *StreamingPipeline) bufferAndSendText(text string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}

	p.textBuffer.WriteString(text)

	currentText := p.textBuffer.String()
	for _, boundary := range p.cfg.SentenceBoundary {
		if strings.Contains(currentText, boundary) {
			parts := strings.Split(currentText, boundary)
			for i := 0; i < len(parts)-1; i++ {
				sentence := parts[i] + boundary
				select {
				case p.llmTextChan <- sentence:
				case <-p.doneChan:
					return
				}
			}
			p.textBuffer.Reset()
			p.textBuffer.WriteString(parts[len(parts)-1])
			break
		}
	}
}

func (p *StreamingPipeline) flushTextBuffer() {
	p.mu.Lock()
	defer p.mu.Unlock()

	remaining := p.textBuffer.String()
	if remaining != "" {
		select {
		case p.llmTextChan <- remaining:
		case <-p.doneChan:
		}
	}

	close(p.llmTextChan)
}

func (p *StreamingPipeline) runASR(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case text, ok := <-p.asrTextChan:
			if !ok {
				return
			}
			if text != "" {
				select {
				case p.llmTextChan <- text:
				case <-p.doneChan:
					return
				}
			}
		}
	}
}

func (p *StreamingPipeline) runLLM(ctx context.Context) {
	if p.llm == nil {
		close(p.llmTextChan)
		return
	}

	chatCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	streamChan, err := p.llm.ChatStreamText(chatCtx, []ai.ChatMessage{{Role: "user", Content: ""}})
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case text, ok := <-streamChan:
			if !ok {
				return
			}
			p.bufferAndSendText(text)
		}
	}
}

func (p *StreamingPipeline) runTTS(ctx context.Context, audioOut chan<- []float32) {
	if p.tts == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case text, ok := <-p.llmTextChan:
			if !ok {
				return
			}

			if text == "" {
				continue
			}

			p.mu.Lock()
			if p.stopped {
				p.mu.Unlock()
				return
			}
			p.mu.Unlock()

			audio, err := p.tts.Synthesize(text, 0, 1.0)
			if err != nil {
				continue
			}

			select {
			case audioOut <- audio:
			case <-ctx.Done():
				return
			case <-p.doneChan:
				return
			}
		}
	}
}

func (p *StreamingPipeline) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopped = true
	close(p.doneChan)
}

type SimpleLLMStreamer struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient interface {
		Do(req interface{}) (interface{}, error)
	}
}

func NewSimpleLLMStreamer(baseURL, apiKey, model string) *SimpleLLMStreamer {
	return &SimpleLLMStreamer{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
	}
}

func (s *SimpleLLMStreamer) ChatStreamText(ctx context.Context, messages []ai.ChatMessage) (<-chan string, error) {
	textChan := make(chan string, 256)

	go func() {
		defer close(textChan)
	}()

	return textChan, nil
}
