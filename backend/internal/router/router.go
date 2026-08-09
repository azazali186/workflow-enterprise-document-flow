// Package router registers every endpoint with its middleware chain and
// exposes the Swagger UI. Routes use only POST/PATCH/DELETE and carry all
// data in the request body.
package router

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/handler"
	"github.com/aeroxe/docu-flow/backend/internal/middleware"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/ws"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Handlers bundles every HTTP handler for wiring.
type Handlers struct {
	Auth         *handler.AuthHandler
	User         *handler.UserHandler
	Role         *handler.RoleHandler
	Permission   *handler.PermissionHandler
	Document     *handler.DocumentHandler
	Version      *handler.VersionHandler
	Category     *handler.CategoryHandler
	Template     *handler.TemplateHandler
	Storage      *handler.StorageHandler
	Verification *handler.VerificationHandler
	Approval     *handler.ApprovalHandler
	Access       *handler.AccessHandler
	AuditLog     *handler.AuditLogHandler
	LoginLog     *handler.LoginLogHandler
	Report       *handler.ReportHandler
	Search       *handler.SearchHandler
	Analytics    *handler.AnalyticsHandler
	Health       *handler.HealthHandler
	WS           *ws.Endpoint
}

// Middlewares bundles the cross-cutting guards.
type Middlewares struct {
	RequestID  *middleware.RequestIDMiddleware
	Auth       *middleware.AuthMiddleware
	RBAC       *middleware.RBACMiddleware
	RateLimit  *middleware.RateLimitMiddleware
	Metrics    *middleware.MetricsMiddleware
	Recovery   *middleware.RecoveryMiddleware
	RequestLog *middleware.RequestLogMiddleware
	CORS       *middleware.CORS
	CSRF       *middleware.CSRFMiddleware
}

// Register wires middleware and routes onto the Hertz engine. Swagger UI is
// mounted only when explicitly enabled (off in production by default) because
// it exposes the full API schema.
func Register(h *server.Hertz, hs *Handlers, mw *Middlewares, enableSwagger bool) {
	h.Use(mw.RequestID.Handle, mw.Recovery.Handle, mw.Metrics.Handle, mw.RequestLog.Handle,
		mw.RateLimit.Handle, mw.CORS.Handle, mw.CSRF.Handle)

	api := h.Group("/api/v1")

	// Public routes (excluded from auth + permission seeding).
	pub := api.Group("")
	pub.POST("/auth/register", hs.Auth.Register)
	pub.POST("/auth/login", hs.Auth.Login)
	pub.POST("/auth/refresh", hs.Auth.Refresh)
	pub.POST("/healthz", hs.Health.Healthz)
	pub.POST("/readyz", hs.Health.Readyz)

	// Real-time events (auth happens inside the endpoint, no API permission).
	h.GET("/ws", hs.WS.Handle)

	// Everything below requires auth + route permission.
	sec := api.Group("", mw.Auth.Handle, mw.RBAC.Handle)
	sec.POST("/auth/logout", hs.Auth.Logout)
	sec.POST("/auth/me", hs.Auth.Me)

	sec.POST("/users/create", hs.User.Create)
	sec.PATCH("/users/update", hs.User.Update)
	sec.POST("/users/delete", hs.User.Delete)
	sec.POST("/users/get", hs.User.Get)
	sec.POST("/users/list", hs.User.List)

	sec.POST("/roles/create", hs.Role.Create)
	sec.PATCH("/roles/update", hs.Role.Update)
	sec.POST("/roles/delete", hs.Role.Delete)
	sec.POST("/roles/get", hs.Role.Get)
	sec.POST("/roles/list", hs.Role.List)
	sec.POST("/roles/assign-permissions", hs.Role.AssignPermissions)

	sec.POST("/permissions/get", hs.Permission.Get)
	sec.POST("/permissions/list", hs.Permission.List)
	sec.POST("/permissions/sync", hs.Permission.Sync)

	sec.POST("/documents/create", hs.Document.Create)
	sec.PATCH("/documents/update", hs.Document.Update)
	sec.POST("/documents/delete", hs.Document.Delete)
	sec.POST("/documents/get", hs.Document.Get)
	sec.POST("/documents/list", hs.Document.List)

	sec.POST("/versions/list", hs.Version.List)
	sec.POST("/versions/get", hs.Version.Get)

	sec.POST("/categories/create", hs.Category.Create)
	sec.PATCH("/categories/update", hs.Category.Update)
	sec.POST("/categories/delete", hs.Category.Delete)
	sec.POST("/categories/get", hs.Category.Get)
	sec.POST("/categories/list", hs.Category.List)

	sec.POST("/templates/create", hs.Template.Create)
	sec.PATCH("/templates/update", hs.Template.Update)
	sec.POST("/templates/delete", hs.Template.Delete)
	sec.POST("/templates/get", hs.Template.Get)
	sec.POST("/templates/list", hs.Template.List)

	sec.POST("/storages/upload", hs.Storage.Upload)
	sec.POST("/storages/register", hs.Storage.Register)
	sec.POST("/storages/get", hs.Storage.Get)
	sec.POST("/storages/delete", hs.Storage.Delete)
	sec.POST("/storages/list", hs.Storage.List)

	sec.POST("/verifications/create", hs.Verification.Create)
	sec.POST("/verifications/decide", hs.Verification.Decide)
	sec.POST("/verifications/get", hs.Verification.Get)
	sec.POST("/verifications/list", hs.Verification.List)

	sec.POST("/approvals/create", hs.Approval.CreateChain)
	sec.POST("/approvals/decide", hs.Approval.Decide)
	sec.POST("/approvals/get", hs.Approval.Get)
	sec.POST("/approvals/list", hs.Approval.List)

	sec.POST("/accesses/grant", hs.Access.Grant)
	sec.POST("/accesses/revoke", hs.Access.Revoke)
	sec.POST("/accesses/list", hs.Access.List)

	sec.POST("/audit-logs/get", hs.AuditLog.Get)
	sec.POST("/audit-logs/list", hs.AuditLog.List)
	sec.POST("/login-logs/get", hs.LoginLog.Get)
	sec.POST("/login-logs/list", hs.LoginLog.List)

	sec.POST("/reports/dashboard", hs.Report.Dashboard)

	sec.POST("/search/documents", hs.Search.Search)

	sec.POST("/analytics/documents", hs.Analytics.Documents)
	sec.POST("/analytics/storage", hs.Analytics.Storage)
	sec.POST("/analytics/workflow", hs.Analytics.Workflow)

	if enableSwagger {
		registerSwagger(h)
	}
	registerMetrics(h)
	h.NoRoute(middleware.NotFoundHandler)
}

// registerMetrics exposes the Prometheus scrape endpoint (GET, infra-only).
func registerMetrics(h *server.Hertz) {
	h.GET("/metrics", func(ctx context.Context, c *app.RequestContext) {
		req, err := adaptor.GetCompatRequest(&c.Request)
		if err != nil {
			response.Fail("bad request").SetCode(400).Json(c)
			return
		}
		req = req.WithContext(ctx)
		rw := adaptor.GetCompatResponseWriter(&c.Response)
		promhttp.Handler().ServeHTTP(rw, req)
	})
}

// registerSwagger mounts the Swagger UI (served as GET by the browser).
func registerSwagger(h *server.Hertz) {
	h.GET("/swagger/*any", func(ctx context.Context, c *app.RequestContext) {
		req, err := adaptor.GetCompatRequest(&c.Request)
		if err != nil {
			response.Fail("bad request").SetCode(400).Json(c)
			return
		}
		req = req.WithContext(ctx)
		rw := adaptor.GetCompatResponseWriter(&c.Response)
		httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json"))(rw, req)
	})
}
