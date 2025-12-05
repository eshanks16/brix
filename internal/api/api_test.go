package api

import (
	"brix-pizza/internal/database"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMenuHandler_Success(t *testing.T) {
	// Setup: Use the real database initialization code
	db := database.InitTestDB(t)
	defer db.Close()
	Init(db)

	// Don't set API key for this test (unsecured mode)
	os.Unsetenv("BRIX_API_KEY")
	apiKey = ""

	req := httptest.NewRequest(http.MethodGet, "/api/menu", nil)
	w := httptest.NewRecorder()

	// Test
	MenuHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that we have the expected fields
	if _, ok := result["pizza_styles"]; !ok {
		t.Error("Response missing 'pizza_styles' field")
	}
	if _, ok := result["pizza_sizes"]; !ok {
		t.Error("Response missing 'pizza_sizes' field")
	}
	if _, ok := result["toppings"]; !ok {
		t.Error("Response missing 'toppings' field")
	}

	// Verify we got the seeded data
	styles := result["pizza_styles"].([]interface{})
	sizes := result["pizza_sizes"].([]interface{})
	toppings := result["toppings"].([]interface{})

	if len(styles) != 8 {
		t.Errorf("Expected 8 pizza styles, got %d", len(styles))
	}
	if len(sizes) != 4 {
		t.Errorf("Expected 4 pizza sizes, got %d", len(sizes))
	}
	if len(toppings) != 15 {
		t.Errorf("Expected 15 toppings, got %d", len(toppings))
	}
}

func TestMenuHandler_WithAPIKey(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	Init(db)

	// Set API key
	testAPIKey := "test-api-key-123"
	apiKey = testAPIKey

	req := httptest.NewRequest(http.MethodGet, "/api/menu", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()

	// Test
	MenuHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestMenuHandler_InvalidAPIKey(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	Init(db)

	// Set API key
	apiKey = "correct-api-key"

	req := httptest.NewRequest(http.MethodGet, "/api/menu", nil)
	req.Header.Set("Authorization", "Bearer wrong-api-key")
	w := httptest.NewRecorder()

	// Test
	MenuHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] != "Invalid or missing API key" {
		t.Errorf("Expected error message, got %v", result["error"])
	}
}

func TestMenuHandler_MissingAPIKey(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	Init(db)

	// Set API key requirement
	apiKey = "required-api-key"

	req := httptest.NewRequest(http.MethodGet, "/api/menu", nil)
	// Don't set Authorization header
	w := httptest.NewRecorder()

	// Test
	MenuHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestMenuHandler_MethodNotAllowed(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	Init(db)
	apiKey = ""

	req := httptest.NewRequest(http.MethodPost, "/api/menu", nil)
	w := httptest.NewRecorder()

	// Test
	MenuHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestCreateOrderHandler_Success(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()

	// Create a test user
	userID := database.SeedTestUser(t, db)

	Init(db)
	apiKey = "" // Unsecured mode

	orderJSON := `{
		"user_id": 1,
		"pizza_style": "New York Style",
		"size_id": 2,
		"left_toppings": ["Pepperoni"],
		"right_toppings": ["Extra Cheese"],
		"whole_toppings": ["Mushrooms"]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(orderJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Test
	CreateOrderHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		body := w.Body.String()
		t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check order fields
	if result["pizza_style"] != "New York Style" {
		t.Errorf("Expected pizza_style 'New York Style', got '%v'", result["pizza_style"])
	}
	if result["size"] != "Medium" {
		t.Errorf("Expected size 'Medium', got '%v'", result["size"])
	}
	if result["status"] != "pending" {
		t.Errorf("Expected status 'pending', got '%v'", result["status"])
	}

	// Total should be base price (16.99) + 3 unique toppings (4.50) = 21.49
	expectedTotal := 21.49
	if total, ok := result["total"].(float64); !ok || total != expectedTotal {
		t.Errorf("Expected total %.2f, got %v", expectedTotal, result["total"])
	}

	// Verify the order was created with the correct user_id
	if int64(result["id"].(float64)) == 0 {
		t.Error("Expected non-zero order ID")
	}

	// Verify order is in database
	var dbTotal float64
	err := db.QueryRow("SELECT total FROM orders WHERE id = ?", int(result["id"].(float64))).Scan(&dbTotal)
	if err != nil {
		t.Errorf("Failed to query order: %v", err)
	}
	if dbTotal != expectedTotal {
		t.Errorf("Database total %.2f doesn't match expected %.2f", dbTotal, expectedTotal)
	}

	_ = userID // Use the variable
}

func TestCreateOrderHandler_InvalidUserID(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	Init(db)
	apiKey = ""

	orderJSON := `{
		"user_id": 999,
		"pizza_style": "New York Style",
		"size_id": 2,
		"left_toppings": ["Pepperoni"],
		"right_toppings": ["Pepperoni"]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(orderJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Test
	CreateOrderHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] != "Invalid user_id" {
		t.Errorf("Expected 'Invalid user_id' error, got %v", result["error"])
	}
}

func TestCreateOrderHandler_InvalidSizeID(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	database.SeedTestUser(t, db)
	Init(db)
	apiKey = ""

	orderJSON := `{
		"user_id": 1,
		"pizza_style": "New York Style",
		"size_id": 999,
		"left_toppings": ["Pepperoni"],
		"right_toppings": ["Pepperoni"]
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(orderJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Test
	CreateOrderHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] != "Invalid pizza size" {
		t.Errorf("Expected 'Invalid pizza size' error, got %v", result["error"])
	}
}

func TestCreateOrderHandler_InvalidJSON(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	Init(db)
	apiKey = ""

	invalidJSON := `{invalid json`

	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBufferString(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Test
	CreateOrderHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] != "Invalid JSON request" {
		t.Errorf("Expected 'Invalid JSON request' error, got %v", result["error"])
	}
}

func TestGetOrdersHandler_AllOrders(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	database.SeedTestUser(t, db)
	Init(db)
	apiKey = ""

	// Insert a test order
	_, err := db.Exec(`
		INSERT INTO orders (user_id, pizza_style, size, left_toppings, right_toppings, whole_toppings, total, status)
		VALUES (1, 'New York Style', 'Medium', 'Pepperoni', 'Olives', 'Mushrooms', 21.49, 'pending')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test order: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/orders/list", nil)
	w := httptest.NewRecorder()

	// Test
	GetOrdersHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var orders []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(orders) == 0 {
		t.Error("Expected at least one order, got none")
	}

	// Verify order data
	order := orders[0]
	if order["pizza_style"] != "New York Style" {
		t.Errorf("Expected pizza_style 'New York Style', got %v", order["pizza_style"])
	}
	if order["total"] != 21.49 {
		t.Errorf("Expected total 21.49, got %v", order["total"])
	}
}

func TestGetOrdersHandler_FilterByUserID(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	database.SeedTestUser(t, db)
	Init(db)
	apiKey = ""

	// Insert test orders for user 1
	_, err := db.Exec(`
		INSERT INTO orders (user_id, pizza_style, size, left_toppings, right_toppings, whole_toppings, total, status)
		VALUES
			(1, 'New York Style', 'Medium', 'Pepperoni', 'Olives', 'Mushrooms', 21.49, 'pending'),
			(1, 'Chicago Deep Dish', 'Large', 'Sausage', 'Peppers', '', 24.99, 'completed')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test orders: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/orders/list?user_id=1", nil)
	w := httptest.NewRecorder()

	// Test
	GetOrdersHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var orders []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(orders) != 2 {
		t.Errorf("Expected 2 orders for user 1, got %d", len(orders))
	}
}

func TestGetOrdersHandler_EmptyResult(t *testing.T) {
	// Setup
	db := database.InitTestDB(t)
	defer db.Close()
	Init(db)
	apiKey = ""

	// Don't insert any orders

	req := httptest.NewRequest(http.MethodGet, "/api/orders/list?user_id=999", nil)
	w := httptest.NewRecorder()

	// Test
	GetOrdersHandler(w, req)

	// Assertions
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var orders []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(orders) != 0 {
		t.Errorf("Expected 0 orders, got %d", len(orders))
	}
}
