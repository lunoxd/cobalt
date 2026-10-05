package middleware

import (
	"net/http"
)

const maxRequestBodySize = 2 * 1024 * 1024 // 2 MB

// SecurityHeaders applies secure HTTP response headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

		// Limit request body size to protect against large payloads / memory exhaustion
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

		next.ServeHTTP(w, r)
	})
}
