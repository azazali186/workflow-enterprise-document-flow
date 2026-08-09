// Package uuidx provides UUID v7 helpers. UUID v7 is time-ordered, which
// enables keyset (cursor) pagination directly on the primary key.
package uuidx

import "github.com/google/uuid"

// New returns a new UUID v7 as a string.
func New() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fall back to a random v4 when v7 is unavailable (never happens in
		// practice; keeps the signature total).
		return uuid.NewString()
	}
	return id.String()
}

// NewID returns a raw uuid.UUID v7 value.
func NewID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}

// Valid reports whether s is a well-formed UUID (v7 or otherwise).
func Valid(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
