# Brix Pizza

<img src="static/img/brix.png" alt="Brix Pizza Mascot" width="150"/>

A demo pizza ordering application built with Go and SQLite.

## Features

- User registration and authentication with bcrypt password hashing
- Session-based login system
- Order pizza with customizable toppings (split left/right)
- 8 different pizza styles (New York, Chicago, Detroit, etc.)
- View order history
- **Brix AI chatbot** — floating chat widget powered by any OpenAI-compatible inference server
- **REST API endpoints** for programmatic access
- **Structured logging** with zerolog for production-ready observability
- **Prometheus metrics** for monitoring and alerting
- SQLite database with automatic migrations
- Clean, responsive UI with brick oven theme
- Database-driven menu system

## Prerequisites

- Go 1.25 or higher
- SQLite3 (for development, usually pre-installed on macOS/Linux)
- MySQL 5.7+ or MariaDB 10.2+ (optional, for production)

> **🚀 Want to deploy to Kubernetes instead?** Skip to the [Kubernetes Deployment](#-kubernetes-deployment) section for quick deployment instructions.

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Run the application:

**Development (SQLite):**
```bash
go run main.go
```

Or build and run:
```bash
go build -o brix-pizza
./brix-pizza
```

**Production (MySQL with API Key):**
```bash
export BRIX_API_KEY="your-secure-api-key-here"
export DATABASE_URL="user:password@tcp(localhost:3306)/brix_pizza"
go run main.go
```

**Note:** The REST API uses session-based authentication. Users must login first to obtain a session cookie. Optionally set `BRIX_API_KEY` environment variable for an additional layer of security with Bearer token authentication.

3. Open your browser and navigate to:
```
http://localhost:8080
```

4. Create an account by clicking "Register" and filling out the form

## Environment Variables

The application can be configured using the following environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP server port. |
| `LOG_LEVEL` | No | `info` | Logging level: `debug`, `info`, `warn`, `error`, `fatal` |
| `BRIX_API_KEY` | No | _(none)_ | API key for securing REST API endpoints. If not set, API runs in unsecured mode (dev only). |
| `DATABASE_URL` | No | SQLite | MySQL connection string: `user:password@tcp(host:port)/database` |
| `CHATBOT_ENABLED` | No | _(none)_ | Set to any non-empty value (e.g. `true`) to show the Brix chat widget. |
| `CHATBOT_INFERENCE_URL` | When chatbot enabled | _(none)_ | Full URL to an OpenAI-compatible `/v1/chat/completions` endpoint. |
| `CHATBOT_MODEL` | No | `brix` | Model ID on the inference server. |
| `CHATBOT_TOKEN` | No | _(none)_ | Bearer token for inference server authentication. |
| `CHATBOT_TLS_SKIP_VERIFY` | No | _(none)_ | Set to `true` to skip TLS verification (self-signed certs). |

**Example configuration:**
```bash
# For production with MySQL and API security
export BRIX_API_KEY="your-secure-api-key-here"
export DATABASE_URL="brix_user:password@tcp(localhost:3306)/brix_pizza"
go run main.go
```

**Generate a secure API key:**
```bash
# Generate a random 32-character hex key
openssl rand -hex 32
```

## Database Configuration

The application supports both SQLite (development) and MySQL (production) databases.

### SQLite (Default)

SQLite is used by default when no `DATABASE_URL` environment variable is set. This is suitable for development and testing but **not recommended for production**.

When using SQLite, you'll see a warning on startup:
```
⚠️  WARNING: Using SQLite database (not recommended for production)
⚠️  Set DATABASE_URL environment variable to use MySQL
```

### MySQL (Production)

To use MySQL, set the `DATABASE_URL` environment variable:

**Format:**
```
DATABASE_URL="username:password@tcp(host:port)/database_name"
```

**Examples:**
```bash
# Local MySQL
export DATABASE_URL="root:password@tcp(localhost:3306)/brix_pizza"

# Remote MySQL
export DATABASE_URL="brix_user:secure_password@tcp(db.example.com:3306)/brix_pizza"

# With additional parameters
export DATABASE_URL="user:pass@tcp(localhost:3306)/brix_pizza?charset=utf8mb4&parseTime=True&loc=Local"
```

**Setting up MySQL database:**
```sql
CREATE DATABASE brix_pizza CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'brix_user'@'localhost' IDENTIFIED BY 'your_password';
GRANT ALL PRIVILEGES ON brix_pizza.* TO 'brix_user'@'localhost';
FLUSH PRIVILEGES;
```

The application will automatically create all necessary tables using migrations when it starts.

## Monitoring and Logging

Brix Pizza includes production-ready monitoring and logging capabilities.

### Structured Logging

The application uses [zerolog](https://github.com/rs/zerolog) for structured JSON logging with human-readable console output in development.

**Key logged events:**

- User login/logout with user ID, email, session ID, and remote address
- Order creation with order ID, user details, pizza configuration, and total
- HTTP requests with request ID, method, path, status, duration, and bytes transferred
- Database errors and validation failures

**Example log output:**

```
2025-12-13T10:00:24-06:00 INF Request completed bytes=5393 duration_ms=0.329042 method=GET path=/ remote_addr=[::1]:60841 request_id=23b3c278-268c-4d7c-b6ba-eda9748e264a status=200
2025-12-13T10:05:12-06:00 INF User logged in successfully email=john@example.com name=John Doe remote_addr=192.168.1.100:54321 session_id=abc123 user_id=42
2025-12-13T10:07:45-06:00 INF Order created successfully email=john@example.com left_toppings=Pepperoni order_id=15 pizza_style=New York Style right_toppings=Mushrooms size=Large total=18.99 user_id=42 whole_toppings=
```

**Configure log level:**
```bash
export LOG_LEVEL=debug  # debug, info, warn, error, fatal
go run main.go
```

### Prometheus Metrics

The application exposes Prometheus metrics at `/metrics` for monitoring:

**Available metrics:**

- `http_requests_total` - Total HTTP requests by method, path, and status
- `http_request_duration_seconds` - HTTP request latency histogram
- `orders_total` - Total orders by pizza style and size
- `orders_revenue` - Total revenue from all orders
- `user_logins_total` - Total successful user logins
- `database_connections` - Current database connections (open, in-use, idle)

**Access metrics:**

```bash
curl http://localhost:8080/metrics
```

**Example output:**

```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/",status="200"} 142

# HELP orders_total Total number of orders
# TYPE orders_total counter
orders_total{pizza_style="New York Style",size="Large"} 25
orders_total{pizza_style="Chicago Deep Dish",size="Medium"} 18

# HELP orders_revenue Total revenue from orders
# TYPE orders_revenue counter
orders_revenue 1247.85
```

## Project Structure

```
brix-pizza/
├── main.go                      # Application entry point
├── internal/                    # Internal packages
│   ├── models/
│   │   └── models.go           # Data structures and types
│   ├── database/
│   │   └── database.go         # Database initialization and migrations
│   ├── handlers/
│   │   ├── handlers.go         # HTML page handlers
│   │   └── chatbot.go          # Brix AI chat proxy handler
│   ├── logger/
│   │   └── logger.go           # Structured logging configuration
│   ├── middleware/
│   │   ├── logging.go          # HTTP request logging middleware
│   │   └── prometheus.go       # Prometheus metrics middleware
│   ├── metrics/
│   │   └── metrics.go          # Prometheus metrics definitions
│   ├── health/
│   │   └── health.go           # Health check handlers
│   └── api/
│       └── api.go              # REST API handlers
├── templates/                   # HTML templates
│   ├── home.html               # Landing page
│   ├── register.html           # User registration
│   ├── login.html              # User login
│   ├── order.html              # Order form
│   └── orders.html             # Orders dashboard
├── static/                      # Static assets
│   ├── css/
│   │   ├── style.css           # Application styles
│   │   └── chatbot.css         # Brix chat widget styles
│   ├── js/
│   │   ├── price-calculator.js      # Real-time price calculator
│   │   ├── pizza-visualizer.js      # Interactive pizza builder
│   │   ├── form-validator.js        # Order form validation
│   │   ├── registration-validator.js # Registration form validation
│   │   ├── smooth-scroll.js         # Smooth scrolling navigation
│   │   └── chatbot.js               # Brix chat widget
│   ├── img/
│   │   ├── brix.png            # Mascot image
│   │   ├── brix2.png           # Alternative mascot
│   │   └── bricks1.jpg         # Brick background
│   └── video/
│       └── smoke.mp4           # Smoke video effect
├── docs/                        # Documentation
│   └── API_EXAMPLES.md         # API usage examples
├── db/                          # Database directory (created on first run)
│   └── orders.db               # SQLite database
└── go.mod                      # Go module definition
```

## Usage

### Register and Login

1. Go to the home page at `http://localhost:8080`
2. Click "Register" to create a new account
3. Fill in your details (first name, last name, email, phone, password)
4. You'll be automatically logged in and redirected to the order page

### Place an Order

1. After logging in, you'll be on the order page
2. Select pizza style (Chicago, New York, Detroit, etc.)
3. Choose size (Small 10", Medium 12", Large 14", Extra Large 16")
4. Select toppings for left and/or right side of the pizza
5. Click "Place Order"

### View Orders

Navigate to "My Orders" to see your order history with order details.

## REST API

The application provides REST API endpoints for programmatic access. All API endpoints are under the `/api/v1/*` path.

**🔐 Authentication:** API endpoints require a Bearer token set via the `BRIX_API_KEY` environment variable.

**📖 For detailed examples and complete workflows, see [docs/API_EXAMPLES.md](docs/API_EXAMPLES.md)**

### Get Menu

Retrieve available pizza styles, sizes, and toppings.

**Endpoint:** `GET /api/v1/menu`

**Authentication:** Bearer token in `Authorization` header

**Example:**
```bash
curl -H "Authorization: Bearer your-api-key-here" \
  http://localhost:8080/api/v1/menu
```

Or using an environment variable:
```bash
export API_KEY="your-api-key-here"
curl -H "Authorization: Bearer $API_KEY" \
  http://localhost:8080/api/v1/menu
```

**Response:**
```json
{
  "pizza_styles": [
    {
      "id": 1,
      "name": "New York Style",
      "description": "Thin, crispy crust with a wide diameter",
      "emoji": "🗽"
    }
  ],
  "pizza_sizes": [
    {
      "id": 1,
      "name": "Small",
      "diameter": "10\"",
      "base_price": 12.99
    }
  ],
  "toppings": [
    {
      "id": 1,
      "name": "Pepperoni",
      "price": 1.50,
      "category": "meat"
    }
  ]
}
```

### Create Order

Place an order for the authenticated user.

**Endpoint:** `POST /api/v1/orders`

**Authentication:** Session cookie required (login first)

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

**Note:** The `user_id` is automatically extracted from your session.

**Example:**
```bash
# First, login to get session cookie
curl -c cookies.txt -X POST http://localhost:8080/login \
  -d "email=user@example.com&password=yourpassword"

# Then, place order with session cookie
curl -b cookies.txt -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "pizza_style": "New York Style",
    "size_id": 2,
    "left_toppings": ["Pepperoni", "Mushrooms"],
    "whole_toppings": ["Extra Cheese"]
  }'
```

**Response:**

```json
{
  "id": 123,
  "pizza_style": "New York Style",
  "size": "Medium",
  "left_toppings": "Pepperoni, Mushrooms",
  "right_toppings": "Pepperoni, Bell Peppers",
  "total": 22.99,
  "status": "pending",
  "created_at": "2025-12-01T10:30:00Z"
}
```

### List Orders

Get order history for the authenticated user.

**Endpoint:** `GET /api/v1/orders/list`

**Authentication:** Session cookie required (returns orders for logged-in user only)

**Example:**

```bash
curl -b cookies.txt http://localhost:8080/api/v1/orders/list
```

**Response:**

```json
[
  {
    "id": 123,
    "pizza_style": "New York Style",
    "size": "Medium",
    "left_toppings": "Pepperoni, Mushrooms",
    "right_toppings": "Pepperoni, Bell Peppers",
    "total": 22.99,
    "status": "pending",
    "created_at": "2025-12-01T10:30:00Z"
  }
]
```

### API Error Responses

All API endpoints return JSON error messages with appropriate HTTP status codes:

```json
{
  "error": "Invalid email or password"
}
```

Common status codes:

- `200 OK` - Success
- `201 Created` - Order created successfully
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Authentication failed
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

## Database Schema

The application uses the following tables:

### users
| Column        | Type     | Description                      |
|---------------|----------|----------------------------------|
| id            | INTEGER  | Primary key (auto-increment)     |
| first_name    | TEXT     | User's first name                |
| last_name     | TEXT     | User's last name                 |
| email         | TEXT     | Email (unique)                   |
| phone         | TEXT     | Phone number                     |
| password_hash | TEXT     | Bcrypt hashed password           |
| created_at    | DATETIME | Account creation timestamp       |

### orders
| Column         | Type     | Description                      |
|----------------|----------|----------------------------------|
| id             | INTEGER  | Primary key (auto-increment)     |
| user_id        | INTEGER  | Foreign key to users table       |
| pizza_style    | TEXT     | Pizza style (e.g., Chicago)      |
| size           | TEXT     | Size name (e.g., Medium)         |
| left_toppings  | TEXT     | Toppings for left side           |
| right_toppings | TEXT     | Toppings for right side          |
| total          | REAL     | Order total in dollars           |
| status         | TEXT     | Order status (default: pending)  |
| created_at     | DATETIME | Order timestamp                  |

### pizza_styles
| Column        | Type     | Description                      |
|---------------|----------|----------------------------------|
| id            | INTEGER  | Primary key (auto-increment)     |
| name          | TEXT     | Style name (unique)              |
| description   | TEXT     | Style description                |
| emoji         | TEXT     | Emoji icon for style             |
| active        | INTEGER  | 1=active, 0=inactive (default: 1)|
| display_order | INTEGER  | Display order (default: 0)       |
| created_at    | DATETIME | Creation timestamp               |

### pizza_sizes
| Column        | Type     | Description                      |
|---------------|----------|----------------------------------|
| id            | INTEGER  | Primary key (auto-increment)     |
| name          | TEXT     | Size name (unique)               |
| diameter      | TEXT     | Size diameter (e.g., "12\"")     |
| base_price    | REAL     | Base price for this size         |
| display_order | INTEGER  | Display order (default: 0)       |
| active        | INTEGER  | 1=active, 0=inactive (default: 1)|
| created_at    | DATETIME | Creation timestamp               |

### toppings
| Column        | Type     | Description                      |
|---------------|----------|----------------------------------|
| id            | INTEGER  | Primary key (auto-increment)     |
| name          | TEXT     | Topping name (unique)            |
| price         | REAL     | Price per topping                |
| category      | TEXT     | Category (meat/veggie/cheese)    |
| active        | INTEGER  | 1=active, 0=inactive (default: 1)|
| display_order | INTEGER  | Display order (default: 0)       |
| created_at    | DATETIME | Creation timestamp               |

### migrations
| Column     | Type     | Description                      |
|------------|----------|----------------------------------|
| id         | INTEGER  | Primary key (auto-increment)     |
| name       | TEXT     | Migration name (unique)          |
| applied_at | DATETIME | Migration applied timestamp      |

## Database Migrations

The application uses a simple migration system to manage database schema changes. Migrations are automatically applied when the application starts.

### How It Works

1. On startup, the app creates a `migrations` table to track applied migrations
2. Each migration has a unique name (e.g., `001_create_users_table`)
3. The app checks which migrations have been applied
4. Only new migrations are executed
5. Once applied, a migration is recorded in the `migrations` table

### Kubernetes-Safe Initialization

The application is designed to safely handle multiple pods starting simultaneously in a Kubernetes deployment:

- **Database-Level Locking**: The seed data migration (002_seed_menu_data) uses database transactions with locking to prevent race conditions
- **Idempotent Seeding**: Before inserting seed data, the application checks if data already exists, preventing duplicates
- **Transaction Safety**: All seed operations are wrapped in transactions that roll back on failure
- **Concurrent Pod Starts**: Multiple pods can start at the same time without corrupting the database or causing duplicate menu items

The `seedMenuData()` function uses:
- SQLite: Exclusive transactions (BEGIN EXCLUSIVE) automatically prevent concurrent writes
- MySQL: Row-level locks (`FOR UPDATE`) ensure only one pod seeds data at a time

### Development Database Reset

During active development, you can reset the database to start fresh:

```bash
rm db/orders.db
go run main.go
```

The application will automatically recreate all tables and seed the menu data on startup.

### Adding a New Migration

To add a new migration, edit the `runMigrations()` function in [internal/database/database.go](internal/database/database.go):

```go
migrations := []struct {
    name string
    sql  string
}{
    // Existing migrations...
    {
        name: "003_add_new_column",
        sql: `ALTER TABLE orders ADD COLUMN delivery_address TEXT;`,
    },
}
```

**Important:**
- Always use sequential numbering (001, 002, 003, etc.)
- Never modify existing migrations after they've been applied
- Use `CREATE TABLE IF NOT EXISTS` for table creation
- Test migrations thoroughly before deploying

### Viewing Applied Migrations

You can check which migrations have been applied by querying the database:

```bash
sqlite3 db/orders.db "SELECT * FROM migrations;"
```

### Resetting the Database (Development Only)

If you need to completely reset your database during development:

```bash
rm db/orders.db
# Then restart the application - all migrations will run from scratch
```

**Warning:** This will delete all data including users and orders!

## Testing

Brix Pizza includes comprehensive unit tests for all packages. Tests use the same database migrations as production to ensure consistency.

### Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Generate coverage report (creates coverage.html)
make test-coverage

# Or use go test directly
go test ./internal/... -v
go test ./internal/... -race -coverprofile=coverage.out
```

### Test Coverage

- **Health package**: 100% coverage (3 tests)
- **Handlers package**: 80.3% coverage (19 tests)
- **API package**: 78.8% coverage (12 tests)
- **Database package**: 63.2% coverage (3 tests)
- **Overall**: 76.7% coverage

### Test Structure

Tests follow Go best practices:

- Each package has its own `*_test.go` files
- Database tests use in-memory SQLite for speed
- Test helpers in `internal/database/testing.go` reuse production migration code
- All tests can run in parallel with `-race` flag

**Example test:**

```go
func TestMenuHandler_Success(t *testing.T) {
    // Setup: Use real database initialization
    db := database.InitTestDB(t)
    defer db.Close()

    // Test the handler
    req := httptest.NewRequest(http.MethodGet, "/api/v1/menu", nil)
    w := httptest.NewRecorder()
    MenuHandler(w, req)

    // Assertions
    if w.Result().StatusCode != http.StatusOK {
        t.Error("Expected 200 OK")
    }
}
```

### CI/CD Integration

To run tests in CI pipeline:

```bash
#!/bin/bash
set -e

# Run tests with coverage
go test ./internal/... -race -coverprofile=coverage.out -covermode=atomic

# Fail if coverage below threshold (optional)
go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' | \
  awk '{if ($1 < 70) exit 1}'
```

## 🚀 Kubernetes Deployment

Deploy Brix Pizza to Kubernetes in minutes with built-in MySQL database, health checks, and auto-scaling.

### Quick Deploy

```bash
# Deploy everything (MySQL + Application)
kubectl apply -f deployment/k8s/brix/
kubectl apply -f deployment/k8s/mysql/

# Check status
kubectl get pods -n brix

# Access locally
kubectl port-forward -n brix svc/brix-pizza 8080:80
```

**That's it!** The app is now running at `http://localhost:8080`

### MySQL via OpenShift Virtualization (optional)

Run MySQL inside a VM instead of a container using OpenShift Virtualization:

```bash
oc apply -f deployment/k8s/mysql-vm/
```

See [deployment/k8s/mysql-vm/README.md](deployment/k8s/mysql-vm/README.md) for prerequisites and configuration.

### Enabling the Brix Chatbot

The chat widget is off by default. To enable it, update `deployment/k8s/brix/02-configmap.yaml`:

```yaml
CHATBOT_ENABLED: "true"
CHATBOT_INFERENCE_URL: "https://your-model.apps.cluster.example.com/v1/chat/completions"
CHATBOT_MODEL: "your-model-id"
CHATBOT_TLS_SKIP_VERIFY: "true"  # only if using self-signed certificates
```

Add the inference server bearer token to `deployment/k8s/brix/03-secret.yaml`:

```bash
oc create secret generic brix-pizza-secrets \
  --from-literal=api-key=$(openssl rand -hex 32) \
  --from-literal=chatbot-token=<your-token> \
  -n brix
```

### What's Included

- ✅ **MySQL Database** - StatefulSet with persistent storage
- ✅ **MySQL VM** - OpenShift Virtualization alternative in `deployment/k8s/mysql-vm/`
- ✅ **Auto-Scaling** - Horizontal Pod Autoscaler (1-5 replicas)
- ✅ **Health Checks** - Liveness and readiness probes
- ✅ **Monitoring** - Prometheus metrics at `/metrics`
- ✅ **Production-Ready** - Security context, resource limits, graceful shutdown

### ⚠️ Before Production

The default deployment uses **demo credentials** for quick testing. Before production:

1. Change passwords in `deployment/k8s/mysql/02-secrets.yaml`
2. Change API key in `deployment/k8s/brix/03-secret.yaml`
3. Update image tag in `deployment/k8s/brix/04-deployment.yaml` (currently `quay.io/rh-ee-eshanks/brix-pizza:v1.1.0`)

### Full Documentation

See **[deployment/README.md](deployment/README.md)** for:

- Detailed configuration options
- Production deployment best practices
- Troubleshooting guide
- Ingress/LoadBalancer setup

## License

MIT
