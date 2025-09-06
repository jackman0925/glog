# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [released]

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
