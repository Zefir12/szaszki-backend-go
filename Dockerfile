# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (needed for some Go modules)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first (for better layer caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary, GOOS=linux for Linux container
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage - minimal image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy any other necessary files (config files, static assets, etc.)
# COPY --from=builder /app/config ./config
# COPY --from=builder /app/static ./static

# Change ownership of the app directory to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:4411/api/health/healthcheck || exit 1

# Run the application
CMD ["./main"]