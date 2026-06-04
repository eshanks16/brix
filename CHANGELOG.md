# Changelog

All notable changes to Brix Pizza will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-06-04

### Added

#### Brix AI Chatbot

- Floating chat widget displayed on every page when `CHATBOT_ENABLED` is set
- Brix mascot persona with pizza-focused system prompt
- Proxies to any OpenAI-compatible inference server (`/v1/chat/completions`)
- Conversation history maintained client-side (last 10 exchanges forwarded for context)
- Bearer token authentication via `CHATBOT_TOKEN` environment variable
- TLS skip-verify support via `CHATBOT_TLS_SKIP_VERIFY` for self-signed certificates
- Structured error logging on all inference failure paths
- `POST /api/chat` backend handler (`internal/handlers/chatbot.go`)
- `static/js/chatbot.js` and `static/css/chatbot.css` — vanilla JS, no dependencies

#### OpenShift Virtualization MySQL Deployment

- New `deployment/k8s/mysql-vm/` manifests to run MySQL inside a KubeVirt VirtualMachine
- CentOS Stream 9 golden image via DataSource in `openshift-virtualization-os-images`
- Cloud-init bootstrap installs and configures MySQL 8.0 on first boot
- Same `mysql-service` ClusterIP name as the StatefulSet deployment — no app changes needed
- README with verification steps, customization options, and tear-down instructions

#### Chatbot Kubernetes Manifests

- `CHATBOT_ENABLED`, `CHATBOT_INFERENCE_URL`, `CHATBOT_MODEL`, `CHATBOT_TLS_SKIP_VERIFY` added to `deployment/k8s/brix/02-configmap.yaml` (empty/disabled by default)
- `chatbot-token` added to `deployment/k8s/brix/03-secret.yaml`
- All five chatbot env vars wired into `deployment/k8s/brix/04-deployment.yaml`

### Changed

- `WriteTimeout` increased from 15 s to 60 s to accommodate inference server response times
- Template parsing now uses a `FuncMap` to expose `chatbotEnabled()` without modifying any handler data structs

## [1.0.0] - 2024-12-14

### Added

#### Core Features
- User registration and authentication with bcrypt password hashing
- Session-based login system with secure cookie handling
- Pizza ordering with customizable toppings (split left/right or whole pizza)
- 8 different pizza styles (New York, Chicago, Detroit, Neapolitan, Sicilian, California, Greek, St. Louis)
- 4 pizza sizes with dynamic pricing
- 15+ toppings across meat, vegetable, and cheese categories
- Order history dashboard for logged-in users
- Database-driven menu system with automatic seeding

#### REST API (v1)
- `/api/v1/menu` - Get available pizza styles, sizes, and toppings
- `/api/v1/orders` - Create orders (POST) and list user orders (GET)
- Bearer token authentication support with `BRIX_API_KEY` environment variable
- Session-based authentication for API endpoints
- Complete API documentation in `docs/API_EXAMPLES.md`

#### Database Support
- SQLite support for local development
- MySQL/MariaDB support for production deployments
- Automatic database migrations with version tracking
- Kubernetes-safe concurrent initialization with transaction locking
- Connection pooling with configurable limits

#### Monitoring & Observability
- Structured logging with zerolog (JSON format in production)
- Request-level logging with unique request IDs
- User event logging (login, logout, order creation)
- HTTP request/response logging with method, path, status, duration
- Health check and metrics endpoints filtered at DEBUG level to reduce noise
- Prometheus metrics exposition at `/metrics` endpoint
- Custom metrics: HTTP requests, order counts, revenue tracking, database connections
- Database connection pool metrics (open, in-use, idle connections)

#### Health & Reliability
- Liveness probe at `/health/live` - Server responsiveness check
- Readiness probe at `/health/ready` - Database connectivity check
- Graceful shutdown with 30-second timeout
- Request timeout configuration (15s read, 15s write, 60s idle)

#### Kubernetes Deployment
- Complete Kubernetes manifests for production deployment
- MySQL StatefulSet with persistent storage (10Gi default)
- Horizontal Pod Autoscaler (1-5 replicas based on CPU)
- ConfigMap for application configuration
- Secrets management for database credentials and API keys
- Service definitions (ClusterIP, LoadBalancer, NodePort options)
- OpenShift Route support
- Prometheus ServiceMonitor for metrics scraping
- PodDisruptionBudget for high availability
- Security context (non-root, no privilege escalation)
- Resource requests and limits

#### Testing
- Comprehensive unit test suite (76.7% coverage)
- Health check tests (100% coverage)
- Handler tests (80.3% coverage)
- API endpoint tests (78.8% coverage)
- Database migration tests (63.2% coverage)
- Middleware tests for logging and Prometheus
- Race condition detection with `-race` flag support

#### User Interface
- Responsive web design with brick oven theme
- Interactive pizza builder with live visualization
- Real-time price calculator
- Client-side form validation
- Smooth scrolling navigation
- Animated smoke effects
- Mobile-friendly layout

#### Developer Experience
- Makefile with common development tasks
- Docker support with multi-stage builds
- Environment variable configuration
- Detailed README with setup instructions
- API usage examples and documentation
- Migration guide for database changes
- Production deployment guide

### Changed
- Health check and metrics endpoints now log at DEBUG level instead of INFO
- MySQL init ConfigMap now empty (migrations handled by application)
- Improved error handling with structured error logging

### Fixed
- Database schema initialization race condition in Kubernetes
- MySQL migration column name mismatch (password vs password_hash)
- Health check request logging verbosity

### Security
- Demo credentials clearly marked for development only
- Warnings in documentation about changing production passwords
- Non-root container execution
- Read-only root filesystem option
- Security context with dropped capabilities
- Session-based authentication
- Bcrypt password hashing

### Documentation
- Comprehensive README with quick start guide
- Kubernetes deployment documentation
- API usage examples with curl commands
- Database configuration guide
- Monitoring and logging setup
- Testing guide with coverage reports
- Troubleshooting section

[1.0.0]: https://github.com/eshanks16/brix/releases/tag/v1.0.0
