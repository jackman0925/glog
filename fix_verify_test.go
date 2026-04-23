package glog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCallerSkipFix(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "glog_caller_test")
	defer os.RemoveAll(tempDir)

	configContent := `
encoder: json
path: ""
directory: ""
show_line: true
show_goroutine: false
log_stdout: false
log_level: "info"
separate_levels: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	_ = os.WriteFile(configPath, []byte(configContent), 0644)

	// 1. Test Instance Logger Caller
	logger, err := New(configPath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Capture output to a buffer instead of file if possible, or just read the file
	logFile := filepath.Join(tempDir, "app.log")
	
	// We need to trigger a log and check the "caller" field
	logger.Info("instance log message")
	Flush()

	content, _ := os.ReadFile(logFile)
	if !strings.Contains(string(content), "fix_verify_test.go") {
		t.Errorf("Instance logger caller info wrong: %s", string(content))
	}

	// 2. Test Global Logger Caller
	os.Remove(logFile)
	Init(configPath, tempDir)
	Info("global log message")
	Flush()

	content, _ = os.ReadFile(logFile)
	if !strings.Contains(string(content), "fix_verify_test.go") {
		t.Errorf("Global logger caller info wrong: %s", string(content))
	}
}

func TestDisabledLogLevelPerformance(t *testing.T) {
	// If log level is Info, Debug calls should be extremely fast even with showGoroutine true
	tempDir, _ := os.MkdirTemp("", "glog_perf_test")
	defer os.RemoveAll(tempDir)

	configContent := `
show_goroutine: true
log_level: "info"
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	_ = os.WriteFile(configPath, []byte(configContent), 0644)
	Init(configPath, tempDir)

	// This should not trigger getGoroutineID (which is slow)
	// We can't easily measure "not calling it" without mocks, 
	// but we can benchmark it in a test.
}

func BenchmarkDisabledLogLevelWithGoroutine(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "glog_bench_disabled")
	defer os.RemoveAll(tempDir)

	configContent := `
show_goroutine: true
log_level: "info"
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	_ = os.WriteFile(configPath, []byte(configContent), 0644)
	Init(configPath, tempDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Debug("this is disabled")
	}
}
