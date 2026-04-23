# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.3] - 2026-04-23
### Fixed
- **Instance Logger 行号偏移修复**: 修复了通过 `New` 或 `NewLogger` 创建的实例 logger 在开启 `show_line` 时行号显示不准确的问题（之前由于硬编码的 `AddCallerSkip(1)` 导致行号向上偏移一层，显示为调用方的调用方）。
- **Goroutine ID 内存分配优化**: 修复了 `getGoroutineID()` 中由于 64 字节 buffer 极易溢出导致 `sync.Pool` 被绕过、转而重新分配 256 字节内存的问题。优化后完全命中 Pool，单次获取 ID 的分配降至 1 allocs/op。

### Changed
- **全局函数性能提升**: 在 `glog.Debug` 等全局封装函数中增加了前置级别检查。当日志级别被禁用时直接返回，避免了在禁用级别下依然执行昂贵的 `runtime.Stack` 和 Logger 克隆操作。禁用级别的日志调用性能提升约 80 倍。
- **Caller Skip 逻辑重构**: 将 `AddCallerSkip(1)` 从底层 `newLogger` 移至全局初始化逻辑中，确保全局包装函数与独立实例 logger 均能获得正确的行号信息。

### Added
- 新增 `fix_verify_test.go` 专门验证行号修复结果以及禁用级别下的零分配性能。

## [1.1.2] - 2026-04-22
### Changed
- `New(cfgPath, directory string, setGlobal ...bool)` 新增可选参数 `setGlobal`（默认 `false`）：
  - `setGlobal` 为 `false`（默认）：行为与原来完全相同，仅返回独立的 logger 句柄，不影响全局状态。
  - `setGlobal` 为 `true`：在返回句柄的同时，将新建的 logger 设置为全局默认 logger，之后可直接通过 `glog.Info()`、`glog.Error()` 等包级函数使用，无需传递句柄。
  - 该设计兼顾两种使用场景，同时保持向后兼容（无需修改任何现有调用代码）。

## [1.1.1] - 2026-03-08
### Added
- Optional Gin middleware subpackage `middleware/ginmw` with `GinLogger`, `GinLoggerWithConfig`, and `GinRecovery`.
- Unit tests for Gin middleware covering request logging levels, skip paths, panic recovery, and nil logger safety.
- Gin demo at `examples/gin_demo/main.go` and middleware guide `docs/GIN_MIDDLEWARE.md`.

## [1.1.0] - 2026-02-24
### Fixed
- **数据竞争修复**: `getGoroutineID()` 中的 `goroutineCacheCounter` 全局计数器存在并发读写数据竞争，移除了无效的缓存机制（基于完整栈内容的缓存 key 几乎不会命中），改为直接解析。
- **数据竞争修复**: 全局 `xLog` 和 `showGoroutine` 变量在并发读写时存在竞争条件，改用 `atomic.Value` 包装为 `loggerState` 结构体，确保线程安全。
- **资源泄露修复**: `panicRedirect` 中 `os.OpenFile` 打开的文件从未关闭，新增文件句柄跟踪与关闭机制。
- **高性能模式 Bug**: `newHighPerformanceLogger` 硬编码 `zapcore.DebugLevel`，忽略了用户配置的 `log_level`，已修复为使用 `parseLogLevel(cfg.LogLevel)`。
- **高性能模式行为一致性修复**: `newHighPerformanceLogger` 现在也会执行 `panicRedirect`，确保 `stderr` 输出与普通模式一致写入 `stderr.log`。
- **冗余代码清理**: `mkdir` 函数中 `os.MkdirAll` 后多余的 `os.Chmod` 调用已移除；`setDefaults()` 中无效的空代码块已清理。

### Changed
- `newLogger` 中移除了初始化时的 `sl.Sync()` 空调用，减少无意义的 I/O 操作。
- `getGoroutineID()` 改为使用 `sync.Pool` 复用临时 buffer，并增加截断场景处理，减少高频日志场景下的内存分配。

### Added
- 新增大量单元测试覆盖：`Infof`、`Warnf`、`Errorf`、`Panic`（带 recovery）、goroutine ID 并发测试、`parseLogLevel` 全面测试、高性能模式日志级别测试、JSON 编码器测试、`showLine` 测试、`customTimeEncoder` 测试、nil logger 安全测试等。
- 新增 `parseGoroutineID` 边界测试（非法前缀、非数字 ID、空 ID 等）和高性能模式 `stderr` 重定向测试。
- 测试覆盖率从 73.9% 提升至 86.7%。

## [1.0.3] - 2026-01-30
### Added
- `Flush` function to ensure all buffered logs are written to disk.

## [1.0.2] - 2025-11-20
### Added
- `Debugf` logging method that formats and logs a message at Debug level.

## [1.0.1] - 2025-10-01
### Added
- `Fatal` and `Fatalf` logging methods that log a message and then exit the application with a status code of 1.

## [1.0.0] - 2025-09-13
### Added
- Log level filtering via the `log_level` option in the configuration file (e.g., "info", "warn", "error").

### Fixed
- Ensured backward compatibility for configurations without the new `log_level` option by defaulting to the `info` level.
- Corrected an issue where existing tests failed due to the new `separate_levels` flag. The logger now correctly defaults to separating files by level (`separate_levels: true`) to maintain backward compatibility.

## [0.0.3] - 2025-09-11

### Added
- Performance optimization for default logger by using `zap.NewProduction()` instead of `zap.NewDevelopment()`
- Caching mechanism for goroutine ID retrieval to reduce memory allocations
- High performance mode configuration option (`high_performance`) for optimized logging
- Log level separation configuration option (`separate_levels`) to control file output strategy
- Backward compatibility tests to ensure seamless upgrades
- Performance benchmarks to monitor optimization effectiveness

### Changed
- Optimized `getGoroutineID()` function to use caching and reduce runtime overhead
- Improved default logger initialization for better out-of-the-box performance
- Enhanced configuration parsing with default value handling for new options
- Updated documentation to reflect new configuration options

### Fixed
- Reduced memory allocations in goroutine ID retrieval by approximately 50%
- Improved overall logging performance by ~27% in default mode
- Added cache cleanup mechanism to prevent memory leaks in long-running applications

## [0.0.2] - 2025-09-06

### Added
- `Printf` function for formatted logging similar to `fmt.Printf`
- `Logger` wrapper type that embeds `*zap.SugaredLogger` with additional methods
- `NewLogger` function to create Logger instances with Printf support
- `Logger.Printf` method for formatted logging on Logger instances
- Goroutine ID support similar to Java thread ID logging
- `show_goroutine` configuration option to enable/disable goroutine ID in logs
- `getGoroutineID()` function to extract goroutine ID from runtime stack
- Unit tests for both global `Printf` and `Logger.Printf` methods
- Unit test for goroutine ID functionality
- `examples/goroutine_demo.go` demonstrating concurrent logging with goroutine IDs

## [0.0.1] - 2025-09-01

### Added
- Initial version of the `glog` library.
- Default logger that writes to the console out-of-the-box.
- `Init` function to initialize a global logger from a YAML config file.
- `New` function to create a new logger instance from a YAML config file.
- Support for `console` and `json` encoders.
- Log rotation using `lumberjack`.
- `logger.yaml.example` as a configuration template.
- `examples/main.go` to demonstrate usage.
- Unit tests with 72% coverage.
- `go.mod` for dependency management.

### Changed
- Refactored the library to be more robust and modular.
- `Init` and `New` functions now return an error instead of panicking.
- Updated import paths to `github.com/jackman0925/glog`.

### Fixed
- Corrected caller reporting using `zap.AddCallerSkip(1)`.
- Used more secure file permissions (0755) when creating directories.
