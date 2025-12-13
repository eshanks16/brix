# Brix Pizza - Kubernetes/OpenShift Deployment Guide

This guide walks you through deploying Brix Pizza to OpenShift or standard Kubernetes clusters.

## Prerequisites

- **OpenShift** cluster with `oc` CLI configured, OR
- **Kubernetes** cluster (1.19+) with `kubectl` configured
- Docker image available at `eshanks16/brix-pizza:latest` on Docker Hub

## Directory Structure

```
k8s/
├── brix/                    # Brix Pizza application manifests
│   ├── 01-namespace.yaml    # Creates the brix namespace
│   ├── 02-configmap.yaml    # Configuration (PORT setting)
│   ├── 03-secret.yaml.template       # API key secret template
│   ├── 04-deployment.yaml   # Main application deployment
│   ├── 05-service.yaml      # ClusterIP service
│   ├── 06-route.yaml        # OpenShift Route (TLS)
│   ├── 07-service-loadbalancer.yaml  # LoadBalancer alternative
│   └── 08-hpa.yaml          # Horizontal Pod Autoscaler
├── mysql/                   # MySQL database manifests (optional)
│   ├── 01-secrets.yaml.template      # MySQL credentials
│   ├── 02-init-configmap.yaml  # Database initialization
│   └── 03-deployment.yaml   # MySQL deployment
├── README.md                # This file
├── MYSQL.md                 # MySQL deployment guide
└── DATABASE_SUPPORT.md      # Database comparison guide
```

Files are numbered in deployment order for easy sequential application.

## Quick Start - OpenShift

### 1. Create the namespace
```bash
oc apply -f brix/01-namespace.yaml
```

### 2. Create the secret

```bash
# Generate a random API key (optional - can be skipped for session-only auth)
export API_KEY=$(openssl rand -hex 32)

# Create the secret
oc create secret generic brix-pizza-secrets \
  --from-literal=api-key=$API_KEY \
  -n brix

# Save the API key for later use
echo "Your API Key: $API_KEY"
```

**Note:** The API key is optional. If not set, the API uses session-only authentication.

### 3. Deploy the application

```bash
# Apply all at once (templates are automatically skipped)
oc apply -f brix/
```

### 4. Get the route URL

```bash
oc get route brix-pizza -n brix -o jsonpath='{.spec.host}'
```

### 5. Test the application

```bash
# Get the route hostname
export BRIX_URL=$(oc get route brix-pizza -n brix -o jsonpath='{.spec.host}')

# Test the web interface
curl https://$BRIX_URL

# Test the health endpoint
curl https://$BRIX_URL/health/live

# Test the API (with session)
# First register/login via web UI, then use the session cookie
```

## Quick Start - Standard Kubernetes

### 1. Create the namespace

```bash
kubectl apply -f brix/01-namespace.yaml
```

### 2. Create the secret

```bash
# Generate a random API key (optional)
export API_KEY=$(openssl rand -hex 32)

# Create the secret
kubectl create secret generic brix-pizza-secrets \
  --from-literal=api-key=$API_KEY \
  -n brix

# Save the API key
echo "Your API Key: $API_KEY"
```

### 3. Deploy the application

```bash
# Apply ConfigMap
kubectl apply -f brix/02-configmap.yaml

# Deploy the application
kubectl apply -f brix/04-deployment.yaml

# For cloud providers with LoadBalancer support:
kubectl apply -f brix/07-service-loadbalancer.yaml

# For clusters with Ingress:
kubectl apply -f brix/05-service.yaml
# Then configure your Ingress controller separately
```

### 4. Get the LoadBalancer IP

```bash
kubectl get svc brix-pizza-lb -n brix
```

## Database

The application uses **SQLite** with a local database file. Each pod maintains its own SQLite database.

### Important Note for Production

For production deployments with multiple replicas, consider:

1. **Single Replica Mode**: Set `replicas: 1` in [deployment.yaml](deployment.yaml:9) for SQLite
2. **Shared Storage**: Use a PersistentVolumeClaim for shared SQLite access (not recommended)
3. **External Database**: Migrate to PostgreSQL or MySQL for true multi-pod deployments

Current deployment runs 3 replicas - each pod will have independent data!

## Application Features

### Session-Based Authentication

The application uses HTTP sessions with cookies. Session affinity is configured:

- **sessionAffinity: ClientIP**
- **timeoutSeconds: 10800** (3 hours)

This ensures users stick to the same pod for the duration of their session.

### Health Checks

- **Liveness**: `/health/live` - Checks if the app is running
- **Readiness**: `/health/ready` - Checks if the app can serve traffic (includes DB check)

### Resource Limits

Default per pod:

- **Requests**: 100m CPU, 128Mi memory
- **Limits**: 500m CPU, 512Mi memory

## Horizontal Pod Autoscaler (Optional)

```bash
# Enable autoscaling based on CPU usage
kubectl apply -f brix/08-hpa.yaml
# or
oc apply -f brix/08-hpa.yaml
```

Scales between 2-10 pods based on CPU utilization (target: 70%).

## Monitoring

### Check pod status

```bash
# OpenShift
oc get pods -n brix

# Kubernetes
kubectl get pods -n brix
```

### View logs

```bash
# OpenShift
oc logs -f deployment/brix-pizza -n brix

# Kubernetes
kubectl logs -f deployment/brix-pizza -n brix
```

### Check pod health

```bash
# OpenShift
oc describe pod -l app=brix-pizza -n brix

# Kubernetes
kubectl describe pod -l app=brix-pizza -n brix
```

## Security

### API Key

The `BRIX_API_KEY` is optional:

- **Not set**: API endpoints use session-only authentication
- **Set**: API endpoints require both session cookie AND Bearer token

### OpenShift Security Context Constraints (SCC)

The deployment uses:

- `runAsNonRoot: true`
- `runAsUser: 1000` (matches the Docker image user)
- `allowPrivilegeEscalation: false`
- `readOnlyRootFilesystem: false` (SQLite needs write access)

Works with the default `restricted` SCC in OpenShift.

### TLS

The OpenShift Route is configured with:

- **Edge termination**: TLS terminates at the OpenShift router
- **HTTP redirect**: All HTTP traffic redirects to HTTPS

## Scaling

### Manual scaling

```bash
# OpenShift
oc scale deployment brix-pizza --replicas=5 -n brix

# Kubernetes
kubectl scale deployment brix-pizza --replicas=5 -n brix
```

**Warning**: With SQLite, each pod has independent data. Use `replicas: 1` or migrate to external DB.

### Auto-scaling (HPA)

```bash
# Apply the HPA
oc apply -f brix/08-hpa.yaml

# View HPA status
oc get hpa -n brix
```

## Troubleshooting

### Pods not starting

```bash
# Check pod events
oc describe pod -l app=brix-pizza -n brix

# Check if image can be pulled
oc get events -n brix | grep -i pull
```

### Readiness probe failing

```bash
# Check the readiness endpoint directly
oc port-forward deployment/brix-pizza 8080:8080 -n brix
curl http://localhost:8080/health/ready
```

### Route/Service not accessible

```bash
# Verify route exists (OpenShift)
oc get route brix-pizza -n brix

# Verify service endpoints
oc get endpoints brix-pizza -n brix

# Test service from within cluster
oc run test-curl --image=curlimages/curl -i --rm --restart=Never -- \
  curl http://brix-pizza.brix.svc.cluster.local/health/live
```

## Image Updates

### Deploy a new version

```bash
# The image is automatically pulled from Docker Hub
# Restart deployment to pick up new image
oc rollout restart deployment/brix-pizza -n brix

# Watch the rollout
oc rollout status deployment/brix-pizza -n brix
```

### Rollback a deployment

```bash
# View rollout history
oc rollout history deployment/brix-pizza -n brix

# Rollback to previous version
oc rollout undo deployment/brix-pizza -n brix
```

## Clean Up

### Remove all resources

```bash
# OpenShift
oc delete namespace brix

# Kubernetes
kubectl delete namespace brix
```

### Remove individual components

```bash
# OpenShift
oc delete -f brix/06-route.yaml
oc delete -f brix/05-service.yaml
oc delete -f brix/04-deployment.yaml
oc delete -f brix/02-configmap.yaml
oc delete secret brix-pizza-secrets -n brix

# Kubernetes (use kubectl instead of oc)
```

## Production Checklist

Before deploying to production:

- [ ] Decide on database strategy (SQLite with 1 replica vs external DB)
- [ ] Configure persistent storage if using SQLite
- [ ] Secrets contain real values (not placeholders)
- [ ] Resource limits are tuned for your workload
- [ ] Route/Ingress is configured with proper TLS certificate
- [ ] Monitoring and alerting are set up
- [ ] Logging is configured
- [ ] Pod Disruption Budget is appropriate for your SLA
- [ ] Backup strategy for SQLite database (if applicable)
- [ ] API key is stored securely and rotated regularly

## Verify Deployment

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
| `LOG_LEVEL` | ConfigMap | Logging level: `debug`, `info`, `warn`, `error`, `fatal` (default: info) |
| `BRIX_API_KEY` | Secret | API authentication key |
| `DATABASE_URL` | Secret | MySQL connection string |

**To change the log level:**

Edit the ConfigMap and restart the pods:
```bash
kubectl edit configmap brix-pizza-config -n brix
# Change LOG_LEVEL to desired level (debug, info, warn, error, fatal)

# Restart pods to pick up new config
kubectl rollout restart deployment brix-pizza -n brix
```

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
  mysql -h mysql-service.brix.svc.cluster.local -u brix_user -p
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
