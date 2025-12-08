# Database Support - SQLite and MySQL

Brix Pizza supports both SQLite and MySQL databases with **automatic detection** based on environment variables.

## How It Works

The application automatically chooses the database based on the `DATABASE_URL` environment variable:

```go
DATABASE_URL not set  →  SQLite (local file: ./db/orders.db)
DATABASE_URL is set   →  MySQL (connection string)
```

## Database Comparison

| Feature | SQLite | MySQL |
|---------|--------|-------|
| **Setup** | Automatic, no configuration | Requires separate MySQL server |
| **Multi-pod** | ❌ Each pod has separate data | ✅ All pods share data |
| **Production** | ⚠️ Single replica only | ✅ Fully scalable |
| **Development** | ✅ Perfect for local dev | ⚠️ Requires MySQL setup |
| **Backups** | Manual file copy | Standard MySQL tools |
| **Performance** | Fast for single pod | Better for concurrent access |

## Using SQLite (Default)

**No configuration needed!** Just deploy the app:

```bash
oc apply -f brix/04-deployment.yaml
```

The app will:
- Create `./db/orders.db` on first run
- Run migrations automatically
- Seed menu data (pizza styles, sizes, toppings)

**Important:** With SQLite, set `replicas: 1` in deployment.yaml or each pod will have independent data.

## Using MySQL (Production)

### 1. Deploy MySQL

```bash
# Apply MySQL init script
oc apply -f mysql/02-init-configmap.yaml

# Deploy MySQL server
oc apply -f mysql/03-deployment.yaml

# Create MySQL secrets
oc create secret generic mysql-secrets \
  --from-literal=root-password=$(openssl rand -base64 32) \
  --from-literal=user=brix_user \
  --from-literal=password=$(openssl rand -base64 32) \
  -n brix
```

### 2. Add connection string to secret

```bash
# Create connection string
export MYSQL_PASSWORD=$(oc get secret mysql-secrets -n brix -o jsonpath='{.data.password}' | base64 -d)
export CONNECTION_STRING="brix_user:${MYSQL_PASSWORD}@tcp(mysql-service.brix.svc.cluster.local:3306)/brix_pizza?parseTime=true"

# Update secret with connection string
oc create secret generic mysql-secrets \
  --from-literal=root-password=$(oc get secret mysql-secrets -n brix -o jsonpath='{.data.root-password}' | base64 -d) \
  --from-literal=user=brix_user \
  --from-literal=password=$MYSQL_PASSWORD \
  --from-literal=connection-string=$CONNECTION_STRING \
  --dry-run=client -o yaml | oc apply -f -
```

### 3. Deploy/restart the app

```bash
# If already deployed, restart to pick up MySQL
oc rollout restart deployment/brix-pizza -n brix

# Or deploy fresh
oc apply -f brix/04-deployment.yaml
```

### 4. Verify MySQL is being used

```bash
# Check logs - should see "Using MySQL database"
oc logs deployment/brix-pizza -n brix | grep -i mysql

# Should output:
# 📊 Using MySQL database
# ✅ Successfully connected to MySQL database
```

## Schema Migrations

Both databases use the same migration system:

1. **Migrations table** tracks which migrations have been applied
2. **Database-specific SQL** is used automatically:
   - SQLite: `INTEGER PRIMARY KEY AUTOINCREMENT`, `TEXT`, `REAL`
   - MySQL: `INT AUTO_INCREMENT PRIMARY KEY`, `VARCHAR`, `DECIMAL`
3. **Seed data** is loaded on first run (only once, even with multiple pods)

## Connection String Format

```
user:password@tcp(host:port)/database?parseTime=true
```

**Examples:**

```bash
# In-cluster MySQL
brix_user:mypass@tcp(mysql-service.brix.svc.cluster.local:3306)/brix_pizza?parseTime=true

# External MySQL (AWS RDS)
brix_user:mypass@tcp(mydb.abc123.us-east-1.rds.amazonaws.com:3306)/brix_pizza?parseTime=true

# Local MySQL (development)
brix_user:mypass@tcp(localhost:3306)/brix_pizza?parseTime=true
```

**Important:** Include `?parseTime=true` to properly handle TIMESTAMP columns.

## Switching Databases

### From SQLite to MySQL

1. Deploy MySQL (see above)
2. Add `DATABASE_URL` to mysql-secrets
3. Restart pods: `oc rollout restart deployment/brix-pizza -n brix`
4. Data will NOT be migrated - start fresh or migrate manually

### From MySQL to SQLite

1. Remove `DATABASE_URL` from environment (or set it to empty)
2. Restart pods
3. Each pod creates its own SQLite database

## Data Migration

To migrate data from SQLite to MySQL:

```bash
# 1. Export data from SQLite
sqlite3 ./db/orders.db .dump > backup.sql

# 2. Convert SQLite SQL to MySQL SQL (manual editing needed)
# - Change AUTOINCREMENT to AUTO_INCREMENT
# - Adjust data types (TEXT → VARCHAR, REAL → DECIMAL)

# 3. Import to MySQL
oc exec -i deployment/mysql -n brix -- \
  mysql -u root -p$MYSQL_ROOT_PASSWORD brix_pizza < backup-mysql.sql
```

**Note:** Migration requires manual SQL syntax conversion.

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_URL` | MySQL connection string | (empty - uses SQLite) |
| `PORT` | HTTP server port | 8080 |
| `BRIX_API_KEY` | Optional API key | (empty - session only) |

## Troubleshooting

### "Using SQLite database" but I deployed MySQL

**Problem:** App is not picking up `DATABASE_URL`

**Solution:**
```bash
# Verify secret exists
oc get secret mysql-secrets -n brix

# Check if connection-string is in the secret
oc get secret mysql-secrets -n brix -o jsonpath='{.data.connection-string}' | base64 -d

# Verify deployment has DATABASE_URL env
oc get deployment brix-pizza -n brix -o yaml | grep -A 5 DATABASE_URL
```

### "Failed to connect to MySQL"

**Problem:** MySQL is not running or connection string is wrong

**Solution:**
```bash
# Check MySQL pod
oc get pods -n brix -l app=mysql

# Test MySQL connection
oc exec -it deployment/mysql -n brix -- \
  mysql -u brix_user -p -e "SHOW DATABASES;"

# Verify connection string format
echo $CONNECTION_STRING
```

### "Lock query note (safe to ignore for SQLite)"

This is **normal** when using SQLite. SQLite doesn't support `FOR UPDATE`, but transactions are exclusive anyway.

### Tables not created in MySQL

**Problem:** Init script didn't run

**Solution:**
```bash
# Check if ConfigMap exists
oc get cm mysql-init -n brix

# Recreate MySQL pod to run init script
oc delete pod -l app=mysql -n brix
```

## Performance Tips

### SQLite
- Use `replicas: 1` to avoid data inconsistency
- Enable WAL mode for better concurrency (automatic)
- Good for < 100 concurrent users

### MySQL
- Scale to multiple replicas: `oc scale deployment brix-pizza --replicas=5`
- Add read replicas for high traffic
- Use connection pooling (Go does this automatically)
- Good for 1000+ concurrent users

## Backup Strategies

### SQLite Backup

```bash
# Copy database file
oc exec deployment/brix-pizza -n brix -- cat /app/db/orders.db > backup.db

# Or use SQLite backup command
oc exec deployment/brix-pizza -n brix -- \
  sqlite3 /app/db/orders.db ".backup /tmp/backup.db"
```

### MySQL Backup

```bash
# mysqldump
oc exec deployment/mysql -n brix -- \
  mysqldump -u root -p$MYSQL_ROOT_PASSWORD brix_pizza > backup.sql

# Restore
oc exec -i deployment/mysql -n brix -- \
  mysql -u root -p$MYSQL_ROOT_PASSWORD brix_pizza < backup.sql
```

## See Also

- [MYSQL.md](MYSQL.md) - Complete MySQL deployment guide
- [README.md](README.md) - General deployment guide
- [deployment.yaml](deployment.yaml) - Application deployment config
