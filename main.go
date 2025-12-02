package main

import (
	"brix-pizza/internal/api"
	"brix-pizza/internal/database"
	"brix-pizza/internal/handlers"
	"html/template"
	"log"
	"net/http"
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

	// API Routes
	http.HandleFunc("/api/menu", api.MenuHandler)
	http.HandleFunc("/api/orders", api.CreateOrderHandler)
	http.HandleFunc("/api/orders/list", api.GetOrdersHandler)

	// Static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("🍕 Brix Pizza is running on http://localhost:8080")
	log.Println("📡 API available at http://localhost:8080/api/*")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
