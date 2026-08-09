package middleware

import (
	"os"
	"testing"

	"github.com/aeroxe/docu-flow/backend/internal/pkg/logger"
)

func TestMain(m *testing.M) {
	// Middleware error paths log through the global zap logger.
	_ = logger.Init("error", false)
	os.Exit(m.Run())
}
