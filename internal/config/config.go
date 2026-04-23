package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server     ServerConfig
	HTTP       HTTPConfig
	WebSocket  WebSocketConfig
	Signaling  SignalingConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	OAuth2     OAuth2Config
	Voice      VoiceConfig
	LLM        LLMConfig
	LogLevel   string
}

type ServerConfig struct {
	Host string
	Port int
}

type HTTPConfig struct {
	Port         int
	ReadTimeout  int
	WriteTimeout int
}

type WebSocketConfig struct {
	Port         int
	ReadTimeout  int
	WriteTimeout int
}

type SignalingConfig struct {
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	Expiration int
}

type OAuth2Config struct {
	GitHub struct {
		ClientID     string
		ClientSecret string
		CallbackURL  string
	}
	Google struct {
		ClientID     string
		ClientSecret string
		CallbackURL  string
	}
}

type VoiceConfig struct {
	ASREncoderPath  string
	ASRDecoderPath  string
	ASRTokensPath   string
	TTSModelPath    string
	TTSVoicesPath  string
	TTSTokensPath   string
	TTSDataDir      string
	SampleRate      int
}

type LLMConfig struct {
	BaseURL      string
	APIKey       string
	Model        string
	MaxTokens    int
	Temperature  float64
	SystemPrompt string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvAsInt("SERVER_PORT", 8080),
		},
		HTTP: HTTPConfig{
			Port:         getEnvAsInt("HTTP_PORT", 8080),
			ReadTimeout:  getEnvAsInt("HTTP_READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("HTTP_WRITE_TIMEOUT", 30),
		},
		WebSocket: WebSocketConfig{
			Port:         getEnvAsInt("WS_PORT", 8081),
			ReadTimeout:  getEnvAsInt("WS_READ_TIMEOUT", 60),
			WriteTimeout: getEnvAsInt("WS_WRITE_TIMEOUT", 60),
		},
		Signaling: SignalingConfig{
			Port: getEnvAsInt("SIGNALING_PORT", 8082),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "voice"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			Expiration: getEnvAsInt("JWT_EXPIRATION", 86400),
		},
		OAuth2: OAuth2Config{
			GitHub: struct {
				ClientID     string
				ClientSecret string
				CallbackURL  string
			}{
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
				CallbackURL:  getEnv("GITHUB_CALLBACK_URL", "http://localhost:8080/api/v1/auth/callback/github"),
			},
			Google: struct {
				ClientID     string
				ClientSecret string
				CallbackURL  string
			}{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				CallbackURL:  getEnv("GOOGLE_CALLBACK_URL", "http://localhost:8080/api/v1/auth/callback/google"),
			},
		},
		Voice: VoiceConfig{
			ASREncoderPath:  getEnv("ASR_ENCODER_PATH", "./models/paraformer/sherpa-onnx-streaming-paraformer-bilingual-zh-en/encoder.onnx"),
			ASRDecoderPath:  getEnv("ASR_DECODER_PATH", "./models/paraformer/sherpa-onnx-streaming-paraformer-bilingual-zh-en/decoder.onnx"),
			ASRTokensPath:   getEnv("ASR_TOKENS_PATH", "./models/paraformer/sherpa-onnx-streaming-paraformer-bilingual-zh-en/tokens.txt"),
			TTSModelPath:    getEnv("TTS_MODEL_PATH", "./models/kokoro/kokoro-en-v0_19/model.onnx"),
			TTSVoicesPath:  getEnv("TTS_VOICES_PATH", "./models/kokoro/kokoro-en-v0_19/voices.bin"),
			TTSTokensPath:   getEnv("TTS_TOKENS_PATH", "./models/kokoro/kokoro-en-v0_19/tokens.txt"),
			TTSDataDir:      getEnv("TTS_DATA_DIR", "./models/kokoro/kokoro-en-v0_19/espeak-ng-data"),
			SampleRate:      getEnvAsInt("VOICE_SAMPLE_RATE", 16000),
		},
		LLM: LLMConfig{
			BaseURL:      getEnv("LLM_BASE_URL", "https://openrouter.ai/api/v1"),
			APIKey:       getEnv("LLM_API_KEY", "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"),
			Model:        getEnv("LLM_MODEL", "minimax/minimax-m2.5:free"),
			MaxTokens:    getEnvAsInt("LLM_MAX_TOKENS", 2048),
			Temperature:  float64(getEnvAsInt("LLM_TEMPERATURE", 70)) / 100.0,
			SystemPrompt: getEnv("LLM_SYSTEM_PROMPT", "You are a helpful AI assistant."),
		},
		LogLevel: getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
