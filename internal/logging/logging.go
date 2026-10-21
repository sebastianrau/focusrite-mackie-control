// Package logging provides the app-wide structured logger.
package logging

import (
	"log/slog"
	"os"
)

// EnvLevel names the environment variable that sets the log level
// (debug, info, warn, error). The default is info.
const EnvLevel = "MONITOR_CONTROL_LOG"

var handler slog.Handler

func init() {
	level := new(slog.LevelVar)
	if v := os.Getenv(EnvLevel); v != "" {
		if err := level.UnmarshalText([]byte(v)); err != nil {
			slog.Warn("invalid log level, using info", "env", EnvLevel, "value", v)
		}
	}
	handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
}

// New returns a logger tagged with the given component name.
func New(component string) *slog.Logger {
	return slog.New(handler).With("component", component)
}
