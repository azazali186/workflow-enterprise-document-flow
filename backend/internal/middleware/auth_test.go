package middleware

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/redis/go-redis/v9"
)

const testJWTSecret = "middleware-test-secret-0123456789"

// buildAuthEngine registers the auth middleware in front of a trivial route.
func buildAuthEngine(t *testing.T, c *cache.Client, statusFn UserStatusCheck) *route.Engine {
	t.Helper()
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(NewAuthMiddleware(c, time.Hour, statusFn).Handle)
	h.POST("/protected", func(_ context.Context, ctx *app.RequestContext) {
		ctx.SetStatusCode(200)
	})
	return h.Engine
}

func requestProtected(engine *route.Engine, token string) *ut.ResponseRecorder {
	var headers []ut.Header
	if token != "" {
		headers = append(headers, ut.Header{Key: "Authorization", Value: "Bearer " + token})
	}
	return ut.PerformRequest(engine, "POST", "/protected", nil, headers...)
}

// bodyCode decodes the business code from the JSON envelope.
func bodyCode(t *testing.T, w *ut.ResponseRecorder) int {
	t.Helper()
	var r struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return r.Code
}

func newTestCache(t *testing.T) (*miniredis.Miniredis, *cache.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	return mr, cache.New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
}

func validTokenFor(t *testing.T, userID string) string {
	t.Helper()
	token, _, err := jwt.Generate(userID, userID+"@example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestAuthAllowsValidSession(t *testing.T) {
	jwt.Init(testJWTSecret, time.Hour)
	_, c := newTestCache(t)
	engine := buildAuthEngine(t, c, nil)

	token := validTokenFor(t, "u1")
	if err := c.Set(service.SSOKey("u1"), service.SSOValue(token, "u1"), time.Hour); err != nil {
		t.Fatal(err)
	}
	if w := requestProtected(engine, token); w.Code != 200 {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestAuthRejectsMissingToken(t *testing.T) {
	jwt.Init(testJWTSecret, time.Hour)
	_, c := newTestCache(t)
	engine := buildAuthEngine(t, c, nil)

	if w := requestProtected(engine, ""); bodyCode(t, w) != 401 {
		t.Fatalf("expected 401, got %d", bodyCode(t, w))
	}
}

func TestAuthRejectsMalformedToken(t *testing.T) {
	jwt.Init(testJWTSecret, time.Hour)
	_, c := newTestCache(t)
	engine := buildAuthEngine(t, c, nil)

	if w := requestProtected(engine, "not-a-jwt"); bodyCode(t, w) != 401 {
		t.Fatalf("expected 401, got %d", bodyCode(t, w))
	}
}

func TestAuthRejectsRevokedSession(t *testing.T) {
	jwt.Init(testJWTSecret, time.Hour)
	_, c := newTestCache(t)
	engine := buildAuthEngine(t, c, nil)

	// Valid token, but no matching Redis session (logged out / revoked).
	token := validTokenFor(t, "u1")
	if w := requestProtected(engine, token); bodyCode(t, w) != 401 {
		t.Fatalf("expected 401, got %d", bodyCode(t, w))
	}
}

func TestAuthBlocksInactiveAccountAndKillsSession(t *testing.T) {
	jwt.Init(testJWTSecret, time.Hour)
	_, c := newTestCache(t)
	engine := buildAuthEngine(t, c, func(_ context.Context, userID string) (bool, error) {
		return false, nil // account locked/deleted
	})

	token := validTokenFor(t, "u1")
	if err := c.Set(service.SSOKey("u1"), service.SSOValue(token, "u1"), time.Hour); err != nil {
		t.Fatal(err)
	}
	if w := requestProtected(engine, token); bodyCode(t, w) != 401 {
		t.Fatalf("expected 401, got %d", bodyCode(t, w))
	}
	if exists, _ := c.Exists(service.SSOKey("u1")); exists {
		t.Fatal("session should be purged for an inactive account")
	}
}

func TestAuthFailsClosedOnStatusCheckError(t *testing.T) {
	jwt.Init(testJWTSecret, time.Hour)
	_, c := newTestCache(t)
	engine := buildAuthEngine(t, c, func(_ context.Context, _ string) (bool, error) {
		return false, context.DeadlineExceeded // e.g. DB outage
	})

	token := validTokenFor(t, "u1")
	if err := c.Set(service.SSOKey("u1"), service.SSOValue(token, "u1"), time.Hour); err != nil {
		t.Fatal(err)
	}
	// Fails closed with 503, but the session survives an outage: a transient
	// DB error must not mass-log-out every user.
	if w := requestProtected(engine, token); bodyCode(t, w) != 503 {
		t.Fatalf("expected 503, got %d", bodyCode(t, w))
	}
	if exists, _ := c.Exists(service.SSOKey("u1")); !exists {
		t.Fatal("session must survive a transient status-check error")
	}
}
