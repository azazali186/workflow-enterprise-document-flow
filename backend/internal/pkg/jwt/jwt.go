// Package jwt issues and parses signed JSON Web Tokens.
package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrTokenInvalid wraps all parse failures.
	ErrTokenInvalid = errors.New("invalid token")
	secret          []byte
	accessTTL       time.Duration
)

// Claims carries identity data embedded in a token.
type Claims struct {
	UserID  string   `json:"user_id"`
	Email   string   `json:"email"`
	RoleIDs []string `json:"role_ids,omitempty"`
	jwt.RegisteredClaims
}

// Init configures the signing secret and default access TTL.
func Init(secretKey string, ttl time.Duration) {
	secret = []byte(secretKey)
	accessTTL = ttl
}

// Generate signs a new access token for a user.
func Generate(userID, email string, roleIDs []string) (string, time.Time, error) {
	if len(secret) == 0 {
		return "", time.Time{}, ErrTokenInvalid
	}
	exp := time.Now().Add(accessTTL)
	claims := Claims{
		UserID:  userID,
		Email:   email,
		RoleIDs: roleIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    "docuflow-api-gateway",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString(secret)
	return s, exp, err
}

// ParseToken validates a token string and returns its claims. Only HS256
// tokens signed by this service and carrying the expected issuer are
// accepted, so tokens minted by other services sharing a signing key (or an
// accidental same-secret deployment) cannot be replayed here.
func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return secret, nil
	}, jwt.WithIssuer("docuflow-api-gateway"), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// CSRFFor derives the double-submit CSRF value bound to a token: a keyed hash
// of the token under the signing secret. The SPA receives it in the login/
// refresh response body and echoes it in the X-CSRF-Token header; the CSRF
// middleware recomputes it from the HttpOnly session cookie, so no additional
// cookie or server state is needed. An attacker cannot forge it without the
// token, which JS cannot read.
func CSRFFor(token string) string {
	m := hmac.New(sha256.New, secret)
	_, _ = m.Write([]byte(token))
	return hex.EncodeToString(m.Sum(nil))[:32]
}
