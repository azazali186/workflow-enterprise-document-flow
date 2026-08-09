package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
)

// stubRBAC implements service.RBACService for middleware tests. The real
// service denies requests from anonymous callers (empty user id).
type stubRBAC struct {
	allowed bool
	err     error
}

func (s stubRBAC) HasPermission(_ context.Context, userID, _, _ string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.allowed && userID != "", nil
}
func (s stubRBAC) UserRoutes(context.Context, string) (map[string]bool, error) { return nil, nil }
func (s stubRBAC) InvalidateUser(context.Context, string) error                { return nil }
func (s stubRBAC) AssignPermissions(context.Context, string, []string) error   { return nil }
func (s stubRBAC) AssignRoles(context.Context, string, []string) error         { return nil }
func (s stubRBAC) RoleCodes(context.Context, string) ([]string, error)         { return nil, nil }

// setUserID injects an authenticated identity into the request context.
func setUserID(uid string) func(ctx *app.RequestContext) {
	return func(ctx *app.RequestContext) {
		ctx.Set("user_id", uid)
	}
}

func buildRBACEngine(rbac service.RBACService, pre ...func(ctx *app.RequestContext)) *route.Engine {
	h := server.New(server.WithHostPorts("127.0.0.1:0"))
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		for _, fn := range pre {
			fn(c)
		}
		c.Next(ctx)
	})
	h.Use(NewRBACMiddleware(rbac).Handle)
	h.POST("/api/v1/documents/list", func(_ context.Context, ctx *app.RequestContext) {
		ctx.SetStatusCode(200)
	})
	return h.Engine
}

func TestRBACAllowsAuthorizedUser(t *testing.T) {
	engine := buildRBACEngine(stubRBAC{allowed: true}, setUserID("u1"))
	if w := ut.PerformRequest(engine, "POST", "/api/v1/documents/list", nil); w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRBACDeniesUnauthorizedUser(t *testing.T) {
	engine := buildRBACEngine(stubRBAC{allowed: false}, setUserID("u1"))
	if w := ut.PerformRequest(engine, "POST", "/api/v1/documents/list", nil); bodyCode(t, w) != 403 {
		t.Fatalf("expected 403, got %d", bodyCode(t, w))
	}
}

func TestRBACFailsClosedOnCheckError(t *testing.T) {
	engine := buildRBACEngine(stubRBAC{allowed: true, err: errors.New("db unavailable")}, setUserID("u1"))
	if w := ut.PerformRequest(engine, "POST", "/api/v1/documents/list", nil); bodyCode(t, w) != 503 {
		t.Fatalf("expected 503, got %d", bodyCode(t, w))
	}
}

func TestRBACAnonymousUserDenied(t *testing.T) {
	// No auth middleware ran → no user_id → treated as anonymous.
	engine := buildRBACEngine(stubRBAC{allowed: true})
	if w := ut.PerformRequest(engine, "POST", "/api/v1/documents/list", nil); bodyCode(t, w) != 403 {
		t.Fatalf("expected 403, got %d", bodyCode(t, w))
	}
}
