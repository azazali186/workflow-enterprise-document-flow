// Package lock provides a Redis-backed distributed lock used to serialise
// critical sections across replicas (double-approval, outbox dispatch).
package lock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lock is a Redis distributed lock.
type Lock struct {
	rdb redis.UniversalClient
}

// New wraps a Redis client as a distributed lock.
func New(rdb redis.UniversalClient) *Lock {
	return &Lock{rdb: rdb}
}

// Acquire tries to take the lock. Returns false when already held.
func (l *Lock) Acquire(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	ok, err := l.rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// releaseScript atomically deletes the key only when the token matches,
// preventing a holder from releasing a lock it no longer owns.
var releaseScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)

// Release frees the lock if token still owns it.
func (l *Lock) Release(ctx context.Context, key, token string) error {
	return releaseScript.Run(ctx, l.rdb, []string{key}, token).Err()
}

// WithLock runs fn while holding the distributed lock, retrying acquisition
// with backoff until acquired or ctx is done.
func (l *Lock) WithLock(ctx context.Context, key, token string, ttl time.Duration, fn func() error) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		ok, err := l.Acquire(ctx, key, token, ttl)
		if err != nil {
			return err
		}
		if ok {
			defer l.Release(ctx, key, token) //nolint:errcheck
			return fn()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
