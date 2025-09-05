# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o grpc-server ./cmd/grpc-server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/grpc-server .

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Create non-root user and set up directories
RUN adduser -D -s /bin/sh appuser && \
    chown -R appuser:appuser /root
USER appuser

# Expose port
EXPOSE 50051

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ./grpc-server --help || exit 1

# Run the application (no config file - relies on environment variables)
CMD ["./grpc-server"]
