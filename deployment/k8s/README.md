# Brix Pizza - Kubernetes Deployment Guide

This guide walks you through deploying Brix Pizza to a Kubernetes cluster.

## Prerequisites

- Kubernetes cluster (1.19+) with kubectl configured
- Docker for building container images
- Container registry (Docker Hub, GCR, ECR, or similar)
- MySQL database (external or in-cluster)

## Quick Start

### 1. Build and Push Docker Image

```bash
# Navigate to deployment directory
cd deployment

# Build the image
docker build -t brix-pizza:latest -f Dockerfile ..

# Tag for your registry (replace with your registry)
docker tag brix-pizza:latest YOUR_REGISTRY/brix-pizza:latest

# Push to registry
docker push YOUR_REGISTRY/brix-pizza:latest
```

**Examples for different registries:**

```bash
# Docker Hub
docker tag brix-pizza:latest username/brix-pizza:latest
docker push username/brix-pizza:latest

# Google Container Registry
docker tag brix-pizza:latest gcr.io/project-id/brix-pizza:latest
docker push gcr.io/project-id/brix-pizza:latest

# AWS ECR
docker tag brix-pizza:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/brix-pizza:latest
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/brix-pizza:latest
```

### 2. Set Up MySQL Database

You need a MySQL database accessible from your Kubernetes cluster. Options:

**Option A: External MySQL** (AWS RDS, Google Cloud SQL, etc.)
- Create database: `brix_pizza`
- Create user with appropriate permissions
- Note the connection details (host, port, username, password)

**Option B: In-Cluster MySQL** (for development/testing)

```bash
# Deploy MySQL in Kubernetes
kubectl create namespace brix
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: mysql
  namespace: brix
spec:
  ports:
  - port: 3306
  selector:
    app: mysql
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql
  namespace: brix
spec:
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - name: mysql
        image: mysql:8.0
        env:
        - name: MYSQL_ROOT_PASSWORD
          value: rootpassword
        - name: MYSQL_DATABASE
          value: brix_pizza
        - name: MYSQL_USER
          value: brix_user
        - name: MYSQL_PASSWORD
          value: brixpassword
        ports:
        - containerPort: 3306
        volumeMounts:
        - name: mysql-storage
          mountPath: /var/lib/mysql
      volumes:
      - name: mysql-storage
        emptyDir: {}
EOF
```

### 3. Create Kubernetes Secrets

Generate a secure API key:

```bash
export BRIX_API_KEY=$(openssl rand -hex 32)
echo "Your API key: $BRIX_API_KEY"
```

Create the secret:

```bash
# For external MySQL
kubectl create secret generic brix-pizza-secrets \
  --from-literal=api-key=$BRIX_API_KEY \
  --from-literal=database-url="brix_user:password@tcp(mysql-host.example.com:3306)/brix_pizza" \
  -n brix

# For in-cluster MySQL
kubectl create secret generic brix-pizza-secrets \
  --from-literal=api-key=$BRIX_API_KEY \
  --from-literal=database-url="brix_user:brixpassword@tcp(mysql.brix.svc.cluster.local:3306)/brix_pizza" \
  -n brix
```

**Important:** Save your API key securely! You'll need it to access the API endpoints.

### 4. Update Image Reference

Edit `deployment.yaml` and replace the image reference:

```yaml
# Change this line:
image: brix-pizza:latest

# To your registry path:
image: YOUR_REGISTRY/brix-pizza:latest
```

### 5. Deploy to Kubernetes

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Create ConfigMap
kubectl apply -f k8s/configmap.yaml

# Deploy application
kubectl apply -f k8s/deployment.yaml

# Create Service
kubectl apply -f k8s/service.yaml

# Optional: Enable autoscaling
kubectl apply -f k8s/hpa.yaml
```

### 6. Verify Deployment

Check pod status:

```bash
kubectl get pods -n brix
```

Expected output:
```
NAME                          READY   STATUS    RESTARTS   AGE
brix-pizza-xxxxxxxxxx-xxxxx   1/1     Running   0          30s
brix-pizza-xxxxxxxxxx-xxxxx   1/1     Running   0          30s
brix-pizza-xxxxxxxxxx-xxxxx   1/1     Running   0          30s
```

Check logs:

```bash
kubectl logs -f deployment/brix-pizza -n brix
```

You should see:
```
🍕 Brix Pizza is running on http://0.0.0.0:8080
📡 API available at http://0.0.0.0:8080/api/*
💚 Health checks: /health/live (liveness) and /health/ready (readiness)
```

### 7. Access the Application

Get the external IP:

```bash
kubectl get svc brix-pizza -n brix
```

Wait for EXTERNAL-IP to be assigned:
```
NAME         TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)        AGE
brix-pizza   LoadBalancer   10.96.123.45    34.123.45.67     80:30123/TCP   2m
```

Access the application:
```bash
# Web UI
http://EXTERNAL-IP/

# Health checks
curl http://EXTERNAL-IP/health/live
curl http://EXTERNAL-IP/health/ready

# API (using your API key)
curl -H "Authorization: Bearer $BRIX_API_KEY" http://EXTERNAL-IP/api/menu
```

## Configuration

### Environment Variables

All configuration is managed through ConfigMap and Secrets:

| Variable | Source | Description |
|----------|--------|-------------|
| `PORT` | ConfigMap | HTTP server port (default: 8080) |
| `BRIX_API_KEY` | Secret | API authentication key |
| `DATABASE_URL` | Secret | MySQL connection string |

### Scaling

**Manual scaling:**
```bash
kubectl scale deployment brix-pizza --replicas=5 -n brix
```

**Autoscaling** (if HPA is enabled):
- Minimum replicas: 2
- Maximum replicas: 10
- Target CPU: 70%

### Resource Limits

Default resource allocation per pod:
- **Requests:** CPU 100m, Memory 128Mi
- **Limits:** CPU 500m, Memory 512Mi

Adjust in `deployment.yaml` based on your workload.

## Troubleshooting

### Pods not starting

```bash
# Check pod status
kubectl get pods -n brix

# Describe pod for events
kubectl describe pod POD_NAME -n brix

# Check logs
kubectl logs POD_NAME -n brix
```

Common issues:
- **ImagePullBackOff**: Image not found in registry or auth required
- **CrashLoopBackOff**: Application error, check logs
- **Pending**: Resource constraints or node selector issues

### Database connection errors

Check logs for database connectivity:

```bash
kubectl logs -f deployment/brix-pizza -n brix | grep -i database
```

Verify secret:
```bash
kubectl get secret brix-pizza-secrets -n brix -o yaml
```

Test database connectivity from a pod:
```bash
kubectl run -it --rm debug --image=mysql:8.0 --restart=Never -n brix -- \
  mysql -h mysql.brix.svc.cluster.local -u brix_user -p
```

### Readiness probe failing

```bash
# Check readiness endpoint
kubectl port-forward deployment/brix-pizza 8080:8080 -n brix

# In another terminal
curl http://localhost:8080/health/ready
```

If it returns "Database not ready", check MySQL connectivity.

### LoadBalancer pending

Some clusters (like local Minikube) don't support LoadBalancer services.

**Option 1: Use NodePort**

Edit `service.yaml`:
```yaml
spec:
  type: NodePort  # Change from LoadBalancer
```

**Option 2: Use port-forward for testing**

```bash
kubectl port-forward svc/brix-pizza 8080:80 -n brix
# Access at http://localhost:8080
```

**Option 3: Use Ingress**

Create an Ingress resource for production environments.

## Updating the Application

```bash
# Build new image with tag
docker build -t YOUR_REGISTRY/brix-pizza:v2 -f deployment/Dockerfile .
docker push YOUR_REGISTRY/brix-pizza:v2

# Update deployment
kubectl set image deployment/brix-pizza brix-pizza=YOUR_REGISTRY/brix-pizza:v2 -n brix

# Monitor rollout
kubectl rollout status deployment/brix-pizza -n brix

# Rollback if needed
kubectl rollout undo deployment/brix-pizza -n brix
```

## Cleanup

Remove all resources:

```bash
kubectl delete -f k8s/hpa.yaml
kubectl delete -f k8s/service.yaml
kubectl delete -f k8s/deployment.yaml
kubectl delete -f k8s/configmap.yaml
kubectl delete secret brix-pizza-secrets -n brix
kubectl delete -f k8s/namespace.yaml
```

## Production Checklist

Before deploying to production:

- [ ] MySQL database is provisioned with backups enabled
- [ ] Secrets contain real values (not placeholders)
- [ ] Image is pushed to a secure private registry
- [ ] Resource limits are tuned for your workload
- [ ] Ingress or LoadBalancer is configured with TLS/SSL
- [ ] Monitoring and alerting are set up (Prometheus, Grafana)
- [ ] Logging is configured (ELK, Loki, CloudWatch)
- [ ] Pod Disruption Budget is appropriate for your SLA
- [ ] Persistent storage for MySQL (if in-cluster)
- [ ] Network policies for security (if required)
- [ ] Backup and disaster recovery plan
- [ ] API key is stored securely and rotated regularly

## Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [MySQL on Kubernetes](https://kubernetes.io/docs/tasks/run-application/run-single-instance-stateful-application/)
- [Horizontal Pod Autoscaling](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [Ingress Controllers](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/)
