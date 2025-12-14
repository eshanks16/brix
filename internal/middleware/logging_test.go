package middleware

import (
	"brix-pizza/internal/logger"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLogger(t *testing.T) {
	// Initialize logger for testing
	logger.Init("info")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with logging middleware
	handler := RequestLogger(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify request ID header was added
	requestID := rec.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("Expected X-Request-ID header to be set")
	}

	// Verify response body
	if rec.Body.String() != "test response" {
		t.Errorf("Expected 'test response', got '%s'", rec.Body.String())
	}
}

func TestRequestLogger_ErrorStatus(t *testing.T) {
	// Initialize logger for testing
	logger.Init("info")

	// Create a test handler that returns an error status
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})

	// Wrap with logging middleware
	handler := RequestLogger(testHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/error", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify error status is captured
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestRequestLogger_BytesTracking(t *testing.T) {
	// Initialize logger for testing
	logger.Init("info")

	responseBody := "This is a test response with some content"

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	})

	// Wrap with logging middleware
	handler := RequestLogger(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response body length
	if rec.Body.Len() != len(responseBody) {
		t.Errorf("Expected body length %d, got %d", len(responseBody), rec.Body.Len())
	}
}

func TestRequestLogger_HealthCheckAndMetrics(t *testing.T) {
	// Initialize logger for testing
	logger.Init("info")

	testCases := []struct {
		name string
		path string
	}{
		{"health check live", "/health/live"},
		{"health check ready", "/health/ready"},
		{"metrics endpoint", "/metrics"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with logging middleware
			handler := RequestLogger(testHandler)

			// Create test request
			req := httptest.NewRequest("GET", tc.path, nil)
			rec := httptest.NewRecorder()

			// Execute request - should only log at DEBUG level
			handler.ServeHTTP(rec, req)

			// Verify response
			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rec.Code)
			}

			// Verify request ID header was still added
			requestID := rec.Header().Get("X-Request-ID")
			if requestID == "" {
				t.Error("Expected X-Request-ID header to be set")
			}
		})
	}
}
