package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParse(t *testing.T) {
	Init("test-secret-key-1234567890", time.Hour)
	token, exp, err := Generate("user-1", "a@b.c", []string{"r1"})
	if err != nil {
		t.Fatal(err)
	}
	if exp.Before(time.Now()) {
		t.Fatal("expiry in the past")
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" || claims.Email != "a@b.c" {
		t.Errorf("claims mismatch: %+v", claims)
	}
	if len(claims.RoleIDs) != 1 || claims.RoleIDs[0] != "r1" {
		t.Errorf("roles mismatch: %v", claims.RoleIDs)
	}
}

func TestParseFailsForWrongSecret(t *testing.T) {
	Init("secret-a", time.Hour)
	token, _, _ := Generate("u1", "a@b.c", nil)
	Init("secret-b", time.Hour)
	if _, err := ParseToken(token); err == nil {
		t.Fatal("expected error with wrong secret")
	}
}

func TestParseFailsForGarbage(t *testing.T) {
	Init("secret-a", time.Hour)
	if _, err := ParseToken("garbage.token.value"); err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestParseFailsWhenExpired(t *testing.T) {
	Init("secret-a", -time.Minute) // negative TTL → already expired
	token, _, err := Generate("u1", "a@b.c", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(token); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseRejectsForeignIssuer(t *testing.T) {
	Init("secret-a", time.Hour)
	// A validly signed token from a different service sharing the secret must
	// still be rejected: the issuer claim is validated on parse.
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
		"user_id": "u1",
		"iss":     "some-other-service",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(s); err == nil {
		t.Fatal("expected error for a token issued by another service")
	}
}

func TestParseRejectsNoneAlgorithm(t *testing.T) {
	Init("secret-a", time.Hour)
	// The "none" algorithm must never be accepted.
	raw := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjoicDAiLCJpc3MiOiJkb2N1Zmxvdy1hcGktZ2F0ZXdheSIsImV4cCI6OTk5OTk5OTk5OX0."
	if _, err := ParseToken(raw); err == nil {
		t.Fatal("expected error for alg=none token")
	}
}
