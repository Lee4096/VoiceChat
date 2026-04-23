package voice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TTSService struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type TTSRequest struct {
	Text      string  `json:"text"`
	Model     string  `json:"model"`
	Voice     string  `json:"voice"`
	Speed     float64 `json:"speed"`
	Language  string  `json:"language"`
	Timestamp int64   `json:"timestamp"`
}

type TTSResponse struct {
	Audio     string  `json:"audio"`
	Timestamp int64   `json:"timestamp"`
	Duration  float64 `json:"duration"`
	IsFinal   bool    `json:"is_final"`
}

func NewTTSService(baseURL string) *TTSService {
	if baseURL == "" {
		baseURL = "http://localhost:8084"
	}
	return &TTSService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (s *TTSService) Synthesize(ctx context.Context, text string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/tts/synthesize", s.baseURL)

	reqBody := TTSRequest{
		Text:      text,
		Model:     "tts-1",
		Voice:     "alloy",
		Speed:     1.0,
		Language:  "auto",
		Timestamp: time.Now().UnixMilli(),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result TTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return base64.StdEncoding.DecodeString(result.Audio)
}

func (s *TTSService) SynthesizeStream(ctx context.Context, text string) (<-chan []byte, error) {
	url := fmt.Sprintf("%s/api/v1/tts/stream", s.baseURL)

	reqBody := TTSRequest{
		Text:      text,
		Model:     "tts-1",
		Voice:     "alloy",
		Speed:     1.0,
		Language:  "auto",
		Timestamp: time.Now().UnixMilli(),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("TTS stream request failed: %s", string(body))
	}

	resultChan := make(chan []byte)

	go func() {
		defer close(resultChan)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var result TTSResponse
				if err := decoder.Decode(&result); err != nil {
					if err == io.EOF {
						return
					}
					return
				}
				audioData, err := base64.StdEncoding.DecodeString(result.Audio)
				if err != nil {
					continue
				}
				resultChan <- audioData
				if result.IsFinal {
					return
				}
			}
		}
	}()

	return resultChan, nil
}
