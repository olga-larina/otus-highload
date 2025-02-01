package tracing

import (
	"context"

	"github.com/olga-larina/otus-highload/pkg/model"
	"github.com/pckilgore/combuuid"
	"go.opentelemetry.io/otel/trace"
)

func GetContext(ctx context.Context, traceIdStr string, spanIdStr string, requestIdStr string) (context.Context, error) {
	// если передан requestId, то используем его, иначе - генерируем новый
	if len(requestIdStr) == 0 {
		requestIdStr = combuuid.NewUuid().String()
	}
	ctx = context.WithValue(ctx, model.RequestIdContextKey, requestIdStr)

	if len(traceIdStr) == 0 {
		return ctx, nil
	}

	var traceId trace.TraceID

	traceId, err := trace.TraceIDFromHex(traceIdStr)
	if err != nil {
		return ctx, err
	}

	spanId, err := trace.SpanIDFromHex(spanIdStr)
	if err != nil {
		return GetContextWithTraceId(ctx, traceId), err
	}

	return GetContextWithTraceIdSpanId(ctx, traceId, spanId), nil
}

func GetContextWithTraceId(ctx context.Context, traceId trace.TraceID) context.Context {
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceId,
	})
	return trace.ContextWithSpanContext(ctx, spanContext)
}

func GetContextWithTraceIdSpanId(ctx context.Context, traceId trace.TraceID, spanId trace.SpanID) context.Context {
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceId,
		SpanID:     spanId,
		Remote:     true,
		TraceFlags: trace.FlagsSampled,
	})
	return trace.ContextWithSpanContext(ctx, spanContext)
}
