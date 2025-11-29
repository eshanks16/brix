# Brix Pizza

A demo pizza ordering application built with Go and SQLite.

## Features

- User registration and authentication with bcrypt password hashing
- Session-based login system
- Order pizza with customizable toppings (split left/right)
- 8 different pizza styles (New York, Chicago, Detroit, etc.)
- View order history
- SQLite database with automatic migrations
- Clean, responsive UI with brick oven theme

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

4. Create an account by clicking "Register" and filling out the form

## Project Structure

```
brix-pizza/
├── main.go              # Main application with HTTP handlers and migrations
├── templates/           # HTML templates
│   ├── home.html       # Landing page
│   ├── register.html   # User registration
│   ├── login.html      # User login
│   ├── order.html      # Order form
│   └── orders.html     # Orders dashboard
├── static/             # Static assets
│   ├── css/
│   │   └── style.css   # Application styles
│   └── brix.png        # Mascot image
├── db/                 # Database directory (created on first run)
│   └── orders.db       # SQLite database
└── go.mod              # Go module definition
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

Navigate to "My Orders" to see your order history with order details and status.

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
| size           | TEXT     | Size (small/medium/large/xl)     |
| left_toppings  | TEXT     | Toppings for left side           |
| right_toppings | TEXT     | Toppings for right side          |
| total          | REAL     | Order total in dollars           |
| status         | TEXT     | Order status (default: pending)  |
| created_at     | DATETIME | Order timestamp                  |

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

### Adding a New Migration

To add a new migration, edit the `runMigrations()` function in [main.go](main.go):

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

## License

MIT
