package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"voicechat/internal/ai"
	"voicechat/internal/signaling"
	"voicechat/internal/voice"
	"voicechat/pkg/utils"

	"github.com/gorilla/websocket"
)

// 安全限制，防止滥用
const (
	MaxAudioPayloadSize = 512 * 1024 // 最大音频负载 512KB
	MaxTextLength       = 2000      // 最大文本输入 2000 字符
	LLMTimeout          = 30 * time.Second  // LLM 请求超时
	TTSTimeout          = 60 * time.Second   // TTS 合成超时
)

// Server 是处理实时语音和文本通信的 WebSocket 服务器。
// 管理客户端连接、WebRTC 信令和 AI 语音/文本交互。
type Server struct {
	cfg           Config
	logger        Logger
	signaling     *signaling.Server       // WebRTC 信令服务器
	upgrader      websocket.Upgrader      // HTTP 到 WebSocket 的升级器
	clients       map[string]*Client     // 按连接 ID 活跃的客户端
	voiceService  *voice.SherpaVoiceService // ASR/TTS 服务
	llm           *ai.LLMService         // LLM 服务用于 AI 响应
	voiceProcessor *voice.VoiceProcessor  // 带缓冲的语音处理
	mu            sync.RWMutex           // 保护 clients map 和 voiceService
}

// Logger 定义服务器使用的日志接口。
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
}

// Config 保存 WebSocket 服务器的所有配置。
type Config struct {
	Port             int    // WebSocket 服务器端口
	ReadTimeout      int    // 读取超时（秒）
	WriteTimeout     int    // 写入超时（秒）
	ASREncoderPath   string // Sherpa ASR 编码器模型路径
	ASRDecoderPath   string // Sherpa ASR 解码器模型路径
	ASRTokensPath    string // ASR tokens 文件路径
	TTSModelPath     string // Kokoro TTS 模型路径
	TTSVoicesPath    string // Kokoro 语音文件路径
	TTSTokensPath    string // TTS tokens 文件路径
	TTSDataDir       string // TTS 数据目录
	LLMEndpoint      string // LLM API 端点
	LLMApiKey        string // LLM API 密钥
}

// Client 代表单个 WebSocket 客户端连接。
// 每个客户端属于一个房间，可以发送/接收语音和文本消息。
type Client struct {
	conn      *websocket.Conn
	server    *Server
	userID    string       // 用户标识符
	roomID    string       // 用户当前所在的房间
	send      chan []byte  // 出站消息通道（带缓冲）
	mu        sync.Mutex   // 保护连接写入
	processor *voice.VoiceProcessor // 该客户端的语音处理
}

// Message 代表具有类型和负载的 WebSocket 消息。
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ClientMessage 代表来自客户端的传入消息。
type ClientMessage struct {
	Type    string          `json:"type"`
	RoomID  string          `json:"room_id,omitempty"`
	UserID  string          `json:"user_id,omitempty"`
	Token   string          `json:"token,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// VoiceDataPayload 包含从客户端发送的音频数据。
type VoiceDataPayload struct {
	Audio     string `json:"audio"`      // Base64 编码的 PCM 音频
	Format    string `json:"format"`    // 音频格式（例如 "pcm"）
	SampleRate int    `json:"sample_rate"` // 采样率（例如 16000）
}

// AIResponsePayload 发送给客户端，包含 AI 响应。
type AIResponsePayload struct {
	Type    string `json:"type"`    // 消息类型（ai_voice_response、ai_text_response）
	Text    string `json:"text,omitempty"`    // 文本响应
	Audio   string `json:"audio,omitempty"`   // Base64 编码的音频响应
	IsFinal bool   `json:"is_final"` // 是否为最终响应
}

// NewServer 使用给定配置创建新的 WebSocket 服务器。
func NewServer(cfg Config, logger Logger, signal *signaling.Server, llm *ai.LLMService) (*Server, error) {
	return &Server{
		cfg:          cfg,
		logger:       logger,
		signaling:    signal,
		llm:          llm,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients:        make(map[string]*Client),
		voiceService:   nil,    // 延迟初始化
		voiceProcessor: nil,
	}, nil
}

// initVoiceService 在首次使用时初始化 Sherpa ASR/TTS 服务。
// 线程安全，采用延迟初始化模式。
func (s *Server) initVoiceService() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.voiceService != nil {
		return nil
	}

	s.logger.Info("Initializing voice service...")
	voiceSvc, err := voice.NewSherpaVoiceService(voice.SherpaConfig{
		ASRConfig: voice.ASRConfig{
			EncoderPath: s.cfg.ASREncoderPath,
			DecoderPath: s.cfg.ASRDecoderPath,
			TokensPath:  s.cfg.ASRTokensPath,
			NThreads:    4,
		},
		TTSConfig: voice.KokoroTTSConfig{
			ModelPath:  s.cfg.TTSModelPath,
			VoicesPath: s.cfg.TTSVoicesPath,
			TokensPath: s.cfg.TTSTokensPath,
			DataDir:    s.cfg.TTSDataDir,
			Lang:       "en-us",
			Speed:      1.0,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create voice service: %w", err)
	}

	s.voiceService = voiceSvc
	s.voiceProcessor = voice.NewVoiceProcessor(voiceSvc, 16000)
	s.logger.Info("Voice service initialized successfully")
	return nil
}

// Run 启动 WebSocket 服务器并阻塞直到上下文被取消。
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.cfg.Port)
	s.logger.Info("WebSocket server starting on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("WebSocket server error: %v", err)
		}
	}()

	<-ctx.Done()
	return server.Close()
}

// Close 优雅关闭服务器，关闭所有客户端连接并释放资源。
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.voiceService != nil {
		s.voiceService.Close()
	}

	for _, client := range s.clients {
		client.conn.Close()
	}
	return nil
}

// handleHealth 返回服务器健康状态用于监控。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleWebSocket 将 HTTP 连接升级为 WebSocket 并注册新客户端。
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if err := s.initVoiceService(); err != nil {
		s.logger.Error("Failed to initialize voice service: %v", err)
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection: %v", err)
		return
	}

	s.mu.RLock()
	processor := s.voiceProcessor
	s.mu.RUnlock()

	client := &Client{
		conn:      conn,
		server:    s,
		send:      make(chan []byte, 256),
		processor: processor,
	}

	s.mu.Lock()
	s.clients[clientID(conn)] = client
	s.mu.Unlock()

	go client.writePump()
	go client.readPump()

	s.logger.Info("New WebSocket connection from %s", r.RemoteAddr)
}

// handleMessage 根据消息类型将传入消息路由到相应的处理程序。
// 支持：join_room、leave_room、offer、answer、ice_candidate、voice_data、ai_voice_chat、ai_text_chat
func (s *Server) handleMessage(client *Client, msg *ClientMessage) {
	s.logger.Info("handleMessage: type=%s, userID=%s, roomID=%s", msg.Type, client.userID, client.roomID)
	switch msg.Type {
	case "join_room":
		s.handleJoinRoom(client, msg)
	case "leave_room":
		s.handleLeaveRoom(client)
	case "offer":
		s.handleOffer(client, msg)
	case "answer":
		s.handleAnswer(client, msg)
	case "ice_candidate":
		s.handleIceCandidate(client, msg)
	case "voice_data":
		s.handleVoiceData(client, msg)
	case "ai_voice_chat":
		s.logger.Info("Routing to handleAIVoiceChat")
		s.handleAIVoiceChat(client, msg)
	case "ai_text_chat":
		s.logger.Info("Routing to handleAITextChat")
		s.handleAITextChat(client, msg)
	default:
		s.logger.Error("Unknown message type: %s", msg.Type)
	}
}

func (s *Server) handleJoinRoom(client *Client, msg *ClientMessage) {
	client.mu.Lock()
	client.userID = msg.UserID
	client.roomID = msg.RoomID
	client.mu.Unlock()

	response := Message{
		Type: "room_joined",
		Payload: json.RawMessage(`{"room_id":"` + msg.RoomID + `"}`),
	}
	client.sendJSON(response)

	broadcast := Message{
		Type: "user_joined",
		Payload: json.RawMessage(`{"user_id":"` + msg.UserID + `"}`),
	}
	s.broadcastToRoom(msg.RoomID, broadcast, client)

	s.logger.Info("User %s joined room %s", msg.UserID, msg.RoomID)
}

func (s *Server) handleLeaveRoom(client *Client) {
	client.mu.Lock()
	roomID := client.roomID
	userID := client.userID
	client.roomID = ""
	client.mu.Unlock()

	broadcast := Message{
		Type: "user_left",
		Payload: json.RawMessage(`{"user_id":"` + userID + `"}`),
	}
	s.broadcastToRoom(roomID, broadcast, nil)

	s.logger.Info("User %s left room %s", userID, roomID)
}

func (s *Server) handleOffer(client *Client, msg *ClientMessage) {
	s.forwardToUser(msg.RoomID, msg.UserID, msg.Type, msg.Payload)
}

func (s *Server) handleAnswer(client *Client, msg *ClientMessage) {
	s.forwardToUser(msg.RoomID, msg.UserID, msg.Type, msg.Payload)
}

func (s *Server) handleIceCandidate(client *Client, msg *ClientMessage) {
	s.forwardToUser(msg.RoomID, msg.UserID, msg.Type, msg.Payload)
}

func (s *Server) handleVoiceData(client *Client, msg *ClientMessage) {
	broadcast := Message{
		Type:    "voice_data",
		Payload: msg.Payload,
	}
	s.broadcastToRoom(client.roomID, broadcast, client)
}

// handleAIVoiceChat 处理语音输入：ASR -> LLM -> TTS，返回音频响应。
// 完整管道：Base64 音频 -> 解码 -> 识别 -> LLM 聊天 -> 清理文本 -> 合成 -> 编码 -> 响应
func (s *Server) handleAIVoiceChat(client *Client, msg *ClientMessage) {
	s.logger.Info("handleAIVoiceChat: processor=%v", client.processor != nil)
	if client.processor == nil {
		s.logger.Error("Voice processor not initialized")
		return
	}

	var payload VoiceDataPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		s.logger.Error("Failed to unmarshal voice data: %v", err)
		return
	}

	if len(payload.Audio) > MaxAudioPayloadSize {
		s.logger.Error("Audio payload too large: %d bytes (max: %d)", len(payload.Audio), MaxAudioPayloadSize)
		return
	}

	s.logger.Info("Audio payload length: %d", len(payload.Audio))

	// 解码 Base64 音频为原始字节
	audioData, err := utils.Base64Decode(payload.Audio)
	if err != nil {
		s.logger.Error("Failed to decode audio: %v", err)
		return
	}

	s.logger.Info("Decoded audio length: %d bytes", len(audioData))

	// 将 16 位 PCM 转换为 float32 用于 ASR
	samples := voice.Int16ToFloat32(utils.BytesToInt16(audioData))
	s.logger.Info("Samples count: %d", len(samples))

	// 步骤 1：语音识别 (ASR)
	result, err := client.processor.Recognize(samples)
	if err != nil {
		s.logger.Error("ASR error: %v", err)
		return
	}

	s.logger.Info("ASR result: %+v", result)

	if result != nil && result.Text != "" {
		s.logger.Info("User %s said: %s", client.userID, result.Text)

		// 步骤 2：LLM 聊天
		ctx, cancel := context.WithTimeout(context.Background(), LLMTimeout)
		defer cancel()

		aiResp, err := s.llm.Chat(ctx, result.Text)
		if err != nil {
			s.logger.Error("LLM error: %v", err)
			return
		}

		s.logger.Info("AI response: %s", aiResp.Content)

		// 步骤 3：清理 TTS 文本（移除 emoji，截断）
		cleanText := cleanTextForTTS(aiResp.Content)
		s.logger.Info("Cleaned text for TTS: %s", cleanText)

		s.logger.Info("Starting TTS synthesis...")

		// 步骤 4：带超时的文本转语音合成
		ttsCtx, ttsCancel := context.WithTimeout(context.Background(), TTSTimeout)
		done := make(chan struct{})
		var audio []float32
		var ttsErr error

		go func() {
			audio, ttsErr = client.processor.Synthesize(cleanText)
			close(done)
		}()

		select {
		case <-ttsCtx.Done():
			s.logger.Error("TTS timeout after %v", TTSTimeout)
			ttsCancel()
			return
		case <-done:
			ttsCancel()
		}

		if ttsErr != nil {
			s.logger.Error("TTS error: %v", ttsErr)
			return
		}
		s.logger.Info("TTS synthesis complete, audio length: %d samples", len(audio))

		// 步骤 5：编码音频并发送响应
		int16Audio := voice.Float32ToInt16(audio)
		audioB64 := utils.Base64Encode(utils.Int16ToBytes(int16Audio))

		response := Message{
			Type: "ai_voice_response",
			Payload: json.RawMessage(fmt.Sprintf(
				`{"text":"%s","audio":"%s","is_final":true}`,
				utils.EscapeJSONString(aiResp.Content),
				audioB64,
			)),
		}
		client.sendJSON(response)
	} else {
		s.logger.Info("ASR returned empty result")
	}
}

// handleAITextChat 处理文本输入：LLM 聊天，返回文本响应。
func (s *Server) handleAITextChat(client *Client, msg *ClientMessage) {
	s.logger.Info("handleAITextChat called")
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		s.logger.Error("Failed to unmarshal text: %v", err)
		return
	}

	if len(payload.Text) == 0 || len(payload.Text) > MaxTextLength {
		s.logger.Error("Invalid text length: %d (max: %d)", len(payload.Text), MaxTextLength)
		return
	}

	s.logger.Info("User said: %s", payload.Text)

	ctx, cancel := context.WithTimeout(context.Background(), LLMTimeout)
	defer cancel()

	aiResp, err := s.llm.Chat(ctx, payload.Text)
	if err != nil {
		s.logger.Error("LLM error: %v", err)
		return
	}

	s.logger.Info("AI response: %s", aiResp.Content)

	response := Message{
		Type: "ai_text_response",
		Payload: json.RawMessage(fmt.Sprintf(
			`{"text":"%s"}`,
			utils.EscapeJSONString(aiResp.Content),
		)),
	}
	client.sendJSON(response)
	s.logger.Info("Sent ai_text_response to client")
}

func (s *Server) broadcastToRoom(roomID string, msg Message, except *Client) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.clients {
		if c.roomID == roomID && c != except {
			c.sendJSON(msg)
		}
	}
}

func (s *Server) forwardToUser(roomID, targetUserID, msgType string, payload json.RawMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg := Message{
		Type:    msgType,
		Payload: payload,
	}

	for _, c := range s.clients {
		if c.roomID == roomID && c.userID == targetUserID {
			c.sendJSON(msg)
			break
		}
	}
}

// readPump 持续从 WebSocket 连接读取消息。
// 在 goroutine 中运行，直到连接关闭。
func (c *Client) readPump() {
	defer func() {
		c.server.removeClient(c)
		c.conn.Close()
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.server.logger.Error("WebSocket read error: %v", err)
			}
			break
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.server.logger.Error("Failed to unmarshal message: %v", err)
			continue
		}

		c.server.handleMessage(c, &msg)
	}
}

// writePump 持续从发送通道向 WebSocket 发送消息。
// 在 goroutine 中运行，当发送通道为空或连接关闭时阻塞。
func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.mu.Lock()
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			c.mu.Unlock()

			if err != nil {
				return
			}
		}
	}
}

// sendJSON 将消息排队发送到客户端。
// 非阻塞：如果发送缓冲区满则丢弃消息。
func (c *Client) sendJSON(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case c.send <- data:
	default:
		c.server.logger.Info("Warning: send buffer full, message dropped. Type: %s", msg.Type)
	}
}

// removeClient 注销客户端并清理资源。
func (s *Server) removeClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[clientID(client.conn)]; ok {
		delete(s.clients, clientID(client.conn))
		close(client.send)
	}
}

// clientID 使用 WebSocket 连接的远程地址生成唯一 ID。
func clientID(conn *websocket.Conn) string {
	return utils.MD5(conn.RemoteAddr().String())
}

// cleanTextForTTS 通过移除有问题内容来准备 TTS 合成的文本。
// 移除 emoji、截断至 200 字符（TTS 限制）、并去除空白。
func cleanTextForTTS(text string) string {
	emojiRegex := regexp.MustCompile(`[\x{1F300}-\x{1F9FF}]`)
	text = emojiRegex.ReplaceAllString(text, "")

	text = strings.ReplaceAll(text, "👋", "")
	text = strings.ReplaceAll(text, "📝", "")
	text = strings.ReplaceAll(text, "💻", "")
	text = strings.ReplaceAll(text, "📚", "")
	text = strings.ReplaceAll(text, "✍️", "")
	text = strings.ReplaceAll(text, "💡", "")
	text = strings.ReplaceAll(text, "🎵", "")
	text = strings.ReplaceAll(text, "❤️", "")
	text = strings.ReplaceAll(text, "🔥", "")

	text = strings.ReplaceAll(text, "😄", "")
	text = strings.ReplaceAll(text, "😊", "")
	text = strings.ReplaceAll(text, "👍", "")

	text = strings.TrimSpace(text)

	if len(text) > 200 {
		text = text[:200]
	}

	return text
}
