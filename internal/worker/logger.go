package worker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents a build log severity.
type LogLevel string

const (
	LogInfo  LogLevel = "INFO"
	LogWarn  LogLevel = "WARN"
	LogError LogLevel = "ERROR"
	LogDebug LogLevel = "DEBUG"
)

// BuildLogger multiplexes build output to a log file and the SSE hub.
type BuildLogger struct {
	deploymentID string
	file         *os.File
	sseHub       *SSEHub
	maxSize      int64
	currentSize  int64
	mu           sync.Mutex
}

// NewBuildLogger creates a logger that writes to both file and SSE.
func NewBuildLogger(logPath string, sseHub *SSEHub, deploymentID string, maxSize int64) (*BuildLogger, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return &BuildLogger{
		deploymentID: deploymentID,
		file:         f,
		sseHub:       sseHub,
		maxSize:      maxSize,
	}, nil
}

// Info logs an informational message.
func (l *BuildLogger) Info(msg string) {
	l.write(LogInfo, msg)
}

// Infof logs a formatted informational message.
func (l *BuildLogger) Infof(format string, args ...interface{}) {
	l.write(LogInfo, fmt.Sprintf(format, args...))
}

// Warn logs a warning message.
func (l *BuildLogger) Warn(msg string) {
	l.write(LogWarn, msg)
}

// Error logs an error message.
func (l *BuildLogger) Error(msg string) {
	l.write(LogError, msg)
}

// Errorf logs a formatted error message.
func (l *BuildLogger) Errorf(format string, args ...interface{}) {
	l.write(LogError, fmt.Sprintf(format, args...))
}

func (l *BuildLogger) write(level LogLevel, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)

	lineBytes := int64(len(line))
	if l.currentSize+lineBytes <= l.maxSize {
		_, _ = l.file.WriteString(line)
		l.currentSize += lineBytes
	} else if l.currentSize < l.maxSize {
		truncMsg := fmt.Sprintf("[%s] [WARN] Log output truncated (exceeded %d bytes)\n", timestamp, l.maxSize)
		_, _ = l.file.WriteString(truncMsg)
		l.currentSize = l.maxSize
	}

	l.sseHub.Publish(l.deploymentID, SSEEventLog, line)
}

// StreamWriter returns an io.Writer that writes each line to the logger.
func (l *BuildLogger) StreamWriter(level LogLevel) io.Writer {
	return &logWriter{logger: l, level: level}
}

// Close flushes and closes the log file.
func (l *BuildLogger) Close() error {
	return l.file.Close()
}

// logWriter adapts BuildLogger to the io.Writer interface.
type logWriter struct {
	logger *BuildLogger
	level  LogLevel
	buf    []byte
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]
		if line != "" {
			w.logger.write(w.level, line)
		}
	}
	return len(p), nil
}
