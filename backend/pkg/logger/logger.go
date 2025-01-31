package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/olga-larina/otus-highload/pkg/tracing"
)

var globalLogger *slog.Logger

const (
	KEY_ERR     = "error"
	KEY_TRACE   = "traceId"
	KEY_SPAN    = "spanId"
	KEY_REQUEST = "requestId"
)

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

func Debug(ctx context.Context, msg string, args ...any) {
	globalLogger.Debug(msg, populateArgs(ctx, args)...)
}

func Info(ctx context.Context, msg string, args ...any) {
	globalLogger.Info(msg, populateArgs(ctx, args)...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	globalLogger.Warn(msg, populateArgs(ctx, args)...)
}

func Error(ctx context.Context, err error, msg string, args ...any) {
	globalLogger.Error(msg, append(populateArgs(ctx, args), KEY_ERR, err)...)
}

func populateArgs(ctx context.Context, args []any) []any {
	if traceId := tracing.GetTraceId(ctx); len(traceId) > 0 {
		args = append(args, KEY_TRACE, traceId)
	}
	if spanId := tracing.GetSpanId(ctx); len(spanId) > 0 {
		args = append(args, KEY_SPAN, spanId)
	}
	if requestId := tracing.GetRequestId(ctx); len(requestId) > 0 {
		args = append(args, KEY_REQUEST, requestId)
	}
	return args
}
