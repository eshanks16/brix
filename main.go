package main

import (
	"brix-pizza/internal/api"
	"brix-pizza/internal/database"
	"brix-pizza/internal/handlers"
	"brix-pizza/internal/health"
	"brix-pizza/internal/logger"
	"brix-pizza/internal/metrics"
	"brix-pizza/internal/middleware"
	"context"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger.Init(logLevel)

	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Database initialization failed")
	}
	defer db.Close()

	// Parse templates
	templates := template.Must(template.ParseGlob("templates/*.html"))

	// Initialize packages
	handlers.Init(db, templates)
	api.Init(db)

	// Start database metrics collector (update every 15 seconds)
	metrics.StartDatabaseMetricsCollector(db, 15*time.Second)

	// HTML Routes
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/specialties", handlers.SpecialtiesHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/order", handlers.OrderPageHandler)
	http.HandleFunc("/place-order", handlers.PlaceOrderHandler)
	http.HandleFunc("/orders", handlers.OrdersHandler)

	// API Routes (v1)
	http.HandleFunc("/api/v1/menu", api.MenuHandler)
	http.HandleFunc("/api/v1/orders", api.CreateOrderHandler)
	http.HandleFunc("/api/v1/orders/list", api.GetOrdersHandler)

	// Health Check Routes (for Kubernetes probes)
	http.HandleFunc("/health/live", health.LivenessHandler)
	http.HandleFunc("/health/ready", health.ReadinessHandler(db))

	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Get port from environment variable (default: 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// Create HTTP server with timeouts and logging + Prometheus middleware
	srv := &http.Server{
		Addr:         addr,
		Handler:      middleware.RequestLogger(middleware.PrometheusMiddleware(http.DefaultServeMux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Logger.Info().
			Str("port", port).
			Str("address", "http://0.0.0.0:"+port).
			Msg("🍕 Brix Pizza server starting")
		logger.Logger.Info().
			Str("api_endpoint", "http://0.0.0.0:"+port+"/api/v1/*").
			Msg("📡 API v1 available")
		logger.Logger.Debug().
			Msg("💚 Health checks: /health/live (liveness) and /health/ready (readiness)")
		logger.Logger.Debug().
			Str("metrics_endpoint", "http://0.0.0.0:"+port+"/metrics").
			Msg("📊 Prometheus metrics available")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// Capture SIGINT (Ctrl+C) and SIGTERM (Kubernetes pod termination)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Logger.Info().Msg("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Warn().Err(err).Msg("Server forced to shutdown")
	}

	logger.Logger.Info().Msg("Server stopped")
}
