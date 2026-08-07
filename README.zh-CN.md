# glog

`glog` 是一个基于 [zap](https://github.com/uber-go/zap) 的 Go 日志库，提供文件分级、日志轮转、JSON/console 编码、Gin 中间件和全局/实例两种使用方式。

## 功能特性

- 支持 `Debug`、`Info`、`Warn`、`Error`、`Panic`、`Fatal` 等日志级别。
- 支持 `console` 和 `json` 输出格式。
- 支持基于 [lumberjack](https://github.com/natefinch/lumberjack) 的日志轮转。
- 支持默认全局 logger，无需初始化即可使用。
- 支持通过 `New()` 创建独立 logger，适合大型项目依赖注入。
- 支持 Gin 请求日志和 panic recovery 中间件。
- Gin 请求日志支持 `request_id` 和 `trace_id`，便于链路追踪。

## 安装

```bash
go get github.com/jackman0925/glog
```

如果使用 Gin 中间件，需要项目本身依赖 Gin：

```bash
go get github.com/gin-gonic/gin
```

## 基础用法

不初始化时，`glog` 会使用默认全局 logger：

```go
package main

import "github.com/jackman0925/glog"

func main() {
	glog.Info("hello")
	glog.Warnf("disk usage: %d%%", 90)
}
```

## 使用配置文件初始化全局 logger

```go
package main

import (
	"log"

	"github.com/jackman0925/glog"
)

func main() {
	if err := glog.Init("./logger.yaml", "/my-app"); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer glog.Flush()

	glog.Info("app started")
}
```

## 大型项目推荐用法

大型项目优先使用 `New()` 创建实例 logger，并通过依赖注入传递给业务模块。

```go
logger, err := glog.New("./logger.yaml", "/my-service")
if err != nil {
	return err
}
defer logger.Sync()

logger.Infow("create user", "user_id", 123)
```

注意：

- `glog.Info()`、`glog.Warn()` 等包级函数使用的是全局 logger。
- `New()` 返回独立 logger 实例，默认不会修改全局 logger。
- 如果业务模块采用实例 logger，就不要混用 `glog.xxx`，否则日志配置和字段可能不一致。
- 如果确实希望 `New()` 同时设置全局 logger，可以使用 `glog.New("./logger.yaml", "/my-service", true)`。

## Gin 中间件

`middleware/ginmw` 提供 Gin 请求日志和 panic recovery：

```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackman0925/glog"
	"github.com/jackman0925/glog/middleware/ginmw"
)

func main() {
	logger, err := glog.New("./logger.yaml", "/gin-app")
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	r := gin.New()
	r.Use(
		ginmw.GinLoggerWithConfig(logger, ginmw.LoggerConfig{
			SkipSuccessfulRequests: true,
		}),
		ginmw.GinRecovery(logger, true),
	)
}
```

`GinLogger` 会记录：

- `method`
- `path`
- `status`
- `latency_ms`
- `client_ip`
- `user_agent`
- `request_id`
- `trace_id`
- `errors`

日志级别规则：

- `status >= 500` 记录为 `Error`
- `status >= 400` 记录为 `Warn`
- 其他状态记录为 `Info`

`trace_id` 默认从 `X-Trace-ID` 读取；如果没有该 header，会尝试从 W3C `traceparent` 中解析。也可以通过 `LoggerConfig.TraceIDHeader` 改成自定义 header。

## SkipPaths 兼容行为

默认情况下，`SkipPaths` 保持旧行为：命中路径后不记录日志，不区分最终状态码。

```go
ginmw.LoggerConfig{
	SkipPaths: []string{"/healthz"},
}
```

如果希望“只跳过成功请求，保留 4xx/5xx”，开启 `SkipSuccessfulPaths`：

```go
ginmw.LoggerConfig{
	SkipPaths:           []string{"/healthz", "/metrics"},
	SkipSuccessfulPaths: true,
}
```

此时：

- `/healthz` 返回 `200` 或 `302`：跳过日志。
- `/healthz` 返回 `400` 或 `500`：保留日志。

如果希望“全站只保留失败请求日志”，开启 `SkipSuccessfulRequests`：

```go
ginmw.LoggerConfig{
	SkipSuccessfulRequests: true,
}
```

此时：

- 任意路径返回 `200` 或 `302`：跳过日志。
- 任意路径返回 `400` 或 `500`：保留日志。

`SkipPaths`、`SkipSuccessfulPaths`、`SkipSuccessfulRequests` 都是 Gin 中间件的 Go 代码配置，不会改变 `logger.yaml` 的结构。

## JSON 输出

生产环境建议使用 JSON 输出，便于日志采集系统按字段查询：

```yaml
encoder: json
path: "/var/log"
directory: "/my-service"
show_line: false
show_goroutine: false
encode_level: Capital
stacktrace_key: stacktrace
log_stdout: false
high_performance: true
separate_levels: false
log_level: info
segment:
  max_size: 100
  max_age: 7
  max_backups: 10
  compress: true
```

如果是旧版本用户继续使用 `encoder: console`，仍然保持兼容；只有需要结构化采集和按 `trace_id` 查询时，才需要改成 `encoder: json`。

## 示例

- `examples/main.go`：基础用法。
- `examples/gin_demo/main.go`：Gin 中间件示例，复用 `examples/logger.yaml`。
- `docs/GIN_MIDDLEWARE.md`：Gin 中间件详细说明。
- `docs/PRODUCTION_INTEGRATION_CHECKLIST.md`：生产集成清单。

运行 Gin 示例：

```bash
go run ./examples/gin_demo
```

可测试：

- `GET /hello`
- `GET /healthz`
- `GET /healthz?fail=true`
- `GET /bad-request`
- `GET /panic`
