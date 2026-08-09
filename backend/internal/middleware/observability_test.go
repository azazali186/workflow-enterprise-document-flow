package middleware

import (
	"context"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/metrics"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/trace"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// buildRequestIDEngine registers the RequestID middleware before a handler
// that echoes the trace id it received in its context.
func buildRequestIDEngine(t *testing.T, seen *string) *route.Engine {
	t.Helper()
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(NewRequestIDMiddleware().Handle)
	h.POST("/ping", func(ctx context.Context, c *app.RequestContext) {
		if seen != nil {
			*seen = trace.ID(ctx)
		}
		c.SetStatusCode(200)
	})
	return h.Engine
}

func TestRequestIDForwardsClientHeader(t *testing.T) {
	var seen string
	engine := buildRequestIDEngine(t, &seen)
	w := ut.PerformRequest(engine, "POST", "/ping", nil,
		ut.Header{Key: "X-Request-ID", Value: "client-trace-42"})
	if w.Header().Get("X-Request-ID") != "client-trace-42" {
		t.Fatalf("expected response header echo, got %q", w.Header().Get("X-Request-ID"))
	}
	if seen != "client-trace-42" {
		t.Fatalf("expected trace id in handler context, got %q", seen)
	}
}

func TestRequestIDGeneratesWhenMissing(t *testing.T) {
	var seen string
	engine := buildRequestIDEngine(t, &seen)
	if w := ut.PerformRequest(engine, "POST", "/ping", nil); w.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected a generated request id header")
	}
	if seen == "" {
		t.Fatal("expected generated trace id in handler context")
	}
}

func TestRequestIDRejectsMalformedHeader(t *testing.T) {
	var seen string
	engine := buildRequestIDEngine(t, &seen)
	ut.PerformRequest(engine, "POST", "/ping", nil,
		ut.Header{Key: "X-Request-ID", Value: "bad id with spaces"})
	if seen == "bad id with spaces" {
		t.Fatal("malformed request id must be replaced with a generated one")
	}
}

// TestMetricsLabelsUseBusinessCode proves the metrics middleware labels a
// request by the envelope's business code (not the literal HTTP 200 the
// envelope always writes), so error-rate alerting can actually fire.
// ToFloat64 on a CounterVec creates a zero-valued probe series, so a value
// of 0 after the request means the middleware never incremented it.
func TestMetricsLabelsUseBusinessCode(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(NewMetricsMiddleware().Handle)
	h.POST("/boom", func(ctx context.Context, c *app.RequestContext) {
		response.Fail("boom").SetCode(500).Json(c)
	})
	engine := h.Engine

	business := func() float64 {
		return testutil.ToFloat64(metrics.ReqTotal.WithLabelValues("/boom", "POST", "500"))
	}
	httpStatus := func() float64 {
		return testutil.ToFloat64(metrics.ReqTotal.WithLabelValues("/boom", "POST", "200"))
	}
	if v := business(); v != 0 {
		t.Fatalf("expected no 500 business-code series before the request, got %v", v)
	}
	ut.PerformRequest(engine, "POST", "/boom", nil)

	if v := business(); v != 1 {
		t.Fatalf("expected the business code 500 to be recorded once, got %v", v)
	}
	// The literal HTTP status the envelope wrote must not be the recorded label.
	if v := httpStatus(); v != 0 {
		t.Fatalf("expected no status=200 series for an error response, got %v", v)
	}
}
