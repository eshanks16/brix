package middleware

import (
	"brix-pizza/internal/logger"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (rw *loggingResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// RequestLogger logs HTTP requests with structured fields
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()

		// Add request ID to context and response header
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)

		// Wrap response writer to capture status code
		wrapped := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Log request start
		logger.Logger.Debug().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Msg("Request started")

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		logEvent := logger.Logger.Info().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapped.statusCode).
			Int("bytes", wrapped.bytes).
			Dur("duration_ms", duration).
			Str("remote_addr", r.RemoteAddr)

		// Add warning/error level for non-2xx responses
		if wrapped.statusCode >= 500 {
			logEvent = logger.Logger.Error().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.statusCode).
				Int("bytes", wrapped.bytes).
				Dur("duration_ms", duration).
				Str("remote_addr", r.RemoteAddr)
		} else if wrapped.statusCode >= 400 {
			logEvent = logger.Logger.Warn().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.statusCode).
				Int("bytes", wrapped.bytes).
				Dur("duration_ms", duration).
				Str("remote_addr", r.RemoteAddr)
		}

		logEvent.Msg("Request completed")
	})
}
