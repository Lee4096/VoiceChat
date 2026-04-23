package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
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
	Mode       string
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
	TTSVoicesPath   string
	TTSTokensPath   string
	TTSDataDir      string
	SampleRate      int
	EnableVAD       bool
	VADAggressiveness int
}

type LLMConfig struct {
	BaseURL      string
	APIKey       string
	Model        string
	MaxTokens    int
	Temperature  float64
	SystemPrompt string
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix("VOICECHAT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("http.port", 8080)
	v.SetDefault("http.read_timeout", 30)
	v.SetDefault("http.write_timeout", 30)
	v.SetDefault("websocket.port", 8081)
	v.SetDefault("websocket.read_timeout", 60)
	v.SetDefault("websocket.write_timeout", 60)
	v.SetDefault("signaling.port", 8082)
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("jwt.secret", "your-secret-key-change-in-production")
	v.SetDefault("jwt.expiration", 86400)
	v.SetDefault("voice.sample_rate", 16000)
	v.SetDefault("voice.enable_vad", false)
	v.SetDefault("voice.vad_aggressiveness", 2)
	v.SetDefault("llm.base_url", "https://openrouter.ai/api/v1")
	v.SetDefault("llm.max_tokens", 2048)
	v.SetDefault("llm.temperature", 0.7)
	v.SetDefault("llm.model", "minimax/minimax-m2.5:free")
	v.SetDefault("llm.system_prompt", "You are a helpful AI assistant.")
	v.SetDefault("log_level", "INFO")
	v.SetDefault("mode", "release")

	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}
