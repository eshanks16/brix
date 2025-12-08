# Kubernetes Deployment Files - Summary

This directory contains all Kubernetes/OpenShift manifests for deploying Brix Pizza.

## 📂 Directory Structure

```
k8s/
├── brix/                  # Main application (numbered for deployment order)
├── mysql/                 # Optional MySQL database
├── README.md              # Full deployment guide
├── QUICKSTART.md          # Fast deployment guide
├── MYSQL.md               # MySQL setup guide
├── DATABASE_SUPPORT.md    # SQLite vs MySQL comparison
└── DEPLOYMENT_SUMMARY.md  # This file
```

## 🚀 Quick Commands

### Deploy with SQLite (development)
```bash
oc apply -f brix/01-namespace.yaml
oc apply -f brix/
```

### Deploy with MySQL (demo - uses demo credentials)
```bash
oc apply -f brix/01-namespace.yaml
oc apply -f mysql/
oc apply -f brix/
```

**⚠️ Demo credentials included!** For production, see [QUICKSTART.md](QUICKSTART.md) to generate secure credentials.

## 📋 Deployment Order

Files are numbered to indicate deployment order:

### Brix Pizza (brix/)
1. `01-namespace.yaml` - Create namespace
2. `02-configmap.yaml` - App configuration  
3. `03-secret.yaml.template` - API key (create via kubectl/oc)
4. `04-deployment.yaml` - Application pods
5. `05-service.yaml` - Internal service
6. `06-route.yaml` - External access (OpenShift)
7. `07-service-loadbalancer.yaml` - Alternative for K8s
8. `08-hpa.yaml` - Auto-scaling (optional)

### MySQL Database (mysql/)
1. `01-secrets.yaml.template` - MySQL credentials (create via kubectl/oc)
2. `02-init-configmap.yaml` - Database initialization
3. `03-deployment.yaml` - MySQL StatefulSet with persistent storage (px-csi-db)

## 🎯 Which Guide to Use?

| Your Situation | Read This |
|----------------|-----------|
| "Just deploy it fast!" | [QUICKSTART.md](QUICKSTART.md) |
| "I need MySQL" | [MYSQL.md](MYSQL.md) |
| "SQLite or MySQL?" | [DATABASE_SUPPORT.md](DATABASE_SUPPORT.md) |
| "Full documentation" | [README.md](README.md) |

## 🔧 Key Features

- **Dual database support**: SQLite (default) or MySQL (production)
- **OpenShift ready**: Includes Routes with TLS
- **Kubernetes compatible**: LoadBalancer service alternative
- **Session-based auth**: Client IP session affinity configured
- **Health checks**: Liveness and readiness probes
- **Auto-scaling**: HPA based on CPU (optional)
- **Security**: Non-root user, restricted SCC compatible

## 📊 Database Comparison

| Feature | SQLite | MySQL |
|---------|--------|-------|
| Setup | Automatic | Manual deployment |
| Multi-pod | ❌ (replicas: 1) | ✅ Full scalability |
| Best for | Development, < 100 users | Production, 1000+ users |
| Configuration | None needed | Set DATABASE_URL |

## 🔐 Security Notes

- **API Key**: Optional, for additional API security
- **Sessions**: 3-hour timeout with client IP affinity
- **TLS**: Automatic with OpenShift Routes
- **User**: Runs as non-root (UID 1000)

## 📦 What Gets Deployed

### With SQLite (default)
- 3 pods (each with own SQLite database)
- ClusterIP service with session affinity
- OpenShift Route with TLS

### With MySQL
- 3+ Brix Pizza pods (shared MySQL database)
- 1 MySQL StatefulSet with persistent storage (px-csi-db, 10Gi)
- ClusterIP services for both (mysql-service for client connections)
- OpenShift Route with TLS

## 🛠️ Common Operations

### View logs
```bash
oc logs -f deployment/brix-pizza -n brix
```

### Scale application
```bash
# MySQL only!
oc scale deployment brix-pizza --replicas=5 -n brix
```

### Update image
```bash
oc rollout restart deployment/brix-pizza -n brix
```

### Get URL
```bash
oc get route brix-pizza -n brix -o jsonpath='{.spec.host}'
```

## 📝 Important Notes

1. **SQLite requires `replicas: 1`** - Each pod has independent data
2. **MySQL allows multiple replicas** - All pods share the same database
3. **MySQL uses StatefulSet** - Ensures stable pod identity and persistent storage (px-csi-db)
4. **Session affinity is required** - Users must stick to same pod for sessions
5. **Demo credentials included** - Secret files contain demo passwords for quick deployment. **Change for production!**
6. **Files are numbered** - Deploy entire directories safely with `oc apply -f brix/` or `oc apply -f mysql/`

## 🐛 Troubleshooting

### Pods not starting
```bash
oc describe pod -l app=brix-pizza -n brix
```

### Check database mode
```bash
oc logs deployment/brix-pizza -n brix | grep -i "using"
# Output: "Using SQLite" or "Using MySQL"
```

### MySQL connection issues
```bash
# Connect to MySQL StatefulSet pod
oc exec -it mysql-0 -n brix -- mysql -u brix_user -p

# Check StatefulSet status
oc get statefulset mysql -n brix

# Check persistent volume
oc get pvc -n brix
```

## 📚 Additional Resources

- Main README: [README.md](README.md)
- Quick Start: [QUICKSTART.md](QUICKSTART.md)
- MySQL Guide: [MYSQL.md](MYSQL.md)
- Database Info: [DATABASE_SUPPORT.md](DATABASE_SUPPORT.md)
- Main Dockerfile: [../../Dockerfile](../../Dockerfile)
- API Documentation: [../../docs/API_EXAMPLES.md](../../docs/API_EXAMPLES.md)
