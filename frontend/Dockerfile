# Build stage
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Runtime stage
FROM nginx:alpine

# Install runtime dependencies
RUN apk add --no-cache curl

# Copy nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Copy built application from builder
COPY --from=builder /app/build /usr/share/nginx/html

# Create non-root user
RUN addgroup -g 1000 -S clicklite && \
    adduser -u 1000 -S clicklite -G clicklite && \
    chown -R clicklite:clicklite /usr/share/nginx/html && \
    chown -R clicklite:clicklite /var/cache/nginx && \
    chown -R clicklite:clicklite /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R clicklite:clicklite /var/run/nginx.pid

# Switch to non-root user
USER clicklite

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:3000/ || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]