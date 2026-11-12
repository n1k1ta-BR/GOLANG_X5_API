package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func requestID(ctx context.Context) string {
	return middleware.GetReqID(ctx)
}

func requestLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			id := requestID(r.Context())
			w.Header().Set("X-Request-ID", id)
			writer := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(writer, r)

			status := writer.Status()
			if status == 0 {
				status = http.StatusOK
			}
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration", time.Since(started),
				"request_id", id,
			)
		}))
	}
}

func recoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				if incident := recover(); incident != nil {
					logger.Error("handler panic", "panic", incident, "request_id", requestID(r.Context()))
					if writer.Status() == 0 {
						writeJSON(writer, 500, errorResponse{"internal server error", "INTERNAL_ERROR"})
					}
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
