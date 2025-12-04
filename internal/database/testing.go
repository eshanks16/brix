package database

import (
	"database/sql"
	"io"
	"log"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// InitTestDB creates an in-memory SQLite database for testing
// It uses the same migrations as the production database
func InitTestDB(t *testing.T) *sql.DB {
	// Suppress migration logs during tests
	log.SetOutput(io.Discard)
	defer func() {
		// Restore log output after test
		log.SetOutput(io.Writer(log.Writer()))
	}()

	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Run the same migrations as production
	err = runMigrations(db)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

// SeedTestUser inserts a test user and returns the user ID
func SeedTestUser(t *testing.T, db *sql.DB) int64 {
	result, err := db.Exec(`
		INSERT INTO users (first_name, last_name, email, phone, password_hash)
		VALUES (?, ?, ?, ?, ?)
	`, "Test", "User", "test@example.com", "555-1234", "hashed_password")

	if err != nil {
		t.Fatalf("Failed to seed test user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get user ID: %v", err)
	}

	return userID
}
