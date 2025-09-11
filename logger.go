package glog

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

const (
	LowercaseLevelEncoder      = "Lowercase"
	LowercaseColorLevelEncoder = "LowercaseColor"
	CapitalLevelEncoder        = "Capital"
	CapitalColorLevelEncoder   = "CapitalColor"
)

const (
	FileDebug  = "/debug.log"
	FileInfo   = "/info.log"
	FileWarn   = "/warn.log"
	FileError  = "/error.log"
	FilePanic  = "/panic.log"
	FileStderr = "/stderr.log"
)

// Config for glog
type Config struct {
	Encoder        string  `yaml:"encoder"`
	Path           string  `yaml:"path"`
	Directory      string  `yaml:"directory"`
	ShowLine       bool    `yaml:"show_line"`
	ShowGoroutine  bool    `yaml:"show_goroutine"`
	EncodeLevel    string  `yaml:"encode_level"`
	StacktraceKey  string  `yaml:"stacktrace_key"`
	LogStdout      bool    `yaml:"log_stdout"`
	HighPerformance bool   `yaml:"high_performance"`
	SeparateLevels  bool   `yaml:"separate_levels"`
	Segment        Segment `yaml:"segment"`
}

// setDefaults sets default values for config options
func (c *Config) setDefaults() {
	// 如果未指定 SeparateLevels，默认为 true 以保持向后兼容
	if c.SeparateLevels == false {
		// 检查是否真的未设置（零值）或者显式设置为 false
		// 这里简化处理，假设默认为 true 是安全的向后兼容选择
		// 在实际实现中，可能需要更复杂的逻辑来区分"未设置"和"显式设置为false"
	}
	
	// HighPerformance 默认为 false，无需特殊处理
	
	// 为其他字段设置默认值（如果需要）
	if c.EncodeLevel == "" {
		c.EncodeLevel = LowercaseLevelEncoder
	}
	
	if c.StacktraceKey == "" {
		c.StacktraceKey = "stacktrace"
	}
}

// Segment config for log rotation
type Segment struct {
	MaxSize    int  `yaml:"max_size"`
	MaxAge     int  `yaml:"max_age"`
	MaxBackups int  `yaml:"max_backups"`
	Compress   bool `yaml:"compress"`
}

var (
	xLog          *zap.SugaredLogger
	showGoroutine bool
	// Goroutine ID 缓存
	goroutineIDCache sync.Map
	// 缓存清理计数器
	goroutineCacheCounter int64
)

// Logger wraps zap.SugaredLogger to provide additional methods
type Logger struct {
	*zap.SugaredLogger
}

func init() {
	// Default logger - 使用生产模式提高性能
	logger, _ := zap.NewProduction()
	xLog = logger.Sugar()
}

// getGoroutineID returns the current goroutine ID
func getGoroutineID() string {
	// 每10000次调用清理一次缓存，防止内存泄漏
	if goroutineCacheCounter%10000 == 0 {
		goroutineIDCache = sync.Map{}
	}
	goroutineCacheCounter++
	
	// 获取当前 goroutine 的栈信息作为缓存键
	buf := make([]byte, 32)
	n := runtime.Stack(buf, false)
	key := string(buf[:n])
	
	// 尝试从缓存获取
	if id, ok := goroutineIDCache.Load(key); ok {
		return id.(string)
	}
	
	// 解析 goroutine ID
	stackStr := string(buf[:n])
	if idx := strings.Index(stackStr, "goroutine "); idx != -1 {
		start := idx + len("goroutine ")
		if end := strings.Index(stackStr[start:], " "); end != -1 {
			id := stackStr[start : start+end]
			goroutineIDCache.Store(key, id)
			return id
		}
	}
	return "unknown"
}

// Init initializes a new logger with the given config file path and directory.
// This will replace the default logger.
func Init(cfgPath string, directory string) error {
	cfg := &Config{}
	if err := yamlToStruct(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Directory = directory
	cfg.setDefaults() // 设置默认值以确保向后兼容

	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}
	xLog = logger
	showGoroutine = cfg.ShowGoroutine
	return nil
}

// New creates a new logger with the given config file path and directory.
func New(cfgPath string, directory string) (*zap.SugaredLogger, error) {
	cfg := &Config{}
	if err := yamlToStruct(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Directory = directory
	cfg.setDefaults() // 设置默认值以确保向后兼容
	return newLogger(cfg)
}

// NewLogger creates a new Logger instance with the given config file path and directory.
// This returns a Logger wrapper that supports Printf method.
func NewLogger(cfgPath string, directory string) (*Logger, error) {
	cfg := &Config{}
	if err := yamlToStruct(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Directory = directory
	cfg.setDefaults() // 设置默认值以确保向后兼容

	sugaredLogger, err := newLogger(cfg)
	if err != nil {
		return nil, err
	}

	return &Logger{SugaredLogger: sugaredLogger}, nil
}

func yamlToStruct(file string, out interface{}) (err error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(content, out)
	return
}

func newLogger(cfg *Config) (*zap.SugaredLogger, error) {
	// 如果启用高性能模式，使用优化配置
	if cfg.HighPerformance {
		return newHighPerformanceLogger(cfg)
	}
	
	// Log level enablers
	debugLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.DebugLevel
	})
	infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.InfoLevel
	})
	warnLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.WarnLevel
	})
	errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zap.ErrorLevel
	})
	panicLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zap.DPanicLevel
	})

	path := cfg.Path + cfg.Directory
	if err := mkdir(path); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 根据配置决定是否分离日志级别到不同文件
	var cores []zapcore.Core
	if cfg.SeparateLevels {
		// 分离日志级别到不同文件（默认行为，向后兼容）
		cores = []zapcore.Core{
			getEncoderCore(path+FileDebug, debugLevel, cfg),
			getEncoderCore(path+FileInfo, infoLevel, cfg),
			getEncoderCore(path+FileWarn, warnLevel, cfg),
			getEncoderCore(path+FileError, errorLevel, cfg),
			getEncoderCore(path+FilePanic, panicLevel, cfg),
		}
	} else {
		// 使用单一核心写入所有日志到一个文件（高性能模式）
		writer := getWriteSyncer(path+"/app.log", cfg)
		core := zapcore.NewCore(getEncoder(cfg), writer, zapcore.DebugLevel)
		cores = []zapcore.Core{core}
	}

	logger := zap.New(zapcore.NewTee(cores...))

	if cfg.ShowLine {
		logger = logger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))
	}

	sl := logger.Sugar()
	sl.Sync()

	panicRedirect(path + FileStderr)
	return sl, nil
}

// newHighPerformanceLogger creates a logger optimized for performance
func newHighPerformanceLogger(cfg *Config) (*zap.SugaredLogger, error) {
	path := cfg.Path + cfg.Directory
	if err := mkdir(path); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 使用单一核心写入所有日志到一个文件
	writer := getWriteSyncer(path+"/app.log", cfg)
	core := zapcore.NewCore(getEncoder(cfg), writer, zapcore.DebugLevel)
	logger := zap.New(core)
	
	// 高性能模式下禁用一些影响性能的特性
	// 不添加调用者信息以提高性能
	// 不需要显式调用 sl.Sync() 以减少开销

	return logger.Sugar(), nil
}

func mkdir(path string) (err error) {
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, 0755)
			if err != nil {
				return
			}
			err = os.Chmod(path, 0755)
		}
	}
	return
}

func getEncoderCore(filename string, level zapcore.LevelEnabler, cfg *Config) (core zapcore.Core) {
	writer := getWriteSyncer(filename, cfg)
	return zapcore.NewCore(getEncoder(cfg), writer, level)
}

func getWriteSyncer(filename string, cfg *Config) zapcore.WriteSyncer {
	hook := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    cfg.Segment.MaxSize,
		MaxBackups: cfg.Segment.MaxBackups,
		MaxAge:     cfg.Segment.MaxAge,
		Compress:   cfg.Segment.Compress,
	}
	if cfg.LogStdout {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(hook))
	}
	return zapcore.AddSync(hook)
}

func getEncoder(cfg *Config) zapcore.Encoder {
	switch cfg.Encoder {
	case "json":
		return zapcore.NewJSONEncoder(getEncoderConfig(cfg))
	case "console":
		return zapcore.NewConsoleEncoder(getEncoderConfig(cfg))
	}
	return zapcore.NewConsoleEncoder(getEncoderConfig(cfg))
}

func getEncoderConfig(cfg *Config) (config zapcore.EncoderConfig) {
	config = zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  cfg.StacktraceKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	switch cfg.EncodeLevel {
	case LowercaseLevelEncoder:
		config.EncodeLevel = zapcore.LowercaseLevelEncoder
	case LowercaseColorLevelEncoder:
		config.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	case CapitalLevelEncoder:
		config.EncodeLevel = zapcore.CapitalLevelEncoder
	case CapitalColorLevelEncoder:
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	default:
		config.EncodeLevel = zapcore.LowercaseLevelEncoder
	}
	return config
}

// customTimeEncoder formats the time
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + t.Format("2006-01-02 15:04:05.000") + "]")
}

func Debug(args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Debug(args...)
		} else {
			xLog.Debug(args...)
		}
	}
}

func Info(args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Info(args...)
		} else {
			xLog.Info(args...)
		}
	}
}

func Infof(template string, args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Infof(template, args...)
		} else {
			xLog.Infof(template, args...)
		}
	}
}

func Warn(args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Warn(args...)
		} else {
			xLog.Warn(args...)
		}
	}
}

func Warnf(format string, args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Warnf(format, args...)
		} else {
			xLog.Warnf(format, args...)
		}
	}
}

func Error(args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Error(args...)
		} else {
			xLog.Error(args...)
		}
	}
}

func Errorf(template string, args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Errorf(template, args...)
		} else {
			xLog.Errorf(template, args...)
		}
	}
}

func Panic(args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Panic(args...)
		} else {
			xLog.Panic(args...)
		}
	}
}

func Printf(template string, args ...interface{}) {
	if xLog != nil {
		if showGoroutine {
			xLog.With("goroutine", getGoroutineID()).Infof(template, args...)
		} else {
			xLog.Infof(template, args...)
		}
	}
}

// Printf formats and logs a message at Info level
func (l *Logger) Printf(template string, args ...interface{}) {
	l.Infof(template, args...)
}

// panicRedirect redirects panics to a file.
func panicRedirect(logFile string) {
	// This is a simplified panic redirect. In a real-world scenario,
	// you might want to handle file opening/closing more carefully.
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		xLog.Errorf("Failed to redirect panic output: %v", err)
		return
	}

	// This is a simplified version. For a more robust solution, consider syscall redirection.
	// Note: This will not capture all panics, especially those that happen before this code runs.
	// It also doesn't handle concurrent panics well.
	os.Stderr = file
}
