package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySucceedsOnSecondAttempt(t *testing.T) {
	attempts := 0
	opts := DefaultOptions()
	opts.MaxAttempts = 3
	opts.BaseDelay = 1 * time.Millisecond
	err := Do(context.Background(), opts, func(_ int) error {
		attempts++
		if attempts == 1 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestRetryGivesUpAfterMaxAttempts(t *testing.T) {
	attempts := 0
	opts := DefaultOptions()
	opts.MaxAttempts = 4
	opts.BaseDelay = 1 * time.Millisecond
	err := Do(context.Background(), opts, func(_ int) error {
		attempts++
		return errors.New("always failing")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 4 {
		t.Errorf("attempts = %d, want 4", attempts)
	}
}

func TestPermanentStopsImmediately(t *testing.T) {
	attempts := 0
	opts := DefaultOptions()
	opts.MaxAttempts = 5
	err := Do(context.Background(), opts, func(_ int) error {
		attempts++
		return Permanent(errors.New("do not retry"))
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := DefaultOptions()
	opts.MaxAttempts = 10
	opts.BaseDelay = 5 * time.Millisecond
	err := Do(ctx, opts, func(_ int) error { return errors.New("x") })
	if err == nil {
		t.Fatal("expected context error")
	}
}
