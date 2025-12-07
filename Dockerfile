# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o camt-csv .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies (poppler for PDF support)
RUN apk add --no-cache ca-certificates poppler-utils

# Create non-root user
RUN adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /app/camt-csv .

# Copy database files (categories, etc.)
COPY --from=builder /app/database ./database

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

ENTRYPOINT ["./camt-csv"]
CMD ["--help"]
