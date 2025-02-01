package http_server

import (
	"net/http"
)

func SkipValidatorForManualRoutes(validator func(next http.Handler) http.Handler, next http.Handler, skippedRoutes []string) http.Handler {
	skippedRoutesSet := make(map[string]struct{})
	for _, v := range skippedRoutes {
		skippedRoutesSet[v] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, exists := skippedRoutesSet[r.URL.Path]; exists {
			next.ServeHTTP(w, r)
			return
		}
		validator(next).ServeHTTP(w, r)
	})
}
