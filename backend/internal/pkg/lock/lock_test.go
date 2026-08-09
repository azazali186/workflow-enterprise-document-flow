package lock

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestLock(t *testing.T) *Lock {
	t.Helper()
	mr := miniredis.RunT(t)
	return New(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
}

func TestAcquireRelease(t *testing.T) {
	l := newTestLock(t)
	ctx := context.Background()

	ok, err := l.Acquire(ctx, "job:1", "token-a", time.Minute)
	if err != nil || !ok {
		t.Fatalf("first acquire should succeed, ok=%v err=%v", ok, err)
	}
	// A second contender cannot take the lock while held.
	ok, err = l.Acquire(ctx, "job:1", "token-b", time.Minute)
	if err != nil || ok {
		t.Fatalf("second acquire should fail, ok=%v err=%v", ok, err)
	}
	// Release with a wrong token is a no-op (still owned by token-a).
	if err := l.Release(ctx, "job:1", "token-b"); err != nil {
		t.Fatal(err)
	}
	ok, _ = l.Acquire(ctx, "job:1", "token-c", time.Minute)
	if ok {
		t.Fatal("lock must still be held after a foreign release")
	}
	// The owner can release.
	if err := l.Release(ctx, "job:1", "token-a"); err != nil {
		t.Fatal(err)
	}
	ok, _ = l.Acquire(ctx, "job:1", "token-c", time.Minute)
	if !ok {
		t.Fatal("lock should be free after owner release")
	}
}

func TestWithLockRunsExactlyOnce(t *testing.T) {
	l := newTestLock(t)
	ctx := context.Background()

	calls := 0
	err := l.WithLock(ctx, "job:2", "token-x", time.Minute, func() error {
		calls++
		// The lock is held inside the critical section.
		ok, _ := l.Acquire(ctx, "job:2", "token-y", time.Minute)
		if ok {
			t.Error("critical section must not be re-entrant")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}
