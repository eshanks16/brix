package handlers

import (
	"brix-pizza/internal/database"
	"brix-pizza/internal/models"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// setupTestHandlers initializes handlers with test database and mock templates
func setupTestHandlers(t *testing.T) *template.Template {
	db := database.InitTestDB(t)

	// Create minimal mock templates for testing
	// These templates just render basic text so we can test handler logic
	tmpl := template.Must(template.New("").Parse(`
		{{define "home.html"}}Home Page{{end}}
		{{define "register.html"}}{{if .Error}}{{.Error}}{{else}}Register Page{{end}}{{end}}
		{{define "login.html"}}{{if .Error}}{{.Error}}{{else}}Login Page{{end}}{{end}}
		{{define "order.html"}}Order Page{{end}}
		{{define "orders.html"}}Orders Page{{end}}
	`))

	Init(db, tmpl)

	// Clear sessions for clean state
	sessions = make(map[string]*models.Session)

	return tmpl
}

// createTestSession creates a test session and returns the session cookie
func createTestSession(userID int, email, name string) *http.Cookie {
	sessionID := generateSessionID()
	sessions[sessionID] = &models.Session{
		UserID: userID,
		Email:  email,
		Name:   name,
	}
	return &http.Cookie{
		Name:  "session_id",
		Value: sessionID,
		Path:  "/",
	}
}

func TestHomeHandler(t *testing.T) {
	setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	HomeHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Home Page") {
		t.Error("Expected home page content")
	}
}

func TestOrderPageHandler_WithoutSession(t *testing.T) {
	setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/order", nil)
	w := httptest.NewRecorder()

	OrderPageHandler(w, req)

	resp := w.Result()
	// Should redirect to login if no session
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status 303 (redirect), got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("Expected redirect to /login, got %s", location)
	}
}

func TestOrderPageHandler_WithSession(t *testing.T) {
	setupTestHandlers(t)

	// Create a test user
	userID := database.SeedTestUser(t, db)

	req := httptest.NewRequest(http.MethodGet, "/order", nil)
	req.AddCookie(createTestSession(int(userID), "test@example.com", "Test User"))
	w := httptest.NewRecorder()

	OrderPageHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Order Page") {
		t.Error("Expected order page content")
	}
}

func TestRegisterHandler_GET(t *testing.T) {
	setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	w := httptest.NewRecorder()

	RegisterHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Register Page") {
		t.Error("Expected register page content")
	}
}

func TestRegisterHandler_POST_Success(t *testing.T) {
	setupTestHandlers(t)

	formData := url.Values{}
	formData.Set("email", "newuser@example.com")
	formData.Set("first_name", "New")
	formData.Set("last_name", "User")
	formData.Set("phone", "555-555-5678")
	formData.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	RegisterHandler(w, req)

	resp := w.Result()
	// Should redirect to /order after successful registration
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status 303 (redirect), got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/order" {
		t.Errorf("Expected redirect to /order, got %s", location)
	}

	// Should have set a session cookie
	cookies := resp.Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "session_id" {
			found = true
			if cookie.Value == "" {
				t.Error("Session cookie has empty value")
			}
			break
		}
	}
	if !found {
		t.Error("Expected session_id cookie to be set")
	}

	// Verify user was created in database
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "newuser@example.com").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 user with email newuser@example.com, got %d", count)
	}
}

func TestRegisterHandler_POST_DuplicateEmail(t *testing.T) {
	setupTestHandlers(t)

	// Create existing user
	database.SeedTestUser(t, db)

	formData := url.Values{}
	formData.Set("email", "test@example.com") // Same as seeded user
	formData.Set("first_name", "Duplicate")
	formData.Set("last_name", "User")
	formData.Set("phone", "555-555-9999")
	formData.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	RegisterHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "already exists") {
		t.Error("Expected error message about existing account")
	}
}

func TestLoginHandler_GET(t *testing.T) {
	setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	LoginHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Login Page") {
		t.Error("Expected login page content")
	}
}

func TestLoginHandler_POST_Success(t *testing.T) {
	setupTestHandlers(t)

	// Create a user with known password
	password := "testpass123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	db.Exec(`INSERT INTO users (first_name, last_name, email, phone, password_hash)
		VALUES (?, ?, ?, ?, ?)`,
		"Login", "Test", "login@example.com", "555-1111", string(hashedPassword))

	formData := url.Values{}
	formData.Set("email", "login@example.com")
	formData.Set("password", password)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	LoginHandler(w, req)

	resp := w.Result()
	// Should redirect to / after successful login
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status 303 (redirect), got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/" {
		t.Errorf("Expected redirect to /, got %s", location)
	}

	// Should have set a session cookie
	cookies := resp.Cookies()
	found := false
	for _, cookie := range cookies {
		if cookie.Name == "session_id" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected session_id cookie to be set")
	}
}

func TestLoginHandler_POST_InvalidEmail(t *testing.T) {
	setupTestHandlers(t)

	formData := url.Values{}
	formData.Set("email", "nonexistent@example.com")
	formData.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	LoginHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid email or password") {
		t.Error("Expected error message about invalid credentials")
	}
}

func TestLoginHandler_POST_InvalidPassword(t *testing.T) {
	setupTestHandlers(t)

	// Create a user
	password := "correctpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	db.Exec(`INSERT INTO users (first_name, last_name, email, phone, password_hash)
		VALUES (?, ?, ?, ?, ?)`,
		"Pass", "Test", "passtest@example.com", "555-2222", string(hashedPassword))

	formData := url.Values{}
	formData.Set("email", "passtest@example.com")
	formData.Set("password", "wrongpassword")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	LoginHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid email or password") {
		t.Error("Expected error message about invalid credentials")
	}
}

func TestLogoutHandler(t *testing.T) {
	setupTestHandlers(t)

	// Create a session
	sessionCookie := createTestSession(1, "test@example.com", "Test User")

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()

	LogoutHandler(w, req)

	resp := w.Result()
	// Should redirect to /login
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status 303 (redirect), got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("Expected redirect to /login, got %s", location)
	}

	// Session should be deleted
	if _, exists := sessions[sessionCookie.Value]; exists {
		t.Error("Session should have been deleted")
	}

	// Cookie should be invalidated
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "session_id" {
			if cookie.MaxAge != -1 {
				t.Error("Expected cookie to be invalidated (MaxAge = -1)")
			}
		}
	}
}

func TestPlaceOrderHandler_WithoutSession(t *testing.T) {
	setupTestHandlers(t)

	formData := url.Values{}
	formData.Set("pizza_style", "New York Style")
	formData.Set("size", "Medium")

	req := httptest.NewRequest(http.MethodPost, "/place-order", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	PlaceOrderHandler(w, req)

	resp := w.Result()
	// Should redirect to login if no session
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status 303 (redirect), got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("Expected redirect to /login, got %s", location)
	}
}

func TestPlaceOrderHandler_Success(t *testing.T) {
	setupTestHandlers(t)

	// Create a test user
	userID := database.SeedTestUser(t, db)

	formData := url.Values{}
	formData.Set("pizza_style", "New York Style")
	formData.Set("size", "2") // Size ID for Medium
	// New radio button format: topping_{name} = "left" | "whole" | "right"
	formData.Set("topping_Pepperoni", "left")
	formData.Set("topping_Mushrooms", "whole")
	formData.Set("topping_Extra Cheese", "right")

	req := httptest.NewRequest(http.MethodPost, "/place-order", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(createTestSession(int(userID), "test@example.com", "Test User"))
	w := httptest.NewRecorder()

	PlaceOrderHandler(w, req)

	resp := w.Result()
	// Should redirect to /orders after successful order
	if resp.StatusCode != http.StatusSeeOther {
		body := w.Body.String()
		t.Errorf("Expected status 303 (redirect), got %d. Body: %s", resp.StatusCode, body)
	}

	location := resp.Header.Get("Location")
	if location != "/orders" {
		t.Errorf("Expected redirect to /orders, got %s", location)
	}

	// Verify order was created with correct toppings
	var leftToppings, wholeToppings, rightToppings string
	err := db.QueryRow("SELECT left_toppings, whole_toppings, right_toppings FROM orders WHERE user_id = ?", userID).
		Scan(&leftToppings, &wholeToppings, &rightToppings)
	if err != nil {
		t.Errorf("Failed to query order: %v", err)
	}

	if leftToppings != "Pepperoni" {
		t.Errorf("Expected left_toppings 'Pepperoni', got '%s'", leftToppings)
	}
	if wholeToppings != "Mushrooms" {
		t.Errorf("Expected whole_toppings 'Mushrooms', got '%s'", wholeToppings)
	}
	if rightToppings != "Extra Cheese" {
		t.Errorf("Expected right_toppings 'Extra Cheese', got '%s'", rightToppings)
	}
}

func TestOrdersHandler_WithoutSession(t *testing.T) {
	setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()

	OrdersHandler(w, req)

	resp := w.Result()
	// Should redirect to login if no session
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Expected status 303 (redirect), got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/login" {
		t.Errorf("Expected redirect to /login, got %s", location)
	}
}

func TestOrdersHandler_WithSession(t *testing.T) {
	setupTestHandlers(t)

	// Create a test user
	userID := database.SeedTestUser(t, db)

	// Create a test order
	db.Exec(`INSERT INTO orders (user_id, pizza_style, size, left_toppings, right_toppings, whole_toppings, total, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, "New York Style", "Medium", "Pepperoni", "Olives", "Mushrooms", 21.49, "pending")

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	req.AddCookie(createTestSession(int(userID), "test@example.com", "Test User"))
	w := httptest.NewRecorder()

	OrdersHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Orders Page") {
		t.Error("Expected orders page content")
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	// Sleep to ensure different nanosecond timestamp
	time.Sleep(1 * time.Microsecond)
	id2 := generateSessionID()

	if id1 == id2 {
		t.Error("Session IDs should be unique")
	}

	if len(id1) == 0 {
		t.Error("Session ID should not be empty")
	}
}

func TestGetSession_NoSession(t *testing.T) {
	setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	session := getSession(req)

	if session != nil {
		t.Error("Expected nil session when no cookie present")
	}
}

func TestGetSession_ValidSession(t *testing.T) {
	setupTestHandlers(t)

	cookie := createTestSession(1, "test@example.com", "Test User")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	session := getSession(req)

	if session == nil {
		t.Fatal("Expected valid session")
	}

	if session.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", session.Email)
	}
}

func TestGetSession_InvalidSessionID(t *testing.T) {
	setupTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "invalid-session-id",
	})
	session := getSession(req)

	if session != nil {
		t.Error("Expected nil session for invalid session ID")
	}
}

// Validation tests

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"Valid email", "test@example.com", false},
		{"Valid email with subdomain", "user@mail.example.com", false},
		{"Valid email with numbers", "user123@example.com", false},
		{"Empty email", "", true},
		{"Email too long", strings.Repeat("a", 100) + "@example.com", true},
		{"Invalid format - no @", "testexample.com", true},
		{"Invalid format - no domain", "test@", true},
		{"Invalid format - no local part", "@example.com", true},
		{"Whitespace trimmed", "  test@example.com  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name      string
		nameInput string
		fieldName string
		wantErr   bool
	}{
		{"Valid name", "John", "First name", false},
		{"Valid name with space", "Mary Jane", "First name", false},
		{"Valid name with hyphen", "Mary-Jane", "First name", false},
		{"Valid name with apostrophe", "O'Brien", "Last name", false},
		{"Empty name", "", "First name", true},
		{"Name too short", "A", "First name", true},
		{"Name too long", strings.Repeat("a", 51), "First name", true},
		{"Invalid characters - numbers", "John123", "First name", true},
		{"Invalid characters - symbols", "John@Doe", "First name", true},
		{"Whitespace trimmed", "  John  ", "First name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.nameInput, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
	}{
		{"Valid US phone", "555-555-1234", false},
		{"Valid phone with parens", "(555) 555-1234", false},
		{"Valid phone with spaces", "555 555 1234", false},
		{"Valid international format", "+1-555-555-1234", false},
		{"Valid 10 digits only", "5555551234", false},
		{"Empty phone", "", true},
		{"Phone too short", "555-5678", true},
		{"Phone too long", strings.Repeat("1", 21), true},
		{"Invalid characters", "555-abc-1234", true},
		{"Whitespace trimmed", "  555-555-1234  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePhone(tt.phone)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePhone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"Valid password", "password123", false},
		{"Valid minimum length", "123456", false},
		{"Empty password", "", true},
		{"Password too short", "12345", true},
		{"Password too long", strings.Repeat("a", 101), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
