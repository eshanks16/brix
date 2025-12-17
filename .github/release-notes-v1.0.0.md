# Brix Pizza v1.0.0 🍕

First stable release of Brix Pizza - a production-ready pizza ordering application built with Go!

## 🚀 Quick Start

### Run Locally

```bash
git clone https://github.com/eshanks16/brix.git
cd brix
go run main.go
```

Open http://localhost:8080

### Deploy to Kubernetes

```bash
kubectl apply -f deployment/k8s/brix/
kubectl apply -f deployment/k8s/mysql/
kubectl get pods -n brix
```

See [deployment/README.md](deployment/README.md) for details.

## ✨ Features

### Core Application
- 🔐 User authentication with bcrypt password hashing
- 🍕 8 pizza styles (New York, Chicago, Detroit, and more)
- 📏 4 sizes with dynamic pricing
- 🎨 Customizable toppings (split left/right or whole pizza)
- 📊 Order history dashboard
- 💾 SQLite (dev) and MySQL (prod) support

### REST API
- 📡 Complete REST API with session-based auth
- 🔑 Optional Bearer token authentication
- 📖 Full API documentation with examples

### Production Ready
- 🔍 Structured logging with zerolog
- 📊 Prometheus metrics at `/metrics`
- ❤️ Health checks (`/health/live`, `/health/ready`)
- 🐳 Docker support with multi-stage builds
- ☸️ Complete Kubernetes manifests
- 📈 Horizontal Pod Autoscaler
- 🔒 Security context and resource limits

### Monitoring & Observability
- Request logging with unique request IDs
- User event tracking (login, logout, orders)
- HTTP metrics (requests, duration, status codes)
- Business metrics (orders, revenue)
- Database connection pool metrics

## 📦 Container Image

```bash
docker pull eshanks16/brix-pizza:v1.0.0
```

**Platforms:** linux/amd64, linux/arm64

## 🔧 What's Included

- **Application Server** - Go HTTP server with graceful shutdown
- **Database Migrations** - Automatic schema management
- **Health Checks** - Kubernetes-ready liveness/readiness probes
- **API Documentation** - Complete examples in `docs/API_EXAMPLES.md`
- **Kubernetes Manifests** - Production-ready deployment files
- **Unit Tests** - 76.7% code coverage

## ⚙️ Configuration

Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `BRIX_API_KEY` | - | API authentication key |
| `DATABASE_URL` | SQLite | MySQL connection string |

## 🐛 Known Issues

- None! This is the first stable release.

## 📚 Documentation

- [README.md](README.md) - Full documentation
- [deployment/README.md](deployment/README.md) - Kubernetes deployment guide
- [docs/API_EXAMPLES.md](docs/API_EXAMPLES.md) - API usage examples
- [CHANGELOG.md](CHANGELOG.md) - Complete changelog

## 🙏 Acknowledgments

Built with:
- [Go](https://golang.org/) - Programming language
- [zerolog](https://github.com/rs/zerolog) - Structured logging
- [Prometheus](https://prometheus.io/) - Monitoring
- [MySQL](https://www.mysql.com/) - Database

## 📝 License

MIT License - see [LICENSE](LICENSE) for details

---

**Docker Hub:** https://hub.docker.com/r/eshanks16/brix-pizza
**Documentation:** https://github.com/eshanks16/brix
**Report Issues:** https://github.com/eshanks16/brix/issues
