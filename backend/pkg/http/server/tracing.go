package http_server

import (
	"net/http"

	"github.com/gorilla/mux"
	internalhttp "github.com/olga-larina/otus-highload/pkg/http"
	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/olga-larina/otus-highload/pkg/tracing"
	"go.opentelemetry.io/otel"
)

func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := tracing.GetContext(
			r.Context(),
			r.Header.Get(internalhttp.HEADER_TRACE_ID),
			r.Header.Get(internalhttp.HEADER_SPAN_ID),
			r.Header.Get(internalhttp.HEADER_REQUEST_ID),
		)
		if err != nil {
			logger.Error(ctx, err, "failed obtaining context")
		}
		if ctx == nil {
			ctx = r.Context()
		}

		route := mux.CurrentRoute(r)
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			logger.Error(ctx, err, "failed obtaining path template")
			pathTemplate = "unknown"
		}

		ctxWithSpan, span := otel.Tracer("default").Start(ctx, pathTemplate)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctxWithSpan))
	})
}
