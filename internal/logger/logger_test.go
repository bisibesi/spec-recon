package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerInit(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "spec-recon-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	consoleBuffer := &bytes.Buffer{}

	// Initialize logger
	err = Init(consoleBuffer, logPath, false)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Test that log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// Test Info logging
	Info("Test info message")
	consoleOutput := consoleBuffer.String()
	if !strings.Contains(consoleOutput, "Test info message") {
		t.Errorf("Console output missing info message: %s", consoleOutput)
	}

	// Read log file
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logStr := string(logContent)
	if !strings.Contains(logStr, "[INFO]") {
		t.Error("Log file missing INFO level")
	}
	if !strings.Contains(logStr, "Test info message") {
		t.Error("Log file missing info message")
	}
}

func TestLoggerLevels(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spec-recon-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	consoleBuffer := &bytes.Buffer{}

	err = Init(consoleBuffer, logPath, false)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Test all log levels
	Debug("Debug message")
	Info("Info message")
	Warn("Warn message")
	Error("Error message")

	logContent, _ := os.ReadFile(logPath)
	logStr := string(logContent)

	// File should contain all levels
	if !strings.Contains(logStr, "[DEBUG]") {
		t.Error("Log file missing DEBUG level")
	}
	if !strings.Contains(logStr, "[INFO]") {
		t.Error("Log file missing INFO level")
	}
	if !strings.Contains(logStr, "[WARN]") {
		t.Error("Log file missing WARN level")
	}
	if !strings.Contains(logStr, "[ERROR]") {
		t.Error("Log file missing ERROR level")
	}

	// Console should NOT contain DEBUG (verbose=false)
	consoleStr := consoleBuffer.String()
	if strings.Contains(consoleStr, "[DEBUG]") {
		t.Error("Console should not show DEBUG when verbose=false")
	}
}

func TestLoggerVerbose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spec-recon-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	consoleBuffer := &bytes.Buffer{}

	// Initialize with verbose=true
	err = Init(consoleBuffer, logPath, true)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	Debug("Debug message")

	consoleStr := consoleBuffer.String()
	if !strings.Contains(consoleStr, "[DEBUG]") {
		t.Error("Console should show DEBUG when verbose=true")
	}
	if !strings.Contains(consoleStr, "Debug message") {
		t.Error("Console missing debug message content")
	}
}

func TestLoggerParseError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spec-recon-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	consoleBuffer := &bytes.Buffer{}

	err = Init(consoleBuffer, logPath, false)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	// Log a parse error
	testErr := os.ErrNotExist
	LogParseError("/path/to/file.java", testErr, "test context")

	// Read log file
	logContent, _ := os.ReadFile(logPath)
	logStr := string(logContent)

	// File should contain parse error details
	if !strings.Contains(logStr, "[PARSE_ERROR]") {
		t.Error("Log file missing PARSE_ERROR marker")
	}
	if !strings.Contains(logStr, "/path/to/file.java") {
		t.Error("Log file missing file path")
	}
	if !strings.Contains(logStr, "test context") {
		t.Error("Log file missing context")
	}

	// Console should NOT show parse error details (only summary if debug enabled)
	consoleStr := consoleBuffer.String()
	if strings.Contains(consoleStr, "[PARSE_ERROR]") {
		t.Error("Console should not show detailed parse errors")
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("Level.String() = %s, expected %s", result, tt.expected)
		}
	}
}

func TestGetLogFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spec-recon-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	consoleBuffer := &bytes.Buffer{}

	err = Init(consoleBuffer, logPath, false)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	retrievedPath := GetLogFilePath()
	if retrievedPath != logPath {
		t.Errorf("GetLogFilePath() = %s, expected %s", retrievedPath, logPath)
	}
}

func TestIsVerbose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spec-recon-log-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")
	consoleBuffer := &bytes.Buffer{}

	// Test with verbose=false
	err = Init(consoleBuffer, logPath, false)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	if IsVerbose() {
		t.Error("IsVerbose() should return false when initialized with verbose=false")
	}
	Close()

	// Test with verbose=true
	err = Init(consoleBuffer, logPath, true)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer Close()

	if !IsVerbose() {
		t.Error("IsVerbose() should return true when initialized with verbose=true")
	}
}
