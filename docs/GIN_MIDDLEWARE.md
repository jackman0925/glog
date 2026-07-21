# Gin Middleware Guide

This project provides an optional Gin middleware subpackage:

- `github.com/jackman0925/glog/middleware/ginmw`

## APIs

- `GinLogger(log *zap.SugaredLogger) gin.HandlerFunc`
- `GinLoggerWithConfig(log *zap.SugaredLogger, cfg LoggerConfig) gin.HandlerFunc`
- `GinRecovery(log *zap.SugaredLogger, includeStack bool) gin.HandlerFunc`

## LoggerConfig

- `SkipPaths []string`: exact paths to skip logging.
- `SkipSuccessfulPaths bool`: when `false` (default), matched `SkipPaths` are fully skipped for all status codes; when `true`, matched paths skip only `<400` responses and still log `4xx/5xx`.
- `RequestIDHeader string`: request ID header key (default: `X-Request-ID`).
- `TraceIDHeader string`: trace ID header key (default: `X-Trace-ID`, with W3C `traceparent` fallback).
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
- `trace_id` (from configured trace header, or W3C `traceparent`)
- `errors` (when Gin context has errors)

## Level Mapping

- `status >= 500` => `Error`
- `status >= 400` => `Warn`
- others => `Info`

## Skip Rules

By default, `SkipPaths` preserves the original behavior and skips matching paths before logging, regardless of the final response status.

Use `SkipSuccessfulPaths: true` when you want quiet health-check or metrics logs but still need failed requests:

```go
r.Use(ginmw.GinLoggerWithConfig(logger, ginmw.LoggerConfig{
	SkipPaths:           []string{"/healthz", "/metrics"},
	SkipSuccessfulPaths: true,
}))
```

With this config, `/healthz` returning `200` or `302` is skipped, while `/healthz` returning `404` or `500` is logged.

## Production Recommendation

For large projects, prefer instance logger injection:

1. Create logger with `glog.New(...)`.
2. Pass logger into middleware constructors.
3. Avoid mixing `glog.xxx` global logging and instance logging in business modules.

Use `encoder: json` in production so request fields such as `trace_id` can be queried reliably by log collection systems.

## Demo

Run demo app:

```bash
go run ./examples/gin_demo
```

Then test endpoints:

- `GET /hello`
- `GET /healthz` (skipped by logger)
- `GET /panic` (recovery + panic log)
