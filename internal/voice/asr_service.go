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

type ASRService struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type ASRRequest struct {
	Audio     string  `json:"audio"`
	Model     string  `json:"model"`
	Language  string  `json:"language"`
	Timestamp int64   `json:"timestamp"`
}

type ASRResponse struct {
	Text      string  `json:"text"`
	Timestamp int64   `json:"timestamp"`
	IsFinal   bool    `json:"is_final"`
}

func NewASRService(baseURL string) *ASRService {
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}
	return &ASRService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *ASRService) Recognize(ctx context.Context, audioData []byte) (*ASRResponse, error) {
	url := fmt.Sprintf("%s/api/v1/asr/recognize", s.baseURL)

	reqBody := ASRRequest{
		Audio:     base64.StdEncoding.EncodeToString(audioData),
		Model:     "whisper-base",
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
		return nil, fmt.Errorf("ASR request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result ASRResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (s *ASRService) RecognizeStream(ctx context.Context, audioData []byte) (<-chan *ASRResponse, error) {
	url := fmt.Sprintf("%s/api/v1/asr/stream", s.baseURL)

	reqBody := ASRRequest{
		Audio:     base64.StdEncoding.EncodeToString(audioData),
		Model:     "whisper-base",
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
		return nil, fmt.Errorf("ASR stream request failed: %s", string(body))
	}

	resultChan := make(chan *ASRResponse)

	go func() {
		defer close(resultChan)
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var result ASRResponse
				if err := decoder.Decode(&result); err != nil {
					if err == io.EOF {
						return
					}
					return
				}
				resultChan <- &result
				if result.IsFinal {
					return
				}
			}
		}
	}()

	return resultChan, nil
}
