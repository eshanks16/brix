package main

import (
	"brix-pizza/internal/api"
	"brix-pizza/internal/database"
	"brix-pizza/internal/handlers"
	"brix-pizza/internal/health"
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initialize database
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("Database initialization failed:", err)
	}
	defer db.Close()

	// Parse templates
	templates := template.Must(template.ParseGlob("templates/*.html"))

	// Initialize packages
	handlers.Init(db, templates)
	api.Init(db)

	// HTML Routes
	http.HandleFunc("/", handlers.HomeHandler)
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

	// Static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Get port from environment variable (default: 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// Create HTTP server with timeouts
	srv := &http.Server{
		Addr:         addr,
		Handler:      nil, // use DefaultServeMux
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🍕 Brix Pizza is running on http://0.0.0.0:%s", port)
		log.Printf("📡 API v1 available at http://0.0.0.0:%s/api/v1/*", port)
		log.Printf("💚 Health checks: /health/live (liveness) and /health/ready (readiness)")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// Capture SIGINT (Ctrl+C) and SIGTERM (Kubernetes pod termination)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
