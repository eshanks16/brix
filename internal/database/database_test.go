package database

import (
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
