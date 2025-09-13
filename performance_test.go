package glog

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func BenchmarkDefaultLogger(b *testing.B) {
	// Reset to default logger
	logger, _ := zap.NewDevelopment()
	xLog = logger.Sugar()
	showGoroutine = false

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("This is a test log message")
		}
	})
}

func BenchmarkGoroutineIDLogger(b *testing.B) {
	// Reset to default logger
	logger, _ := zap.NewDevelopment()
	xLog = logger.Sugar()
	showGoroutine = true

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("This is a test log message with goroutine ID")
		}
	})
}

func BenchmarkCustomLogger(b *testing.B) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file
	configContent := `
encoder: json
path: ""
directory: ""
show_line: false
show_goroutine: false
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
		b.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the logger
	if err := Init(configPath, tempDir); err != nil {
		b.Fatalf("Failed to initialize logger: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("This is a test log message")
		}
	})
}

func BenchmarkCustomLoggerWithGoroutineID(b *testing.B) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_bench_goroutine")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a logger.yaml file with goroutine ID enabled
	configContent := `
encoder: json
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
		b.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the logger
	if err := Init(configPath, tempDir); err != nil {
		b.Fatalf("Failed to initialize logger: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("This is a test log message with goroutine ID")
		}
	})
}

func BenchmarkGetGoroutineID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getGoroutineID()
	}
}

// BenchmarkProductionStressTest simulates a production-like logging scenario under high concurrency.
func BenchmarkProductionStressTest(b *testing.B) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "glog_bench_prod")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// A production-like configuration: single file, json encoder, info level.
	configContent := `
encoder: json
path: ""
directory: ""
show_line: false
show_goroutine: false
encode_level: Capital
log_stdout: false
separate_levels: false
log_level: "info"
segment:
  max_size: 100
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "logger.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the logger
	if err := Init(configPath, tempDir); err != nil {
		b.Fatalf("Failed to initialize logger: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// In a real-world scenario, logs often have arguments.
			Info("A typical production log message", "user_id", 12345, "request_id", "abc-xyz-123")
		}
	})
}

func BenchmarkZapDirect(b *testing.B) {
	// Create a zap logger directly for comparison
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sugar.Info("This is a test log message")
		}
	})
}