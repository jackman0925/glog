package glog

import (
	"fmt"
	"os"
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
	Encoder       string  `yaml:"encoder"`
	Path          string  `yaml:"path"`
	Directory     string  `yaml:"directory"`
	ShowLine      bool    `yaml:"show_line"`
	EncodeLevel   string  `yaml:"encode_level"`
	StacktraceKey string  `yaml:"stacktrace_key"`
	LogStdout     bool    `yaml:"log_stdout"`
	Segment       Segment `yaml:"segment"`
}

// Segment config for log rotation
type Segment struct {
	MaxSize    int  `yaml:"max_size"`
	MaxAge     int  `yaml:"max_age"`
	MaxBackups int  `yaml:"max_backups"`
	Compress   bool `yaml:"compress"`
}

var (
	xLog *zap.SugaredLogger
)

func init() {
	// Default logger
	logger, _ := zap.NewDevelopment()
	xLog = logger.Sugar()
}

// Init initializes a new logger with the given config file path and directory.
// This will replace the default logger.
func Init(cfgPath string, directory string) error {
	cfg := &Config{}
	if err := yamlToStruct(cfgPath, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Directory = directory

	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}
	xLog = logger
	return nil
}

// New creates a new logger with the given config file path and directory.
func New(cfgPath string, directory string) (*zap.SugaredLogger, error) {
	cfg := &Config{}
	if err := yamlToStruct(cfgPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	cfg.Directory = directory
	return newLogger(cfg)
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

	cores := [...]zapcore.Core{
		getEncoderCore(path+FileDebug, debugLevel, cfg),
		getEncoderCore(path+FileInfo, infoLevel, cfg),
		getEncoderCore(path+FileWarn, warnLevel, cfg),
		getEncoderCore(path+FileError, errorLevel, cfg),
		getEncoderCore(path+FilePanic, panicLevel, cfg),
	}

	logger := zap.New(zapcore.NewTee(cores[:]...))

	if cfg.ShowLine {
		logger = logger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))
	}

	sl := logger.Sugar()
	sl.Sync()

	panicRedirect(path + FileStderr)
	return sl, nil
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
	xLog.Debug(args...)
}

func Info(args ...interface{}) {
	xLog.Info(args...)
}

func Infof(template string, args ...interface{}) {
	xLog.Infof(template, args...)
}

func Warn(args ...interface{}) {
	xLog.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	xLog.Warnf(format, args...)
}

func Error(args ...interface{}) {
	xLog.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	xLog.Errorf(template, args...)
}

func Panic(args ...interface{}) {
	xLog.Panic(args...)
}

func Printf(template string, args ...interface{}) {
	xLog.Infof(template, args...)
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
