# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies including C compiler and webp library
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev libwebp-dev

# Copy go mod files
COPY go.mod go.sum ./

# Copy source code first (needed for go mod tidy)
COPY . .

# Download and tidy dependencies
RUN go mod tidy && go mod download

# Build the application with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o news-server ./cmd/server

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies including webp library
RUN apk add --no-cache ca-certificates tzdata curl libwebp

# Copy binary from builder
COPY --from=builder /app/news-server .

# Copy web assets and templates
COPY --from=builder /app/web ./web
COPY --from=builder /app/configs ./configs

# Create directories for uploads and logs
RUN mkdir -p /app/uploads /app/logs

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
CMD ["./news-server"]
