package middleware

import (
	"net/http"
	"strconv"
	"time"

	"brix-pizza/internal/metrics"
)

// PrometheusMiddleware wraps an HTTP handler to collect Prometheus metrics
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(wrapped.statusCode)

		metrics.HttpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusCode).Inc()
		metrics.HttpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
