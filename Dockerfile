# ==========================================
# Build Stage
# ==========================================
FROM golang:1.22-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and other build tools
RUN apk add --no-cache git tzdata

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# CGO_ENABLED=0 creates a statically linked binary (no C dependencies)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/server ./cmd/server

# ==========================================
# Run Stage
# ==========================================
FROM alpine:3.19

# Add CA certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy binary and necessary assets from builder
COPY --from=builder /app/bin/server ./server
COPY --from=builder /app/migrations ./migrations

# Create uploads directory and set permissions
RUN mkdir -p ./uploads && chown -R appuser:appgroup ./uploads

# Switch to non-root user for security
USER appuser

# Expose port (must match .env APP_PORT)
EXPOSE 8080

# Health check to ensure the container is running properly
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Command to run the executable
CMD ["./server"]
