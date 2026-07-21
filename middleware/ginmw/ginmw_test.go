package ginmw

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newObservedSugaredLogger(level zapcore.Level) (*zap.SugaredLogger, *observer.ObservedLogs) {
	core, observed := observer.New(level)
	return zap.New(core).Sugar(), observed
}

// go test -race ./middleware/ginmw
func TestGinLoggerInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, observed := newObservedSugaredLogger(zapcore.DebugLevel)

	r := gin.New()
	r.Use(GinLogger(log))
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok?x=1", nil)
	req.Header.Set("X-Request-ID", "req-1")
	req.Header.Set("X-Trace-ID", "trace-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Level != zapcore.InfoLevel {
		t.Fatalf("expected info level, got %s", entry.Level)
	}
	if entry.Message != "gin request" {
		t.Fatalf("unexpected message: %s", entry.Message)
	}

	ctx := entry.ContextMap()
	if ctx["status"] != int64(http.StatusOK) {
		t.Fatalf("expected status 200, got %#v", ctx["status"])
	}
	if ctx["path"] != "/ok?x=1" {
		t.Fatalf("expected full path with query, got %#v", ctx["path"])
	}
	if ctx["request_id"] != "req-1" {
		t.Fatalf("expected request_id req-1, got %#v", ctx["request_id"])
	}
	if ctx["trace_id"] != "trace-1" {
		t.Fatalf("expected trace_id trace-1, got %#v", ctx["trace_id"])
	}
}

func TestGinLoggerTraceIDFromTraceparent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, observed := newObservedSugaredLogger(zapcore.DebugLevel)

	r := gin.New()
	r.Use(GinLogger(log))
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	r.ServeHTTP(httptest.NewRecorder(), req)

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	traceID := entries[0].ContextMap()["trace_id"]
	if traceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("unexpected trace_id: %#v", traceID)
	}
}

func TestGinLoggerCustomTraceIDHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, observed := newObservedSugaredLogger(zapcore.DebugLevel)

	r := gin.New()
	r.Use(GinLoggerWithConfig(log, LoggerConfig{TraceIDHeader: "X-Correlation-ID"}))
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("X-Correlation-ID", "correlation-1")
	r.ServeHTTP(httptest.NewRecorder(), req)

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	traceID := entries[0].ContextMap()["trace_id"]
	if traceID != "correlation-1" {
		t.Fatalf("unexpected trace_id: %#v", traceID)
	}
}

func TestGinLoggerWarnAndErrorLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, observed := newObservedSugaredLogger(zapcore.DebugLevel)

	r := gin.New()
	r.Use(GinLogger(log))
	r.GET("/bad", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad")
	})
	r.GET("/err", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "err")
	})

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/bad", nil))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/err", nil))

	if w1.Code != http.StatusBadRequest || w2.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected statuses: %d, %d", w1.Code, w2.Code)
	}

	entries := observed.All()
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}

	if entries[0].Level != zapcore.WarnLevel {
		t.Fatalf("expected warn for 4xx, got %s", entries[0].Level)
	}
	if entries[1].Level != zapcore.ErrorLevel {
		t.Fatalf("expected error for 5xx, got %s", entries[1].Level)
	}
}

func TestGinLoggerSkipPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, observed := newObservedSugaredLogger(zapcore.DebugLevel)

	r := gin.New()
	r.Use(GinLoggerWithConfig(log, LoggerConfig{SkipPaths: []string{"/healthz"}}))
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/healthz-fail", func(c *gin.Context) { c.String(http.StatusInternalServerError, "fail") })
	r.GET("/api", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz-fail", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api", nil))

	entries := observed.FilterField(zap.String("path", "/healthz")).All()
	if len(entries) != 0 {
		t.Fatalf("expected /healthz to be skipped, got %d log entries", len(entries))
	}

	entries = observed.All()
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}
	if entries[0].ContextMap()["path"] != "/healthz-fail" {
		t.Fatalf("expected /healthz-fail log entry, got %#v", entries[0].ContextMap()["path"])
	}
	if entries[1].ContextMap()["path"] != "/api" {
		t.Fatalf("expected /api log entry, got %#v", entries[1].ContextMap()["path"])
	}
}

func TestGinLoggerSkipPathsKeepsOriginalBehaviorForFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, observed := newObservedSugaredLogger(zapcore.DebugLevel)

	r := gin.New()
	r.Use(GinLoggerWithConfig(log, LoggerConfig{SkipPaths: []string{"/healthz"}}))
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusInternalServerError, "fail") })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	entries := observed.All()
	if len(entries) != 0 {
		t.Fatalf("expected matched SkipPaths to preserve old behavior and skip failures, got %d log entries", len(entries))
	}
}

func TestGinLoggerSkipSuccessfulPathsKeepsFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, observed := newObservedSugaredLogger(zapcore.DebugLevel)

	r := gin.New()
	r.Use(GinLoggerWithConfig(log, LoggerConfig{
		SkipPaths:           []string{"/healthz"},
		SkipSuccessfulPaths: true,
	}))
	r.GET("/healthz", func(c *gin.Context) {
		switch c.Query("status") {
		case "302":
			c.Redirect(http.StatusFound, "/healthz")
		case "400":
			c.String(http.StatusBadRequest, "bad")
		case "500":
			c.String(http.StatusInternalServerError, "error")
		default:
			c.String(http.StatusOK, "ok")
		}
	})
	r.GET("/api", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz?status=302", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz?status=400", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz?status=500", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api", nil))

	entries := observed.All()
	if len(entries) != 3 {
		t.Fatalf("expected 3 log entries, got %d", len(entries))
	}
	if entries[0].ContextMap()["path"] != "/healthz?status=400" {
		t.Fatalf("expected /healthz?status=400 log entry, got %#v", entries[0].ContextMap()["path"])
	}
	if entries[0].Level != zapcore.WarnLevel {
		t.Fatalf("expected warn level for 400, got %s", entries[0].Level)
	}
	if entries[1].ContextMap()["path"] != "/healthz?status=500" {
		t.Fatalf("expected /healthz?status=500 log entry, got %#v", entries[1].ContextMap()["path"])
	}
	if entries[1].Level != zapcore.ErrorLevel {
		t.Fatalf("expected error level for 500, got %s", entries[1].Level)
	}
	if entries[2].ContextMap()["path"] != "/api" {
		t.Fatalf("expected /api log entry, got %#v", entries[2].ContextMap()["path"])
	}
}

func TestGinRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, observed := newObservedSugaredLogger(zapcore.DebugLevel)

	r := gin.New()
	r.Use(GinRecovery(log, true))
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("X-Request-ID", "req-panic")
	req.Header.Set("X-Trace-ID", "trace-panic")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Level != zapcore.ErrorLevel {
		t.Fatalf("expected error level, got %s", entry.Level)
	}
	if entry.Message != "gin panic recovered" {
		t.Fatalf("unexpected message: %s", entry.Message)
	}
	ctx := entry.ContextMap()
	if ctx["panic"] != "boom" {
		t.Fatalf("expected panic field boom, got %#v", ctx["panic"])
	}
	if ctx["request_id"] != "req-panic" {
		t.Fatalf("expected request_id req-panic, got %#v", ctx["request_id"])
	}
	if ctx["trace_id"] != "trace-panic" {
		t.Fatalf("expected trace_id trace-panic, got %#v", ctx["trace_id"])
	}
	stack, ok := ctx["stack"].(string)
	if !ok || strings.TrimSpace(stack) == "" {
		t.Fatalf("expected non-empty stack")
	}
}

func TestTraceIDFromTraceparentInvalid(t *testing.T) {
	tests := []string{
		"",
		"invalid",
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",
		"00-4BF92F3577B34DA6A3CE929D0E0E4736-00f067aa0ba902b7-01",
		"00-short-00f067aa0ba902b7-01",
	}

	for _, tt := range tests {
		if got := traceIDFromTraceparent(tt); got != "" {
			t.Fatalf("traceIDFromTraceparent(%q) = %q, want empty", tt, got)
		}
	}
}

func TestNilLoggerDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(GinLogger(nil), GinRecovery(nil, false))
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
