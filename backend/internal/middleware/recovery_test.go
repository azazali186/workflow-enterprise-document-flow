package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestRecoveryConvertsPanicTo500(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(NewRecoveryMiddleware().Handle)
	h.POST("/boom", func(_ context.Context, _ *app.RequestContext) {
		panic("unexpected failure")
	})
	engine := h.Engine

	w := ut.PerformRequest(engine, "POST", "/boom", nil)
	if bodyCode(t, w) != 500 {
		t.Fatalf("expected 500 envelope, got %d (%s)", bodyCode(t, w), w.Body.String())
	}
}

func TestRecoveryPassesHealthyRequests(t *testing.T) {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(NewRecoveryMiddleware().Handle)
	h.POST("/ok", func(_ context.Context, ctx *app.RequestContext) { ctx.SetStatusCode(200) })
	engine := h.Engine

	if w := ut.PerformRequest(engine, "POST", "/ok", nil); w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
