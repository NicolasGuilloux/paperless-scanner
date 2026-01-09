# Multi-stage Dockerfile for paperless-scanner
# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /build

# Copy go mod files from src directory
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code from src directory
COPY src/*.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o paperless-scanner .

# Stage 2: Runtime image
FROM alpine:latest

# Install SANE backends for scanner support
# Install ca-certificates for HTTPS connections
RUN apk add --no-cache \
    sane \
    sane-backends \
    ca-certificates \
    tzdata

# Create a non-root user to run the application
# Note: 'scanner' group already exists from SANE installation, so we use 'appuser'
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/paperless-scanner /app/paperless-scanner

# Create directory for scans (temporary storage)
RUN mkdir -p /tmp/paperless-scanner && \
    chown -R appuser:appuser /tmp/paperless-scanner /app

# Switch to non-root user
USER appuser

# Set environment variables (can be overridden at runtime)
ENV SCANNER_URL="" \
    PAPERLESS_URL="" \
    PAPERLESS_TOKEN=""

# Default command - use environment variables or command-line flags
ENTRYPOINT ["/app/paperless-scanner"]

# Default arguments (can be overridden)
CMD ["--help"]
