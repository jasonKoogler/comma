package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger defines the interface for logging
type Logger interface {
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
	Debug(format string, v ...interface{})
}

// LogLevel represents log severity
type LogLevel string

const (
	InfoLevel  LogLevel = "INFO"
	WarnLevel  LogLevel = "WARN"
	ErrorLevel LogLevel = "ERROR"
	DebugLevel LogLevel = "DEBUG"
)

// FileLogger implements Logger with file-based logging
type FileLogger struct {
	logger *log.Logger
	file   *os.File
	mu     sync.Mutex
}

// NewFileLogger creates a new file logger
func NewFileLogger(appName string) (*FileLogger, error) {
	// Create log directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	logDir := filepath.Join(homeDir, "."+appName, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file
	logFile := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", appName, time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(file, "", log.LstdFlags)
	return &FileLogger{
		logger: logger,
		file:   file,
	}, nil
}

// log logs a message with the given level
func (l *FileLogger) log(level LogLevel, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, v...)
	l.logger.Printf("[%s] %s", level, msg)
}

// Info logs an info message
func (l *FileLogger) Info(format string, v ...interface{}) {
	l.log(InfoLevel, format, v...)
}

// Warn logs a warning message
func (l *FileLogger) Warn(format string, v ...interface{}) {
	l.log(WarnLevel, format, v...)
}

// Error logs an error message
func (l *FileLogger) Error(format string, v ...interface{}) {
	l.log(ErrorLevel, format, v...)
}

// Debug logs a debug message
func (l *FileLogger) Debug(format string, v ...interface{}) {
	l.log(DebugLevel, format, v...)
}

// Close closes the logger file
func (l *FileLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// ConsoleLogger implements Logger with console output
type ConsoleLogger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		warnLogger:  log.New(os.Stdout, "[WARN] ", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags),
	}
}

// Info logs an info message
func (l *ConsoleLogger) Info(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

// Warn logs a warning message
func (l *ConsoleLogger) Warn(format string, v ...interface{}) {
	l.warnLogger.Printf(format, v...)
}

// Error logs an error message
func (l *ConsoleLogger) Error(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

// Debug logs a debug message
func (l *ConsoleLogger) Debug(format string, v ...interface{}) {
	l.debugLogger.Printf(format, v...)
}

// NullLogger implements Logger without any output
type NullLogger struct{}

// NewNullLogger creates a new null logger
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// Info does nothing
func (l *NullLogger) Info(format string, v ...interface{}) {}

// Warn does nothing
func (l *NullLogger) Warn(format string, v ...interface{}) {}

// Error does nothing
func (l *NullLogger) Error(format string, v ...interface{}) {}

// Debug does nothing
func (l *NullLogger) Debug(format string, v ...interface{}) {}
