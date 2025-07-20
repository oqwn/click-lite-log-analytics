#!/bin/bash

echo "=== Quick Monitoring Test ==="
echo "This will generate test data to populate the monitoring dashboard"
echo

# Check if backend is running
if ! curl -s http://localhost:20002/api/v1/health > /dev/null; then
    echo "❌ Backend is not running on port 20002"
    echo "Please start it with: cd backend && go run main.go"
    exit 1
fi

echo "✅ Backend is running"

# Generate some test logs
echo "📊 Generating test logs..."

for i in {1..100}; do
    # Create a batch of logs
    logs='[
        {
            "level": "info",
            "message": "User login successful",
            "service": "auth-service",
            "attributes": {
                "user_id": "user'$i'",
                "ip": "192.168.1.'$((i % 255 + 1))'",
                "success": true
            }
        },
        {
            "level": "debug",
            "message": "Database query executed",
            "service": "db-service",
            "attributes": {
                "query_time_ms": '$((RANDOM % 100 + 10))',
                "table": "users"
            }
        }
    ]'
    
    # Send logs to the backend
    curl -s -X POST http://localhost:20002/api/v1/ingest/logs \
        -H "Content-Type: application/json" \
        -d "$logs" > /dev/null
    
    # Show progress
    if [ $((i % 20)) -eq 0 ]; then
        echo "  → Sent $i batches"
    fi
done

echo "✅ Sent 200 test logs"

# Execute some test queries to generate query metrics
echo "🔍 Executing test queries..."

for i in {1..10}; do
    # Execute different types of queries
    queries=(
        "SELECT COUNT(*) FROM logs WHERE level = 'info'"
        "SELECT service, COUNT(*) as count FROM logs GROUP BY service"
        "SELECT * FROM logs ORDER BY timestamp DESC LIMIT 10"
        "SELECT level, COUNT(*) FROM logs WHERE timestamp > datetime('now', '-1 hour') GROUP BY level"
    )
    
    for query in "${queries[@]}"; do
        curl -s -X POST http://localhost:20002/api/v1/query/execute \
            -H "Content-Type: application/json" \
            -d "{\"sql\": \"$query\"}" > /dev/null
    done
    
    echo "  → Executed query batch $i"
done

echo "✅ Executed 40 test queries"

# Wait a moment for metrics to update
echo "⏳ Waiting for metrics to update..."
sleep 3

# Show current metrics
echo "📈 Current Metrics:"
curl -s http://localhost:20002/api/v1/monitoring/metrics | jq '.metrics[] | select(.name | contains("total") or contains("rate")) | {name: .name, value: .value}'

echo
echo "🎉 Test data generated!"
echo
echo "Now open http://localhost:5173/monitoring to see:"
echo "  • Health Status: Should show 'OK' status"
echo "  • Metrics: Should show ingestion and query rates" 
echo "  • Alerts: Should clear the 'no recent logs' alert"
echo
echo "The dashboard refreshes every 5 seconds automatically."