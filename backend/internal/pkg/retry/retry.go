// Package retry provides exponential backoff with jitter for transient
// operations such as NATS publishes and external calls.
package retry

import (
	"context"
	"math/rand/v2"
	"time"
)

// Options tunes the retry loop.
type Options struct {
	MaxAttempts int           // total attempts, including the first (default 3)
	BaseDelay   time.Duration // first backoff (default 100ms)
	MaxDelay    time.Duration // backoff ceiling (default 2s)
	Factor      float64       // multiplier between attempts (default 2)
	Jitter      float64       // 0..1 randomisation (default 0.2)
}

// DefaultOptions returns a sane production configuration.
func DefaultOptions() Options {
	return Options{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		Factor:      2,
		Jitter:      0.2,
	}
}

// ErrAborted is returned when the context is cancelled before completion.
var ErrAborted = context.Canceled

// Do runs fn with exponential backoff. fn should return retryable errors for
// transient failures; permanent errors should be wrapped with retry.Permanent
// to stop immediately.
func Do(ctx context.Context, opts Options, fn func(attempt int) error) error {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.BaseDelay <= 0 {
		opts.BaseDelay = 100 * time.Millisecond
	}
	if opts.MaxDelay <= 0 {
		opts.MaxDelay = 2 * time.Second
	}
	if opts.Factor <= 1 {
		opts.Factor = 2
	}
	delay := opts.BaseDelay
	var err error
	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		if err = fn(attempt); err == nil {
			return nil
		}
		if isPermanent(err) {
			return err
		}
		if attempt == opts.MaxAttempts {
			break
		}
		delay = backoffDelay(delay, opts)
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
	return err
}

func backoffDelay(prev time.Duration, o Options) time.Duration {
	next := time.Duration(float64(prev) * o.Factor)
	if next > o.MaxDelay {
		next = o.MaxDelay
	}
	if o.Jitter > 0 {
		j := 1 + (rand.Float64()*2-1)*o.Jitter //nolint:gosec // G404: jitter is a timing spread, not a secret
		next = time.Duration(float64(next) * j)
	}
	return next
}

type permanentError struct{ err error }

func (p permanentError) Error() string { return p.err.Error() }
func (p permanentError) Unwrap() error { return p.err }

// Permanent wraps err so retry.Do stops immediately.
func Permanent(err error) error { return permanentError{err: err} }

func isPermanent(err error) bool {
	_, ok := err.(permanentError)
	return ok
}
