package trace

import (
	"context"
	"testing"
)

func TestWithIDAndID(t *testing.T) {
	ctx := WithID(context.Background(), "abc-123")
	if got := ID(ctx); got != "abc-123" {
		t.Fatalf("got %q", got)
	}
}

func TestIDAbsent(t *testing.T) {
	if got := ID(context.Background()); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestWithEmptyIDReturnsSameContext(t *testing.T) {
	ctx := context.Background()
	if got := WithID(ctx, ""); got != ctx {
		t.Fatal("empty id must not wrap the context")
	}
}

func TestWithIDDoesNotMutateParent(t *testing.T) {
	parent := context.Background()
	child := WithID(parent, "x")
	if ID(parent) != "" {
		t.Fatal("parent context must stay clean")
	}
	if ID(child) != "x" {
		t.Fatal("child must carry the id")
	}
}
