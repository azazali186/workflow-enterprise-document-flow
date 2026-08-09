package service

import (
	"crypto/md5" //nolint:gosec // G501: session fingerprint, not auth
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/constant"
	"github.com/aeroxe/docu-flow/backend/internal/pkg/cache"
)

// SSOKey returns the Redis key tracking a user's active session.
func SSOKey(userID string) string { return constant.AdminToken + userID }

// SSOValue derives the stored fingerprint for a token (md5 of token+user).
func SSOValue(tokenStr, userID string) string {
	sum := md5.Sum([]byte(tokenStr + userID)) //nolint:gosec // fingerprint, not auth
	return hex.EncodeToString(sum[:])
}

// RenewIfNeeded extends the session TTL when it drops below half of ttl.
// Returns true when the session still exists and was validated.
func RenewIfNeeded(c *cache.Client, userID, fingerprint string, ttl time.Duration) (bool, error) {
	str, remaining, err := c.GetWithTTL(SSOKey(userID))
	if err != nil {
		return false, err
	}
	if str != fingerprint {
		return false, nil
	}
	if remaining < ttl/2 {
		if err := c.Set(SSOKey(userID), fingerprint, ttl); err != nil {
			return false, err
		}
	}
	return true, nil
}

// Describe is a tiny helper used in logs.
func Describe(v any) string { return fmt.Sprintf("%v", v) }
