// Package breaker implements a concurrency-safe circuit breaker protecting
// downstream dependencies (NATS, external services) from cascading failure.
package breaker

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen is returned while the circuit is open (fail fast).
var ErrOpen = errors.New("circuit breaker is open")

// State is the breaker state machine position.
type State int

const (
	// Closed lets all calls through.
	Closed State = iota
	// Open fails fast for the cooldown window.
	Open
	// HalfOpen admits a limited number of probe calls.
	HalfOpen
)

// Options configures a breaker.
type Options struct {
	FailureThreshold int           // consecutive failures to open (default 5)
	Cooldown         time.Duration // time in Open before probing (default 10s)
	HalfOpenMax      int           // probe calls allowed in HalfOpen (default 1)
}

// DefaultOptions returns a production-default breaker config.
func DefaultOptions() Options {
	return Options{FailureThreshold: 5, Cooldown: 10 * time.Second, HalfOpenMax: 1}
}

// Breaker guards a single dependency.
type Breaker struct {
	mu        sync.Mutex
	state     State
	failures  int
	probes    int
	successes int
	openedAt  time.Time
	opts      Options
}

// New builds a breaker. Zero options fall back to defaults.
func New(opts Options) *Breaker {
	if opts.FailureThreshold <= 0 {
		opts.FailureThreshold = 5
	}
	if opts.Cooldown <= 0 {
		opts.Cooldown = 10 * time.Second
	}
	if opts.HalfOpenMax <= 0 {
		opts.HalfOpenMax = 1
	}
	return &Breaker{state: Closed, opts: opts}
}

// State returns the current state (for metrics/tests).
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Execute runs fn if the circuit permits; otherwise returns ErrOpen.
func (b *Breaker) Execute(fn func() error) error {
	if !b.allow() {
		return ErrOpen
	}
	err := fn()
	if err != nil {
		b.recordFailure()
	} else {
		b.recordSuccess()
	}
	return err
}

func (b *Breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case Closed:
		return true
	case Open:
		if time.Since(b.openedAt) >= b.opts.Cooldown {
			b.state = HalfOpen
			b.probes = 0
			b.successes = 0
			b.probes++
			return true
		}
		return false
	default: // HalfOpen
		if b.probes >= b.opts.HalfOpenMax {
			return false
		}
		b.probes++
		return true
	}
}

func (b *Breaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case Closed:
		b.failures++
		if b.failures >= b.opts.FailureThreshold {
			b.openLocked()
		}
	case HalfOpen:
		b.openLocked()
	}
}

func (b *Breaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case HalfOpen:
		b.successes++
		if b.successes >= b.opts.HalfOpenMax {
			b.state = Closed
			b.failures = 0
		}
	case Closed:
		b.failures = 0
	}
}

func (b *Breaker) openLocked() {
	b.state = Open
	b.openedAt = time.Now()
	b.failures = 0
	b.probes = 0
}
