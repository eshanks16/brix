package models

import "time"

// User represents a registered user
type User struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	Phone     string
	Password  string
	CreatedAt time.Time
}

// Order represents a pizza order
type Order struct {
	ID            int
	UserID        int
	PizzaStyle    string
	Size          string
	LeftToppings  string
	RightToppings string
	WholeToppings string
	Total         float64
	Status        string
	CreatedAt     time.Time
	CustomerName  string
	CustomerPhone string
}

// Session represents a user session
type Session struct {
	UserID int
	Email  string
	Name   string
}

// MenuData holds all menu information for the order page
type MenuData struct {
	Session     *Session
	PizzaStyles []PizzaStyle
	PizzaSizes  []PizzaSize
	Toppings    []Topping
}

// PizzaStyle represents a pizza style from the database
type PizzaStyle struct {
	ID          int
	Name        string
	Description string
	Emoji       string
}

// PizzaSize represents a pizza size from the database
type PizzaSize struct {
	ID        int
	Name      string
	Diameter  string
	BasePrice float64
}

// Topping represents a pizza topping from the database
type Topping struct {
	ID       int
	Name     string
	Price    float64
	Category string
}

// API Request/Response types

// CreateOrderRequest represents the JSON request to create an order
type CreateOrderRequest struct {
	UserID        int      `json:"user_id"`        // ID of user placing order
	PizzaStyle    string   `json:"pizza_style"`
	SizeID        int      `json:"size_id"`
	LeftToppings  []string `json:"left_toppings"`
	RightToppings []string `json:"right_toppings"`
	WholeToppings []string `json:"whole_toppings"`
}

// OrderResponse represents the JSON response for an order
type OrderResponse struct {
	ID            int       `json:"id"`
	PizzaStyle    string    `json:"pizza_style"`
	Size          string    `json:"size"`
	LeftToppings  string    `json:"left_toppings"`
	RightToppings string    `json:"right_toppings"`
	WholeToppings string    `json:"whole_toppings"`
	Total         float64   `json:"total"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// MenuResponse represents the JSON response for the menu
type MenuResponse struct {
	PizzaStyles []PizzaStyle `json:"pizza_styles"`
	PizzaSizes  []PizzaSize  `json:"pizza_sizes"`
	Toppings    []Topping    `json:"toppings"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error string `json:"error"`
}
