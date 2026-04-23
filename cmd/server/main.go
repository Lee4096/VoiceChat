package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"voicechat/internal/ai"
	"voicechat/internal/auth"
	"voicechat/internal/config"
	httpgateway "voicechat/internal/gateway/http"
	"voicechat/internal/gateway/websocket"
	"voicechat/internal/repository/postgres"
	"voicechat/internal/repository/redis"
	"voicechat/internal/signaling"
	"voicechat/pkg/utils"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	var logger *utils.Logger
	if cfg.LogFormat == "json" {
		logger = utils.NewJSONLogger(cfg.LogLevel)
	} else {
		logger = utils.NewLogger(cfg.LogLevel)
	}
	ctx := context.Background()

	pg, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL: %v", err)
	}
	defer pg.Close()

	if err := pg.InitSchema(ctx); err != nil {
		logger.Error("Failed to init schema: %v", err)
	}

	rd, err := redis.NewClient(ctx, cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis: %v", err)
	}
	defer rd.Close()

	signalServer := signaling.NewServer(signaling.Config{Port: cfg.Signaling.Port}, logger)
	defer signalServer.Close()

	llmService := ai.NewLLMService(ai.LLMConfig{
		BaseURL:      cfg.LLM.BaseURL,
		APIKey:       cfg.LLM.APIKey,
		Model:        cfg.LLM.Model,
		MaxTokens:    cfg.LLM.MaxTokens,
		Temperature:  cfg.LLM.Temperature,
		SystemPrompt: cfg.LLM.SystemPrompt,
	})

	wsServer, err := websocket.NewServer(websocket.Config{
		Port:           cfg.WebSocket.Port,
		ReadTimeout:    cfg.WebSocket.ReadTimeout,
		WriteTimeout:   cfg.WebSocket.WriteTimeout,
		ASREncoderPath: cfg.Voice.ASREncoderPath,
		ASRDecoderPath: cfg.Voice.ASRDecoderPath,
		ASRTokensPath:  cfg.Voice.ASRTokensPath,
		TTSModelPath:   cfg.Voice.TTSModelPath,
		TTSVoicesPath:  cfg.Voice.TTSVoicesPath,
		TTSTokensPath:  cfg.Voice.TTSTokensPath,
		TTSDataDir:     cfg.Voice.TTSDataDir,
	}, logger, signalServer, llmService)
	if err != nil {
		logger.Fatal("Failed to create WebSocket server: %v", err)
	}
	defer wsServer.Close()

	httpCfg := httpgateway.Config{
		Port:         cfg.HTTP.Port,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}
	jwtCfg := auth.JWTConfig{
		Secret:     cfg.JWT.Secret,
		Expiration: cfg.JWT.Expiration,
	}
	oauthCfg := httpgateway.OAuth2ConfigInput{
		GitHub: struct {
			ClientID     string
			ClientSecret string
			CallbackURL  string
		}{
			ClientID:     cfg.OAuth2.GitHub.ClientID,
			ClientSecret: cfg.OAuth2.GitHub.ClientSecret,
			CallbackURL:  cfg.OAuth2.GitHub.CallbackURL,
		},
		Google: struct {
			ClientID     string
			ClientSecret string
			CallbackURL  string
		}{
			ClientID:     cfg.OAuth2.Google.ClientID,
			ClientSecret: cfg.OAuth2.Google.ClientSecret,
			CallbackURL:  cfg.OAuth2.Google.CallbackURL,
		},
	}
	httpServer := httpgateway.NewServer(httpCfg, logger, pg, rd, signalServer, jwtCfg, oauthCfg)
	defer httpServer.Close()

	go signalServer.Run(ctx)
	go wsServer.Run(ctx)

	go func() {
		logger.Info("Server started on :%d", cfg.HTTP.Port)
		if err := httpServer.Run(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
}