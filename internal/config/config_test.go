package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("VOICECHAT_SERVER_PORT")
	os.Unsetenv("VOICECHAT_LLM_MAX_TOKENS")
	os.Unsetenv("VOICECHAT_LLM_API_KEY")
	os.Unsetenv("VOICECHAT_VOICE_SAMPLE_RATE")
	os.Unsetenv("VOICECHAT_VOICE_VAD_AGGRESSIVENESS")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	os.Setenv("VOICECHAT_SERVER_PORT", "9090")
	os.Setenv("VOICECHAT_LLM_MODEL", "test-model")
	defer func() {
		os.Unsetenv("VOICECHAT_SERVER_PORT")
		os.Unsetenv("VOICECHAT_LLM_MODEL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}

	if cfg.LLM.Model != "test-model" {
		t.Errorf("LLM.Model = %s, want test-model", cfg.LLM.Model)
	}
}

func TestLoadServerConfig(t *testing.T) {
	os.Setenv("VOICECHAT_SERVER_HOST", "127.0.0.1")
	os.Setenv("VOICECHAT_SERVER_PORT", "9000")
	defer func() {
		os.Unsetenv("VOICECHAT_SERVER_HOST")
		os.Unsetenv("VOICECHAT_SERVER_PORT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %s, want 127.0.0.1", cfg.Server.Host)
	}

	if cfg.Server.Port != 9000 {
		t.Errorf("Server.Port = %d, want 9000", cfg.Server.Port)
	}
}
