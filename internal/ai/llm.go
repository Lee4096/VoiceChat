package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMConfig 保存 LLM API 客户端的配置。
type LLMConfig struct {
	BaseURL      string  // API 基础 URL（例如 https://openrouter.ai/api/v1）
	APIKey       string  // 认证的 API 密钥
	Model        string  // 模型标识符（例如 "minimax/minimax-m2.5:free"）
	MaxTokens    int     // 响应中的最大 token 数
	Temperature  float64 // 采样温度（0.0-2.0）
	Stream       bool    // 启用流式响应
	SystemPrompt string  // 所有对话的系统提示
}

// LLMClient 封装用于 LLM API 通信的 HTTP 客户端。
type LLMClient struct {
	cfg    LLMConfig
	client *http.Client
}

// ChatMessage 表示对话中的单条消息。
type ChatMessage struct {
	Role    string `json:"role"`    // "system"、"user" 或 "assistant"
	Content string `json:"content"` // 消息文本
	Name    string `json:"name,omitempty"` // 说话人的可选名称
}

// ChatRequest 发送到 LLM 聊天补全端点。
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatResponse 从 LLM 聊天补全端点接收的响应。
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 表示 LLM 的一个可能的响应。
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"` // "stop"、"length" 等
}

// Usage 包含请求的 token 使用统计。
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamResponse 是流式响应中的单个块。
type StreamResponse struct {
	ID      string         `json:"id"`
	Choices []StreamChoice `json:"choices"`
}

// StreamChoice 是流式响应 choices 数组中的单个块。
type StreamChoice struct {
	Index        int           `json:"index"`
	Delta        ChatMessage   `json:"delta"`   // 增量消息内容
	FinishReason string        `json:"finish_reason,omitempty"`
}

// NewLLMClient 使用默认配置覆盖创建新的 LLM 客户端。
// 如果未指定值，则应用合理的默认值。
func NewLLMClient(cfg LLMConfig) *LLMClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://openrouter.ai/api/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "minimax/minimax-m2.5:free"
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 2048
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}

	return &LLMClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// LLMService 提供简化的 LLM 交互接口。
type LLMService struct {
	*LLMClient
}

// NewLLMService 使用给定配置创建新的 LLM 服务。
func NewLLMService(cfg LLMConfig) *LLMService {
	return &LLMService{
		LLMClient: NewLLMClient(cfg),
	}
}

// Chat 发送单条用户消息并返回助手的响应。
// 便捷方法，处理单轮对话。
func (s *LLMService) Chat(ctx context.Context, text string) (*ChatMessage, error) {
	resp, err := s.LLMClient.Chat(ctx, []ChatMessage{{Role: "user", Content: text}})
	if err != nil {
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}
	return &resp.Choices[0].Message, nil
}

// Chat 发送消息列表到 LLM 并返回完整响应。
// 如果配置了系统提示，会自动预填。
func (c *LLMClient) Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	url := fmt.Sprintf("%s/chat/completions", c.cfg.BaseURL)

	if c.cfg.SystemPrompt != "" {
		messages = append([]ChatMessage{
			{Role: "system", Content: c.cfg.SystemPrompt},
		}, messages...)
	}

	reqBody := ChatRequest{
		Model:       c.cfg.Model,
		Messages:    messages,
		MaxTokens:   c.cfg.MaxTokens,
		Temperature: c.cfg.Temperature,
		Stream:      false,
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
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ChatStream 启动流式聊天会话，返回响应块的通道。
// 调用者应从返回的通道读取直到它关闭。
// 上下文取消将停止流。
func (c *LLMClient) ChatStream(ctx context.Context, messages []ChatMessage) (<-chan *StreamResponse, error) {
	url := fmt.Sprintf("%s/api/chat", c.cfg.BaseURL)

	if c.cfg.SystemPrompt != "" {
		messages = append([]ChatMessage{
			{Role: "system", Content: c.cfg.SystemPrompt},
		}, messages...)
	}

	reqBody := ChatRequest{
		Model:       c.cfg.Model,
		Messages:    messages,
		MaxTokens:   c.cfg.MaxTokens,
		Temperature: c.cfg.Temperature,
		Stream:      true,
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
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("LLM stream request failed with status %d: %s", resp.StatusCode, string(body))
	}

	resultChan := make(chan *StreamResponse)

	go func() {
		defer close(resultChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)

		buf := make([]byte, 1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			// 跳过非 JSON 行（例如空行或 SSE 注释）
			if line[0] != '{' {
				continue
			}

			var result StreamResponse
			if err := json.Unmarshal(line, &result); err != nil {
				continue
			}

			select {
			case resultChan <- &result:
			case <-ctx.Done():
				return
			}
		}
	}()

	return resultChan, nil
}

// CompletionRequest 发送到 LLM 补全端点，用于简单的 prompt 补全。
type CompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

// CompletionResponse 从 LLM 补全端点接收的响应。
type CompletionResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

// Complete 发送简单的 prompt 补全请求。
func (c *LLMClient) Complete(ctx context.Context, prompt string) (*CompletionResponse, error) {
	url := fmt.Sprintf("%s/api/generate", c.cfg.BaseURL)

	reqBody := CompletionRequest{
		Model:       c.cfg.Model,
		Prompt:      prompt,
		MaxTokens:   c.cfg.MaxTokens,
		Temperature: c.cfg.Temperature,
		Stream:      false,
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
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM completion request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
