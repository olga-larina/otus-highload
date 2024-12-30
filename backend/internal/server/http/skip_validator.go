package internalhttp

import (
	"net/http"
)

func SkipValidatorForManualRoutes(validator func(next http.Handler) http.Handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == METRICS_ROUTE || r.URL.Path == CACHE_INVALIDATE_ROUTE || r.URL.Path == POST_FEED_ROUTE {
			next.ServeHTTP(w, r)
			return
		}
		validator(next).ServeHTTP(w, r)
	})
}
