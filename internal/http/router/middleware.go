package router

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func requestLogger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			startedAt := time.Now()
			next.ServeHTTP(recorder, r)

			logger.Printf(
				"request: method=%s host=%s path=%s status=%d duration=%s",
				r.Method,
				r.Host,
				r.URL.Path,
				recorder.Status(),
				time.Since(startedAt).String(),
			)
		})
	}
}
