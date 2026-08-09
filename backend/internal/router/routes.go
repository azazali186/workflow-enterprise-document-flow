package router

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aeroxe/docu-flow/backend/internal/database"
	"github.com/aeroxe/docu-flow/backend/internal/model"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/cloudwego/hertz/pkg/app/server"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

// RouteInfo describes one registered API route.
type RouteInfo struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Guard  string `json:"guard"`
	Method string `json:"method"`
}

// excludedPrefixes are non-API routes that are never stored as permissions:
// auth bootstrap, infra probes, metrics, Swagger UI and the WebSocket hub.
var excludedPrefixes = []string{
	"/swagger",
	"/metrics",
	"/ws",
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
	"/api/v1/healthz",
	"/api/v1/readyz",
	// The options lookup is auth-gated but not a permission-gated operation
	// (see Register: it sits on the auth-only group), so it must not appear
	// in the assignable permission catalog.
	"/api/v1/options",
}

// IsExcludedRoute reports whether a path is public (no permission required).
func IsExcludedRoute(path string) bool {
	for _, p := range excludedPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// CollectRoutes mirrors the gateway PrintAllRoutes logic: it walks every
// registered Hertz route and builds the RouteInfo list, skipping duplicates
// and excluded routes.
func CollectRoutes(h *server.Hertz) []RouteInfo {
	seen := make(map[string]struct{})
	var out []RouteInfo
	for _, r := range h.Routes() {
		routeKey := fmt.Sprintf("%s %s", r.Method, r.Path)
		if _, exists := seen[routeKey]; exists {
			continue
		}
		seen[routeKey] = struct{}{}
		if IsExcludedRoute(r.Path) {
			continue
		}
		out = append(out, RouteInfo{
			Name:   formatRouteName(r.Path),
			URL:    r.Path,
			Guard:  "API",
			Method: r.Method,
		})
	}
	return out
}

// SyncPermissionsFromRoutes upserts every collected route as a permission and
// caches the manifest in Redis under api-gateway-permissions.
func SyncPermissionsFromRoutes(h *server.Hertz) (int, error) {
	routes := CollectRoutes(h)
	jsonData, err := json.MarshalIndent(routes, "", "    ")
	if err != nil {
		return 0, err
	}
	if err := database.Cache.Set("api-gateway-permissions", jsonData, 0); err != nil {
		logger.Warn("failed to cache route manifest", zap.Error(err))
	}
	n, err := storeNewPermissions(database.DB, routes)
	if err != nil {
		return 0, err
	}
	logger.Info("permissions synced from routes", zap.Int("upserted", n), zap.Int("total", len(routes)))
	return n, nil
}

// storeNewPermissions upserts permission rows keyed by "METHOD path".
func storeNewPermissions(db *gorm.DB, routes []RouteInfo) (int, error) {
	upserted := 0
	for _, route := range routes {
		uniqueRoute := fmt.Sprintf("%s %s", route.Method, route.URL)
		var existing model.Permission
		err := db.Where("route = ?", uniqueRoute).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			if err := db.Create(&model.Permission{
				Name: route.Name, Route: uniqueRoute, Path: route.URL,
				Method: route.Method, Service: "api-gateway",
			}).Error; err != nil {
				return upserted, err
			}
			upserted++
		case err != nil:
			return upserted, err
		default:
			changed := false
			if existing.Name != route.Name {
				existing.Name = route.Name
				changed = true
			}
			if existing.Path != route.URL {
				existing.Path = route.URL
				changed = true
			}
			if existing.Service != "api-gateway" {
				existing.Service = "api-gateway"
				changed = true
			}
			if changed {
				if err := db.Save(&existing).Error; err != nil {
					return upserted, err
				}
				upserted++
			}
		}
	}
	return upserted, nil
}

// formatRouteName converts a path to a human friendly title (mirrors gateway).
func formatRouteName(path string) string {
	cleaned := strings.Replace(path, "/api/v1", "", 1)
	cleaned = strings.ReplaceAll(cleaned, "/", " ")
	return cases.Title(language.English).String(strings.TrimSpace(cleaned))
}
