package internalhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/olga-larina/otus-highload/backend/internal/logger"
)

// логирование на выходе сервиса, есть респонс, но ещё нет статуса запроса
func StrictLoggingMiddleware(f nethttp.StrictHTTPHandlerFunc, operationID string) nethttp.StrictHTTPHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		startTime := time.Now()
		response, err := f(ctx, w, r, request)
		elapsed := time.Since(startTime)

		var errValue string
		if err != nil {
			errValue = err.Error()
		}

		var responseValue string
		if response != nil {
			responseBytes, err := json.Marshal(response)
			if responseBytes != nil && err == nil {
				responseValue = string(responseBytes)
			}
		}

		logger.Info(r.Context(), "http request",
			"operationID", operationID,
			"ip", r.RemoteAddr,
			"startTime", startTime.Format(timeLayout),
			"method", r.Method,
			"path", r.URL.Path,
			"version", r.Proto,
			"latency", elapsed.Milliseconds(),
			"userAgent", r.UserAgent(),
			"error", errValue,
			"response", responseValue,
		)

		return response, err
	}
}

// логирование на выходе http, есть статус ответа, но респонс уже записан в writer
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		var status int
		if isWebSocketRequest(r) {
			next.ServeHTTP(w, r)
		} else {
			rec := statusRecorder{w, 200}
			next.ServeHTTP(&rec, r)
			status = rec.status
		}
		elapsed := time.Since(startTime)

		logger.Info(r.Context(), "http request",
			"ip", r.RemoteAddr,
			"startTime", startTime.Format(timeLayout),
			"method", r.Method,
			"path", r.URL.Path,
			"version", r.Proto,
			"statusCode", status,
			"latency", elapsed.Milliseconds(),
			"userAgent", r.UserAgent(),
		)
	})
}

func isWebSocketRequest(r *http.Request) bool {
	return r.Header.Get("Upgrade") == "websocket" &&
		r.Header.Get("Connection") == "Upgrade" &&
		r.Header.Get("Sec-WebSocket-Key") != "" &&
		r.Header.Get("Sec-WebSocket-Version") == "13"
}
