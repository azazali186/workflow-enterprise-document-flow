// DocuFlow API gateway — Hertz + GORM + Redis + NATS JetStream.
//
// On startup the server:
//  1. boots the DI container (Postgres/Redis/NATS, migrations, RBAC seed)
//  2. registers every route with auth + RBAC middleware
//  3. scans registered routes and seeds them as permissions (upsert)
//  4. starts the transactional outbox dispatcher
//  5. serves Swagger UI at /swagger
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/aeroxe/docu-flow/backend/docs"
	"github.com/aeroxe/docu-flow/backend/internal/bootstrap"
	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/database"
	"github.com/aeroxe/docu-flow/backend/internal/handler"
	"github.com/aeroxe/docu-flow/backend/internal/middleware"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/router"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	logger.FatalIf(err, "load config")
	logger.FatalIf(logger.Init(cfg.LogLevel, cfg.IsProduction()), "init logger")

	container, err := bootstrap.New(ctx, cfg)
	logger.FatalIf(err, "bootstrap application")
	logger.Info("infrastructure ready",
		zap.String("env", cfg.Env), zap.String("port", cfg.Port))

	h := server.New(
		server.WithHostPorts(":"+cfg.Port),
		server.WithMaxRequestBodySize(cfg.MaxBodyBytes),
		server.WithReadTimeout(30*time.Second),
		server.WithExitWaitTime(10*time.Second),
	)

	// Client IP resolution honours X-Forwarded-For / X-Real-IP only when the
	// direct peer is inside TRUSTED_PROXIES. With an empty list the peer
	// address is used, so the rate-limit key can't be spoofed via headers.
	trustedNets, err := cfg.TrustedProxyNets()
	logger.FatalIf(err, "parse TRUSTED_PROXIES")
	h.SetClientIPFunc(app.ClientIPWithOption(app.ClientIPOptions{
		RemoteIPHeaders: []string{"X-Forwarded-For", "X-Real-IP"},
		TrustedCIDRs:    trustedNets,
	}))

	// Middleware wiring. The auth guard additionally verifies the account is
	// still active on every request (revoked/locked users lose access at once).
	mw := &router.Middlewares{
		RequestID: middleware.NewRequestIDMiddleware(),
		Auth: middleware.NewAuthMiddleware(container.Cache, cfg.JWTExpiry,
			bootstrap.ActiveUserStatus(container.DB)),
		RBAC:       middleware.NewRBACMiddleware(container.RBAC),
		RateLimit:  middleware.NewRateLimitMiddleware(container.Cache, cfg.RateLimit),
		Metrics:    middleware.NewMetricsMiddleware(),
		Recovery:   middleware.NewRecoveryMiddleware(),
		RequestLog: middleware.NewRequestLogMiddleware(),
		CORS:       middleware.NewCORS(cfg.CORSOrigins),
		CSRF:       middleware.NewCSRFMiddleware(),
	}

	router.Register(h, container.Handlers, mw, cfg.SwaggerEnabled)

	// Seed permissions from the live route table (idempotent upsert).
	handler.PermissionSyncFn = func() (int, error) { return router.SyncPermissionsFromRoutes(h) }
	n, err := router.SyncPermissionsFromRoutes(h)
	if err != nil {
		logger.Fatal("failed to seed permissions", zap.Error(err))
	}
	logger.Info("permission seed complete", zap.Int("changed", n))

	// Outbox dispatcher + event worker background loops.
	go container.Dispatcher.Run(ctx, container.Lock)
	go func() {
		if err := container.Worker.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("worker stopped", zap.Error(err))
		}
	}()

	// Spin blocks until shutdown; unexpected exits are logged internally.
	go func() {
		h.Spin()
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = h.Shutdown(shutdownCtx)
	database.CloseNATS()
	logger.FatalIf(database.CloseRedis(), "close redis")
	logger.Sync()
}
