package database

import (
	"os"
	"testing"
)

func TestRunMigrations(t *testing.T) {
	db := InitTestDB(t)
	defer db.Close()

	// Verify all tables exist
	tables := []string{"users", "orders", "pizza_styles", "pizza_sizes", "toppings", "migrations"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s does not exist: %v", table, err)
		}
	}

	// Verify menu data was seeded
	var styleCount, sizeCount, toppingCount int
	db.QueryRow("SELECT COUNT(*) FROM pizza_styles").Scan(&styleCount)
	db.QueryRow("SELECT COUNT(*) FROM pizza_sizes").Scan(&sizeCount)
	db.QueryRow("SELECT COUNT(*) FROM toppings").Scan(&toppingCount)

	if styleCount != 8 {
		t.Errorf("Expected 8 pizza styles, got %d", styleCount)
	}
	if sizeCount != 4 {
		t.Errorf("Expected 4 pizza sizes, got %d", sizeCount)
	}
	if toppingCount != 15 {
		t.Errorf("Expected 15 toppings, got %d", toppingCount)
	}
}

func TestSeedMenuData_Idempotent(t *testing.T) {
	db := InitTestDB(t)
	defer db.Close()

	// Get initial counts
	var initialStyleCount, initialSizeCount, initialToppingCount int
	db.QueryRow("SELECT COUNT(*) FROM pizza_styles").Scan(&initialStyleCount)
	db.QueryRow("SELECT COUNT(*) FROM pizza_sizes").Scan(&initialSizeCount)
	db.QueryRow("SELECT COUNT(*) FROM toppings").Scan(&initialToppingCount)

	// Try to seed again - should be idempotent (no duplicates)
	err := seedMenuData(db)
	if err != nil {
		t.Fatalf("Second seed failed: %v", err)
	}

	// Verify counts haven't changed
	var finalStyleCount, finalSizeCount, finalToppingCount int
	db.QueryRow("SELECT COUNT(*) FROM pizza_styles").Scan(&finalStyleCount)
	db.QueryRow("SELECT COUNT(*) FROM pizza_sizes").Scan(&finalSizeCount)
	db.QueryRow("SELECT COUNT(*) FROM toppings").Scan(&finalToppingCount)

	if finalStyleCount != initialStyleCount {
		t.Errorf("Seed not idempotent: styles changed from %d to %d", initialStyleCount, finalStyleCount)
	}
	if finalSizeCount != initialSizeCount {
		t.Errorf("Seed not idempotent: sizes changed from %d to %d", initialSizeCount, finalSizeCount)
	}
	if finalToppingCount != initialToppingCount {
		t.Errorf("Seed not idempotent: toppings changed from %d to %d", initialToppingCount, finalToppingCount)
	}
}

func TestSeedTestUser(t *testing.T) {
	db := InitTestDB(t)
	defer db.Close()

	userID := SeedTestUser(t, db)

	if userID == 0 {
		t.Error("Expected non-zero user ID")
	}

	// Verify user was created
	var email string
	err := db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
	if err != nil {
		t.Errorf("Failed to query user: %v", err)
	}

	if email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", email)
	}
}

func TestSplitSQLStatements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name: "single statement",
			input: `CREATE TABLE test (
				id INT PRIMARY KEY
			);`,
			expected: 1,
		},
		{
			name: "multiple statements",
			input: `
				CREATE TABLE users (id INT);
				CREATE TABLE orders (id INT);
				CREATE TABLE products (id INT);
			`,
			expected: 3,
		},
		{
			name: "with comments",
			input: `
				-- This is a comment
				CREATE TABLE test1 (id INT);
				-- Another comment
				CREATE TABLE test2 (id INT);
			`,
			expected: 2,
		},
		{
			name:     "empty input",
			input:    "",
			expected: 0,
		},
		{
			name: "statement without semicolon",
			input: `CREATE TABLE test (id INT)`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitSQLStatements(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d statements, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestInitDB(t *testing.T) {
	// Ensure db directory exists before test
	err := os.MkdirAll("./db", 0755)
	if err != nil {
		t.Fatalf("Failed to create db directory: %v", err)
	}

	// Clean up test database file after test
	defer os.Remove("./db/orders.db")

	// Test InitDB with SQLite (default)
	db, err := InitDB()
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer db.Close()

	// Verify database is functional
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pizza_styles").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query database: %v", err)
	}

	if count == 0 {
		t.Error("Expected pizza styles to be seeded")
	}
}
