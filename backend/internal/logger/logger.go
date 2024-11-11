package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

var globalLogger *slog.Logger

func New(level string) error {
	var slogLevel slog.Level
	if err := slogLevel.UnmarshalText([]byte(level)); err != nil {
		return fmt.Errorf("cannot parse logger level: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	}))
	slog.SetDefault(logger)
	globalLogger = logger
	return nil
}

func Debug(_ context.Context, msg string, args ...any) {
	globalLogger.Debug(msg, args...)
}

func Info(_ context.Context, msg string, args ...any) {
	globalLogger.Info(msg, args...)
}

func Warn(_ context.Context, msg string, args ...any) {
	globalLogger.Warn(msg, args...)
}

func Error(_ context.Context, err error, msg string, args ...any) {
	globalLogger.Error(msg, append(args, "error", err)...)
}
