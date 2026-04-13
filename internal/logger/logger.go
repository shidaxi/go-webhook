package logger

import (
	"fmt"

	"go.uber.org/zap"
)

var global *zap.Logger

// Init initializes the global logger.
// format: "json" (production) or "text" (development).
func Init(format string) error {
	var (
		l   *zap.Logger
		err error
	)

	switch format {
	case "text":
		l, err = zap.NewDevelopment()
	case "json", "":
		l, err = zap.NewProduction()
	default:
		return fmt.Errorf("unknown log format: %q (expected json or text)", format)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	global = l
	return nil
}

// L returns the global logger. Panics if Init was not called.
func L() *zap.Logger {
	if global == nil {
		// Fallback to nop so tests without Init don't panic
		return zap.NewNop()
	}
	return global
}

// Sync flushes any buffered log entries.
func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}
