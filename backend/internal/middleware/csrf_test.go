package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/sessioncookie"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
)

func buildCSRFEngine() *route.Engine {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(NewCSRFMiddleware().Handle)
	ok := func(_ context.Context, c *app.RequestContext) { response.Success(nil).Json(c) }
	h.POST("/api/v1/things/create", ok)
	h.POST("/api/v1/healthz", ok)
	h.POST("/api/v1/auth/refresh", ok)
	h.GET("/ws", ok)
	return h.Engine
}

func TestCSRFRequiresHeaderWhenSessionCookiePresent(t *testing.T) {
	jwt.Init("csrf-test-secret", time.Hour)
	token := "signed-token-abc"
	engine := buildCSRFEngine()
	cookie := sessioncookie.TokenCookieName + "=" + token

	// Cookie present but header missing → business code 403.
	w := ut.PerformRequest(engine, "POST", "/api/v1/things/create", nil,
		ut.Header{Key: "Cookie", Value: cookie})
	if code := bodyCode(t, w); code != 403 {
		t.Fatalf("expected 403 without header, got %d (body %s)", code, w.Body.String())
	}
	// Correct header → code 0.
	w2 := ut.PerformRequest(engine, "POST", "/api/v1/things/create", nil,
		ut.Header{Key: "Cookie", Value: cookie},
		ut.Header{Key: "X-CSRF-Token", Value: jwt.CSRFFor(token)})
	if code := bodyCode(t, w2); code != 0 {
		t.Fatalf("expected success with matching header, got %d", code)
	}
}

func TestCSRFRejectsMismatchedHeader(t *testing.T) {
	jwt.Init("csrf-test-secret", time.Hour)
	engine := buildCSRFEngine()
	cookie := sessioncookie.TokenCookieName + "=signed-token-abc"
	w := ut.PerformRequest(engine, "POST", "/api/v1/things/create", nil,
		ut.Header{Key: "Cookie", Value: cookie},
		ut.Header{Key: "X-CSRF-Token", Value: "forged-value"})
	if code := bodyCode(t, w); code != 403 {
		t.Fatalf("expected 403 on mismatch, got %d", code)
	}
}

func TestCSRFSkipsWhenNoSessionCookie(t *testing.T) {
	// No session cookie (login/register) → passes without the header.
	engine := buildCSRFEngine()
	w := ut.PerformRequest(engine, "POST", "/api/v1/things/create", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200 without cookie, got %d", w.Code)
	}
}

func TestCSRFExemptsHealthRefreshAndGET(t *testing.T) {
	jwt.Init("csrf-test-secret", time.Hour)
	engine := buildCSRFEngine()
	cookie := sessioncookie.TokenCookieName + "=signed-token-abc"

	// healthz and refresh carry a session cookie but no header → still pass.
	for _, path := range []string{"/api/v1/healthz", "/api/v1/auth/refresh"} {
		w := ut.PerformRequest(engine, "POST", path, nil,
			ut.Header{Key: "Cookie", Value: cookie})
		if w.Code != 200 {
			t.Fatalf("%s must be exempt, got %d", path, w.Code)
		}
	}
	// GET (websocket upgrade) with cookie but no header → passes.
	w2 := ut.PerformRequest(engine, "GET", "/ws", nil,
		ut.Header{Key: "Cookie", Value: cookie})
	if w2.Code != 200 {
		t.Fatalf("GET must be exempt, got %d", w2.Code)
	}
}
