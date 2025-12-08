# Using MySQL with Brix Pizza

This guide explains how to deploy Brix Pizza with MySQL instead of SQLite.

## Important Note

**The Brix Pizza application supports both SQLite and MySQL!**

- **SQLite** (default): Used when `DATABASE_URL` is not set - great for development and single-pod deployments
- **MySQL** (recommended for production): Used when `DATABASE_URL` is set - supports multiple pods with shared data

This guide shows you how to deploy MySQL and configure the app to use it.

## Files for MySQL Deployment

- **mysql-deployment.yaml** - MySQL server deployment with persistent storage
- **mysql-secrets.yaml** - Template for MySQL credentials
- **mysql-init-configmap.yaml** - Database initialization script

## Quick Start - Deploy MySQL

### 1. Create MySQL secrets

```bash
# Generate strong passwords
export MYSQL_ROOT_PASSWORD=$(openssl rand -base64 32)
export MYSQL_USER_PASSWORD=$(openssl rand -base64 32)

# Create the secret
oc create secret generic mysql-secrets \
  --from-literal=root-password=$MYSQL_ROOT_PASSWORD \
  --from-literal=user=brix_user \
  --from-literal=password=$MYSQL_USER_PASSWORD \
  -n brix

# Save credentials securely!
echo "MySQL Root Password: $MYSQL_ROOT_PASSWORD"
echo "MySQL User: brix_user"
echo "MySQL User Password: $MYSQL_USER_PASSWORD"
```

### 2. Deploy MySQL

```bash
# Apply the init script ConfigMap
oc apply -f mysql/02-init-configmap.yaml

# Deploy MySQL
oc apply -f mysql/03-deployment.yaml
```

### 3. Verify MySQL is running

```bash
# Check pod status
oc get pods -n brix -l app=mysql

# Check logs
oc logs -f deployment/mysql -n brix

# Verify database was created
oc exec -it deployment/mysql -n brix -- \
  mysql -u brix_user -p$MYSQL_USER_PASSWORD -e "SHOW DATABASES;"
```

You should see `brix_pizza` in the database list.

### 4. Test the connection

```bash
# Connect to MySQL from within the cluster
oc run mysql-client --image=mysql:8.0 -i --rm --restart=Never -n brix -- \
  mysql -h mysql-service.brix.svc.cluster.local -u brix_user -p$MYSQL_USER_PASSWORD -e "USE brix_pizza; SHOW TABLES;"
```

Expected output:
```
+----------------------+
| Tables_in_brix_pizza |
+----------------------+
| orders               |
| pizza_sizes          |
| pizza_styles         |
| toppings             |
| users                |
+----------------------+
```

## How It Works

The application automatically detects which database to use:

1. **Check for `DATABASE_URL` environment variable**
   - If set → use MySQL with the connection string
   - If not set → use SQLite (local file)

2. **Schema migrations**
   - Automatically creates tables on first startup
   - Uses database-specific SQL syntax (SQLite vs MySQL)
   - Seeds initial menu data (pizza styles, sizes, toppings)

3. **No code changes needed!**
   - The application already includes both database drivers
   - Schema migrations handle syntax differences automatically

## Deploy MySQL and Configure the App

### 1. Update the MySQL secret with connection string

After creating MySQL secrets (see above), add the connection string:

```bash
# Connection string format: user:password@tcp(host:3306)/database?parseTime=true
export CONNECTION_STRING="brix_user:${MYSQL_USER_PASSWORD}@tcp(mysql-service.brix.svc.cluster.local:3306)/brix_pizza?parseTime=true"

# Update the secret to include connection string
oc create secret generic mysql-secrets \
  --from-literal=root-password=$MYSQL_ROOT_PASSWORD \
  --from-literal=user=brix_user \
  --from-literal=password=$MYSQL_USER_PASSWORD \
  --from-literal=connection-string=$CONNECTION_STRING \
  --dry-run=client -o yaml | oc apply -f -
```

**Note:** The `deployment.yaml` already includes the `DATABASE_URL` environment variable (optional), so no changes needed!

### 2. Restart the deployment to pick up MySQL

```bash
# Restart to pick up new image and MySQL connection
oc rollout restart deployment/brix-pizza -n brix

# Watch rollout
oc rollout status deployment/brix-pizza -n brix
```

### 3. Verify the application

```bash
# Check logs - should see successful database connection
oc logs -f deployment/brix-pizza -n brix

# Test health check (includes database check)
curl https://$(oc get route brix-pizza -n brix -o jsonpath='{.spec.host}')/health/ready
```

## Benefits of MySQL over SQLite

1. **Multi-pod support**: All pods share the same database
2. **Better concurrency**: MySQL handles concurrent writes better
3. **Scalability**: Can scale horizontally with MySQL replication
4. **Backup/Recovery**: Standard MySQL backup tools
5. **Production ready**: Battle-tested for production workloads

## Persistent Storage

The MySQL deployment uses a PersistentVolumeClaim for data storage:

```yaml
resources:
  requests:
    storage: 10Gi
```

**OpenShift**: Will automatically provision storage using the default StorageClass.

**Kubernetes**: May require manual PV creation or StorageClass configuration depending on your cluster.

### Check storage

```bash
# View PVC status
oc get pvc mysql-data -n brix

# View PV details
oc get pv
```

## Backup MySQL Database

### Manual backup

```bash
# Backup to local file
oc exec deployment/mysql -n brix -- \
  mysqldump -u root -p$MYSQL_ROOT_PASSWORD brix_pizza > backup.sql

# Restore from backup
oc exec -i deployment/mysql -n brix -- \
  mysql -u root -p$MYSQL_ROOT_PASSWORD brix_pizza < backup.sql
```

### Automated backups

Consider setting up:
- **CronJob** for scheduled backups
- **Velero** for cluster-level backups
- **Cloud provider backups** (AWS RDS Snapshots, Azure Backup, etc.)

## Monitoring MySQL

### Check MySQL status

```bash
# Connect to MySQL shell
oc exec -it deployment/mysql -n brix -- \
  mysql -u root -p$MYSQL_ROOT_PASSWORD

# Inside MySQL shell:
SHOW DATABASES;
USE brix_pizza;
SHOW TABLES;
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM orders;
```

### Check resource usage

```bash
# View pod resources
oc top pod -l app=mysql -n brix

# View logs
oc logs deployment/mysql -n brix --tail=50
```

## Troubleshooting

### MySQL pod not starting

```bash
# Check pod events
oc describe pod -l app=mysql -n brix

# Check PVC status
oc get pvc mysql-data -n brix
```

### Connection refused errors

```bash
# Verify MySQL service
oc get svc mysql -n brix

# Test connection from another pod
oc run mysql-test --image=mysql:8.0 -i --rm --restart=Never -n brix -- \
  mysql -h mysql-service.brix.svc.cluster.local -u brix_user -p$MYSQL_USER_PASSWORD -e "SELECT 1;"
```

### Application can't connect to MySQL

1. Check DATABASE_URL secret is set correctly
2. Verify MySQL pod is running and ready
3. Check application logs: `oc logs deployment/brix-pizza -n brix`
4. Verify network connectivity between pods

## Clean Up MySQL

### Remove MySQL deployment

```bash
# Delete MySQL resources
oc delete -f mysql/03-deployment.yaml
oc delete -f mysql/02-init-configmap.yaml

# Delete PVC (WARNING: This deletes all data!)
oc delete pvc mysql-data -n brix

# Delete secrets
oc delete secret mysql-secrets -n brix
```

## Production Considerations

For production MySQL deployments:

1. **High Availability**: Use MySQL replication or Galera cluster
2. **Resource Limits**: Tune CPU/memory based on workload
3. **Connection Pooling**: Configure max connections in MySQL
4. **Monitoring**: Set up Prometheus + Grafana for MySQL
5. **Backups**: Automated daily backups with retention policy
6. **Security**: Use TLS for MySQL connections
7. **Performance**: Optimize indexes and query performance

## Alternative: Managed MySQL Services

Instead of self-hosting, consider managed services:

- **AWS RDS for MySQL**
- **Azure Database for MySQL**
- **Google Cloud SQL for MySQL**
- **OpenShift Database as a Service**

Benefits:
- Automated backups
- High availability built-in
- Automatic updates
- Better performance
- No operational overhead
