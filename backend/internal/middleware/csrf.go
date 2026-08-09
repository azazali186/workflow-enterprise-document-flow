package middleware

import (
	"context"
	"crypto/subtle"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/sessioncookie"
	"github.com/cloudwego/hertz/pkg/app"
)

// CSRFMiddleware enforces the double-submit pattern: any state-changing
// request carrying the session cookie must send X-CSRF-Token equal to the
// HMAC-derived value bound to that token (jwt.CSRFFor). The SPA learns the
// value from the login/refresh response body; a cross-site attacker can
// neither read the HttpOnly cookie nor recompute the HMAC, so forged state
// changes are rejected. SameSite=Lax already blocks cross-site cookie
// attachment for these POSTs; the header check is defense in depth.
type CSRFMiddleware struct{}

// NewCSRFMiddleware wires the guard.
func NewCSRFMiddleware() *CSRFMiddleware { return &CSRFMiddleware{} }

// Handle implements app.HandlerFunc.
func (m *CSRFMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	method := string(c.Request.Method())
	// Read-only methods need no protection; the API surface is POST/PATCH/
	// DELETE but OPTIONS preflight and infra GETs pass through untouched.
	if method == "GET" || method == "OPTIONS" {
		c.Next(ctx)
		return
	}
	// Infra probes and the token-rotation endpoint are exempt: probes are
	// never called from a browser, and refresh rotates a SameSite=Lax cookie
	// whose response body is unreadable cross-origin (CORS), so a forged
	// refresh cannot leak a token.
	if path := string(c.Request.Path()); path == "/api/v1/healthz" || path == "/api/v1/readyz" ||
		path == "/api/v1/auth/refresh" {
		c.Next(ctx)
		return
	}
	tokenCookie := sessioncookie.Token(c)
	if tokenCookie == "" {
		// No session cookie yet (login/register) → nothing to protect.
		c.Next(ctx)
		return
	}
	header := string(c.Request.Header.Peek("X-CSRF-Token"))
	expected := jwt.CSRFFor(tokenCookie)
	if subtle.ConstantTimeCompare([]byte(header), []byte(expected)) != 1 {
		response.Fail("csrf token mismatch").SetCode(apperror.CodeForbidden).Json(c)
		c.Abort()
		return
	}
	c.Next(ctx)
}
