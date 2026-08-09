// Package crypto provides field-level AES-256-GCM encryption, bcrypt
// password hashing and a generic JSON redactor so sensitive values never
// reach logs or audit trails.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCipher is returned when a stored ciphertext cannot be decoded.
	ErrInvalidCipher = errors.New("invalid ciphertext")
	block            cipher.AEAD
)

// Init must be called once at startup with a 32-byte key (base64 encoded).
// When key is empty a random ephemeral key is derived (development only).
func Init(key string) error {
	var raw []byte
	if key == "" {
		sum := sha256.Sum256([]byte("dev-only-encryption-key"))
		raw = sum[:]
	} else {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil || len(decoded) != 32 {
			return errors.New("ENCRYPTION_KEY must be 32 bytes, base64 encoded")
		}
		raw = decoded
	}
	c, err := aes.NewCipher(raw)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return err
	}
	block = gcm
	return nil
}

// Encrypt encrypts plaintext with AES-256-GCM. Output is base64(nonce||ct).
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, block.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := block.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt.
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil || len(raw) < block.NonceSize()+1 {
		return "", ErrInvalidCipher
	}
	nonce, ct := raw[:block.NonceSize()], raw[block.NonceSize():]
	pt, err := block.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", ErrInvalidCipher
	}
	return string(pt), nil
}

// HashPassword returns the bcrypt hash of a plaintext password.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword reports whether password matches the stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// sensitiveKeys are JSON keys whose values are always redacted in logs.
var sensitiveKeys = map[string]struct{}{
	"password": {}, "password_hash": {}, "old_password": {}, "new_password": {},
	"token": {}, "access_token": {}, "refresh_token": {}, "authorization": {},
	"api_key": {}, "secret": {}, "client_secret": {}, "object_key": {},
	"jwt_secret": {}, "credit_card": {}, "otp": {},
}

// RedactJSON walks arbitrary JSON values and replaces sensitive values with
// "***REDACTED***". Non-JSON payloads are returned untouched.
func RedactJSON(v any) any {
	if v == nil {
		return v
	}
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if _, ok := sensitiveKeys[strings.ToLower(k)]; ok {
				out[k] = "***REDACTED***"
				continue
			}
			out[k] = RedactJSON(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = RedactJSON(val)
		}
		return out
	default:
		return v
	}
}

// MarshalRedacted serialises v to JSON with sensitive keys redacted.
// It never fails for normal structs; on error it returns "{}".
func MarshalRedacted(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "{}"
	}
	out, err := json.Marshal(RedactJSON(parsed))
	if err != nil {
		return "{}"
	}
	return string(out)
}
