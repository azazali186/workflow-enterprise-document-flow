// Package sessioncookie owns the browser-session cookie contract shared by
// the auth handlers (which set it) and the middleware (which reads it).
//
// Exactly ONE cookie is used — the HttpOnly session token. CSRF protection is
// bound to that same token (HMAC-derived value delivered in the login/refresh
// response body and echoed back in X-CSRF-Token), because hertz v0.9 serializes
// multiple Set-Cookie headers into a single value that browsers would corrupt.
package sessioncookie

import (
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
)

// TokenCookieName is the HttpOnly session cookie. Not a credential itself —
// just the cookie key the browser stores it under (gosec: G101 false positive).
const TokenCookieName = "docuflow_token" // #nosec G101

// MaxAgeFor derives the cookie Max-Age in seconds from a token expiry.
func MaxAgeFor(expiresAt time.Time) int {
	ttl := int(time.Until(expiresAt).Seconds())
	if ttl < 0 {
		return 0
	}
	return ttl
}

// SetToken stores the session token in an HttpOnly, SameSite=Lax cookie so
// client-side JS cannot read it. secure must be true only when the app is
// served over HTTPS (production).
func SetToken(c *app.RequestContext, token string, maxAge int, secure bool) {
	c.SetCookie(TokenCookieName, token, maxAge, "/", "",
		protocol.CookieSameSiteLaxMode, secure, true)
}

// Token returns the session token cookie, if any.
func Token(c *app.RequestContext) string { return string(c.Cookie(TokenCookieName)) }

// Clear expires the session cookie (logout / session teardown). Written raw
// because hertz's SetCookie API cannot express a deletion (negative Max-Age
// serializes to no expiry attribute at all); a past Expires plus Max-Age=0
// makes every browser drop the cookie immediately.
func Clear(c *app.RequestContext, secure bool) {
	raw := "docuflow_token=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; HttpOnly; SameSite=Lax"
	if secure {
		raw += "; Secure"
	}
	c.Response.Header.Set("Set-Cookie", raw)
}
