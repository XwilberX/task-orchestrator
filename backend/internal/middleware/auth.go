package middleware

import (
	"net/http"

	"github.com/XwilberX/task-orchestrator/pkg/response"
)

func APIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Acepta la clave por header (API clients) o query param (SSE/EventSource)
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.URL.Query().Get("api_key")
			}
			if key != apiKey {
				response.Unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
