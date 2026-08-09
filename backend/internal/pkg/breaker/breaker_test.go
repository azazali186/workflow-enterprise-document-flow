package breaker

import (
	"errors"
	"testing"
	"time"
)

func TestOpensAfterThreshold(t *testing.T) {
	b := New(Options{FailureThreshold: 3, Cooldown: 50 * time.Millisecond, HalfOpenMax: 1})
	var err error
	for i := 0; i < 3; i++ {
		err = b.Execute(func() error { return errors.New("boom") })
	}
	if err == nil || b.State() != Open {
		t.Fatalf("expected open state, got %v", b.State())
	}
	// While open, calls fail fast without invoking fn.
	called := false
	err = b.Execute(func() error { called = true; return nil })
	if err != ErrOpen {
		t.Fatalf("expected ErrOpen, got %v", err)
	}
	if called {
		t.Fatal("fn must not run while open")
	}
}

func TestHalfOpenRecoversAfterCooldown(t *testing.T) {
	b := New(Options{FailureThreshold: 2, Cooldown: 30 * time.Millisecond, HalfOpenMax: 1})
	_ = b.Execute(func() error { return errors.New("x") })
	_ = b.Execute(func() error { return errors.New("x") })
	if b.State() != Open {
		t.Fatal("expected open")
	}
	time.Sleep(40 * time.Millisecond)
	// Probe succeeds → back to closed.
	if err := b.Execute(func() error { return nil }); err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	if b.State() != Closed {
		t.Fatalf("expected closed after recovery, got %v", b.State())
	}
	// Normal traffic flows.
	if err := b.Execute(func() error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHalfOpenFailureReopens(t *testing.T) {
	b := New(Options{FailureThreshold: 2, Cooldown: 20 * time.Millisecond, HalfOpenMax: 1})
	_ = b.Execute(func() error { return errors.New("x") })
	_ = b.Execute(func() error { return errors.New("x") })
	time.Sleep(30 * time.Millisecond)
	_ = b.Execute(func() error { return errors.New("still broken") })
	if b.State() != Open {
		t.Fatalf("expected open again, got %v", b.State())
	}
}

func TestSuccessResetsFailureCount(t *testing.T) {
	b := New(Options{FailureThreshold: 3, Cooldown: time.Second, HalfOpenMax: 1})
	_ = b.Execute(func() error { return errors.New("x") })
	_ = b.Execute(func() error { return nil })
	_ = b.Execute(func() error { return errors.New("x") })
	_ = b.Execute(func() error { return errors.New("x") })
	if b.State() != Closed {
		t.Fatalf("expected closed (failures reset), got %v", b.State())
	}
}
