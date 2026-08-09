package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// These tests run with L == nil (Init is never called), which is exactly the
// early-boot window where a nil dereference used to panic on config failures.

func TestNilLoggerLevelCallsDoNotPanic(t *testing.T) {
	Debug("x")
	Info("x")
	Warn("x")
	Error("x")
	Sync()
}

func TestFatalIfNilErrorIsNoOp(t *testing.T) {
	FatalIf(nil, "nothing to report") // must not panic or exit
}

func TestFallbackFatalCarriesTheRealError(t *testing.T) {
	var buf bytes.Buffer
	writeFallbackFatal(&buf, "load config",
		zap.Error(errors.New("JWT_SECRET is required")))

	got := buf.String()
	if !strings.Contains(got, "FATAL load config") {
		t.Fatalf("missing FATAL prefix+msg, got %q", got)
	}
	if !strings.Contains(got, "JWT_SECRET is required") {
		t.Fatalf("missing the underlying error, got %q", got)
	}
}
