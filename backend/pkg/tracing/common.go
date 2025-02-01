package tracing

import (
	"context"

	"github.com/olga-larina/otus-highload/pkg/model"
	"go.opentelemetry.io/otel/trace"
)

func GetTraceId(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		traceId := spanCtx.TraceID()
		return traceId.String()
	}
	return ""
}

func GetSpanId(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		spanId := spanCtx.SpanID()
		return spanId.String()
	}
	return ""
}

func GetRequestId(ctx context.Context) string {
	if requestId, ok := ctx.Value(model.RequestIdContextKey).(string); ok {
		return requestId
	}
	return ""
}
