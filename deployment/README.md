# Brix Pizza - Kubernetes Deployment

Deploy Brix Pizza to Kubernetes in minutes.

## Quick Start

```bash
# Deploy everything
kubectl apply -f deployment/k8s/brix/
kubectl apply -f deployment/k8s/mysql/

# Check status
kubectl get pods -n brix

# Access the app (locally)
kubectl port-forward -n brix svc/brix-pizza 8080:80
# Open http://localhost:8080
```

## What Gets Deployed

- **MySQL Database** - StatefulSet with persistent storage (10Gi)
- **Brix Pizza App** - Web application with auto-scaling (1-5 replicas)
- **Services** - Internal networking
- **Monitoring** - Prometheus metrics endpoint

All resources deploy to the `brix` namespace.

## ⚠️  Before Production

**The default deployment uses demo credentials for quick testing.**

Before production use:

1. **Change passwords** in `deployment/k8s/mysql/02-secrets.yaml`
2. **Change API key** in `deployment/k8s/brix/03-secret.yaml`
3. **Pin image version** in `deployment/k8s/brix/04-deployment.yaml` (don't use `latest`)

Generate secure passwords:
```bash
# Generate password
openssl rand -base64 32

# Base64 encode for Kubernetes
echo -n "your-password" | base64
```

## Configuration

### Application Settings

Edit `deployment/k8s/brix/02-configmap.yaml`:
- `LOG_LEVEL` - debug, info, warn, error (default: info)
- `PORT` - HTTP port (default: 8080)

### Resource Limits

Edit `deployment/k8s/brix/04-deployment.yaml` to adjust:
- CPU/memory requests and limits
- Replica count

### Storage

Edit `deployment/k8s/mysql/04-mysql-sts.yaml` to change:
- Storage size (default: 10Gi)
- Storage class (default: px-csi-db)

## Accessing the Application

### Local Access

```bash
kubectl port-forward -n brix svc/brix-pizza 8080:80
```

### Production Access

Choose based on your environment:

**OpenShift** (Route):
```bash
kubectl apply -f deployment/k8s/brix/06-route.yaml
```

**Cloud LoadBalancer**:
```bash
kubectl apply -f deployment/k8s/brix/07a-service-loadbalancer.yaml
```

**Bare Metal** (NodePort):
```bash
kubectl apply -f deployment/k8s/brix/07b-service-nodeport.yaml
```

## Troubleshooting

### Check Pod Status
```bash
kubectl get pods -n brix
kubectl describe pod -n brix <pod-name>
```

### View Application Logs
```bash
kubectl logs -n brix -l app=brix-pizza --tail=100 -f
```

### Database Issues
```bash
# Check MySQL status
kubectl get pod -n brix -l app=mysql

# View MySQL logs
kubectl logs -n brix -l app=mysql

# Connect to database
kubectl exec -it -n brix $(kubectl get pod -n brix -l app=mysql -o jsonpath='{.items[0].metadata.name}') -- mysql -u brix_user -p
# Default password: demo-brix-pass-CHANGE-ME
```

### Common Issues

**Pods pending**: Check persistent volume claims
```bash
kubectl get pvc -n brix
```

**CrashLoopBackOff**: Check logs for errors
```bash
kubectl logs -n brix <pod-name> --previous
```

## Cleanup

```bash
kubectl delete namespace brix
```

This removes all resources including persistent volumes.

## Learn More

- [Main Project README](../README.md) - Application overview
- [Kubernetes Documentation](https://kubernetes.io/docs/home/)
