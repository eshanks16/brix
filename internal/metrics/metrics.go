package metrics

import (
	"database/sql"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "brix_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "brix_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Order metrics
	OrdersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "brix_orders_total",
			Help: "Total number of pizza orders",
		},
		[]string{"pizza_style", "size"},
	)

	OrdersRevenue = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "brix_orders_revenue_total",
			Help: "Total revenue from orders in dollars",
		},
	)

	// User metrics
	UsersRegistered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "brix_users_registered_total",
			Help: "Total number of registered users",
		},
	)

	UserLoginsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "brix_user_logins_total",
			Help: "Total number of user logins",
		},
	)

	// Database metrics
	DatabaseConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "brix_database_connections_open",
			Help: "Number of open database connections",
		},
	)

	DatabaseConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "brix_database_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	DatabaseConnectionsInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "brix_database_connections_in_use",
			Help: "Number of database connections in use",
		},
	)

	DatabaseConnectionsWaitCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "brix_database_connections_wait_count",
			Help: "Total number of connections waited for",
		},
	)

	DatabaseConnectionsWaitDuration = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "brix_database_connections_wait_duration_seconds",
			Help: "Total time blocked waiting for new connections",
		},
	)

	DatabaseConnectionsMaxIdleClosed = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "brix_database_connections_max_idle_closed",
			Help: "Total number of connections closed due to SetMaxIdleConns",
		},
	)

	DatabaseConnectionsMaxLifetimeClosed = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "brix_database_connections_max_lifetime_closed",
			Help: "Total number of connections closed due to SetConnMaxLifetime",
		},
	)
)

// UpdateDatabaseMetrics updates database connection pool metrics
func UpdateDatabaseMetrics(db *sql.DB) {
	stats := db.Stats()
	DatabaseConnectionsOpen.Set(float64(stats.OpenConnections))
	DatabaseConnectionsIdle.Set(float64(stats.Idle))
	DatabaseConnectionsInUse.Set(float64(stats.InUse))
	DatabaseConnectionsWaitCount.Set(float64(stats.WaitCount))
	DatabaseConnectionsWaitDuration.Set(stats.WaitDuration.Seconds())
	DatabaseConnectionsMaxIdleClosed.Set(float64(stats.MaxIdleClosed))
	DatabaseConnectionsMaxLifetimeClosed.Set(float64(stats.MaxLifetimeClosed))
}

// StartDatabaseMetricsCollector starts a goroutine that periodically collects database metrics
func StartDatabaseMetricsCollector(db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			UpdateDatabaseMetrics(db)
		}
	}()
}
