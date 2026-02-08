package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Level represents the logging level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger handles dual-output logging (console + file)
type Logger struct {
	consoleLogger *log.Logger
	fileLogger    *log.Logger
	logFile       *os.File
	verbose       bool
	minLevel      Level
}

var globalLogger *Logger

// Init initializes the global logger
// consoleOutput: where to write INFO logs (typically os.Stdout)
// logFilePath: path to the log file for DEBUG/ERROR logs
// verbose: if true, show DEBUG logs on console as well
func Init(consoleOutput io.Writer, logFilePath string, verbose bool) error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Create loggers
	consoleLogger := log.New(consoleOutput, "", 0) // No prefix for clean console output
	fileLogger := log.New(logFile, "", log.LstdFlags)

	minLevel := LevelInfo
	if verbose {
		minLevel = LevelDebug
	}

	globalLogger = &Logger{
		consoleLogger: consoleLogger,
		fileLogger:    fileLogger,
		logFile:       logFile,
		verbose:       verbose,
		minLevel:      minLevel,
	}

	return nil
}

// Close closes the log file
func Close() {
	if globalLogger != nil && globalLogger.logFile != nil {
		globalLogger.logFile.Close()
	}
}

// Debug logs a debug message (file only, unless verbose)
func Debug(format string, args ...interface{}) {
	if globalLogger == nil {
		return
	}
	globalLogger.log(LevelDebug, format, args...)
}

// Info logs an info message (console + file)
func Info(format string, args ...interface{}) {
	if globalLogger == nil {
		fmt.Printf(format+"\n", args...)
		return
	}
	globalLogger.log(LevelInfo, format, args...)
}

// Warn logs a warning message (console + file)
func Warn(format string, args ...interface{}) {
	if globalLogger == nil {
		fmt.Printf("WARN: "+format+"\n", args...)
		return
	}
	globalLogger.log(LevelWarn, format, args...)
}

// Error logs an error message (console + file)
func Error(format string, args ...interface{}) {
	if globalLogger == nil {
		fmt.Printf("ERROR: "+format+"\n", args...)
		return
	}
	globalLogger.log(LevelError, format, args...)
}

// log handles the actual logging logic
func (l *Logger) log(level Level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	// Always log to file with timestamp and level (regardless of minLevel)
	l.fileLogger.Printf("[%s] %s", level.String(), message)

	// Only log to console if level >= minLevel
	if level < l.minLevel {
		return
	}

	// Log to console based on level
	switch level {
	case LevelDebug:
		if l.verbose {
			l.consoleLogger.Printf("[DEBUG] %s", message)
		}
	case LevelInfo:
		l.consoleLogger.Printf("%s", message) // Clean output for INFO
	case LevelWarn:
		l.consoleLogger.Printf("⚠️  %s", message)
	case LevelError:
		l.consoleLogger.Printf("❌ %s", message)
	}
}

// InfoClean logs an info message without any prefix (console only)
// Useful for progress updates that shouldn't go to log file
func InfoClean(format string, args ...interface{}) {
	if globalLogger == nil {
		fmt.Printf(format+"\n", args...)
		return
	}
	globalLogger.consoleLogger.Printf(format, args...)
}

// LogParseError logs a parsing error (file only, not console)
// This keeps the console clean while preserving error details in the log file
func LogParseError(filePath string, err error, context string) {
	if globalLogger == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf("[PARSE_ERROR] File: %s, Context: %s, Error: %v", filePath, context, err)
	globalLogger.fileLogger.Printf("[%s] %s", timestamp, message)

	// Only show count on console, details in file
	Debug("Parse error in %s: %v", filePath, err)
}

// GetLogFilePath returns the path to the current log file
func GetLogFilePath() string {
	if globalLogger != nil && globalLogger.logFile != nil {
		return globalLogger.logFile.Name()
	}
	return ""
}

// IsVerbose returns whether verbose logging is enabled
func IsVerbose() bool {
	if globalLogger == nil {
		return false
	}
	return globalLogger.verbose
}
