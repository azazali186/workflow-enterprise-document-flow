// Package logger provides a production-grade zap-based structured logger.
package logger

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// L is the process-wide structured logger.
	L *zap.Logger
	// S is a sugar wrapper around L for quick formatting calls.
	S *zap.SugaredLogger
)

// Init builds the global logger. level is one of debug|info|warn|error.
// In production the output is JSON; in development it is human readable.
func Init(level string, production bool) error {
	cfg := zap.NewProductionConfig()
	if !production {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	lvl, err := zapcore.ParseLevel(level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	z, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		return err
	}
	L = z
	S = z.Sugar()
	return nil
}

// Debug logs a debug message with optional fields. Nil-safe before Init runs.
func Debug(msg string, fields ...zap.Field) {
	if L != nil {
		L.Debug(msg, fields...)
	}
}

// Info logs an info message with optional fields. Nil-safe before Init runs.
func Info(msg string, fields ...zap.Field) {
	if L != nil {
		L.Info(msg, fields...)
	}
}

// Warn logs a warning message with optional fields. Nil-safe before Init runs.
func Warn(msg string, fields ...zap.Field) {
	if L != nil {
		L.Warn(msg, fields...)
	}
}

// Error logs an error message with optional fields. Nil-safe before Init runs.
func Error(msg string, fields ...zap.Field) {
	if L != nil {
		L.Error(msg, fields...)
	}
}

// Fatal logs a fatal message and exits the process. Nil-safe: early boot
// failures (before Init) fall back to stderr so the real error is reported
// instead of panicking on a nil logger.
func Fatal(msg string, fields ...zap.Field) {
	if L == nil {
		fallbackFatal(msg, fields...)
	}
	L.Fatal(msg, fields...)
}

// FatalIf logs err and terminates when err != nil. Nil-safe like Fatal.
func FatalIf(err error, msg string) {
	if err != nil {
		Fatal(msg, zap.Error(err))
	}
}

// formatField renders a zap field for the pre-init stderr fallback. zap's
// own String() method is shadowed by the Field.String member, so format the
// public members directly — this path only runs during early-boot failures.
func formatField(f zap.Field) string {
	if f.Type == zapcore.BoolType {
		return fmt.Sprintf("%s=%t", f.Key, f.Integer != 0)
	}
	if f.Interface != nil {
		return fmt.Sprintf("%s=%v", f.Key, f.Interface)
	}
	if f.String != "" {
		return fmt.Sprintf("%s=%s", f.Key, f.String)
	}
	if f.Integer != 0 {
		return fmt.Sprintf("%s=%d", f.Key, f.Integer)
	}
	return f.Key
}

// writeFallbackFatal formats a fatal message the same way regardless of the
// destination so tests can assert on the output without triggering os.Exit.
func writeFallbackFatal(w io.Writer, msg string, fields ...zap.Field) {
	_, _ = fmt.Fprintf(w, "FATAL %s", msg)
	for _, f := range fields {
		_, _ = fmt.Fprintf(w, " %s", formatField(f))
	}
	_, _ = fmt.Fprintln(w)
}

// fallbackFatal reports a fatal startup error on stderr and exits.
func fallbackFatal(msg string, fields ...zap.Field) {
	writeFallbackFatal(os.Stderr, msg, fields...)
	os.Exit(1)
}

// Sync flushes buffered log entries. Call during graceful shutdown.
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
	_ = os.Stderr.Sync()
}
