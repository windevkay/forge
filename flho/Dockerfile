# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user for build
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# -ldflags='-w -s' to strip debug info and reduce binary size
# -a to force rebuilding of packages
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o flho ./cmd/flho

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

# Create app directory
RUN mkdir -p /app && chown -R appuser:appgroup /app

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder --chown=appuser:appgroup /build/flho .

# Copy .env file if it exists (optional)
COPY --chown=appuser:appgroup .env* ./

# Switch to non-root user
USER appuser

# Expose port (default is 4000 based on your code)
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-4000}/health || exit 1

# Command to run the application
# Use exec form to ensure proper signal handling
CMD ["./flho", "-PORT=4000"]
