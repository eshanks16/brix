# Multi-stage Dockerfile for Brix Pizza
# Stage 1: Build the Go application
FROM golang:1.25-alpine AS builder

# Install build dependencies (gcc, musl-dev for CGO support needed by SQLite)
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# -ldflags="-w -s" strips debug information to reduce binary size
# CGO_ENABLED=1 is needed for sqlite3 driver
# Set CC and disable problematic functions for musl compatibility
# Buildx will automatically build natively for each target platform
RUN CGO_ENABLED=1 CGO_CFLAGS="-D_LARGEFILE64_SOURCE" go build -ldflags="-w -s" -o brix-pizza .

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates wget tzdata

# Create non-root user for security
RUN adduser -D -u 1000 -g brix brix

# Create app directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/brix-pizza .

# Copy templates and static files
COPY templates/ ./templates/
COPY static/ ./static/

# Change ownership to non-root user
RUN chown -R brix:brix /app

# Switch to non-root user
USER brix

# Expose port 8080
EXPOSE 8080

# Health check using wget
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

# Run the application
CMD ["/app/brix-pizza"]
