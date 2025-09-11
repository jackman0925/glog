package glog

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBackwardCompatibilityWithOldConfig 测试使用旧配置文件的向后兼容性
func TestBackwardCompatibilityWithOldConfig(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "glog_compat_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建一个没有新选项的旧配置文件
	oldConfigContent := `
encoder: console
path: ""
directory: ""
show_line: true
show_goroutine: true
encode_level: CapitalColor
log_stdout: true
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "old_logger.yaml")
	if err := os.WriteFile(configPath, []byte(oldConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 使用旧配置初始化日志器应该成功（不检查文件路径，只验证不报错）
	if err := Init(configPath, tempDir); err != nil {
		t.Fatalf("Failed to initialize logger with old config: %v", err)
	}

	// 记录一些日志确保功能正常
	Info("This is a test message with old config")
	Warn("This is a warning message with old config")
	
	// 如果能执行到这里而没有 panic 或错误，说明兼容性良好
}

// TestBackwardCompatibilityWithNewConfig 测试使用新配置文件的功能
func TestBackwardCompatibilityWithNewConfig(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "glog_new_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建一个包含新选项的配置文件
	newConfigContent := `
encoder: json
path: ""
directory: ""
show_line: false
show_goroutine: false
encode_level: Capital
log_stdout: false
high_performance: true
separate_levels: false
segment:
  max_size: 10
  max_age: 7
  max_backups: 10
  compress: false
`
	configPath := filepath.Join(tempDir, "new_logger.yaml")
	if err := os.WriteFile(configPath, []byte(newConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 使用新配置初始化日志器应该成功
	logger, err := New(configPath, tempDir)
	if err != nil {
		t.Fatalf("Failed to create logger with new config: %v", err)
	}

	// 记录一些日志确保功能正常
	logger.Info("This is a test message with new config")
	logger.Warn("This is a warning message with new config")
	
	// 如果能执行到这里而没有 panic 或错误，说明新功能正常
}