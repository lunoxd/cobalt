package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
)

// Recoverer catches panics, logs them, and responds with a safe 500 error.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				err, ok := rvr.(error)
				if !ok {
					err = fmt.Errorf("%v", rvr)
				}
				slog.Error("panic recovered",
					"error", err,
					"path", r.URL.Path,
					"method", r.Method,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_SERVER_ERROR","message":"A fatal server error occurred."}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
