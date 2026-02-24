package glog

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- Helper ---

func checkLogFile(t *testing.T, filePath, level, message string) {
	t.Helper()
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

// writeConfig creates a logger.yaml in tempDir and returns the config path.
func writeConfig(t *testing.T, tempDir, content string) string {
	t.Helper()
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	return configPath
}

const baseConsoleConfig = `
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

// --- Tests ---

func TestFileLogging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := writeConfig(t, tempDir, baseConsoleConfig)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Info("This is an info message")
	Warn("This is a warning message")

	checkLogFile(t, filepath.Join(tempDir, FileInfo), "INFO", "This is an info message")
	checkLogFile(t, filepath.Join(tempDir, FileWarn), "WARN", "This is a warning message")
}

func TestNewLogger(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_new")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	configPath := writeConfig(t, tempDir, configContent)

	logger, err := New(configPath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create new logger: %v", err)
	}

	logger.Info("This is an info message from new logger")

	checkLogFile(t, filepath.Join(tempDir, FileInfo), `"level":"INFO"`, `"message":"This is an info message from new logger"`)
}

func TestInitError(t *testing.T) {
	err := Init("non_existent_config.yaml", "somedir")
	if err == nil {
		t.Error("Expected an error when initializing with a non-existent config file, but got nil")
	}
}

func TestNewError(t *testing.T) {
	_, err := New("non_existent_config.yaml", "somedir")
	if err == nil {
		t.Error("Expected an error when creating logger with a non-existent config file, but got nil")
	}
}

func TestNewLoggerError(t *testing.T) {
	_, err := NewLogger("non_existent_config.yaml", "somedir")
	if err == nil {
		t.Error("Expected an error when creating logger with a non-existent config file, but got nil")
	}
}

func TestPrintf(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_printf")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := writeConfig(t, tempDir, baseConsoleConfig)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	username := "testuser"
	loginTime := "2024-01-01 12:00:00"
	Printf("用户 %s 登录成功，时间: %s", username, loginTime)

	expectedMessage := "用户 testuser 登录成功，时间: 2024-01-01 12:00:00"
	checkLogFile(t, filepath.Join(tempDir, FileInfo), "INFO", expectedMessage)
}

func TestNewLoggerWithPrintf(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_newlogger_printf")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := writeConfig(t, tempDir, baseConsoleConfig)

	logger, err := NewLogger(configPath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create new logger: %v", err)
	}

	username := "testuser"
	loginTime := "2024-01-01 12:00:00"
	logger.Printf("用户 %s 登录成功，时间: %s", username, loginTime)

	expectedMessage := "用户 testuser 登录成功，时间: 2024-01-01 12:00:00"
	checkLogFile(t, filepath.Join(tempDir, FileInfo), "INFO", expectedMessage)
}

func TestGoroutineID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_goroutine")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Info("This is a test message with goroutine ID")

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

	content2, err := os.ReadFile(filepath.Join(tempDir+"_new", FileInfo))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent2 := string(content2)
	if !strings.Contains(logContent2, "goroutine") {
		t.Errorf("Log file should contain goroutine ID, but it doesn't. Content: %s", logContent2)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_level_filtering")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Debug("This is a debug message")
	Info("This is an info message")
	Warn("This is a warning message")
	Error("This is an error message")

	logFilePath := filepath.Join(tempDir, "app.log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file %s: %v", logFilePath, err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "This is a warning message") {
		t.Errorf("Log file should contain the warning message, but it doesn't. Content: %s", logContent)
	}
	if !strings.Contains(logContent, "This is an error message") {
		t.Errorf("Log file should contain the error message, but it doesn't. Content: %s", logContent)
	}

	if strings.Contains(logContent, "This is a debug message") {
		t.Errorf("Log file should NOT contain the debug message, but it does. Content: %s", logContent)
	}
	if strings.Contains(logContent, "This is an info message") {
		t.Errorf("Log file should NOT contain the info message, but it does. Content: %s", logContent)
	}
}

func TestDebugf(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_debugf")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
log_level: "debug"
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	component := "database"
	status := "connected"
	Debugf("Component %s status: %s", component, status)

	expectedMessage := "Component database status: connected"
	checkLogFile(t, filepath.Join(tempDir, FileDebug), "DEBUG", expectedMessage)
}

func TestMissingLogLevelDefaultsToInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_missing_level")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Debug("This is a debug message")
	Info("This is an info message")

	logFilePath := filepath.Join(tempDir, "app.log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file %s: %v", logFilePath, err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "This is an info message") {
		t.Errorf("Log file should contain the info message, but it doesn't. Content: %s", logContent)
	}

	if strings.Contains(logContent, "This is a debug message") {
		t.Errorf("Log file should NOT contain the debug message, but it does. Content: %s", logContent)
	}
}

func TestFlush(t *testing.T) {
	err := Flush()
	if err != nil {
		t.Logf("Flush on default logger returned error (expected on some envs): %v", err)
	}

	tempDir, err := os.MkdirTemp("", "glog_test_flush")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `
encoder: console
path: ""
directory: ""
log_stdout: false
segment:
  max_size: 1
  max_age: 1
  max_backups: 1
  compress: false
`
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Info("Message to flush")

	if err := Flush(); err != nil {
		t.Errorf("Flush failed on file logger: %v", err)
	}
}

// --- New tests for improved coverage ---

func TestInfof(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_infof")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := writeConfig(t, tempDir, baseConsoleConfig)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Infof("User %s logged in at %s", "alice", "10:00")

	checkLogFile(t, filepath.Join(tempDir, FileInfo), "INFO", "User alice logged in at 10:00")
}

func TestWarnf(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_warnf")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := writeConfig(t, tempDir, baseConsoleConfig)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Warnf("Disk usage at %d%%", 95)

	checkLogFile(t, filepath.Join(tempDir, FileWarn), "WARN", "Disk usage at 95%")
}

func TestErrorAndErrorf(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := writeConfig(t, tempDir, baseConsoleConfig)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Error("database connection failed")
	Errorf("failed to connect to %s: %s", "localhost:5432", "timeout")

	errorLogPath := filepath.Join(tempDir, FileError)
	content, err := os.ReadFile(errorLogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "database connection failed") {
		t.Errorf("Expected error message not found. Content: %s", logContent)
	}
	if !strings.Contains(logContent, "failed to connect to localhost:5432: timeout") {
		t.Errorf("Expected formatted error message not found. Content: %s", logContent)
	}
}

func TestPanicFunction(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_panic")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := writeConfig(t, tempDir, baseConsoleConfig)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Panic should have been called, but recover returned nil")
		}

		// Verify the panic message was logged
		panicLogPath := filepath.Join(tempDir, FilePanic)
		content, err := os.ReadFile(panicLogPath)
		if err != nil {
			t.Fatalf("Failed to read panic log file: %v", err)
		}
		logContent := string(content)
		if !strings.Contains(logContent, "something went wrong") {
			t.Errorf("Panic log should contain the message. Content: %s", logContent)
		}
	}()

	Panic("something went wrong")
}

func TestGetGoroutineID(t *testing.T) {
	id := getGoroutineID()
	if id == "" || id == "unknown" {
		t.Errorf("Expected a valid goroutine ID, got: %q", id)
	}

	// ID should be a numeric string
	for _, c := range id {
		if c < '0' || c > '9' {
			t.Errorf("Goroutine ID should be numeric, got: %q", id)
			break
		}
	}
}

func TestGetGoroutineIDConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	ids := make([]string, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ids[idx] = getGoroutineID()
		}(i)
	}
	wg.Wait()

	// All IDs should be non-empty and non-"unknown"
	for i, id := range ids {
		if id == "" || id == "unknown" {
			t.Errorf("Goroutine %d returned invalid ID: %q", i, id)
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"DEBUG", "debug"},
		{"info", "info"},
		{"INFO", "info"},
		{"warn", "warn"},
		{"warning", "warn"},
		{"WARN", "warn"},
		{"error", "error"},
		{"ERROR", "error"},
		{"panic", "panic"},
		{"fatal", "fatal"},
		{"", "info"},        // default
		{"unknown", "info"}, // default
		{"INVALID", "info"}, // default
	}

	for _, tt := range tests {
		t.Run("level_"+tt.input, func(t *testing.T) {
			level := parseLogLevel(tt.input)
			if level.String() != tt.expected {
				t.Errorf("parseLogLevel(%q) = %s, want %s", tt.input, level.String(), tt.expected)
			}
		})
	}
}

func TestHighPerformanceMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_high_perf")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `
encoder: json
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
high_performance: true
log_level: "warn"
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Debug and Info should be filtered out in high performance mode with warn level
	Debug("debug in high perf")
	Info("info in high perf")
	Warn("warn in high perf")
	Error("error in high perf")

	logFilePath := filepath.Join(tempDir, "app.log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	if strings.Contains(logContent, "debug in high perf") {
		t.Errorf("High performance mode with warn level should NOT contain debug messages")
	}
	if strings.Contains(logContent, "info in high perf") {
		t.Errorf("High performance mode with warn level should NOT contain info messages")
	}
	if !strings.Contains(logContent, "warn in high perf") {
		t.Errorf("High performance mode with warn level should contain warn messages. Content: %s", logContent)
	}
	if !strings.Contains(logContent, "error in high perf") {
		t.Errorf("High performance mode with warn level should contain error messages. Content: %s", logContent)
	}
}

func TestHighPerformanceModeDefaultLevel(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_high_perf_default")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
encode_level: Capital
log_stdout: false
high_performance: true
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Info("info message in default high perf")

	logFilePath := filepath.Join(tempDir, "app.log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "info message in default high perf") {
		t.Errorf("High performance mode with default level should contain info messages. Content: %s", logContent)
	}
}

func TestShowLine(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_show_line")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `
encoder: console
path: ""
directory: ""
show_line: true
encode_level: Capital
log_stdout: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Info("message with caller info")

	content, err := os.ReadFile(filepath.Join(tempDir, FileInfo))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	// Should contain caller info (file:line)
	if !strings.Contains(logContent, ".go:") {
		t.Errorf("Log should contain caller info (.go:), but it doesn't. Content: %s", logContent)
	}
}

func TestEncoderJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_json")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Info("json encoded message")

	content, err := os.ReadFile(filepath.Join(tempDir, FileInfo))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	// JSON format should contain key-value pairs
	if !strings.Contains(logContent, `"level":"INFO"`) {
		t.Errorf("JSON log should contain level key. Content: %s", logContent)
	}
	if !strings.Contains(logContent, `"message":"json encoded message"`) {
		t.Errorf("JSON log should contain message key. Content: %s", logContent)
	}
}

func TestEncodeLevelVariants(t *testing.T) {
	tests := []struct {
		name        string
		encodeLevel string
	}{
		{"Lowercase", "Lowercase"},
		{"LowercaseColor", "LowercaseColor"},
		{"Capital", "Capital"},
		{"CapitalColor", "CapitalColor"},
		{"Default", "UnknownEncoder"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "glog_test_encode_level")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configContent := `
encoder: console
path: ""
directory: ""
show_line: false
encode_level: "` + tt.encodeLevel + `"
log_stdout: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
			configPath := writeConfig(t, tempDir, configContent)

			if err := Init(configPath, tempDir); err != nil {
				t.Fatalf("Failed to initialize logger with encode_level %s: %v", tt.encodeLevel, err)
			}

			Info("encode level test message")

			content, err := os.ReadFile(filepath.Join(tempDir, FileInfo))
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}

			logContent := string(content)
			if !strings.Contains(logContent, "encode level test message") {
				t.Errorf("Log should contain the test message. Content: %s", logContent)
			}
		})
	}
}

func TestGoroutineIDWithFormattedFunctions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_goroutine_fmt")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `
encoder: console
path: ""
directory: ""
show_line: false
show_goroutine: true
encode_level: Capital
log_stdout: false
log_level: "debug"
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Debugf("debug msg %d", 1)
	Infof("info msg %d", 2)
	Warnf("warn msg %d", 3)
	Errorf("error msg %d", 4)
	Printf("printf msg %d", 5)

	// Verify goroutine IDs appear in each log file
	for _, pair := range []struct {
		file string
		msg  string
	}{
		{filepath.Join(tempDir, FileDebug), "debug msg 1"},
		{filepath.Join(tempDir, FileInfo), "info msg 2"},
		{filepath.Join(tempDir, FileWarn), "warn msg 3"},
		{filepath.Join(tempDir, FileError), "error msg 4"},
		{filepath.Join(tempDir, FileInfo), "printf msg 5"},
	} {
		content, err := os.ReadFile(pair.file)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", pair.file, err)
		}
		logContent := string(content)
		if !strings.Contains(logContent, "goroutine") {
			t.Errorf("Expected goroutine ID in %s. Content: %s", pair.file, logContent)
		}
		if !strings.Contains(logContent, pair.msg) {
			t.Errorf("Expected message %q in %s. Content: %s", pair.msg, pair.file, logContent)
		}
	}
}

func TestConcurrentLogging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_concurrent")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	configPath := writeConfig(t, tempDir, configContent)

	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			Infof("concurrent log from goroutine %d", idx)
		}(i)
	}
	wg.Wait()

	// Verify messages were logged (at least some should be present)
	content, err := os.ReadFile(filepath.Join(tempDir, "app.log"))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	logContent := string(content)
	if !strings.Contains(logContent, "concurrent log from goroutine") {
		t.Errorf("Expected concurrent log messages. Content: %s", logContent)
	}
}

func TestSetDefaults(t *testing.T) {
	t.Run("empty config gets defaults", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()

		if cfg.EncodeLevel != LowercaseLevelEncoder {
			t.Errorf("Expected EncodeLevel=%s, got %s", LowercaseLevelEncoder, cfg.EncodeLevel)
		}
		if cfg.StacktraceKey != "stacktrace" {
			t.Errorf("Expected StacktraceKey=stacktrace, got %s", cfg.StacktraceKey)
		}
	})

	t.Run("existing values preserved", func(t *testing.T) {
		cfg := &Config{
			EncodeLevel:   CapitalLevelEncoder,
			StacktraceKey: "custom_stack",
		}
		cfg.setDefaults()

		if cfg.EncodeLevel != CapitalLevelEncoder {
			t.Errorf("Expected EncodeLevel=%s, got %s", CapitalLevelEncoder, cfg.EncodeLevel)
		}
		if cfg.StacktraceKey != "custom_stack" {
			t.Errorf("Expected StacktraceKey=custom_stack, got %s", cfg.StacktraceKey)
		}
	})
}

func TestMkdir(t *testing.T) {
	t.Run("creates new directory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "glog_test_mkdir")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		newDir := filepath.Join(tempDir, "sub", "dir")
		if err := mkdir(newDir); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}

		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("Expected a directory")
		}
	})

	t.Run("existing directory is ok", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "glog_test_mkdir_existing")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Call mkdir on an existing directory should not error
		if err := mkdir(tempDir); err != nil {
			t.Fatalf("mkdir on existing dir failed: %v", err)
		}
	})
}

func TestYamlToStructInvalidYaml(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_invalid_yaml")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write invalid YAML
	invalidPath := filepath.Join(tempDir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	cfg := &Config{}
	err = yamlToStruct(invalidPath, cfg)
	if err == nil {
		t.Error("Expected error from invalid YAML, got nil")
	}
}

func TestCustomTimeEncoder(t *testing.T) {
	testTime, err := time.Parse("2006-01-02 15:04:05.000", "2024-06-15 10:30:45.123")
	if err != nil {
		t.Fatalf("Failed to parse test time: %v", err)
	}
	var result []string
	enc := &testArrayEncoder{values: &result}
	customTimeEncoder(testTime, enc)

	if len(result) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(result))
	}
	if !strings.Contains(result[0], "2024-06-15 10:30:45.123") {
		t.Errorf("Expected formatted time, got: %s", result[0])
	}
	if !strings.HasPrefix(result[0], "[") || !strings.HasSuffix(result[0], "]") {
		t.Errorf("Expected time wrapped in brackets, got: %s", result[0])
	}
}

func TestFlushWithNilState(t *testing.T) {
	// Temporarily store nil state and restore
	old := currentState.Load()
	defer currentState.Store(old)

	currentState.Store(&loggerState{logger: nil, showGoroutine: false})
	err := Flush()
	if err != nil {
		t.Errorf("Flush with nil logger should not error, got: %v", err)
	}
}

func TestLogFunctionsWithNilLogger(t *testing.T) {
	// Temporarily store nil logger state
	old := currentState.Load()
	defer currentState.Store(old)

	currentState.Store(&loggerState{logger: nil, showGoroutine: false})

	// These should not panic
	Debug("test")
	Debugf("test %s", "msg")
	Info("test")
	Infof("test %s", "msg")
	Warn("test")
	Warnf("test %s", "msg")
	Error("test")
	Errorf("test %s", "msg")
	Printf("test %s", "msg")
}

func TestPathConfiguration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glog_test_path")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Use path + directory combination
	configContent := `
encoder: console
path: "` + tempDir + `"
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
	configPath := writeConfig(t, tempDir, configContent)

	subDir := "/myapp"
	if err := Init(configPath, subDir); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Info("path test message")

	expectedPath := filepath.Join(tempDir+subDir, FileInfo)
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read log file at %s: %v", expectedPath, err)
	}

	if !strings.Contains(string(content), "path test message") {
		t.Errorf("Expected message in log file. Content: %s", string(content))
	}
}

// --- Test helpers ---

type testArrayEncoder struct {
	values *[]string
}

func (e *testArrayEncoder) AppendString(v string) {
	*e.values = append(*e.values, v)
}

// Implement remaining PrimitiveArrayEncoder methods as no-ops
func (e *testArrayEncoder) AppendBool(bool)             {}
func (e *testArrayEncoder) AppendByteString([]byte)     {}
func (e *testArrayEncoder) AppendComplex128(complex128) {}
func (e *testArrayEncoder) AppendComplex64(complex64)   {}
func (e *testArrayEncoder) AppendFloat64(float64)       {}
func (e *testArrayEncoder) AppendFloat32(float32)       {}
func (e *testArrayEncoder) AppendInt(int)               {}
func (e *testArrayEncoder) AppendInt64(int64)           {}
func (e *testArrayEncoder) AppendInt32(int32)           {}
func (e *testArrayEncoder) AppendInt16(int16)           {}
func (e *testArrayEncoder) AppendInt8(int8)             {}
func (e *testArrayEncoder) AppendUint(uint)             {}
func (e *testArrayEncoder) AppendUint64(uint64)         {}
func (e *testArrayEncoder) AppendUint32(uint32)         {}
func (e *testArrayEncoder) AppendUint16(uint16)         {}
func (e *testArrayEncoder) AppendUint8(uint8)           {}
func (e *testArrayEncoder) AppendUintptr(uintptr)       {}
