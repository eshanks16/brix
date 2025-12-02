package api

import (
	"brix-pizza/internal/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

var (
	db     *sql.DB
	apiKey string
)

// Init initializes the API package with database connection
func Init(database *sql.DB) {
	db = database
	// Load API key from environment variable
	apiKey = os.Getenv("BRIX_API_KEY")
	if apiKey == "" {
		log.Println("⚠️  WARNING: BRIX_API_KEY not set - API endpoints will be unsecured")
		log.Println("⚠️  Set BRIX_API_KEY environment variable to secure API access")
	} else {
		log.Println("✅ API key authentication enabled")
	}
}

// requireAPIKey middleware checks for valid API key in request header
func requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If no API key is configured, allow access (for development)
		if apiKey == "" {
			next(w, r)
			return
		}

		// Check for API key in Authorization header
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + apiKey

		if authHeader != expectedAuth {
			respondError(w, "Invalid or missing API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// MenuHandler returns the menu as JSON (with API key protection)
var MenuHandler = requireAPIKey(menuHandler)

func menuHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Fetch pizza styles
	styleRows, err := db.Query(`SELECT id, name, description, emoji FROM pizza_styles WHERE active = 1 ORDER BY display_order`)
	if err != nil {
		log.Printf("Error fetching pizza styles: %v", err)
		respondError(w, "Error loading menu", http.StatusInternalServerError)
		return
	}
	defer styleRows.Close()

	var pizzaStyles []models.PizzaStyle
	for styleRows.Next() {
		var style models.PizzaStyle
		err := styleRows.Scan(&style.ID, &style.Name, &style.Description, &style.Emoji)
		if err != nil {
			continue
		}
		pizzaStyles = append(pizzaStyles, style)
	}

	// Fetch pizza sizes
	sizeRows, err := db.Query(`SELECT id, name, diameter, base_price FROM pizza_sizes WHERE active = 1 ORDER BY display_order`)
	if err != nil {
		log.Printf("Error fetching pizza sizes: %v", err)
		respondError(w, "Error loading menu", http.StatusInternalServerError)
		return
	}
	defer sizeRows.Close()

	var pizzaSizes []models.PizzaSize
	for sizeRows.Next() {
		var size models.PizzaSize
		err := sizeRows.Scan(&size.ID, &size.Name, &size.Diameter, &size.BasePrice)
		if err != nil {
			continue
		}
		pizzaSizes = append(pizzaSizes, size)
	}

	// Fetch toppings
	toppingRows, err := db.Query(`SELECT id, name, price, category FROM toppings WHERE active = 1 ORDER BY display_order`)
	if err != nil {
		log.Printf("Error fetching toppings: %v", err)
		respondError(w, "Error loading menu", http.StatusInternalServerError)
		return
	}
	defer toppingRows.Close()

	var toppings []models.Topping
	for toppingRows.Next() {
		var topping models.Topping
		err := toppingRows.Scan(&topping.ID, &topping.Name, &topping.Price, &topping.Category)
		if err != nil {
			continue
		}
		toppings = append(toppings, topping)
	}

	response := models.MenuResponse{
		PizzaStyles: pizzaStyles,
		PizzaSizes:  pizzaSizes,
		Toppings:    toppings,
	}

	respondJSON(w, response, http.StatusOK)
}

// CreateOrderHandler creates an order via JSON API (with API key protection)
var CreateOrderHandler = requireAPIKey(createOrderHandler)

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Verify user exists
	var userExists int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", req.UserID).Scan(&userExists)
	if err != nil || userExists == 0 {
		respondError(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	// Get base price from database by size ID
	var basePrice float64
	var sizeName string
	err = db.QueryRow("SELECT base_price, name FROM pizza_sizes WHERE id = ?", req.SizeID).Scan(&basePrice, &sizeName)
	if err != nil {
		respondError(w, "Invalid pizza size", http.StatusBadRequest)
		return
	}

	// Count unique toppings and calculate topping costs from database
	uniqueToppings := make(map[string]bool)
	for _, t := range req.LeftToppings {
		uniqueToppings[t] = true
	}
	for _, t := range req.RightToppings {
		uniqueToppings[t] = true
	}

	// Calculate total topping cost from database prices
	var toppingCost float64
	for toppingName := range uniqueToppings {
		var price float64
		err := db.QueryRow("SELECT price FROM toppings WHERE name = ?", toppingName).Scan(&price)
		if err != nil {
			log.Printf("Error fetching topping price for %s: %v", toppingName, err)
			continue
		}
		toppingCost += price
	}

	total := basePrice + toppingCost

	// Join toppings into comma-separated strings
	leftToppingsStr := ""
	if len(req.LeftToppings) > 0 {
		leftToppingsStr = joinStrings(req.LeftToppings, ", ")
	}

	rightToppingsStr := ""
	if len(req.RightToppings) > 0 {
		rightToppingsStr = joinStrings(req.RightToppings, ", ")
	}

	// Insert order
	stmt, err := db.Prepare(`INSERT INTO orders (user_id, pizza_style, size, left_toppings, right_toppings, total)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		respondError(w, "Error creating order", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		req.UserID,
		req.PizzaStyle,
		sizeName,
		leftToppingsStr,
		rightToppingsStr,
		total,
	)

	if err != nil {
		log.Printf("Error executing statement: %v", err)
		respondError(w, "Error saving order", http.StatusInternalServerError)
		return
	}

	orderID, _ := result.LastInsertId()

	// Fetch the created order
	var order models.OrderResponse
	err = db.QueryRow(`SELECT id, pizza_style, size, left_toppings, right_toppings, total, status, created_at
		FROM orders WHERE id = ?`, orderID).Scan(
		&order.ID, &order.PizzaStyle, &order.Size, &order.LeftToppings, &order.RightToppings,
		&order.Total, &order.Status, &order.CreatedAt)

	if err != nil {
		log.Printf("Error fetching created order: %v", err)
		respondError(w, "Order created but error fetching details", http.StatusInternalServerError)
		return
	}

	respondJSON(w, order, http.StatusCreated)
}

// GetOrdersHandler returns all orders for a user (with API key protection)
var GetOrdersHandler = requireAPIKey(getOrdersHandler)

func getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user_id from query parameter (optional - if not provided, returns all orders)
	userIDStr := r.URL.Query().Get("user_id")

	var query string
	var args []interface{}

	if userIDStr != "" {
		// Filter by specific user
		query = `SELECT id, pizza_style, size, left_toppings, right_toppings, total, status, created_at
			FROM orders WHERE user_id = ? ORDER BY created_at DESC`
		args = append(args, userIDStr)
	} else {
		// Return all orders (admin view)
		query = `SELECT id, pizza_style, size, left_toppings, right_toppings, total, status, created_at
			FROM orders ORDER BY created_at DESC`
	}

	// Fetch orders
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error fetching orders: %v", err)
		respondError(w, "Error fetching orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []models.OrderResponse
	for rows.Next() {
		var order models.OrderResponse
		err := rows.Scan(&order.ID, &order.PizzaStyle, &order.Size,
			&order.LeftToppings, &order.RightToppings, &order.Total, &order.Status, &order.CreatedAt)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	respondJSON(w, orders, http.StatusOK)
}

// Helper functions for API responses

func respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, message string, statusCode int) {
	respondJSON(w, models.ErrorResponse{Error: message}, statusCode)
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
