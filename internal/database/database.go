package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

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

		// Configure connection pool for MySQL
		// With 3 replicas, each pod gets these limits:
		// - Max 25 open connections per pod = 75 total max connections to MySQL
		// - 5 idle connections per pod = 15 total idle connections
		database.SetMaxOpenConns(25)
		database.SetMaxIdleConns(5)
		database.SetConnMaxLifetime(5 * time.Minute)
		database.SetConnMaxIdleTime(1 * time.Minute)

		// Test the connection
		err = database.Ping()
		if err != nil {
			return nil, fmt.Errorf("failed to ping MySQL database: %w", err)
		}

		log.Println("✅ Successfully connected to MySQL database")
		log.Printf("📊 Connection pool: max_open=25, max_idle=5, max_lifetime=5m")
	} else {
		// Use SQLite (default)
		log.Println("⚠️  WARNING: Using SQLite database (not recommended for production)")
		log.Println("⚠️  Set DATABASE_URL environment variable to use MySQL")
		log.Println("⚠️  Example: DATABASE_URL=user:password@tcp(localhost:3306)/brix_pizza")

		database, err = sql.Open("sqlite3", "./db/orders.db?_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
		}

		// SQLite connection pool settings
		// WAL mode allows better concurrency, but still keep it conservative
		database.SetMaxOpenConns(10)
		database.SetMaxIdleConns(5)
		database.SetConnMaxLifetime(0) // Don't close connections in SQLite
	}

	// Run migrations
	err = runMigrations(database)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func runMigrations(database *sql.DB) error {
	// Detect database type
	databaseURL := os.Getenv("DATABASE_URL")
	isMySQL := databaseURL != ""

	// Create migrations table to track applied migrations
	var createMigrationsTableSQL string
	if isMySQL {
		createMigrationsTableSQL = `CREATE TABLE IF NOT EXISTS migrations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`
	} else {
		createMigrationsTableSQL = `CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
	}

	_, err := database.Exec(createMigrationsTableSQL)
	if err != nil {
		return err
	}

	// Define all migrations in order
	migrations := []struct {
		name      string
		sqlSQLite string
		sqlMySQL  string
	}{
		{
			name: "001_initial_schema",
			sqlSQLite: `
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
			sqlMySQL: `
				CREATE TABLE IF NOT EXISTS users (
					id INT AUTO_INCREMENT PRIMARY KEY,
					first_name VARCHAR(100) NOT NULL,
					last_name VARCHAR(100) NOT NULL,
					email VARCHAR(255) UNIQUE NOT NULL,
					phone VARCHAR(20) NOT NULL,
					password_hash VARCHAR(255) NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					INDEX idx_email (email)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

				CREATE TABLE IF NOT EXISTS orders (
					id INT AUTO_INCREMENT PRIMARY KEY,
					user_id INT NOT NULL,
					pizza_style VARCHAR(100) NOT NULL,
					size VARCHAR(50) NOT NULL,
					left_toppings TEXT,
					right_toppings TEXT,
					total DECIMAL(10,2) NOT NULL,
					status VARCHAR(20) DEFAULT 'pending',
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
					INDEX idx_user_id (user_id),
					INDEX idx_created_at (created_at)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

				CREATE TABLE IF NOT EXISTS pizza_styles (
					id INT AUTO_INCREMENT PRIMARY KEY,
					name VARCHAR(100) UNIQUE NOT NULL,
					description TEXT,
					emoji VARCHAR(10),
					active TINYINT(1) DEFAULT 1,
					display_order INT DEFAULT 0,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

				CREATE TABLE IF NOT EXISTS pizza_sizes (
					id INT AUTO_INCREMENT PRIMARY KEY,
					name VARCHAR(50) UNIQUE NOT NULL,
					diameter VARCHAR(20) NOT NULL,
					base_price DECIMAL(10,2) NOT NULL,
					display_order INT DEFAULT 0,
					active TINYINT(1) DEFAULT 1,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

				CREATE TABLE IF NOT EXISTS toppings (
					id INT AUTO_INCREMENT PRIMARY KEY,
					name VARCHAR(100) UNIQUE NOT NULL,
					price DECIMAL(10,2) NOT NULL,
					category VARCHAR(50),
					active TINYINT(1) DEFAULT 1,
					display_order INT DEFAULT 0,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
			`,
		},
		{
			name:      "002_seed_menu_data",
			sqlSQLite: "", // Special migration handled separately with locking
			sqlMySQL:  "", // Special migration handled separately with locking
		},
		{
			name: "003_add_whole_toppings",
			sqlSQLite: `
				ALTER TABLE orders ADD COLUMN whole_toppings TEXT;
			`,
			sqlMySQL: `
				ALTER TABLE orders ADD COLUMN whole_toppings TEXT;
			`,
		},
	}

	// Apply each migration if not already applied
	for _, migration := range migrations {
		// Use a transaction with locking to prevent race conditions across multiple pods
		tx, err := database.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Lock the migrations table row for this migration (or entire table if row doesn't exist)
		var count int
		if isMySQL {
			// Use FOR UPDATE to lock the row (or wait if another pod is working on it)
			err = tx.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ? FOR UPDATE", migration.name).Scan(&count)
		} else {
			err = tx.QueryRow("SELECT COUNT(*) FROM migrations WHERE name = ?", migration.name).Scan(&count)
		}
		if err != nil {
			return err
		}

		if count == 0 {
			// Migration not applied yet
			log.Printf("Applying migration: %s", migration.name)

			// Special handling for seed data migration with K8s-safe locking
			if migration.name == "002_seed_menu_data" {
				// Seed data migration handles its own transaction
				tx.Rollback() // Roll back our transaction first
				err = seedMenuData(database)
				if err != nil {
					return err
				}
				// Record migration in a new transaction
				_, err = database.Exec("INSERT INTO migrations (name) VALUES (?)", migration.name)
				if err != nil {
					return err
				}
			} else {
				// Use appropriate SQL based on database type
				var sql string
				if isMySQL {
					sql = migration.sqlMySQL
				} else {
					sql = migration.sqlSQLite
				}

				if sql != "" {
					// For MySQL, split multiple statements and execute separately
					if isMySQL {
						statements := splitSQLStatements(sql)
						for _, stmt := range statements {
							if stmt != "" {
								_, err = tx.Exec(stmt)
								if err != nil {
									// Check if it's a duplicate column error (MySQL error 1060)
									// This can happen in race conditions when multiple pods start simultaneously
									if strings.Contains(err.Error(), "Error 1060") || strings.Contains(err.Error(), "Duplicate column") {
										log.Printf("Column already exists (race condition), continuing...")
									} else {
										return fmt.Errorf("failed to execute statement in migration %s: %v", migration.name, err)
									}
								}
							}
						}
					} else {
						// SQLite can handle multiple statements in one Exec
						_, err = tx.Exec(sql)
						if err != nil {
							return err
						}
					}
				}

				// Record migration as applied
				_, err = tx.Exec("INSERT INTO migrations (name) VALUES (?)", migration.name)
				if err != nil {
					return err
				}

				// Commit the transaction
				err = tx.Commit()
				if err != nil {
					return err
				}
			}

			log.Printf("Migration applied: %s", migration.name)
		} else {
			// Migration already applied, just roll back the transaction
			tx.Rollback()
		}
	}

	return nil
}

func seedMenuData(database *sql.DB) error {
	// Detect database type
	databaseURL := os.Getenv("DATABASE_URL")
	isMySQL := databaseURL != ""

	// Use transaction with explicit locking to prevent race conditions in K8s
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Lock the migrations table to ensure only one pod seeds the data
	if isMySQL {
		// MySQL supports FOR UPDATE for row-level locking
		_, err = tx.Exec("SELECT COUNT(*) FROM migrations WHERE name = '002_seed_menu_data' FOR UPDATE")
		if err != nil {
			log.Printf("Failed to acquire lock: %v", err)
		}
	}
	// SQLite doesn't support FOR UPDATE, but the transaction itself is exclusive

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

// splitSQLStatements splits a multi-statement SQL string into individual statements
// This is needed because MySQL's Go driver doesn't support executing multiple statements at once
func splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		current.WriteString(line)
		current.WriteString("\n")

		// If line ends with semicolon, it's the end of a statement
		if strings.HasSuffix(trimmed, ";") {
			statements = append(statements, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}

	// Add any remaining statement that doesn't end with semicolon
	if current.Len() > 0 {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}
