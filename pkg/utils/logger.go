package utils

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

type Logger struct {
	level  LogLevel
	output *log.Logger
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
		level:  level,
		output: log.New(os.Stdout, "", 0),
	}
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
		l.output.Println(l.format(DEBUG, msg, args...))
	}
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= INFO {
		l.output.Println(l.format(INFO, msg, args...))
	}
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= WARN {
		l.output.Println(l.format(WARN, msg, args...))
	}
}

func (l *Logger) Error(msg string, args ...interface{}) {
	if l.level <= ERROR {
		l.output.Println(l.format(ERROR, msg, args...))
	}
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	if l.level <= FATAL {
		l.output.Println(l.format(FATAL, msg, args...))
	}
	os.Exit(1)
}
