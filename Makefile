.PHONY: help test test-verbose test-coverage run build clean docker-build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run all tests
	@echo "Running tests..."
	@go test ./internal/... -race 2>&1 | grep -v "LC_DYSYMTAB"

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	@go test ./internal/... -v -race 2>&1 | grep -v "LC_DYSYMTAB"

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test ./internal/... -race -coverprofile=coverage.out 2>&1 | grep -v "LC_DYSYMTAB"
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

run: ## Run the application locally
	@echo "Starting Brix Pizza..."
	@go run main.go

build: ## Build the application binary
	@echo "Building brix-pizza..."
	@go build -o brix-pizza .
	@echo "Binary created: ./brix-pizza"

clean: ## Clean build artifacts and test files
	@echo "Cleaning..."
	@rm -f brix-pizza coverage.out coverage.html
	@rm -rf db/
	@echo "Clean complete"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@cd deployment && docker build -t brix-pizza:latest -f Dockerfile ..
	@echo "Docker image built: brix-pizza:latest"

docker-run: ## Run Docker container locally
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --rm brix-pizza:latest
