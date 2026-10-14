// Package logging configures the application's structured logger.
//
// It uses the standard library's log/slog so the process emits machine-readable
// logs (one JSON object per line by default) that can be shipped to a log
// aggregator later without code changes.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Init builds the default slog logger from the environment and installs it as
// the process-wide default (slog.SetDefault).
//
//	LOG_FORMAT  json | text                (default json)
//	LOG_LEVEL   debug | info | warn | error (default info)
func Init() *slog.Logger {
	opts := &slog.HandlerOptions{Level: levelFromEnv()}

	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LOG_FORMAT")), "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// levelFromEnv maps LOG_LEVEL onto an slog.Level, defaulting to info.
func levelFromEnv() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
