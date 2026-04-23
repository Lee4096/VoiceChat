package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		levelStr string
		want     LogLevel
	}{
		{"DEBUG", DEBUG},
		{"INFO", INFO},
		{"WARN", WARN},
		{"ERROR", ERROR},
		{"debug", DEBUG},
		{"info", INFO},
		{"invalid", INFO},
	}

	for _, tt := range tests {
		t.Run(tt.levelStr, func(t *testing.T) {
			logger := NewLogger(tt.levelStr)
			if logger.level != tt.want {
				t.Errorf("NewLogger(%s).level = %v, want %v", tt.levelStr, logger.level, tt.want)
			}
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	logger := NewLogger("DEBUG")

	var buf bytes.Buffer
	logger.output.SetOutput(&buf)

	logger.Info("test message %s", "arg")

	if !strings.Contains(buf.String(), "test message arg") {
		t.Errorf("Info() output = %s, want to contain 'test message arg'", buf.String())
	}
}

func TestLoggerLevelFilter(t *testing.T) {
	logger := NewLogger("ERROR")

	var buf bytes.Buffer
	logger.output.SetOutput(&buf)

	logger.Debug("debug should not appear")
	logger.Info("info should not appear")
	logger.Warn("warn should not appear")
	logger.Error("error should appear")

	if buf.Len() == 0 {
		t.Error("Error() should produce output at ERROR level")
	}
}

func TestLogFormat(t *testing.T) {
	logger := NewLogger("INFO")

	var buf bytes.Buffer
	logger.output.SetOutput(&buf)

	logger.Info("test")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Info() output = %s, want to contain 'INFO'", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("Info() output = %s, want to contain 'test'", output)
	}
}
