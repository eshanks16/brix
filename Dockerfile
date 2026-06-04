# Multi-stage Dockerfile for Brix Pizza
#
# Stage 1: Build using the Red Hat UBI 9 Go toolset.
# registry.access.redhat.com/ubi9/go-toolset ships gcc and the Go toolchain
# on a RHEL UBI 9 base — no apk, no musl, full glibc for CGO.
# The official Go image is used only for building — it is discarded by the
# multi-stage build and does not appear in the final image. Security scanning
# of the shipped image therefore reflects only the UBI runtime stage below.
FROM golang:1.26 AS builder

WORKDIR /build

# Download dependencies before copying source so Docker layer cache is reused
# when only source files change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=1 is required by the sqlite3 driver.
# -ldflags="-w -s" strips debug info to reduce binary size.
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o brix-pizza .

# Stage 2: Minimal Red Hat UBI 9 runtime image.
FROM registry.access.redhat.com/ubi9/ubi-minimal:9

# Update all OS packages first to apply the latest security patches, then
# install runtime dependencies. microdnf clean all removes the package cache
# to keep the layer small.
# ca-certificates: TLS roots for outbound HTTPS (inference server).
# wget:            used by the HEALTHCHECK below.
# tzdata:          timezone data for time formatting.
# libgcc:          required by the CGO-compiled sqlite3 driver at runtime.
RUN microdnf update -y && \
    microdnf install -y ca-certificates wget tzdata libgcc && \
    microdnf clean all

WORKDIR /app

# Copy binary and assets from the builder stage.
COPY --from=builder /build/brix-pizza .
COPY templates/ ./templates/
COPY static/ ./static/

RUN chmod +x /app/brix-pizza && \
    chmod -R 755 /app/templates && \
    chmod -R 755 /app/static

# Run as non-root (uid 1000) — consistent with the K8s securityContext.
USER 1000

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

CMD ["/app/brix-pizza"]
