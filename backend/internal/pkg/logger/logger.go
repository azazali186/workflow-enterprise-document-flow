// Package logger provides a production-grade zap-based structured logger.
package logger

import (
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

// Debug logs a debug message with optional fields.
func Debug(msg string, fields ...zap.Field) { L.Debug(msg, fields...) }

// Info logs an info message with optional fields.
func Info(msg string, fields ...zap.Field) { L.Info(msg, fields...) }

// Warn logs a warning message with optional fields.
func Warn(msg string, fields ...zap.Field) { L.Warn(msg, fields...) }

// Error logs an error message with optional fields.
func Error(msg string, fields ...zap.Field) { L.Error(msg, fields...) }

// Fatal logs a fatal message and exits the process.
func Fatal(msg string, fields ...zap.Field) { L.Fatal(msg, fields...) }

// FatalIf panics-safe exit helper: logs err and terminates when err != nil.
func FatalIf(err error, msg string) {
	if err != nil {
		L.Fatal(msg, zap.Error(err))
	}
}

// Sync flushes buffered log entries. Call during graceful shutdown.
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
	_ = os.Stderr.Sync()
}
