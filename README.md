# Brix Pizza

A demo pizza ordering application built with Go and SQLite.

## Features

- Order pizza through a web interface
- View all orders in a dashboard
- SQLite database for persistent storage
- Clean, responsive UI

## Prerequisites

- Go 1.25 or higher
- SQLite3 (usually pre-installed on macOS/Linux)

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Run the application:
```bash
go run main.go
```

3. Open your browser and navigate to:
```
http://localhost:8080
```

## Project Structure

```
brix-pizza/
├── main.go              # Main application with HTTP handlers
├── templates/           # HTML templates
│   ├── index.html      # Order form
│   └── orders.html     # Orders dashboard
├── static/             # Static assets
│   └── css/
│       └── style.css   # Application styles
├── db/                 # Database directory (created on first run)
│   └── orders.db       # SQLite database
└── go.mod              # Go module definition
```

## Usage

### Place an Order

1. Go to the home page at `http://localhost:8080`
2. Fill in customer details (name, phone)
3. Select pizza type, size, and quantity
4. Optionally add extra toppings
5. Click "Place Order"

### View Orders

Navigate to `http://localhost:8080/orders` to see all orders in the system.

## Database Schema

The application uses a single `orders` table:

| Column     | Type     | Description                    |
|------------|----------|--------------------------------|
| id         | INTEGER  | Primary key (auto-increment)   |
| name       | TEXT     | Customer name                  |
| phone      | TEXT     | Customer phone                 |
| pizza      | TEXT     | Pizza type                     |
| size       | TEXT     | Pizza size (small/medium/large)|
| toppings   | TEXT     | Extra toppings                 |
| quantity   | INTEGER  | Number of pizzas               |
| total      | REAL     | Order total in dollars         |
| status     | TEXT     | Order status (default: pending)|
| created_at | DATETIME | Order timestamp                |

## License

MIT
