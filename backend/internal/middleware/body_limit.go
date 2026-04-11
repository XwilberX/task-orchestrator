package middleware

import (
	"net/http"

	"github.com/XwilberX/task-orchestrator/pkg/response"
)

// MaxBodySize rechaza requests cuyo body supere maxBytes.
// 5 MB es suficiente para cualquier definición de código real.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				response.BadRequest(w, nil, "request body demasiado grande")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
