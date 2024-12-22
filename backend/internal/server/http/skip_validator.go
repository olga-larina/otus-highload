package internalhttp

import (
	"net/http"
	"strings"
)

func SkipValidatorForMetricsAndInternal(validator func(next http.Handler) http.Handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" || strings.HasPrefix(r.URL.Path, "/internal") {
			next.ServeHTTP(w, r)
			return
		}
		validator(next).ServeHTTP(w, r)
	})
}
