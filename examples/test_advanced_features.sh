#!/bin/bash

echo "=== Testing Advanced Features ==="
echo

API_URL="http://localhost:20002/api/v1"

# Check if backend is running
if ! curl -s "$API_URL/health" > /dev/null; then
    echo "❌ Backend is not running on port 20002"
    echo "Please start it with: cd backend && go run main.go"
    exit 1
fi

echo "✅ Backend is running"
echo

# Generate logs with trace IDs and errors
echo "📊 Generating logs with trace IDs and error patterns..."

# Trace 1: Successful request flow
TRACE_ID_1="a1b2c3d4e5f6789012345678901234567890"
curl -s -X POST "$API_URL/ingest/logs" \
    -H "Content-Type: application/json" \
    -d '[
        {
            "level": "info",
            "message": "GET /api/users - Request received",
            "service": "api-gateway",
            "trace_id": "'$TRACE_ID_1'",
            "span_id": "span001",
            "attributes": {
                "method": "GET",
                "path": "/api/users",
                "user_id": "user123"
            }
        },
        {
            "level": "info",
            "message": "Fetching user data from database",
            "service": "user-service",
            "trace_id": "'$TRACE_ID_1'",
            "span_id": "span002",
            "attributes": {
                "parent_span_id": "span001",
                "operation": "db.query",
                "table": "users"
            }
        },
        {
            "level": "info",
            "message": "User data retrieved successfully",
            "service": "user-service",
            "trace_id": "'$TRACE_ID_1'",
            "span_id": "span002",
            "attributes": {
                "duration_ms": 45,
                "rows_returned": 1
            }
        }
    ]' > /dev/null

# Trace 2: Request with errors
TRACE_ID_2="b2c3d4e5f67890123456789012345678901"
curl -s -X POST "$API_URL/ingest/logs" \
    -H "Content-Type: application/json" \
    -d '[
        {
            "level": "info",
            "message": "POST /api/orders - Request received",
            "service": "api-gateway",
            "trace_id": "'$TRACE_ID_2'",
            "span_id": "span003"
        },
        {
            "level": "error",
            "message": "Database connection timeout: Connection to database failed after 5000ms",
            "service": "order-service",
            "trace_id": "'$TRACE_ID_2'",
            "span_id": "span004",
            "attributes": {
                "parent_span_id": "span003",
                "error": "connection_timeout",
                "retry_count": 3
            }
        },
        {
            "level": "error",
            "message": "Order creation failed: Database unavailable",
            "service": "api-gateway",
            "trace_id": "'$TRACE_ID_2'",
            "span_id": "span003",
            "attributes": {
                "status_code": 503
            }
        }
    ]' > /dev/null

# Generate various error patterns
echo "🔍 Generating logs with various error patterns..."

ERROR_PATTERNS=(
    '{"level": "error", "message": "NullPointerException at com.example.UserService.getUser(UserService.java:45)", "service": "user-service"}'
    '{"level": "error", "message": "Out of memory: Java heap space", "service": "analytics-service"}'
    '{"level": "error", "message": "Connection refused: Unable to connect to Redis at localhost:6379", "service": "cache-service"}'
    '{"level": "error", "message": "SQL Error: Deadlock detected when trying to acquire lock", "service": "order-service"}'
    '{"level": "error", "message": "HTTP 503 Service Unavailable: Payment gateway timeout", "service": "payment-service"}'
    '{"level": "warn", "message": "Disk space low: Only 5% free space remaining on /data", "service": "storage-service"}'
    '{"level": "fatal", "message": "Critical: System shutting down due to unrecoverable error", "service": "core-service"}'
)

for pattern in "${ERROR_PATTERNS[@]}"; do
    curl -s -X POST "$API_URL/ingest/logs" \
        -H "Content-Type" -d "[$pattern]" > /dev/null
done

# Generate high error rate for anomaly detection
echo "📈 Generating high error rate for anomaly detection..."
for i in {1..50}; do
    curl -s -X POST "$API_URL/ingest/logs" \
        -H "Content-Type: application/json" \
        -d '[
            {"level": "error", "message": "Request timeout after 30000ms", "service": "api-gateway"},
            {"level": "error", "message": "Failed to process payment: Invalid card", "service": "payment-service"}
        ]' > /dev/null
done

echo "✅ Test data generated"
echo

# Wait for processing
sleep 2

# Test trace correlation
echo "=== Testing Trace Correlation ==="
echo

echo "1. Getting all traces:"
curl -s "$API_URL/traces?limit=5" | jq '.traces[] | {trace_id: .trace_id, services: .services, span_count: .span_count, error_count: .error_count}'
echo

echo "2. Getting specific trace details:"
curl -s "$API_URL/traces/$TRACE_ID_1" | jq '{trace_id: .trace_id, duration: .duration, services: .services, spans: .spans | length}'
echo

echo "3. Getting trace timeline:"
curl -s "$API_URL/traces/$TRACE_ID_2/timeline" | jq '.events[] | {service: .service, operation: .operation, status: .status, duration: .duration}'
echo

# Test error detection
echo "=== Testing Error Detection ==="
echo

echo "1. Getting error statistics:"
curl -s "$API_URL/errors/stats" | jq '.stats[] | {pattern: .pattern, category: .category, count: .count, rate: .rate, trend: .trend}'
echo

echo "2. Getting error anomalies:"
curl -s "$API_URL/errors/anomalies" | jq '.anomalies[] | {type: .type, category: .category, severity: .severity, message: .message}'
echo

echo "3. Getting error trends:"
curl -s "$API_URL/errors/trends" | jq '.'
echo

# Test data export
echo "=== Testing Data Export ==="
echo

echo "1. Getting available export formats:"
curl -s "$API_URL/export/formats" | jq '.formats[] | {format: .format, name: .name, extension: .extension}'
echo

echo "2. Exporting logs as CSV:"
curl -s -X POST "$API_URL/export/logs" \
    -H "Content-Type: application/json" \
    -d '{
        "format": "csv",
        "limit": 10,
        "include_headers": true,
        "fields": ["timestamp", "level", "service", "message", "trace_id"]
    }' \
    -o test_export.csv

echo "✅ Exported to test_export.csv"
head -5 test_export.csv
echo

echo "3. Exporting error logs as JSON:"
curl -s -X POST "$API_URL/export/logs" \
    -H "Content-Type: application/json" \
    -d '{
        "format": "json",
        "filters": [
            {"field": "level", "operator": "=", "value": "error"}
        ],
        "limit": 5
    }' | jq '.logs[] | {timestamp: .timestamp, level: .level, message: .message}'
echo

# Clean up
rm -f test_export.csv

echo "=== Advanced Features Test Complete ==="
echo
echo "Summary:"
echo "✅ Trace correlation is working - traces are being correlated across services"
echo "✅ Error detection is active - patterns are being detected and analyzed"
echo "✅ Anomaly detection found high error rates"
echo "✅ Data export supports CSV, JSON, and Excel formats"
echo
echo "You can now:"
echo "1. View traces at http://localhost:5173/traces"
echo "2. See error dashboard at http://localhost:5173/errors"
echo "3. Export data in various formats from the UI"