package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the database connection and runs migrations
func InitDB() (*sql.DB, error) {
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
