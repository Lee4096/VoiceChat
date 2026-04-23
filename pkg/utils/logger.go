package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	level    LogLevel
	output   *log.Logger
	jsonMode bool
	mu       sync.Mutex
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"msg"`
	TraceID string `json:"trace_id,omitempty"`
	Caller  string `json:"caller"`
}

func NewLogger(levelStr string) *Logger {
	level := INFO
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = DEBUG
	case "INFO":
		level = INFO
	case "WARN":
		level = WARN
	case "ERROR":
		level = ERROR
	case "FATAL":
		level = FATAL
	}

	return &Logger{
		level:    level,
		output:   log.New(os.Stdout, "", 0),
		jsonMode: false,
	}
}

func NewJSONLogger(levelStr string) *Logger {
	logger := NewLogger(levelStr)
	logger.jsonMode = true
	return logger
}

func (l *Logger) SetJSONMode(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.jsonMode = enabled
}

var globalTraceID string
var traceIDMu sync.Mutex

func SetTraceID(traceID string) {
	traceIDMu.Lock()
	defer traceIDMu.Unlock()
	globalTraceID = traceID
}

func GetTraceID() string {
	traceIDMu.Lock()
	defer traceIDMu.Unlock()
	return globalTraceID
}

func (l *Logger) format(level LogLevel, msg string, args ...interface{}) string {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}

	short := file
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		short = file[idx+1:]
	}

	if l.jsonMode {
		entry := LogEntry{
			Time:    time.Now().UTC().Format(time.RFC3339),
			Level:   levelNames[level],
			Message: msg,
			TraceID: GetTraceID(),
			Caller:  fmt.Sprintf("%s:%d", short, line),
		}
		data, _ := json.Marshal(entry)
		return string(data)
	}

	return fmt.Sprintf("%s [%s] %s:%d - %s",
		time.Now().Format("2006-01-02 15:04:05"),
		levelNames[level],
		short,
		line,
		msg,
	)
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level <= DEBUG {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.output.Println(l.format(DEBUG, msg, args...))
	}
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= INFO {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.output.Println(l.format(INFO, msg, args...))
	}
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= WARN {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.output.Println(l.format(WARN, msg, args...))
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	if l.level <= ERROR {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.output.Println(l.format(ERROR, msg, args...))
	}
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	if l.level <= FATAL {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.output.Println(l.format(FATAL, msg, args...))
	}
	os.Exit(1)
}
