# Quick Start Guide

Deploy Brix Pizza in minutes!

## Option 1: SQLite (Single Pod) - Simplest

Perfect for development or low-traffic deployments.

```bash
# 1. Create namespace
oc apply -f brix/01-namespace.yaml

# 2. Create secret (optional for API key)
oc create secret generic brix-pizza-secrets \
  --from-literal=api-key=$(openssl rand -hex 32) \
  -n brix

# 3. Deploy everything
oc apply -f brix/

# 4. Get your URL
echo "https://$(oc get route brix-pizza -n brix -o jsonpath='{.spec.host}')"
```

**Done!** Visit the URL and start ordering pizza.

**Important:** Set `replicas: 1` in `brix/04-deployment.yaml` for SQLite.

## Option 2: MySQL (Multi-Pod) - Production

For production deployments with multiple replicas.

### Quick Demo (uses demo credentials)

```bash
# Deploy everything at once (demo credentials included)
oc apply -f brix/01-namespace.yaml
oc apply -f mysql/
oc apply -f brix/

# Get your URL
echo "https://$(oc get route brix-pizza -n brix -o jsonpath='{.spec.host}')"
```

**⚠️ Demo credentials:** The secret files contain demo passwords. **Change them for production!**

### Production Deployment (secure credentials)

```bash
# 1. Create namespace
oc apply -f brix/01-namespace.yaml

# 2. Generate secure MySQL secrets
export MYSQL_ROOT_PWD=$(openssl rand -base64 32)
export MYSQL_USER_PWD=$(openssl rand -base64 32)
export CONN_STRING="brix_user:${MYSQL_USER_PWD}@tcp(mysql-service.brix.svc.cluster.local:3306)/brix_pizza?parseTime=true"

oc create secret generic mysql-secrets \
  --from-literal=root-password=$MYSQL_ROOT_PWD \
  --from-literal=user=brix_user \
  --from-literal=password=$MYSQL_USER_PWD \
  --from-literal=connection-string=$CONN_STRING \
  -n brix

# 3. Generate secure Brix API key
oc create secret generic brix-pizza-secrets \
  --from-literal=api-key=$(openssl rand -hex 32) \
  -n brix

# 4. Deploy MySQL (skip secret file since we created it above)
oc apply -f mysql/02-init-configmap.yaml
oc apply -f mysql/03-deployment.yaml

# 5. Deploy Brix Pizza (skip secret file)
oc apply -f brix/01-namespace.yaml
oc apply -f brix/02-configmap.yaml
oc apply -f brix/04-deployment.yaml
oc apply -f brix/05-service.yaml
oc apply -f brix/06-route.yaml
oc apply -f brix/08-hpa.yaml

# 6. Get your URL
echo "https://$(oc get route brix-pizza -n brix -o jsonpath='{.spec.host}')"
```

**Done!** MySQL handles shared data across all pods.

## Verify Deployment

```bash
# Check pods
oc get pods -n brix

# Check logs
oc logs -f deployment/brix-pizza -n brix

# Should see one of:
# ⚠️ WARNING: Using SQLite database (SQLite mode)
# 📊 Using MySQL database (MySQL mode)
```

## Access the Application

```bash
# Get URL
export BRIX_URL=$(oc get route brix-pizza -n brix -o jsonpath='{.spec.host}')

# Open in browser
echo "https://$BRIX_URL"

# Test API
curl -s https://$BRIX_URL/health/live | jq .
```

## Scale Up (MySQL only)

```bash
# Scale to 5 replicas (only with MySQL!)
oc scale deployment brix-pizza --replicas=5 -n brix

# Enable autoscaling
oc apply -f brix/08-hpa.yaml
```

## Clean Up

```bash
# Remove everything
oc delete namespace brix
```

## Next Steps

- **SQLite → MySQL migration**: See [DATABASE_SUPPORT.md](DATABASE_SUPPORT.md)
- **MySQL setup details**: See [MYSQL.md](MYSQL.md)
- **Full documentation**: See [README.md](README.md)

## File Organization

All files are numbered for sequential deployment:

```
brix/
  01-namespace.yaml       # Always first
  02-configmap.yaml       # Configuration
  03-secret.yaml.template          # Secret template
  04-deployment.yaml      # Main app
  05-service.yaml         # Internal service
  06-route.yaml           # External access
  07-service-loadbalancer.yaml  # Cloud alternative
  08-hpa.yaml             # Autoscaling

mysql/
  01-secrets.yaml.template         # MySQL credentials
  02-init-configmap.yaml  # Database init
  03-deployment.yaml      # MySQL server
```

Apply entire directories at once: `oc apply -f brix/`
