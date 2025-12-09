package metrics

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestUpdateDatabaseMetrics(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Update metrics
	UpdateDatabaseMetrics(db)

	// Verify metrics were set (we can't easily verify exact values, but we can ensure no panic)
	// The function should complete without error
}

func TestStartDatabaseMetricsCollector(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Start the collector with a very short interval
	StartDatabaseMetricsCollector(db, 10*time.Millisecond)

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	// If we get here without panicking, the collector is working
}

func TestMetricsExist(t *testing.T) {
	// Test that all metrics are initialized
	metrics := []interface{}{
		HttpRequestsTotal,
		HttpRequestDuration,
		OrdersTotal,
		OrdersRevenue,
		UsersRegistered,
		UserLoginsTotal,
		DatabaseConnectionsOpen,
		DatabaseConnectionsIdle,
		DatabaseConnectionsInUse,
		DatabaseConnectionsWaitCount,
		DatabaseConnectionsWaitDuration,
		DatabaseConnectionsMaxIdleClosed,
		DatabaseConnectionsMaxLifetimeClosed,
	}

	for i, metric := range metrics {
		if metric == nil {
			t.Errorf("Metric %d is nil", i)
		}
	}
}
