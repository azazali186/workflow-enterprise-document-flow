package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	if err := Init(base64.StdEncoding.EncodeToString(make([]byte, 32))); err != nil {
		t.Fatal(err)
	}
	plain := "s3://bucket/private/object-key-123"
	ct, err := Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if ct == plain || strings.Contains(ct, plain) {
		t.Fatal("ciphertext leaks plaintext")
	}
	back, err := Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}
	if back != plain {
		t.Errorf("round trip mismatch: %q", back)
	}
}

func TestDecryptInvalid(t *testing.T) {
	if _, err := Decrypt("not-base64!!!"); err == nil {
		t.Error("expected error for invalid ciphertext")
	}
	if _, err := Decrypt(""); err != nil {
		t.Error("empty string should decode to empty")
	}
}

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "s3cret-pass" {
		t.Fatal("hash leaks password")
	}
	if !CheckPassword(hash, "s3cret-pass") {
		t.Error("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("wrong password accepted")
	}
}

func TestRedactJSON(t *testing.T) {
	payload := map[string]any{
		"email":         "a@b.c",
		"password":      "hunter2",
		"token":         "abc",
		"nested":        map[string]any{"api_key": "k", "ok": true},
		"refresh_token": "xyz",
	}
	out := RedactJSON(payload).(map[string]any)
	if out["password"] != "***REDACTED***" {
		t.Errorf("password not redacted: %v", out["password"])
	}
	if out["token"] != "***REDACTED***" || out["refresh_token"] != "***REDACTED***" {
		t.Error("token fields not redacted")
	}
	if out["email"] != "a@b.c" {
		t.Error("non-sensitive value should stay")
	}
	nested := out["nested"].(map[string]any)
	if nested["api_key"] != "***REDACTED***" {
		t.Error("nested api_key not redacted")
	}
	if nested["ok"] != true {
		t.Error("nested boolean changed")
	}
}

func TestMarshalRedactedNeverFails(t *testing.T) {
	if s := MarshalRedacted(map[string]any{"password": "x"}); !strings.Contains(s, "REDACTED") {
		t.Errorf("expected redacted json, got %s", s)
	}
	if s := MarshalRedacted(nil); s == "" {
		t.Error("nil should marshal")
	}
}
