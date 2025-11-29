package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	Phone     string
	Password  string
	CreatedAt time.Time
}

type Order struct {
	ID            int
	UserID        int
	PizzaStyle    string
	Size          string
	LeftToppings  string
	RightToppings string
	Total         float64
	Status        string
	CreatedAt     time.Time
	CustomerName  string
	CustomerPhone string
}

type Session struct {
	UserID int
	Email  string
	Name   string
}

var db *sql.DB
var templates *template.Template
var sessions = make(map[string]*Session)

func main() {
	var err error

	// Initialize database
	db, err = initDB()
	if err != nil {
		log.Fatal("Database initialization failed:", err)
	}
	defer db.Close()

	// Parse templates
	templates = template.Must(template.ParseGlob("templates/*.html"))

	// Routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/order", orderPageHandler)
	http.HandleFunc("/place-order", placeOrderHandler)
	http.HandleFunc("/orders", ordersHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("🍕 Brix Pizza is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB() (*sql.DB, error) {
	database, err := sql.Open("sqlite3", "./db/orders.db")
	if err != nil {
		return nil, err
	}

	// Run migrations
	err = runMigrations(database)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func runMigrations(database *sql.DB) error {
	// Create migrations table to track applied migrations
	createMigrationsTableSQL := `CREATE TABLE IF NOT EXISTS migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := database.Exec(createMigrationsTableSQL)
	if err != nil {
		return err
	}

	// Define all migrations in order
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "001_create_users_table",
			sql: `CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				first_name TEXT NOT NULL,
				last_name TEXT NOT NULL,
				email TEXT UNIQUE NOT NULL,
				phone TEXT NOT NULL,
				password_hash TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);`,
		},
		{
			name: "002_create_orders_table",
			sql: `CREATE TABLE IF NOT EXISTS orders (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				pizza_style TEXT NOT NULL,
				size TEXT NOT NULL,
				left_toppings TEXT,
				right_toppings TEXT,
				total REAL NOT NULL,
				status TEXT DEFAULT 'pending',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (user_id) REFERENCES users(id)
			);`,
		},
	}

	// Apply each migration if not already applied
	for _, migration := range migrations {
		var count int
		err := database.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", migration.name).Scan(&count)
		if err != nil {
			return err
		}

		if count == 0 {
			// Migration not applied yet
			log.Printf("Applying migration: %s", migration.name)

			_, err = database.Exec(migration.sql)
			if err != nil {
				return err
			}

			// Record migration as applied
			_, err = database.Exec("INSERT INTO migrations (name) VALUES (?)", migration.name)
			if err != nil {
				return err
			}

			log.Printf("Migration applied: %s", migration.name)
		}
	}

	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	templates.ExecuteTemplate(w, "home.html", session)
}

func orderPageHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	templates.ExecuteTemplate(w, "order.html", session)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
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
		sessions[sessionID] = &Session{
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

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		templates.ExecuteTemplate(w, "login.html", nil)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()

		email := r.FormValue("email")
		password := r.FormValue("password")

		var user User
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
		sessions[sessionID] = &Session{
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

func logoutHandler(w http.ResponseWriter, r *http.Request) {
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

func getSession(r *http.Request) *Session {
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

func placeOrderHandler(w http.ResponseWriter, r *http.Request) {
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
	size := r.FormValue("size")
	leftToppings := r.Form["left_toppings[]"]
	rightToppings := r.Form["right_toppings[]"]

	// Simple pricing logic based on size
	var basePrice float64
	switch size {
		case "small": basePrice = 12.99
		case "medium": basePrice = 16.99
		case "large": basePrice = 20.99
		case "extra_large": basePrice = 24.99
		default: basePrice = 16.99
	}

	// Add topping costs ($1.50 per topping)
	toppingPrice := 1.50

	// Count unique toppings (if same topping on both sides, only count once)
	uniqueToppings := make(map[string]bool)
	for _, t := range leftToppings {
		uniqueToppings[t] = true
	}
	for _, t := range rightToppings {
		uniqueToppings[t] = true
	}

	total := basePrice + (float64(len(uniqueToppings)) * toppingPrice)

	// Join toppings into comma-separated strings
	leftToppingsStr := ""
	if len(leftToppings) > 0 {
		leftToppingsStr = joinStrings(leftToppings, ", ")
	}

	rightToppingsStr := ""
	if len(rightToppings) > 0 {
		rightToppingsStr = joinStrings(rightToppings, ", ")
	}

	// Insert order
	stmt, err := db.Prepare(`INSERT INTO orders (user_id, pizza_style, size, left_toppings, right_toppings, total)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		http.Error(w, "Error creating order", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		session.UserID,
		pizzaStyle,
		size,
		leftToppingsStr,
		rightToppingsStr,
		total,
	)

	if err != nil {
		http.Error(w, "Error saving order", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

func ordersHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.Query(`
		SELECT o.id, o.user_id, o.pizza_style, o.size, o.left_toppings, o.right_toppings, o.total, o.status, o.created_at,
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

	var orders []Order
	for rows.Next() {
		var order Order
		err := rows.Scan(&order.ID, &order.UserID, &order.PizzaStyle, &order.Size,
			&order.LeftToppings, &order.RightToppings, &order.Total, &order.Status, &order.CreatedAt,
			&order.CustomerName, &order.CustomerPhone)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	templates.ExecuteTemplate(w, "orders.html", orders)
}
