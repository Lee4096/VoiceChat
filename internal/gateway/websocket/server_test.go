package websocket

import (
	"strings"
	"testing"
)

func TestCleanTextForTTS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "normal text",
			input:    "Hello, how are you?",
			expected: "Hello, how are you?",
		},
		{
			name:     "with emojis - removes emoji chars",
			input:    "Hello! 👋 How are you?",
			expected: "Hello!  How are you?",
		},
		{
			name:     "chinese with emojis",
			input:    "你好！😄 有什么可以帮助你的吗？",
			expected: "你好！ 有什么可以帮助你的吗？",
		},
		{
			name:     "long text truncated",
			input:    strings.Repeat("a", 300),
			expected: strings.Repeat("a", 200),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanTextForTTS(tt.input)
			if result != tt.expected {
				t.Errorf("cleanTextForTTS(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
