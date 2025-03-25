package http_server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/olga-larina/otus-highload/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	METRICS_ROUTE = "/metrics"
)

var (
	// RED Metrics
	RequestsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "highload",
		Subsystem: "http_server",
		Name:      "requests_total",
	}, []string{"path", "status"})
	ErrorsCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "highload",
		Subsystem: "http_server",
		Name:      "errors_total",
	}, []string{"path", "status"})
	ResponseTimeHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "highload",
		Subsystem: "http_server",
		Name:      "response_time",
		Buckets:   prometheus.DefBuckets,
	}, []string{"path", "status"})
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		route := mux.CurrentRoute(r)
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			logger.Error(ctx, err, "failed obtaining path template")
			pathTemplate = "unknown"
		}

		timeStart := time.Now()
		var status int
		if isWebSocketRequest(r) {
			next.ServeHTTP(w, r)
		} else {
			rec := statusRecorder{w, 200}
			next.ServeHTTP(&rec, r)
			status = rec.status
		}
		elapsed := time.Since(timeStart)

		RequestsCounter.WithLabelValues(pathTemplate, strconv.Itoa(status)).Inc()
		ResponseTimeHistogram.WithLabelValues(pathTemplate, strconv.Itoa(status)).Observe(elapsed.Seconds())
		if status >= 400 {
			ErrorsCounter.WithLabelValues(pathTemplate, strconv.Itoa(status)).Inc()
		}
	})
}
