package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/metrics"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/trace"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/uuidx"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.uber.org/zap"
)

// RequestIDMiddleware assigns or forwards a request ID so logs, outbox events
// and worker processing can be correlated end to end. It must run first.
type RequestIDMiddleware struct{}

// NewRequestIDMiddleware wires the request-id guard.
func NewRequestIDMiddleware() *RequestIDMiddleware { return &RequestIDMiddleware{} }

// Handle implements app.HandlerFunc.
func (m *RequestIDMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	id := sanitizeRequestID(string(c.Request.Header.Peek("X-Request-ID")))
	if id == "" {
		id = uuidx.New()
	}
	c.Set("request_id", id)
	c.Response.Header.Set("X-Request-ID", id)
	c.Next(trace.WithID(ctx, id))
}

// sanitizeRequestID accepts only short, printable IDs so a caller-supplied
// header can't pollute logs or response headers.
func sanitizeRequestID(id string) string {
	if id == "" || len(id) > 128 {
		return ""
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == ':' {
			continue
		}
		return ""
	}
	return strings.TrimSpace(id)
}

// MetricsMiddleware records Prometheus counters, latency and in-flight gauge.
type MetricsMiddleware struct{}

// NewMetricsMiddleware wires the instrumentation.
func NewMetricsMiddleware() *MetricsMiddleware { return &MetricsMiddleware{} }

// Handle implements app.HandlerFunc.
func (m *MetricsMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	start := time.Now()
	metrics.InFlight.Inc()
	defer metrics.InFlight.Dec()
	c.Next(ctx)
	route := c.FullPath()
	if route == "" {
		route = "unmatched"
	}
	// The API envelope always answers HTTP 200 and carries the outcome in the
	// body business code (recorded by response.Json). Label with that code
	// when present so 5xx business errors are visible to alerting; fall back
	// to the HTTP status for infra routes (204 OPTIONS, unmatched, etc.).
	status := fmt.Sprintf("%d", c.Response.StatusCode())
	if bc, ok := c.Get("business_code"); ok {
		if code, ok := bc.(int); ok && code != 0 {
			status = fmt.Sprintf("%d", code)
		}
	}
	metrics.Observe(route, string(c.Request.Method()), status, start)
}

// RecoveryMiddleware converts panics into 500 responses and logs stack traces.
type RecoveryMiddleware struct{}

// NewRecoveryMiddleware wires the panic guard.
func NewRecoveryMiddleware() *RecoveryMiddleware { return &RecoveryMiddleware{} }

// Handle implements app.HandlerFunc.
func (m *RecoveryMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic recovered",
				zap.String("route", c.FullPath()),
				zap.Any("panic", r),
				zap.ByteString("stack", debug.Stack()))
			hlog.Error("panic recovered: ", fmt.Sprintf("%v", r))
			response.Fail("internal server error").SetCode(500).Json(c)
			c.Abort()
		}
	}()
	c.Next(ctx)
}

// RequestLogMiddleware writes a structured access log (never the body).
type RequestLogMiddleware struct{}

// NewRequestLogMiddleware wires the access logger.
func NewRequestLogMiddleware() *RequestLogMiddleware { return &RequestLogMiddleware{} }

// Handle implements app.HandlerFunc.
func (m *RequestLogMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	start := time.Now()
	c.Next(ctx)
	logger.Info("http_request",
		zap.String("request_id", requestID(c)),
		zap.String("method", string(c.Request.Method())),
		zap.String("path", string(c.Request.Path())),
		zap.Int("status", c.Response.StatusCode()),
		zap.Duration("latency", time.Since(start)),
		zap.String("ip", c.ClientIP()),
	)
}

// requestID reads the id attached by the RequestIDMiddleware.
func requestID(c *app.RequestContext) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// CORS handles cross-origin requests for the web client.
type CORS struct{ origins []string }

// NewCORS wires allowed origins.
func NewCORS(origins []string) *CORS { return &CORS{origins: origins} }

// Handle implements app.HandlerFunc.
func (crs *CORS) Handle(ctx context.Context, c *app.RequestContext) {
	origin := string(c.Request.Header.Peek("Origin"))
	if crs.allows(origin) {
		c.Response.Header.Set("Access-Control-Allow-Origin", origin)
		c.Response.Header.Set("Access-Control-Allow-Methods", "POST, PATCH, DELETE, OPTIONS")
		c.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		c.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		c.Response.Header.Set("Access-Control-Max-Age", "86400")
	}
	if string(c.Request.Method()) == "OPTIONS" {
		c.SetStatusCode(204)
		c.Abort()
		return
	}
	c.Next(ctx)
}

func (crs *CORS) allows(origin string) bool {
	for _, o := range crs.origins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}
