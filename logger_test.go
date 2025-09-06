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
