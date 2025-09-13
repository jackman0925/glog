package glog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileLogging(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file
	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the logger
	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Log some messages
	Info("This is an info message")
	Warn("This is a warning message")

	// Check if log files were created and contain the correct messages
	checkLogFile(t, filepath.Join(tempDir, FileInfo), "INFO", "This is an info message")
	checkLogFile(t, filepath.Join(tempDir, FileWarn), "WARN", "This is a warning message")
}

func TestNewLogger(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_test_new")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file
	configContent := `
encoder: json
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create a new logger
	logger, err := New(configPath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create new logger: %v", err)
	}

	// Log some messages
	logger.Info("This is an info message from new logger")

	// Check if log file was created and contains the correct message
	checkLogFile(t, filepath.Join(tempDir, FileInfo), `"level":"INFO"`, `"message":"This is an info message from new logger"`)
}

func TestInitError(t *testing.T) {
	err := Init("non_existent_config.yaml", "somedir")
	if err == nil {
		t.Error("Expected an error when initializing with a non-existent config file, but got nil")
	}
}

func TestPrintf(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_test_printf")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file
	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the logger
	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Test Printf with formatted message
	username := "testuser"
	loginTime := "2024-01-01 12:00:00"
	Printf("用户 %s 登录成功，时间: %s", username, loginTime)

	// Check if log file was created and contains the formatted message
	expectedMessage := "用户 testuser 登录成功，时间: 2024-01-01 12:00:00"
	checkLogFile(t, filepath.Join(tempDir, FileInfo), "INFO", expectedMessage)
}

func TestNewLoggerWithPrintf(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_test_newlogger_printf")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file
	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create a new Logger instance
	logger, err := NewLogger(configPath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create new logger: %v", err)
	}

	// Test Printf with formatted message
	username := "testuser"
	loginTime := "2024-01-01 12:00:00"
	logger.Printf("用户 %s 登录成功，时间: %s", username, loginTime)

	// Check if log file was created and contains the formatted message
	expectedMessage := "用户 testuser 登录成功，时间: 2024-01-01 12:00:00"
	checkLogFile(t, filepath.Join(tempDir, FileInfo), "INFO", expectedMessage)
}

func TestGoroutineID(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_test_goroutine")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file with goroutine ID enabled
	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
show_goroutine: true
encode_level: Capital
log_stdout: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the logger
	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Log a message
	Info("This is a test message with goroutine ID")

	// Check if log file was created and contains goroutine ID
	content, err := os.ReadFile(filepath.Join(tempDir, FileInfo))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "goroutine") {
		t.Errorf("Log file should contain goroutine ID, but it doesn't. Content: %s", logContent)
	}

	// Test with NewLogger as well
	logger, err := NewLogger(configPath, tempDir+"_new")
	if err != nil {
		t.Fatalf("Failed to create new logger: %v", err)
	}

	logger.Info("This is a test message from NewLogger with goroutine ID")

	// Check if log file was created and contains goroutine ID
	content2, err := os.ReadFile(filepath.Join(tempDir+"_new", FileInfo))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent2 := string(content2)
	if !strings.Contains(logContent2, "goroutine") {
		t.Errorf("Log file should contain goroutine ID, but it doesn't. Content: %s", logContent2)
	}
}

func checkLogFile(t *testing.T, filePath, level, message string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read log file %s: %v", filePath, err)
	}

	if !strings.Contains(string(content), level) {
		t.Errorf("Log file %s should contain level %s, but it doesn't. Content: %s", filePath, level, string(content))
	}

	if !strings.Contains(string(content), message) {
		t.Errorf("Log file %s should contain message '%s', but it doesn't. Content: %s", filePath, message, string(content))
	}
}

func TestLogLevelFiltering(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_test_level_filtering")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file with log level set to "warn" and separate_levels: false
	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
separate_levels: false
log_level: "warn"
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the logger
	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Log messages at different levels
	Debug("This is a debug message")
	Info("This is an info message")
	Warn("This is a warning message")
	Error("This is an error message")

	// Check the combined log file
	logFilePath := filepath.Join(tempDir, "app.log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file %s: %v", logFilePath, err)
	}

	logContent := string(content)

	// Should contain WARN and ERROR messages
	if !strings.Contains(logContent, "This is a warning message") {
		t.Errorf("Log file should contain the warning message, but it doesn't. Content: %s", logContent)
	}
	if !strings.Contains(logContent, "This is an error message") {
		t.Errorf("Log file should contain the error message, but it doesn't. Content: %s", logContent)
	}

	// Should NOT contain DEBUG and INFO messages
	if strings.Contains(logContent, "This is a debug message") {
		t.Errorf("Log file should NOT contain the debug message, but it does. Content: %s", logContent)
	}
	if strings.Contains(logContent, "This is an info message") {
		t.Errorf("Log file should NOT contain the info message, but it does. Content: %s", logContent)
	}
}

func TestMissingLogLevelDefaultsToInfo(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_test_missing_level")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file without log_level, which should default to INFO
	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
separate_levels: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the logger
	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Log messages at different levels
	Debug("This is a debug message")
	Info("This is an info message")

	// Check the combined log file
	logFilePath := filepath.Join(tempDir, "app.log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file %s: %v", logFilePath, err)
	}

	logContent := string(content)

	// Should contain INFO message
	if !strings.Contains(logContent, "This is an info message") {
		t.Errorf("Log file should contain the info message, but it doesn't. Content: %s", logContent)
	}

	// Should NOT contain DEBUG message
	if strings.Contains(logContent, "This is a debug message") {
		t.Errorf("Log file should NOT contain the debug message, but it does. Content: %s", logContent)
	}
}
