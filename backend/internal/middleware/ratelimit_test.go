package middleware

import (
	"context"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/redis/go-redis/v9"
)

func buildRateLimitEngine(c *cache.Client, limit int) *route.Engine {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	// Deterministic client IP so all requests share one limiter key.
	h.SetClientIPFunc(func(*app.RequestContext) string { return "203.0.113.9" })
	h.Use(NewRateLimitMiddleware(c, limit).Handle)
	h.POST("/ping", func(_ context.Context, ctx *app.RequestContext) { ctx.SetStatusCode(200) })
	return h.Engine
}

func TestRateLimitBlocksOverLimit(t *testing.T) {
	_, c := newTestCache(t)
	engine := buildRateLimitEngine(c, 3)

	for i := 1; i <= 3; i++ {
		if w := ut.PerformRequest(engine, "POST", "/ping", nil); w.Code != 200 {
			t.Fatalf("request %d should pass, got %d", i, w.Code)
		}
	}
	if w := ut.PerformRequest(engine, "POST", "/ping", nil); bodyCode(t, w) != 429 {
		t.Fatalf("expected 429, got %d", bodyCode(t, w))
	}
}

func TestRateLimitFailsOpenWhenRedisDown(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close() // simulate Redis outage

	c := cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: addr}))
	engine := buildRateLimitEngine(c, 1)

	// Fail open: traffic keeps flowing instead of a 500 storm.
	for i := 0; i < 3; i++ {
		if w := ut.PerformRequest(engine, "POST", "/ping", nil); w.Code != 200 {
			t.Fatalf("request %d should fail open, got %d", i, w.Code)
		}
	}
}
