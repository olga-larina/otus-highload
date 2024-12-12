package internalhttp

import (
	"net/http"
)

func SkipValidatorForMetrics(validator func(next http.Handler) http.Handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		validator(next).ServeHTTP(w, r)
	})
}
