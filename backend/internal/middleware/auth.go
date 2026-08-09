// Package middleware implements the HTTP cross-cutting concerns: auth, RBAC,
// rate limiting, metrics, recovery and request logging.
package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
)

// UserStatusCheck reports whether a user account may keep its session
// (active users only). Injected so the middleware stays DB-agnostic.
type UserStatusCheck func(ctx context.Context, userID string) (bool, error)

// AuthMiddleware validates the bearer token and the Redis SSO session.
type AuthMiddleware struct {
	cache    *cache.Client
	ttl      time.Duration
	statusFn UserStatusCheck
}

// NewAuthMiddleware wires the auth guard. statusFn is optional; when nil the
// account-status check is skipped.
func NewAuthMiddleware(c *cache.Client, ttl time.Duration, statusFn UserStatusCheck) *AuthMiddleware {
	return &AuthMiddleware{cache: c, ttl: ttl, statusFn: statusFn}
}

// Handle implements app.HandlerFunc.
func (m *AuthMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	authorization := string(c.Request.Header.Peek("Authorization"))
	if len(authorization) <= 7 || !strings.HasPrefix(authorization, "Bearer ") {
		response.Fail("invalid token").SetCode(apperror.CodeUnauthorized).Json(c)
		c.Abort()
		return
	}
	tokenStr := authorization[7:]
	claims, err := jwt.ParseToken(tokenStr)
	if err != nil {
		logger.Warn("auth: token parse failed", zap.Error(err))
		response.Fail("invalid token").SetCode(apperror.CodeUnauthorized).Json(c)
		c.Abort()
		return
	}
	// Single-sign-on: the fingerprint must match the active session in Redis.
	ok, err := service.RenewIfNeeded(m.cache, claims.UserID, service.SSOValue(tokenStr, claims.UserID), m.ttl)
	if err != nil || !ok {
		response.Fail("token expired or session revoked").SetCode(apperror.CodeUnauthorized).Json(c)
		c.Abort()
		return
	}
	// Revoked or locked accounts lose access immediately. A transient status-
	// check error (e.g. DB blip) fails closed with 503 but does NOT destroy
	// the session, so an outage doesn't mass-log-out every user.
	if m.statusFn != nil {
		active, err := m.statusFn(ctx, claims.UserID)
		if err != nil {
			response.Fail("authorization unavailable").SetCode(apperror.CodeUnavailable).Json(c)
			c.Abort()
			return
		}
		if !active {
			_ = m.cache.Del(service.SSOKey(claims.UserID))
			response.Fail("account is not active").SetCode(apperror.CodeUnauthorized).Json(c)
			c.Abort()
			return
		}
	}
	c.Set("user_id", claims.UserID)
	c.Set("user_email", claims.Email)
	c.Set("role_ids", claims.RoleIDs)
	c.Set("token", tokenStr)
	c.Next(ctx)
}

// NotFoundHandler answers unmatched routes.
func NotFoundHandler(_ context.Context, c *app.RequestContext) {
	response.Fail("route not found").SetCode(404).Json(c)
	c.Abort()
}
