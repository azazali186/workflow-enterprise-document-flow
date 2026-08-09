package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
)

func buildCORSEngine(origins []string) *route.Engine {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(NewCORS(origins).Handle)
	h.POST("/ping", func(_ context.Context, ctx *app.RequestContext) { ctx.SetStatusCode(200) })
	return h.Engine
}

func TestCORSAllowListedOrigin(t *testing.T) {
	engine := buildCORSEngine([]string{"https://app.example.com"})
	w := ut.PerformRequest(engine, "POST", "/ping", nil,
		ut.Header{Key: "Origin", Value: "https://app.example.com"})
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected echoed origin, got %q", got)
	}
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCORSRejectsForeignOrigin(t *testing.T) {
	engine := buildCORSEngine([]string{"https://app.example.com"})
	w := ut.PerformRequest(engine, "POST", "/ping", nil,
		ut.Header{Key: "Origin", Value: "https://evil.example.net"})
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header, got %q", got)
	}
}

func TestCORSWildcardAllowsAnyOrigin(t *testing.T) {
	engine := buildCORSEngine([]string{"*"})
	w := ut.PerformRequest(engine, "POST", "/ping", nil,
		ut.Header{Key: "Origin", Value: "https://anything.example.com"})
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://anything.example.com" {
		t.Fatalf("expected echoed origin for wildcard, got %q", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	engine := buildCORSEngine([]string{"https://app.example.com"})
	w := ut.PerformRequest(engine, "OPTIONS", "/ping", nil,
		ut.Header{Key: "Origin", Value: "https://app.example.com"})
	if w.Code != 204 {
		t.Fatalf("expected 204 for preflight, got %d", w.Code)
	}
}
