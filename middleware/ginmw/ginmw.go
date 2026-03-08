package ginmw

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerConfig controls GinLogger behavior.
type LoggerConfig struct {
	// SkipPaths bypasses logging for exact path matches.
	SkipPaths []string
	// RequestIDHeader is the HTTP header key used to extract request ID.
	RequestIDHeader string
	// Message is the log message emitted for each request.
	Message string
}

// GinLogger returns a request logging middleware using default config.
func GinLogger(log *zap.SugaredLogger) gin.HandlerFunc {
	return GinLoggerWithConfig(log, LoggerConfig{})
}

// GinLoggerWithConfig returns a request logging middleware with config.
func GinLoggerWithConfig(log *zap.SugaredLogger, cfg LoggerConfig) gin.HandlerFunc {
	logger := ensureLogger(log)
	skip := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skip[p] = struct{}{}
	}

	requestIDHeader := cfg.RequestIDHeader
	if requestIDHeader == "" {
		requestIDHeader = "X-Request-ID"
	}

	message := cfg.Message
	if message == "" {
		message = "gin request"
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if _, ok := skip[path]; ok {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		fullPath := path
		if raw := c.Request.URL.RawQuery; raw != "" {
			fullPath = path + "?" + raw
		}

		fields := []interface{}{
			"method", c.Request.Method,
			"path", fullPath,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if requestID := c.GetHeader(requestIDHeader); requestID != "" {
			fields = append(fields, "request_id", requestID)
		}
		if errMsg := c.Errors.String(); errMsg != "" {
			fields = append(fields, "errors", errMsg)
		}

		switch {
		case status >= http.StatusInternalServerError:
			logger.Errorw(message, fields...)
		case status >= http.StatusBadRequest:
			logger.Warnw(message, fields...)
		default:
			logger.Infow(message, fields...)
		}
	}
}

// GinRecovery recovers from panics and logs them.
func GinRecovery(log *zap.SugaredLogger, includeStack bool) gin.HandlerFunc {
	logger := ensureLogger(log)
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				fields := []interface{}{
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"client_ip", c.ClientIP(),
					"panic", fmt.Sprint(recovered),
				}
				if includeStack {
					fields = append(fields, "stack", string(debug.Stack()))
				}
				logger.Errorw("gin panic recovered", fields...)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()
	}
}

func ensureLogger(log *zap.SugaredLogger) *zap.SugaredLogger {
	if log != nil {
		return log
	}
	return zap.NewNop().Sugar()
}
