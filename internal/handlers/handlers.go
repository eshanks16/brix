package handlers

import (
	"brix-pizza/internal/models"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	db        *sql.DB
	templates *template.Template
	sessions  = make(map[string]*models.Session)
)

// Init initializes the handlers package with database and templates
func Init(database *sql.DB, tmpl *template.Template) {
	db = database
	templates = tmpl
}

// HomeHandler handles the home page
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	templates.ExecuteTemplate(w, "home.html", session)
}

// OrderPageHandler displays the order form
func OrderPageHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Fetch pizza styles from database
	styleRows, err := db.Query(`SELECT id, name, description, emoji FROM pizza_styles WHERE active = 1 ORDER BY display_order`)
	if err != nil {
		http.Error(w, "Error loading menu", http.StatusInternalServerError)
		log.Printf("Error fetching pizza styles: %v", err)
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

	// Fetch pizza sizes from database
	sizeRows, err := db.Query(`SELECT id, name, diameter, base_price FROM pizza_sizes WHERE active = 1 ORDER BY display_order`)
	if err != nil {
		http.Error(w, "Error loading menu", http.StatusInternalServerError)
		log.Printf("Error fetching pizza sizes: %v", err)
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

	// Fetch toppings from database
	toppingRows, err := db.Query(`SELECT id, name, price, category FROM toppings WHERE active = 1 ORDER BY display_order`)
	if err != nil {
		http.Error(w, "Error loading menu", http.StatusInternalServerError)
		log.Printf("Error fetching toppings: %v", err)
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

	menuData := models.MenuData{
		Session:     session,
		PizzaStyles: pizzaStyles,
		PizzaSizes:  pizzaSizes,
		Toppings:    toppings,
	}

	templates.ExecuteTemplate(w, "order.html", menuData)
}

// RegisterHandler handles user registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		templates.ExecuteTemplate(w, "register.html", nil)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()

		email := r.FormValue("email")
		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")
		phone := r.FormValue("phone")
		password := r.FormValue("password")

		// Check if user already exists
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&exists)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if exists > 0 {
			templates.ExecuteTemplate(w, "register.html", map[string]string{
				"Error": "An account with this email already exists",
			})
			return
		}

		// Hash password using bcrypt
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}

		// Insert new user
		stmt, err := db.Prepare(`INSERT INTO users (first_name, last_name, email, phone, password_hash)
			VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		result, err := stmt.Exec(firstName, lastName, email, phone, string(hashedPassword))
		if err != nil {
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}

		// Auto-login: create session for newly registered user
		userID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Error creating session", http.StatusInternalServerError)
			return
		}

		sessionID := generateSessionID()
		sessions[sessionID] = &models.Session{
			UserID: int(userID),
			Email:  email,
			Name:   firstName + " " + lastName,
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			MaxAge:   86400, // 24 hours
			HttpOnly: true,
		})

		http.Redirect(w, r, "/order", http.StatusSeeOther)
		return
	}
}

// LoginHandler handles user login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		templates.ExecuteTemplate(w, "login.html", nil)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()

		email := r.FormValue("email")
		password := r.FormValue("password")

		var user models.User
		err := db.QueryRow(`SELECT id, first_name, last_name, email, password_hash
			FROM users WHERE email = ?`, email).Scan(
			&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password)

		if err == sql.ErrNoRows {
			templates.ExecuteTemplate(w, "login.html", map[string]string{
				"Error": "Invalid email or password",
			})
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Verify password using bcrypt
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			templates.ExecuteTemplate(w, "login.html", map[string]string{
				"Error": "Invalid email or password",
			})
			return
		}

		// Create session
		sessionID := generateSessionID()
		sessions[sessionID] = &models.Session{
			UserID: user.ID,
			Email:  user.Email,
			Name:   user.FirstName + " " + user.LastName,
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			MaxAge:   86400, // 24 hours
			HttpOnly: true,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
}

// LogoutHandler handles user logout
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		delete(sessions, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// PlaceOrderHandler processes a new pizza order
func PlaceOrderHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/order", http.StatusSeeOther)
		return
	}

	r.ParseForm()

	// Get form values
	pizzaStyle := r.FormValue("pizza_style")
	sizeID := r.FormValue("size")

	// Parse topping selections - now using radio buttons with format: topping_{name} = "left" | "right" | "whole"
	var leftToppings, rightToppings, wholeToppings []string
	for key, values := range r.Form {
		if len(values) > 0 && len(key) > 8 && key[:8] == "topping_" {
			toppingName := key[8:] // Remove "topping_" prefix
			placement := values[0]

			switch placement {
			case "left":
				leftToppings = append(leftToppings, toppingName)
			case "right":
				rightToppings = append(rightToppings, toppingName)
			case "whole":
				wholeToppings = append(wholeToppings, toppingName)
			}
		}
	}

	// Get base price from database by size ID
	var basePrice float64
	var sizeName string
	err := db.QueryRow("SELECT base_price, name FROM pizza_sizes WHERE id = ?", sizeID).Scan(&basePrice, &sizeName)
	if err != nil {
		http.Error(w, "Invalid pizza size", http.StatusBadRequest)
		log.Printf("Error fetching size: %v", err)
		return
	}

	// Count unique toppings and calculate topping costs from database
	uniqueToppings := make(map[string]bool)
	for _, t := range leftToppings {
		uniqueToppings[t] = true
	}
	for _, t := range rightToppings {
		uniqueToppings[t] = true
	}
	for _, t := range wholeToppings {
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
	if len(leftToppings) > 0 {
		leftToppingsStr = joinStrings(leftToppings, ", ")
	}

	rightToppingsStr := ""
	if len(rightToppings) > 0 {
		rightToppingsStr = joinStrings(rightToppings, ", ")
	}

	wholeToppingsStr := ""
	if len(wholeToppings) > 0 {
		wholeToppingsStr = joinStrings(wholeToppings, ", ")
	}

	// Insert order
	stmt, err := db.Prepare(`INSERT INTO orders (user_id, pizza_style, size, left_toppings, right_toppings, whole_toppings, total)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		http.Error(w, "Error creating order", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		session.UserID,
		pizzaStyle,
		sizeName,
		leftToppingsStr,
		rightToppingsStr,
		wholeToppingsStr,
		total,
	)

	if err != nil {
		http.Error(w, "Error saving order", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// OrdersHandler displays the user's order history
func OrdersHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.Query(`
		SELECT o.id, o.user_id, o.pizza_style, o.size, o.left_toppings, o.right_toppings, o.whole_toppings, o.total, o.status, o.created_at,
		       u.first_name || ' ' || u.last_name as customer_name, u.phone
		FROM orders o
		JOIN users u ON o.user_id = u.id
		WHERE o.user_id = ?
		ORDER BY o.created_at DESC`, session.UserID)
	if err != nil {
		http.Error(w, "Error fetching orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.ID, &order.UserID, &order.PizzaStyle, &order.Size,
			&order.LeftToppings, &order.RightToppings, &order.WholeToppings, &order.Total, &order.Status, &order.CreatedAt,
			&order.CustomerName, &order.CustomerPhone)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	templates.ExecuteTemplate(w, "orders.html", orders)
}

// Helper functions

func getSession(r *http.Request) *models.Session {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}

	session, exists := sessions[cookie.Value]
	if !exists {
		return nil
	}

	return session
}

func generateSessionID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
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
