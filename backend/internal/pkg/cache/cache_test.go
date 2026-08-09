package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) *Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return New(context.Background(), redis.NewClient(&redis.Options{Addr: mr.Addr()}))
}

func TestSetGetRoundtrip(t *testing.T) {
	c := newTestCache(t)
	if err := c.Set("k", "v", time.Minute); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get("k")
	if err != nil {
		t.Fatal(err)
	}
	if got != "v" {
		t.Fatalf("got %q", got)
	}
}

func TestGetWithTTL(t *testing.T) {
	c := newTestCache(t)
	if err := c.Set("k", "v", time.Hour); err != nil {
		t.Fatal(err)
	}
	val, ttl, err := c.GetWithTTL("k")
	if err != nil {
		t.Fatal(err)
	}
	if val != "v" {
		t.Fatalf("got %q", val)
	}
	if ttl <= 0 || ttl > time.Hour {
		t.Fatalf("ttl out of range: %v", ttl)
	}
}

func TestSetNX(t *testing.T) {
	c := newTestCache(t)
	ok, err := c.SetNX("lock", "1")
	if err != nil || !ok {
		t.Fatalf("first SetNX should win, ok=%v err=%v", ok, err)
	}
	ok, err = c.SetNX("lock", "2")
	if err != nil || ok {
		t.Fatalf("second SetNX should lose, ok=%v err=%v", ok, err)
	}
}

func TestIncrAndExpire(t *testing.T) {
	c := newTestCache(t)
	for i := 1; i <= 3; i++ {
		n, err := c.Incr("counter")
		if err != nil {
			t.Fatal(err)
		}
		if n != int64(i) {
			t.Fatalf("expected %d, got %d", i, n)
		}
	}
	if err := c.Expire("counter", time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestDelAndExists(t *testing.T) {
	c := newTestCache(t)
	if err := c.Set("k", "v", 0); err != nil {
		t.Fatal(err)
	}
	exists, err := c.Exists("k")
	if err != nil || !exists {
		t.Fatalf("expected exists, ok=%v err=%v", exists, err)
	}
	if err := c.Del("k"); err != nil {
		t.Fatal(err)
	}
	exists, _ = c.Exists("k")
	if exists {
		t.Fatal("key should be gone")
	}
}

func TestJSONHelpers(t *testing.T) {
	c := newTestCache(t)
	type obj struct {
		A string `json:"a"`
		B int    `json:"b"`
	}
	if err := c.SetJSON("o", obj{A: "x", B: 7}, time.Minute); err != nil {
		t.Fatal(err)
	}
	var out obj
	if err := c.GetJSON("o", &out); err != nil {
		t.Fatal(err)
	}
	if out != (obj{A: "x", B: 7}) {
		t.Fatalf("got %+v", out)
	}
}

func TestSetMembers(t *testing.T) {
	c := newTestCache(t)
	if err := c.SAdd("set", "a", "b", "c"); err != nil {
		t.Fatal(err)
	}
	members, err := c.SMembers("set")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 3 {
		t.Fatalf("expected 3 members, got %v", members)
	}
}

func TestPing(t *testing.T) {
	c := newTestCache(t)
	if err := c.Ping(); err != nil {
		t.Fatal(err)
	}
}
