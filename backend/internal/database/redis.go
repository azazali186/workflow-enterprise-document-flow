package database

import (
	"context"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
	"github.com/redis/go-redis/v9"
)

// RDB is the raw Redis client (locks, rate limiting, sets).
var RDB redis.UniversalClient

// Cache is the shared distributed cache facade.
var Cache *cache.Client

// InitRedis parses REDIS_URL and builds the client + cache facade.
func InitRedis(ctx context.Context, cfg *config.Config) error {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return err
	}
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		return err
	}
	RDB = rdb
	Cache = cache.New(ctx, rdb)
	return nil
}

// CloseRedis gracefully closes the connection pool.
func CloseRedis() error {
	if RDB != nil {
		return RDB.Close()
	}
	return nil
}
