package middleware

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/response"
	"github.com/cloudwego/hertz/pkg/app"
)

// RateLimitMiddleware caps requests per minute per identity (user or IP).
type RateLimitMiddleware struct {
	cache *cache.Client
	limit int64
}

// NewRateLimitMiddleware wires the limiter.
func NewRateLimitMiddleware(c *cache.Client, limit int) *RateLimitMiddleware {
	if limit <= 0 {
		limit = 120
	}
	return &RateLimitMiddleware{cache: c, limit: int64(limit)}
}

// Handle implements app.HandlerFunc.
func (m *RateLimitMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	key := c.ClientIP()
	if uid, ok := c.Get("user_id"); ok {
		if s, ok := uid.(string); ok && s != "" {
			key = "u:" + s
		}
	}
	redisKey := constant.RateLimit + key
	count, err := m.cache.Incr(redisKey)
	if err != nil {
		c.Next(ctx) // fail open if Redis is down
		return
	}
	if count == 1 {
		_ = m.cache.Expire(redisKey, constant.RateLimitWindow)
	}
	if count > m.limit {
		response.Fail("rate limit exceeded").SetCode(429).Json(c)
		c.Abort()
		return
	}
	c.Next(ctx)
}
