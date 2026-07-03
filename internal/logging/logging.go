package logging

import (
	"log/slog"
	"os"
	"strings"
)

// DefaultAnvilSignerKey is Anvil/Hardhat account #0 (0xf39F…2266). Public test fixture only.
const DefaultAnvilSignerKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

// NewJSONHandler returns a JSON slog handler with source location (file, function, line).
func NewJSONHandler(level slog.Level) slog.Handler {
	return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
}

// NewTextHandler returns a text slog handler with source location (local dev).
func NewTextHandler(level slog.Level) slog.Handler {
	return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
}

// LevelFromEnv parses LOG_LEVEL (debug, info, warn, error). Defaults to info.
func LevelFromEnv() slog.Level {
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

// SetupDefault configures the process-wide default logger.
func SetupDefault() {
	slog.SetDefault(slog.New(NewJSONHandler(LevelFromEnv())))
}
