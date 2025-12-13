package handlers

import (
	"brix-pizza/internal/logger"
	"brix-pizza/internal/metrics"
	"brix-pizza/internal/models"
	"database/sql"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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

// SpecialtiesHandler displays the specialty pizzas page
func SpecialtiesHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	data := map[string]interface{}{
		"Session": session,
	}
	templates.ExecuteTemplate(w, "specialties.html", data)
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
		logger.Logger.Error().Err(err).Msg("Error fetching pizza styles")
		http.Error(w, "Error loading menu", http.StatusInternalServerError)
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
		logger.Logger.Error().Err(err).Msg("Error fetching pizza sizes")
		http.Error(w, "Error loading menu", http.StatusInternalServerError)
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
		logger.Logger.Error().Err(err).Msg("Error fetching toppings")
		http.Error(w, "Error loading menu", http.StatusInternalServerError)
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

		email := strings.TrimSpace(r.FormValue("email"))
		firstName := strings.TrimSpace(r.FormValue("first_name"))
		lastName := strings.TrimSpace(r.FormValue("last_name"))
		phone := strings.TrimSpace(r.FormValue("phone"))
		password := r.FormValue("password")

		// Validate input
		if err := validateEmail(email); err != nil {
			templates.ExecuteTemplate(w, "register.html", map[string]string{
				"Error": err.Error(),
			})
			return
		}

		if err := validateName(firstName, "First name"); err != nil {
			templates.ExecuteTemplate(w, "register.html", map[string]string{
				"Error": err.Error(),
			})
			return
		}

		if err := validateName(lastName, "Last name"); err != nil {
			templates.ExecuteTemplate(w, "register.html", map[string]string{
				"Error": err.Error(),
			})
			return
		}

		if err := validatePhone(phone); err != nil {
			templates.ExecuteTemplate(w, "register.html", map[string]string{
				"Error": err.Error(),
			})
			return
		}

		if err := validatePassword(password); err != nil {
			templates.ExecuteTemplate(w, "register.html", map[string]string{
				"Error": err.Error(),
			})
			return
		}

		// Check if user already exists
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&exists)
		if err != nil {
			logger.Logger.Error().Err(err).Str("email", email).Msg("Failed to check existing user")
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
			logger.Logger.Error().Err(err).Str("email", email).Msg("Failed to hash password")
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}

		// Insert new user
		stmt, err := db.Prepare(`INSERT INTO users (first_name, last_name, email, phone, password_hash)
			VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			logger.Logger.Error().Err(err).Str("email", email).Msg("Failed to prepare insert statement")
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		result, err := stmt.Exec(firstName, lastName, email, phone, string(hashedPassword))
		if err != nil {
			logger.Logger.Error().Err(err).Str("email", email).Msg("Failed to insert user")
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}

		// Auto-login: create session for newly registered user
		userID, err := result.LastInsertId()
		if err != nil {
			logger.Logger.Error().Err(err).Str("email", email).Msg("Failed to get last insert ID")
			http.Error(w, "Error creating session", http.StatusInternalServerError)
			return
		}

		sessionID := generateSessionID()
		sessions[sessionID] = &models.Session{
			UserID: int(userID),
			Email:  email,
			Name:   firstName + " " + lastName,
		}

		// Track user registration metric
		metrics.UsersRegistered.Inc()

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

		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")

		// Validate input
		if err := validateEmail(email); err != nil {
			templates.ExecuteTemplate(w, "login.html", map[string]string{
				"Error": err.Error(),
			})
			return
		}

		if err := validatePassword(password); err != nil {
			templates.ExecuteTemplate(w, "login.html", map[string]string{
				"Error": err.Error(),
			})
			return
		}

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
			logger.Logger.Error().Err(err).Str("email", email).Msg("Failed to query user for login")
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

		// Log successful login
		logger.Logger.Info().
			Int("user_id", user.ID).
			Str("email", user.Email).
			Str("name", user.FirstName+" "+user.LastName).
			Str("session_id", sessionID).
			Str("remote_addr", r.RemoteAddr).
			Msg("User logged in successfully")

		// Track user login metric
		metrics.UserLoginsTotal.Inc()

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
	var sessionID string
	var userID int
	var email string

	if err == nil {
		sessionID = cookie.Value
		// Get session info before deleting for logging
		if session, exists := sessions[sessionID]; exists {
			userID = session.UserID
			email = session.Email
		}
		delete(sessions, sessionID)
	}

	// Log logout event
	if userID > 0 {
		logger.Logger.Info().
			Int("user_id", userID).
			Str("email", email).
			Str("session_id", sessionID).
			Str("remote_addr", r.RemoteAddr).
			Msg("User logged out")
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
	pizzaStyle := strings.TrimSpace(r.FormValue("pizza_style"))
	sizeID := strings.TrimSpace(r.FormValue("size"))

	// Validate pizza style is not empty
	if pizzaStyle == "" {
		http.Error(w, "Pizza style is required", http.StatusBadRequest)
		return
	}

	// Validate size is not empty
	if sizeID == "" {
		http.Error(w, "Pizza size is required", http.StatusBadRequest)
		return
	}

	// Validate pizza style exists in database
	var styleExists int
	err := db.QueryRow("SELECT COUNT(*) FROM pizza_styles WHERE name = ? AND active = 1", pizzaStyle).Scan(&styleExists)
	if err != nil || styleExists == 0 {
		logger.Logger.Warn().
			Str("pizza_style", pizzaStyle).
			Int("user_id", session.UserID).
			Msg("Invalid pizza style selected")
		http.Error(w, "Invalid pizza style", http.StatusBadRequest)
		return
	}

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
	err = db.QueryRow("SELECT base_price, name FROM pizza_sizes WHERE id = ?", sizeID).Scan(&basePrice, &sizeName)
	if err != nil {
		logger.Logger.Error().
			Err(err).
			Str("size_id", sizeID).
			Int("user_id", session.UserID).
			Msg("Error fetching pizza size")
		http.Error(w, "Invalid pizza size", http.StatusBadRequest)
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
			logger.Logger.Warn().
				Err(err).
				Str("topping_name", toppingName).
				Int("user_id", session.UserID).
				Msg("Error fetching topping price")
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
		logger.Logger.Error().
			Err(err).
			Int("user_id", session.UserID).
			Msg("Error preparing order insert statement")
		http.Error(w, "Error creating order", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		session.UserID,
		pizzaStyle,
		sizeName,
		leftToppingsStr,
		rightToppingsStr,
		wholeToppingsStr,
		total,
	)

	if err != nil {
		logger.Logger.Error().
			Err(err).
			Int("user_id", session.UserID).
			Str("pizza_style", pizzaStyle).
			Str("size", sizeName).
			Msg("Error saving order to database")
		http.Error(w, "Error saving order", http.StatusInternalServerError)
		return
	}

	// Get the order ID
	orderID, _ := result.LastInsertId()

	// Log successful order creation
	logger.Logger.Info().
		Int64("order_id", orderID).
		Int("user_id", session.UserID).
		Str("email", session.Email).
		Str("pizza_style", pizzaStyle).
		Str("size", sizeName).
		Str("left_toppings", leftToppingsStr).
		Str("right_toppings", rightToppingsStr).
		Str("whole_toppings", wholeToppingsStr).
		Float64("total", total).
		Msg("Order created successfully")

	// Track order metrics
	metrics.OrdersTotal.WithLabelValues(pizzaStyle, sizeName).Inc()
	metrics.OrdersRevenue.Add(total)

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

// GetSession retrieves the session from the request cookie (exported for use by API package)
func GetSession(r *http.Request) *models.Session {
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

// getSession is a convenience wrapper for internal use
func getSession(r *http.Request) *models.Session {
	return GetSession(r)
}

// CreateSessionForTesting creates a session for testing (exported for use by other test packages)
func CreateSessionForTesting(userID int, email, name string) string {
	sessionID := generateSessionID()
	sessions[sessionID] = &models.Session{
		UserID: userID,
		Email:  email,
		Name:   name,
	}
	return sessionID
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

// Validation functions

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	nameRegex  = regexp.MustCompile(`^[A-Za-z\s\-']+$`)
	phoneRegex = regexp.MustCompile(`^[\d\s\-\(\)\+]+$`)
)

// validateEmail validates an email address
func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if len(email) == 0 {
		return &ValidationError{"Email is required"}
	}
	if len(email) > 100 {
		return &ValidationError{"Email must be 100 characters or less"}
	}
	if !emailRegex.MatchString(email) {
		return &ValidationError{"Invalid email format"}
	}
	return nil
}

// validateName validates a first or last name
func validateName(name, fieldName string) error {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return &ValidationError{fieldName + " is required"}
	}
	if len(name) < 2 {
		return &ValidationError{fieldName + " must be at least 2 characters"}
	}
	if len(name) > 50 {
		return &ValidationError{fieldName + " must be 50 characters or less"}
	}
	if !nameRegex.MatchString(name) {
		return &ValidationError{fieldName + " can only contain letters, spaces, hyphens, and apostrophes"}
	}
	return nil
}

// validatePhone validates a phone number
func validatePhone(phone string) error {
	phone = strings.TrimSpace(phone)
	if len(phone) == 0 {
		return &ValidationError{"Phone number is required"}
	}
	// Remove non-digit characters for length check
	digits := regexp.MustCompile(`\d`).FindAllString(phone, -1)
	digitCount := len(digits)
	if digitCount < 10 {
		return &ValidationError{"Phone number must contain at least 10 digits"}
	}
	if len(phone) > 20 {
		return &ValidationError{"Phone number must be 20 characters or less"}
	}
	if !phoneRegex.MatchString(phone) {
		return &ValidationError{"Invalid phone number format"}
	}
	return nil
}

// validatePassword validates a password
func validatePassword(password string) error {
	if len(password) == 0 {
		return &ValidationError{"Password is required"}
	}
	if len(password) < 6 {
		return &ValidationError{"Password must be at least 6 characters"}
	}
	if len(password) > 100 {
		return &ValidationError{"Password must be 100 characters or less"}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
