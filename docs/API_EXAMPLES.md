# Brix Pizza API Examples

This document provides practical examples for using the Brix Pizza REST API.

## Authentication

The API uses **session-based authentication**. Users must login first to obtain a session cookie, then use that cookie for subsequent API requests.

### Two-Layer Authentication

1. **Session Cookie** (Required): Identifies the authenticated user
2. **API Key** (Optional): Additional security layer for production environments

### Logging In

Before using the API, you must authenticate via the web login endpoint:

```bash
# Login to get session cookie
curl -c cookies.txt -X POST http://localhost:8080/login \
  -d "email=user@example.com&password=yourpassword"
```

This creates a `cookies.txt` file containing your session cookie.

### Using the Session Cookie

Include the cookie file in all API requests:

```bash
curl -b cookies.txt http://localhost:8080/api/v1/menu
```

### Optional API Key (Production)

For additional security in production, you can set an API key:

```bash
export BRIX_API_KEY="your-secret-api-key-here"
go run main.go
```

Then include both the session cookie AND API key:

```bash
curl -b cookies.txt \
  -H "Authorization: Bearer your-secret-api-key-here" \
  http://localhost:8080/api/v1/menu
```

**Note:** If `BRIX_API_KEY` is not set, only session authentication is required (for development).

## Base URL

```
http://localhost:8080/api/v1
```

**Note:** All API endpoints are versioned. The current version is `v1`.

---

## API Endpoints

### 1. Get Menu

Retrieve all available pizza styles, sizes, and toppings.

**Endpoint:** `GET /api/v1/menu`

**Authentication:** Session cookie required (login first)

**Request:**
```bash
# First, login to get session cookie
curl -c cookies.txt -X POST http://localhost:8080/login \
  -d "email=user@example.com&password=yourpassword"

# Then, use the session cookie to get the menu
curl -b cookies.txt http://localhost:8080/api/v1/menu
```

**Response (200 OK):**
```json
{
  "pizza_styles": [
    {
      "ID": 1,
      "Name": "New York Style",
      "Description": "Thin, crispy crust with a wide diameter",
      "Emoji": "🗽"
    },
    {
      "ID": 2,
      "Name": "Chicago Deep Dish",
      "Description": "Thick, buttery crust with layers of cheese and toppings",
      "Emoji": "🏙️"
    }
  ],
  "pizza_sizes": [
    {
      "ID": 1,
      "Name": "Small",
      "Diameter": "10\"",
      "BasePrice": 12.99
    },
    {
      "ID": 2,
      "Name": "Medium",
      "Diameter": "12\"",
      "BasePrice": 16.99
    },
    {
      "ID": 3,
      "Name": "Large",
      "Diameter": "14\"",
      "BasePrice": 20.99
    },
    {
      "ID": 4,
      "Name": "Extra Large",
      "Diameter": "16\"",
      "BasePrice": 24.99
    }
  ],
  "toppings": [
    {
      "ID": 1,
      "Name": "Pepperoni",
      "Price": 1.5,
      "Category": "meat"
    },
    {
      "ID": 6,
      "Name": "Mushrooms",
      "Price": 1.5,
      "Category": "veggie"
    },
    {
      "ID": 14,
      "Name": "Extra Cheese",
      "Price": 1.5,
      "Category": "cheese"
    }
  ]
}
```

**Using jq to format:**
```bash
curl -s -b cookies.txt http://localhost:8080/api/v1/menu | jq '.'
```

**Extract only pizza styles:**
```bash
curl -s -b cookies.txt http://localhost:8080/api/v1/menu | jq '.pizza_styles'
```

---

### 2. Create Order

Place a new pizza order for the authenticated user.

**Endpoint:** `POST /api/v1/orders`

**Authentication:** Session cookie required (user is identified from session)

**Request Body:**
```json
{
  "pizza_style": "New York Style",
  "size_id": 2,
  "left_toppings": ["Pepperoni", "Mushrooms"],
  "right_toppings": ["Pepperoni", "Bell Peppers"],
  "whole_toppings": ["Extra Cheese"]
}
```

**Note:** The `user_id` is automatically extracted from your session. You don't need to provide it in the request body.

**Example 1: Whole pizza with pepperoni**
```bash
curl -b cookies.txt -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "pizza_style": "New York Style",
    "size_id": 2,
    "whole_toppings": ["Pepperoni"]
  }'
```

**Example 2: Half-and-half pizza**
```bash
curl -b cookies.txt -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "pizza_style": "Chicago Deep Dish",
    "size_id": 3,
    "left_toppings": ["Pepperoni", "Italian Sausage", "Mushrooms"],
    "right_toppings": ["Mushrooms", "Bell Peppers", "Onions"]
  }'
```

**Example 3: Vegetarian whole pizza**
```bash
curl -b cookies.txt -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "pizza_style": "Neapolitan",
    "size_id": 2,
    "whole_toppings": ["Mushrooms", "Bell Peppers", "Onions", "Tomatoes"]
  }'
```

**Example 4: Three-way split (left, whole, right)**
```bash
curl -b cookies.txt -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "pizza_style": "Detroit Style",
    "size_id": 4,
    "left_toppings": ["Pepperoni"],
    "whole_toppings": ["Extra Cheese"],
    "right_toppings": ["Mushrooms"]
  }'
```

**Response (201 Created):**
```json
{
  "id": 42,
  "pizza_style": "New York Style",
  "size": "Medium",
  "left_toppings": "Pepperoni, Mushrooms",
  "right_toppings": "Pepperoni, Bell Peppers",
  "total": 22.99,
  "status": "pending",
  "created_at": "2025-12-01T20:30:00Z"
}
```

**Error Response (401 Unauthorized - No Session):**
```json
{
  "error": "Unauthorized - valid session required"
}
```

**Error Response (400 Bad Request - Invalid Size):**
```json
{
  "error": "Invalid pizza size"
}
```

---

### 3. List Orders

Get order history for the authenticated user.

**Endpoint:** `GET /api/v1/orders/list`

**Authentication:** Session cookie required (returns orders for the logged-in user only)

**Note:** The API automatically returns orders for the authenticated user. You cannot query other users' orders.

**Example 1: Get your orders**
```bash
curl -b cookies.txt http://localhost:8080/api/v1/orders/list
```

**Example 2: Using formatted output with jq**
```bash
curl -s -b cookies.txt http://localhost:8080/api/v1/orders/list | jq '.'
```

**Example 3: Get only order IDs and totals**
```bash
curl -s -b cookies.txt http://localhost:8080/api/v1/orders/list | \
  jq '.[] | {id: .id, total: .total}'
```

**Example 4: Calculate total amount spent**
```bash
curl -s -b cookies.txt http://localhost:8080/api/v1/orders/list | \
  jq '[.[] | .total] | add'
```

**Response (200 OK):**
```json
[
  {
    "id": 42,
    "pizza_style": "New York Style",
    "size": "Medium",
    "left_toppings": "Pepperoni, Mushrooms",
    "right_toppings": "Pepperoni, Bell Peppers",
    "total": 22.99,
    "status": "pending",
    "created_at": "2025-12-01T20:30:00Z"
  },
  {
    "id": 41,
    "pizza_style": "Chicago Deep Dish",
    "size": "Large",
    "left_toppings": "Pepperoni, Italian Sausage",
    "right_toppings": "Pepperoni, Italian Sausage",
    "total": 24.99,
    "status": "completed",
    "created_at": "2025-11-30T18:15:00Z"
  }
]
```

**Error Response (401 Unauthorized):**
```json
{
  "error": "Invalid or missing API key"
}
```

---

## Complete Workflow Example

Here's a complete workflow from getting the menu to placing an order:

```bash
#!/bin/bash

# Set your API key
API_KEY="your-secret-api-key-here"
BASE_URL="http://localhost:8080"

# Step 1: Get the menu to see available options
echo "=== Getting menu ==="
curl -s -H "Authorization: Bearer $API_KEY" \
  $BASE_URL/api/menu | jq '.pizza_styles[] | {id: .ID, name: .Name}'

# Step 2: Place an order for user ID 1
echo -e "\n=== Placing order ==="
ORDER_RESPONSE=$(curl -s -X POST $BASE_URL/api/orders \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "pizza_style": "New York Style",
    "size_id": 2,
    "left_toppings": ["Pepperoni", "Mushrooms"],
    "right_toppings": ["Pepperoni", "Extra Cheese"]
  }')

echo "$ORDER_RESPONSE" | jq '.'

# Step 3: Get order ID
ORDER_ID=$(echo "$ORDER_RESPONSE" | jq -r '.id')
echo -e "\n=== Order created with ID: $ORDER_ID ==="

# Step 4: View all orders for user
echo -e "\n=== Viewing orders for user 1 ==="
curl -s -H "Authorization: Bearer $API_KEY" \
  $BASE_URL/api/orders/list?user_id=1 | jq '.'

# Step 5: View all orders (admin)
echo -e "\n=== Viewing all orders (admin) ==="
curl -s -H "Authorization: Bearer $API_KEY" \
  $BASE_URL/api/orders/list | jq 'length' -
```

---

## Setting Up for Production

### 1. Generate a Secure API Key

```bash
# Generate a random 32-character API key
export BRIX_API_KEY=$(openssl rand -hex 32)
echo "Your API key: $BRIX_API_KEY"
```

### 2. Run the Application

```bash
export BRIX_API_KEY="your-secure-key-here"
export DATABASE_URL="user:password@tcp(localhost:3306)/brix_pizza"
go run main.go
```

### 3. Store API Key Securely

**For development:**
```bash
# Add to your ~/.bashrc or ~/.zshrc
export BRIX_API_KEY="dev-key-12345"
```

**For production:**
- Use environment variable management (Docker secrets, Kubernetes secrets, etc.)
- Never commit API keys to version control
- Rotate keys regularly

---

## Error Codes Reference

| Status Code | Meaning | Common Causes |
|-------------|---------|---------------|
| 200 OK | Success | Request completed successfully |
| 201 Created | Order created | New order was successfully created |
| 400 Bad Request | Invalid input | Missing fields, invalid user_id/size_id, or malformed JSON |
| 401 Unauthorized | Auth failed | Missing or invalid API key |
| 405 Method Not Allowed | Wrong method | Using GET when POST required, or vice versa |
| 500 Internal Server Error | Server error | Database connection issues or server bug |

---

## Tips and Tricks

### Save API key in a variable (for testing)
```bash
export API_KEY="your-secret-api-key-here"

curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/menu
```

### Create a reusable function
```bash
# Add to ~/.bashrc or ~/.zshrc
brix_api() {
  curl -H "Authorization: Bearer $BRIX_API_KEY" "$@"
}

# Usage:
brix_api http://localhost:8080/api/v1/menu
```

### Pretty print with Python
```bash
curl -s -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/menu | python -m json.tool
```

### Save order response to file
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{...}' \
  -o order_receipt.json
```

### Check API availability
```bash
curl -I -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/menu
```

### Batch order creation
```bash
# Create orders for multiple users from a file
cat orders.json | while read line; do
  curl -X POST http://localhost:8080/api/v1/orders \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "$line"
done
```
