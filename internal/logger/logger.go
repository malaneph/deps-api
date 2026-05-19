package logger

import (
	"deps-api/internal/config"
	"log/slog"
	"os"
)

func New(cfg *config.Config) {
	if cfg.AppEnv == "development" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Default()
		return
	}

	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	l := slog.New(h)
	slog.SetDefault(l)
}
