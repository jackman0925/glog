# Gin Middleware Guide

This project provides an optional Gin middleware subpackage:

- `github.com/jackman0925/glog/middleware/ginmw`

## APIs

- `GinLogger(log *zap.SugaredLogger) gin.HandlerFunc`
- `GinLoggerWithConfig(log *zap.SugaredLogger, cfg LoggerConfig) gin.HandlerFunc`
- `GinRecovery(log *zap.SugaredLogger, includeStack bool) gin.HandlerFunc`

## LoggerConfig

- `SkipPaths []string`: exact paths to skip logging.
- `RequestIDHeader string`: request ID header key (default: `X-Request-ID`).
- `Message string`: request log message (default: `gin request`).

## Structured Fields

`GinLogger` writes:

- `method`
- `path` (with query string)
- `status`
- `latency_ms`
- `client_ip`
- `user_agent`
- `request_id` (when header is present)
- `errors` (when Gin context has errors)

## Level Mapping

- `status >= 500` => `Error`
- `status >= 400` => `Warn`
- others => `Info`

## Production Recommendation

For large projects, prefer instance logger injection:

1. Create logger with `glog.New(...)`.
2. Pass logger into middleware constructors.
3. Avoid mixing `glog.xxx` global logging and instance logging in business modules.

## Demo

Run demo app:

```bash
go run ./examples/gin_demo
```

Then test endpoints:

- `GET /hello`
- `GET /healthz` (skipped by logger)
- `GET /panic` (recovery + panic log)
