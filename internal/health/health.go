package health

import (
	"database/sql"
	"log"
	"net/http"
)

// LivenessHandler returns 200 OK if the application is running
// This is used by Kubernetes liveness probe
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ReadinessHandler checks if the application and database are ready
// This is used by Kubernetes readiness probe
func ReadinessHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check database connectivity
		if err := db.Ping(); err != nil {
			log.Printf("Readiness check failed: database not ready: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Database not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
