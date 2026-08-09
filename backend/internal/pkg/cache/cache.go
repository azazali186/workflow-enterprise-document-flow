// Package cache is the distributed cache layer backed by Redis. It mirrors
// the classic db.Cache API used by the gateway: Set(key, value, ttl) and
// GetWithTTL(key) -> (value, ttl, err).
package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client is the Redis-backed distributed cache.
type Client struct {
	rdb redis.UniversalClient
	ctx context.Context
}

// New builds a cache client. ctx controls cache-wide cancellation.
func New(ctx context.Context, rdb redis.UniversalClient) *Client {
	return &Client{rdb: rdb, ctx: ctx}
}

// Set stores value under key with TTL (0 = no expiry).
func (c *Client) Set(key string, value any, ttl time.Duration) error {
	return c.rdb.Set(c.ctx, key, value, ttl).Err()
}

// SetJSON serialises v and stores it under key.
func (c *Client) SetJSON(key string, v any, ttl time.Duration) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.rdb.Set(c.ctx, key, raw, ttl).Err()
}

// Get returns the raw string stored under key.
func (c *Client) Get(key string) (string, error) {
	return c.rdb.Get(c.ctx, key).Result()
}

// GetWithTTL returns the value and its remaining TTL.
func (c *Client) GetWithTTL(key string) (string, time.Duration, error) {
	pipe := c.rdb.TxPipeline()
	g := pipe.Get(c.ctx, key)
	t := pipe.TTL(c.ctx, key)
	if _, err := pipe.Exec(c.ctx); err != nil {
		return "", 0, err
	}
	val, err := g.Result()
	if err != nil {
		return "", 0, err
	}
	ttl, err := t.Result()
	if err != nil {
		return "", 0, err
	}
	return val, ttl, nil
}

// GetJSON loads and unmarshals a cached JSON value.
func (c *Client) GetJSON(key string, out any) error {
	raw, err := c.rdb.Get(c.ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// Del removes one or more keys.
func (c *Client) Del(keys ...string) error {
	return c.rdb.Del(c.ctx, keys...).Err()
}

// SetNX stores a key only if it does not exist (distributed dedupe/locks).
func (c *Client) SetNX(key, value string) (bool, error) {
	return c.rdb.SetNX(c.ctx, key, value, 0).Result()
}

// Incr atomically increments and returns the new value.
func (c *Client) Incr(key string) (int64, error) {
	return c.rdb.Incr(c.ctx, key).Result()
}

// Expire sets a TTL on an existing key.
func (c *Client) Expire(key string, ttl time.Duration) error {
	return c.rdb.Expire(c.ctx, key, ttl).Err()
}

// SAdd adds members to a Redis set (used for permission sets).
func (c *Client) SAdd(key string, members ...string) error {
	return c.rdb.SAdd(c.ctx, key, members).Err()
}

// SMembers returns all members of a Redis set.
func (c *Client) SMembers(key string) ([]string, error) {
	return c.rdb.SMembers(c.ctx, key).Result()
}

// Exists reports whether the key exists.
func (c *Client) Exists(key string) (bool, error) {
	n, err := c.rdb.Exists(c.ctx, key).Result()
	return n > 0, err
}

// Ping verifies connectivity.
func (c *Client) Ping() error {
	return c.rdb.Ping(c.ctx).Err()
}
