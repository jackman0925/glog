# glog

A simple and easy-to-use logging library for Go, built on top of [zap](https://github.com/uber-go/zap).

## Features

*   Multiple log levels (Debug, Info, Warn, Error, Panic).
*   Log rotation with [lumberjack](https://github.com/natefinch/lumberjack).
*   Configurable output format (console or json).
*   Out-of-the-box default logger, no configuration required.
*   Easy to initialize and customize.

## Installation

```bash
go get github.com/jackman0925/glog
```

## Usage

### Default Logger

By default, `glog` provides a logger that writes to the console. You can use it without any initialization.

```go
package main

import (
	"github.com/jackman0925/glog"
)

func main() {
	glog.Info("This is an info message.")
	glog.Warnf("This is a %s message.", "warning")
}
```

### Custom Logger

You can customize the logger by creating a `logger.yaml` file and initializing `glog` with it.

1.  Create a `logger.yaml` file (you can use `logger.yaml.example` as a template).

2.  Initialize `glog` in your application:

```go
package main

import (
	"log"
	"github.com/jackman0925/glog"
)

func main() {
	if err := glog.Init("./logger.yaml", "my-app"); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	glog.Info("This is an info message from the custom logger.")
}
```

### Create a new logger instance

If you need a separate logger instance, you can use the `New` function.

```go
package main

import (
	"log"
	"github.com/jackman0925/glog"
)

func main() {
	logger, err := glog.New("./logger.yaml", "my-other-app")
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("This is a message from a new logger instance.")
}

```

## Configuration

The following options are available in the `logger.yaml` file:

*   `encoder`: `console` or `json`.
*   `path`: Log file path.
*   `directory`: Log file directory.
*   `show_line`: Show file and line number (`true` or `false`).
*   `show_goroutine`: Show goroutine ID (`true` or `false`).
*   `encode_level`: `Lowercase`, `LowercaseColor`, `Capital`, `CapitalColor`.
*   `stacktrace_key`: Stacktrace key.
*   `log_stdout`: Log to stdout (`true` or `false`).
*   `high_performance`: Enable high performance mode (`true` or `false`). When enabled, reduces features for better performance.
*   `separate_levels`: Separate log levels to different files (`true` or `false`). When disabled, logs all levels to a single file for better performance.
*   `segment`:
    *   `max_size`: Max size of log file before rotation (MB).
    *   `max_age`: Max age of log file before rotation (days).
    *   `max_backups`: Max number of backups.
    *   `compress`: Compress rotated log files (`true` or `false`).