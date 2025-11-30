package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
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
	var database *sql.DB
	var err error

	// Check for DATABASE_URL environment variable
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL != "" {
		// Use MySQL
		log.Println("📊 Using MySQL database")
		database, err = sql.Open("mysql", databaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
		}

		// Test the connection
		err = database.Ping()
		if err != nil {
			return nil, fmt.Errorf("failed to ping MySQL database: %w", err)
		}

		log.Println("✅ Successfully connected to MySQL database")
	} else {
		// Use SQLite (default)
		log.Println("⚠️  WARNING: Using SQLite database (not recommended for production)")
		log.Println("⚠️  Set DATABASE_URL environment variable to use MySQL")
		log.Println("⚠️  Example: DATABASE_URL=user:password@tcp(localhost:3306)/brix_pizza")

		database, err = sql.Open("sqlite3", "./db/orders.db")
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
		}
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
			name: "001_initial_schema",
			sql: `
				CREATE TABLE IF NOT EXISTS users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					first_name TEXT NOT NULL,
					last_name TEXT NOT NULL,
					email TEXT UNIQUE NOT NULL,
					phone TEXT NOT NULL,
					password_hash TEXT NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS orders (
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
				);

				CREATE TABLE IF NOT EXISTS pizza_styles (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT UNIQUE NOT NULL,
					description TEXT,
					emoji TEXT,
					active INTEGER DEFAULT 1,
					display_order INTEGER DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS pizza_sizes (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT UNIQUE NOT NULL,
					diameter TEXT NOT NULL,
					base_price REAL NOT NULL,
					display_order INTEGER DEFAULT 0,
					active INTEGER DEFAULT 1,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS toppings (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT UNIQUE NOT NULL,
					price REAL NOT NULL,
					category TEXT,
					active INTEGER DEFAULT 1,
					display_order INTEGER DEFAULT 0,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);
			`,
		},
		{
			name: "002_seed_menu_data",
			sql:  "", // Special migration handled separately with locking
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

			// Special handling for seed data migration with K8s-safe locking
			if migration.name == "002_seed_menu_data" {
				err = seedMenuData(database)
				if err != nil {
					return err
				}
			} else {
				_, err = database.Exec(migration.sql)
				if err != nil {
					return err
				}
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

func seedMenuData(database *sql.DB) error {
	// Use transaction with explicit locking to prevent race conditions in K8s
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Lock the migrations table to ensure only one pod seeds the data
	// This works for both SQLite (BEGIN EXCLUSIVE via tx) and MySQL (row lock)
	_, err = tx.Exec("SELECT COUNT(*) FROM migrations WHERE name = '002_seed_menu_data' FOR UPDATE")
	if err != nil {
		// SQLite doesn't support FOR UPDATE, but the transaction itself is exclusive
		// So we can ignore this error for SQLite and just continue
		log.Printf("Lock query note (safe to ignore for SQLite): %v", err)
	}

	// Double-check if seed data was already applied by another pod
	var styleCount, sizeCount, toppingCount int
	tx.QueryRow("SELECT COUNT(*) FROM pizza_styles").Scan(&styleCount)
	tx.QueryRow("SELECT COUNT(*) FROM pizza_sizes").Scan(&sizeCount)
	tx.QueryRow("SELECT COUNT(*) FROM toppings").Scan(&toppingCount)

	if styleCount > 0 || sizeCount > 0 || toppingCount > 0 {
		log.Println("Menu data already seeded by another instance, skipping...")
		return tx.Commit()
	}

	log.Println("Seeding menu data...")

	// Seed pizza styles
	styles := []struct {
		name, description, emoji string
		order                    int
	}{
		{"New York Style", "Thin, crispy crust with a wide diameter", "🗽", 1},
		{"Chicago Deep Dish", "Thick, buttery crust with layers of cheese and toppings", "🏙️", 2},
		{"Detroit Style", "Square, thick crust with crispy edges", "🚗", 3},
		{"Neapolitan", "Traditional Italian with soft, chewy crust", "🇮🇹", 4},
		{"Sicilian", "Thick, rectangular with fluffy dough", "🌋", 5},
		{"California Style", "Creative toppings on a crispy, thin crust", "🌴", 6},
		{"Greek Style", "Oil-based dough with puffy, chewy texture", "🏛️", 7},
		{"St. Louis Style", "Ultra-thin, cracker-like crust", "🎺", 8},
	}

	for _, style := range styles {
		_, err = tx.Exec(`INSERT INTO pizza_styles (name, description, emoji, display_order) VALUES (?, ?, ?, ?)`,
			style.name, style.description, style.emoji, style.order)
		if err != nil {
			return err
		}
	}

	// Seed pizza sizes
	sizes := []struct {
		name, diameter string
		price          float64
		order          int
	}{
		{"Small", "10\"", 12.99, 1},
		{"Medium", "12\"", 16.99, 2},
		{"Large", "14\"", 20.99, 3},
		{"Extra Large", "16\"", 24.99, 4},
	}

	for _, size := range sizes {
		_, err = tx.Exec(`INSERT INTO pizza_sizes (name, diameter, base_price, display_order) VALUES (?, ?, ?, ?)`,
			size.name, size.diameter, size.price, size.order)
		if err != nil {
			return err
		}
	}

	// Seed toppings
	toppings := []struct {
		name, category string
		price          float64
		order          int
	}{
		{"Pepperoni", "meat", 1.50, 1},
		{"Italian Sausage", "meat", 1.50, 2},
		{"Bacon", "meat", 1.50, 3},
		{"Ham", "meat", 1.50, 4},
		{"Chicken", "meat", 1.50, 5},
		{"Mushrooms", "veggie", 1.50, 6},
		{"Bell Peppers", "veggie", 1.50, 7},
		{"Onions", "veggie", 1.50, 8},
		{"Black Olives", "veggie", 1.50, 9},
		{"Tomatoes", "veggie", 1.50, 10},
		{"Spinach", "veggie", 1.50, 11},
		{"Jalapeños", "veggie", 1.50, 12},
		{"Pineapple", "veggie", 1.50, 13},
		{"Extra Cheese", "cheese", 1.50, 14},
		{"Feta", "cheese", 1.50, 15},
	}

	for _, topping := range toppings {
		_, err = tx.Exec(`INSERT INTO toppings (name, category, price, display_order) VALUES (?, ?, ?, ?)`,
			topping.name, topping.category, topping.price, topping.order)
		if err != nil {
			return err
		}
	}

	log.Println("Menu data seeded successfully")
	return tx.Commit()
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	templates.ExecuteTemplate(w, "home.html", session)
}

type MenuData struct {
	Session     *Session
	PizzaStyles []PizzaStyle
	PizzaSizes  []PizzaSize
	Toppings    []Topping
}

type PizzaStyle struct {
	ID          int
	Name        string
	Description string
	Emoji       string
}

type PizzaSize struct {
	ID        int
	Name      string
	Diameter  string
	BasePrice float64
}

type Topping struct {
	ID       int
	Name     string
	Price    float64
	Category string
}

func orderPageHandler(w http.ResponseWriter, r *http.Request) {
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

	var pizzaStyles []PizzaStyle
	for styleRows.Next() {
		var style PizzaStyle
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

	var pizzaSizes []PizzaSize
	for sizeRows.Next() {
		var size PizzaSize
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

	var toppings []Topping
	for toppingRows.Next() {
		var topping Topping
		err := toppingRows.Scan(&topping.ID, &topping.Name, &topping.Price, &topping.Category)
		if err != nil {
			continue
		}
		toppings = append(toppings, topping)
	}

	menuData := MenuData{
		Session:     session,
		PizzaStyles: pizzaStyles,
		PizzaSizes:  pizzaSizes,
		Toppings:    toppings,
	}

	templates.ExecuteTemplate(w, "order.html", menuData)
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
	sizeID := r.FormValue("size")
	leftToppings := r.Form["left_toppings[]"]
	rightToppings := r.Form["right_toppings[]"]

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
		sizeName,
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
