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

			logger.Printf(
				"request_started: method=%s host=%s path=%s remote_addr=%s proto=%s",
				r.Method,
				r.Host,
				r.URL.Path,
				r.RemoteAddr,
				r.Proto,
			)

			defer func() {
				logger.Printf(
					"request_finished: method=%s host=%s path=%s status=%d duration=%s",
					r.Method,
					r.Host,
					r.URL.Path,
					recorder.Status(),
					time.Since(startedAt).String(),
				)
			}()

			next.ServeHTTP(recorder, r)
		})
	}
}
