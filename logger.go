package glog

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	Encoder         string  `yaml:"encoder"`
	Path            string  `yaml:"path"`
	Directory       string  `yaml:"directory"`
	ShowLine        bool    `yaml:"show_line"`
	ShowGoroutine   bool    `yaml:"show_goroutine"`
	EncodeLevel     string  `yaml:"encode_level"`
	StacktraceKey   string  `yaml:"stacktrace_key"`
	LogStdout       bool    `yaml:"log_stdout"`
	HighPerformance bool    `yaml:"high_performance"`
	SeparateLevels  bool    `yaml:"separate_levels"`
	LogLevel        string  `yaml:"log_level"`
	Segment         Segment `yaml:"segment"`
}

// setDefaults sets default values for config options
func (c *Config) setDefaults() {
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

// loggerState holds the logger and its associated configuration atomically.
type loggerState struct {
	logger        *zap.SugaredLogger
	showGoroutine bool
}

var (
	// currentState stores the current *loggerState atomically for safe concurrent access.
	currentState atomic.Value

	// goroutineIDBufPool reuses stack buffers to reduce per-log allocations.
	goroutineIDBufPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 64)
			return &buf
		},
	}

	// stderrFile tracks the file used for panic redirect so it can be closed on re-init.
	stderrFileMu sync.Mutex
	stderrFile   *os.File
)

// Logger wraps zap.SugaredLogger to provide additional methods
type Logger struct {
	*zap.SugaredLogger
}

func init() {
	// Default logger - use production mode for performance
	logger, _ := zap.NewProduction()
	currentState.Store(&loggerState{
		logger:        logger.Sugar(),
		showGoroutine: false,
	})
}

// getState returns the current loggerState safely.
func getState() *loggerState {
	if v := currentState.Load(); v != nil {
		return v.(*loggerState)
	}
	return nil
}

// getGoroutineID returns the current goroutine ID.
// It parses the goroutine ID from the runtime stack trace.
func getGoroutineID() string {
	bufPtr := goroutineIDBufPool.Get().(*[]byte)
	defer goroutineIDBufPool.Put(bufPtr)

	buf := *bufPtr
	n := runtime.Stack(buf, false)

	// runtime.Stack may truncate output when n == len(buf), so retry with a bigger buffer.
	if n == len(buf) {
		biggerBuf := make([]byte, 256)
		n = runtime.Stack(biggerBuf, false)
		return parseGoroutineID(biggerBuf[:n])
	}

	return parseGoroutineID(buf[:n])
}

func parseGoroutineID(stack []byte) string {
	// Stack prefix format: "goroutine <id> ["
	const prefix = "goroutine "
	if !bytes.HasPrefix(stack, []byte(prefix)) {
		return "unknown"
	}

	idStart := len(prefix)
	idEnd := bytes.IndexByte(stack[idStart:], ' ')
	if idEnd == -1 {
		return "unknown"
	}

	id := stack[idStart : idStart+idEnd]
	if len(id) == 0 {
		return "unknown"
	}

	for _, b := range id {
		if b < '0' || b > '9' {
			return "unknown"
		}
	}

	return string(id)
}

// Init initializes a new logger with the given config file path and directory.
// This will replace the default logger.
func Init(cfgPath string, directory string) error {
	cfg := &Config{SeparateLevels: true}
	if err := yamlToStruct(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Directory = directory
	cfg.setDefaults()

	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}
	currentState.Store(&loggerState{
		logger:        logger,
		showGoroutine: cfg.ShowGoroutine,
	})
	return nil
}

// New creates a new logger with the given config file path and directory.
func New(cfgPath string, directory string) (*zap.SugaredLogger, error) {
	cfg := &Config{SeparateLevels: true}
	if err := yamlToStruct(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Directory = directory
	cfg.setDefaults()
	return newLogger(cfg)
}

// NewLogger creates a new Logger instance with the given config file path and directory.
// This returns a Logger wrapper that supports Printf method.
func NewLogger(cfgPath string, directory string) (*Logger, error) {
	cfg := &Config{SeparateLevels: true}
	if err := yamlToStruct(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Directory = directory
	cfg.setDefaults()

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
	// If high performance mode is enabled, use optimized config
	if cfg.HighPerformance {
		return newHighPerformanceLogger(cfg)
	}

	// Parse log level
	logLevel := parseLogLevel(cfg.LogLevel)

	path := cfg.Path + cfg.Directory
	if err := mkdir(path); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Build cores based on config
	var cores []zapcore.Core
	if cfg.SeparateLevels {
		// Separate log levels to different files (default behavior, backward compatible)
		debugLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.DebugLevel && logLevel <= level
		})
		infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.InfoLevel && logLevel <= level
		})
		warnLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.WarnLevel && logLevel <= level
		})
		errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.ErrorLevel && logLevel <= level
		})
		panicLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= zap.DPanicLevel && logLevel <= level
		})

		cores = []zapcore.Core{
			getEncoderCore(path+FileDebug, debugLevel, cfg),
			getEncoderCore(path+FileInfo, infoLevel, cfg),
			getEncoderCore(path+FileWarn, warnLevel, cfg),
			getEncoderCore(path+FileError, errorLevel, cfg),
			getEncoderCore(path+FilePanic, panicLevel, cfg),
		}
	} else {
		// Use a single core writing all logs to one file
		writer := getWriteSyncer(path+"/app.log", cfg)
		core := zapcore.NewCore(getEncoder(cfg), writer, logLevel)
		cores = []zapcore.Core{core}
	}

	logger := zap.New(zapcore.NewTee(cores...))

	if cfg.ShowLine {
		logger = logger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))
	}

	sl := logger.Sugar()

	panicRedirect(path + FileStderr)
	return sl, nil
}

// newHighPerformanceLogger creates a logger optimized for performance
func newHighPerformanceLogger(cfg *Config) (*zap.SugaredLogger, error) {
	path := cfg.Path + cfg.Directory
	if err := mkdir(path); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Respect configured log level instead of hardcoding DebugLevel
	logLevel := parseLogLevel(cfg.LogLevel)

	// Use a single core writing all logs to one file
	writer := getWriteSyncer(path+"/app.log", cfg)
	core := zapcore.NewCore(getEncoder(cfg), writer, logLevel)
	logger := zap.New(core)

	// High performance mode disables some features:
	// - No caller info for better performance
	// - No explicit Sync() call to reduce overhead

	sl := logger.Sugar()
	panicRedirect(path + FileStderr)
	return sl, nil
}

func mkdir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, 0755)
		}
		return err
	}
	return nil
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
		LocalTime:  true,
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

// parseLogLevel parses the log level from string
func parseLogLevel(levelStr string) zapcore.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		// Default to info level
		return zap.InfoLevel
	}
}

func Debug(args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Debug(args...)
		} else {
			s.logger.Debug(args...)
		}
	}
}

func Debugf(template string, args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Debugf(template, args...)
		} else {
			s.logger.Debugf(template, args...)
		}
	}
}

func Info(args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Info(args...)
		} else {
			s.logger.Info(args...)
		}
	}
}

func Infof(template string, args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Infof(template, args...)
		} else {
			s.logger.Infof(template, args...)
		}
	}
}

func Warn(args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Warn(args...)
		} else {
			s.logger.Warn(args...)
		}
	}
}

func Warnf(format string, args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Warnf(format, args...)
		} else {
			s.logger.Warnf(format, args...)
		}
	}
}

func Error(args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Error(args...)
		} else {
			s.logger.Error(args...)
		}
	}
}

func Errorf(template string, args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Errorf(template, args...)
		} else {
			s.logger.Errorf(template, args...)
		}
	}
}

func Fatal(args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Fatal(args...)
		} else {
			s.logger.Fatal(args...)
		}
	}
}

func Fatalf(template string, args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Fatalf(template, args...)
		} else {
			s.logger.Fatalf(template, args...)
		}
	}
}

func Panic(args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Panic(args...)
		} else {
			s.logger.Panic(args...)
		}
	}
}

func Printf(template string, args ...interface{}) {
	if s := getState(); s != nil && s.logger != nil {
		if s.showGoroutine {
			s.logger.With("goroutine", getGoroutineID()).Infof(template, args...)
		} else {
			s.logger.Infof(template, args...)
		}
	}
}

// Printf formats and logs a message at Info level
func (l *Logger) Printf(template string, args ...interface{}) {
	l.Infof(template, args...)
}

// Flush flushes any buffered log entries.
func Flush() error {
	if s := getState(); s != nil && s.logger != nil {
		return s.logger.Sync()
	}
	return nil
}

// panicRedirect redirects panics to a file.
func panicRedirect(logFile string) {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		if s := getState(); s != nil && s.logger != nil {
			s.logger.Errorf("Failed to redirect panic output: %v", err)
		}
		return
	}

	// Close previous stderr redirect file if any
	stderrFileMu.Lock()
	if stderrFile != nil {
		stderrFile.Close()
	}
	stderrFile = file
	stderrFileMu.Unlock()

	os.Stderr = file
}
