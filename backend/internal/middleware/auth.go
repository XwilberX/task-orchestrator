package middleware

import (
	"net/http"

	"github.com/XwilberX/task-orchestrator/pkg/response"
)

func APIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-API-Key") != apiKey {
				response.Unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
