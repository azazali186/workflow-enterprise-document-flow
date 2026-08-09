package handler

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/database"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// HealthHandler exposes liveness and readiness probes. Both are POST to stay
// within the project's method convention (no GET anywhere in the API).
type HealthHandler struct{}

// NewHealthHandler wires the handler.
func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

// Healthz handles POST /api/v1/healthz.
// @Summary      Liveness probe
// @Tags         system
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /api/v1/healthz [post]
func (h *HealthHandler) Healthz(ctx context.Context, c *app.RequestContext) {
	response.Success(map[string]any{"status": "ok"}).Json(c)
}

// Readyz handles POST /api/v1/readyz.
// @Summary      Readiness probe (checks Postgres, Redis, NATS)
// @Tags         system
// @Accept       json
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /api/v1/readyz [post]
func (h *HealthHandler) Readyz(ctx context.Context, c *app.RequestContext) {
	checks := map[string]string{}
	if err := database.Ping(ctx); err != nil {
		checks["postgres"] = "down"
	} else {
		checks["postgres"] = "up"
	}
	if err := database.Cache.Ping(); err != nil {
		checks["redis"] = "down"
	} else {
		checks["redis"] = "up"
	}
	if database.NC == nil || !database.NC.IsConnected() {
		checks["nats"] = "down"
	} else {
		checks["nats"] = "up"
	}
	for _, v := range checks {
		if v == "down" {
			response.Fail("readiness checks failed").SetCode(503).SetData(checks).Json(c)
			return
		}
	}
	response.Success(checks).Json(c)
}
