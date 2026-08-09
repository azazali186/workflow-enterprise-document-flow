package ws

import (
	"context"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/apperror"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/jwt"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/aeroxe/docu-flow/backend/internal/service"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
	"go.uber.org/zap"
)

// Keepalive constants: the server pings clients so idle connections are
// reaped by the read deadline instead of lingering forever.
const (
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	writeWait  = 10 * time.Second
	maxMsgSize = 1024
)

// ActiveCheck reports whether the account behind a token is still active
// (mirrors the HTTP auth middleware's status guard). Injected so the endpoint
// stays DB-agnostic.
type ActiveCheck func(ctx context.Context, userID string) (bool, error)

// Endpoint upgrades authenticated HTTP requests to WebSocket connections.
type Endpoint struct {
	hub      *Hub
	cache    *cache.Client
	ttl      time.Duration
	statusFn ActiveCheck
	upgrader websocket.HertzUpgrader
	// writeTimeout / readTimeout guard idle connections.
	writeTimeout time.Duration
}

// NewEndpoint wires the WebSocket endpoint. origins controls the handshake
// Origin allow-list: an empty list or a lone "*" allows any origin (dev);
// production passes the explicit CORS allow-list. statusFn is optional; when
// nil the account-status check is skipped.
func NewEndpoint(hub *Hub, c *cache.Client, ttl time.Duration, origins []string, statusFn ActiveCheck) *Endpoint {
	return &Endpoint{
		hub:          hub,
		cache:        c,
		ttl:          ttl,
		statusFn:     statusFn,
		writeTimeout: writeWait,
		upgrader: websocket.HertzUpgrader{
			HandshakeTimeout: 5 * time.Second,
			CheckOrigin: func(ctx *app.RequestContext) bool {
				return originAllowed(string(ctx.Request.Header.Peek("Origin")), origins)
			},
		},
	}
}

// originAllowed mirrors the HTTP CORS policy for WebSocket handshakes.
func originAllowed(origin string, origins []string) bool {
	if origin == "" || len(origins) == 0 {
		return true // non-browser clients (CLI, servers) send no Origin
	}
	for _, o := range origins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

// Handle implements the /ws route. The token travels in the Authorization
// header (query params are banned project-wide).
// @Summary      Upgrade to a real-time WebSocket session
// @Tags         system
// @Description  Authenticates via the Authorization header, then pushes document events.
// @Security     BearerAuth
// @Success      101
// @Router       /ws [get]
func (e *Endpoint) Handle(ctx context.Context, c *app.RequestContext) {
	tokenStr, err := bearerToken(c)
	if err != nil {
		response.Fail(err.Error()).SetCode(apperror.CodeUnauthorized).Json(c)
		return
	}
	claims, err := jwt.ParseToken(tokenStr)
	if err != nil {
		response.Fail("invalid token").SetCode(apperror.CodeUnauthorized).Json(c)
		return
	}
	if ok, err := service.RenewIfNeeded(e.cache, claims.UserID, service.SSOValue(tokenStr, claims.UserID), e.ttl); err != nil || !ok {
		response.Fail("session expired").SetCode(apperror.CodeUnauthorized).Json(c)
		return
	}
	// Revoked or locked accounts cannot hold real-time sessions. A transient
	// status-check error fails closed with 503 and does not destroy the
	// session (same semantics as the HTTP auth guard).
	if e.statusFn != nil {
		active, err := e.statusFn(ctx, claims.UserID)
		if err != nil {
			response.Fail("authorization unavailable").SetCode(apperror.CodeUnavailable).Json(c)
			return
		}
		if !active {
			response.Fail("account is not active").SetCode(apperror.CodeUnauthorized).Json(c)
			return
		}
	}
	userID := claims.UserID
	err = e.upgrader.Upgrade(c, func(conn *websocket.Conn) {
		client := &Client{
			hub:    e.hub,
			conn:   conn,
			userID: userID,
			send:   make(chan []byte, 32),
		}
		e.hub.Register(client)
		logger.Info("ws: client connected", zap.String("user", userID))
		defer func() {
			e.hub.Unregister(client)
			logger.Info("ws: client disconnected", zap.String("user", userID))
		}()

		// Idle connections are reaped: a missing frame (or pong) within
		// pongWait closes the socket, so dead clients can't hold the
		// connection forever. The deadline is refreshed before every read so
		// healthy clients that don't answer pings are never dropped.
		conn.SetReadLimit(maxMsgSize)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})
		go e.pingLoop(conn)

		// Read loop: keeps the connection alive and surfaces client closes.
		for {
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	if err != nil {
		logger.Warn("ws: upgrade failed", zap.Error(err))
	}
}

// pingLoop sends periodic pings via WriteControl, which is safe to call
// concurrently with the hub's writer goroutine.
func (e *Endpoint) pingLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for range ticker.C {
		if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(e.writeTimeout)); err != nil {
			_ = conn.Close()
			return
		}
	}
}

func bearerToken(c *app.RequestContext) (string, error) {
	auth := string(c.Request.Header.Peek("Authorization"))
	if len(auth) <= 7 || auth[:7] != "Bearer " {
		return "", apperror.Unauthorized("missing bearer token")
	}
	return auth[7:], nil
}
