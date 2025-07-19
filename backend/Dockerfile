# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o click-lite .

# Runtime stage
FROM alpine:3.22

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S clicklite && \
    adduser -u 1000 -S clicklite -G clicklite

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/click-lite .

# Copy any config files if needed
# COPY --from=builder /app/config ./config

# Change ownership
RUN chown -R clicklite:clicklite /app

# Switch to non-root user
USER clicklite

# Expose ports
EXPOSE 8080 5514

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./click-lite"]