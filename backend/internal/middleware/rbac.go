package middleware

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
)

// RBACMiddleware enforces route permissions using the registered route
// pattern as the permission key (e.g. "POST /api/v1/documents/list").
type RBACMiddleware struct {
	rbac service.RBACService
}

// NewRBACMiddleware wires the guard.
func NewRBACMiddleware(rbac service.RBACService) *RBACMiddleware {
	return &RBACMiddleware{rbac: rbac}
}

// Handle implements app.HandlerFunc. It must run after AuthMiddleware.
func (m *RBACMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(string)
	route := c.FullPath()
	method := string(c.Request.Method())
	if route == "" {
		// Unregistered path inside a group; let the 404 handler answer.
		c.Next(ctx)
		return
	}
	allowed, err := m.rbac.HasPermission(ctx, uid, method, route)
	if err != nil {
		logger.Error("rbac check failed", zap.String("route", route), zap.Error(err))
		response.Fail("authorization unavailable").SetCode(apperror.CodeUnavailable).Json(c)
		c.Abort()
		return
	}
	if !allowed {
		response.Fail("insufficient permissions: " + method + " " + route).
			SetCode(apperror.CodeForbidden).Json(c)
		c.Abort()
		return
	}
	c.Next(ctx)
}

