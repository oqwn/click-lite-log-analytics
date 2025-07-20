#!/bin/bash

# Test Prometheus metrics endpoint

echo "=== Testing Prometheus Metrics Endpoint ==="
echo

# First, ensure backend is running
if ! curl -s http://localhost:20002/api/v1/health > /dev/null; then
    echo "❌ Backend is not running on port 20002"
    echo "Please start it with: cd backend && go run main.go"
    exit 1
fi

echo "✅ Backend is running"
echo

# Test the Prometheus metrics endpoint
echo "📊 Fetching Prometheus metrics from /metrics endpoint..."
echo
curl -s http://localhost:20002/metrics | head -50

echo
echo "=== Generating some activity to populate metrics ==="
echo

# Generate some logs to increase counters
echo "📤 Sending test logs..."
for i in {1..5}; do
    curl -s -X POST http://localhost:20002/api/v1/ingest/logs \
        -H "Content-Type: application/json" \
        -d '[
            {"level": "info", "message": "Test log '${i}'", "service": "prometheus-test"},
            {"level": "error", "message": "Test error '${i}'", "service": "prometheus-test"}
        ]' > /dev/null
    echo -n "."
done
echo " Done!"

# Execute some queries
echo "🔍 Executing test queries..."
for i in {1..3}; do
    curl -s -X POST http://localhost:20002/api/v1/query/execute \
        -H "Content-Type: application/json" \
        -d '{"sql": "SELECT COUNT(*) FROM logs WHERE service = \"prometheus-test\""}' > /dev/null
    echo -n "."
done
echo " Done!"

# Wait for metrics to update
sleep 2

echo
echo "=== Updated Prometheus Metrics ==="
echo

# Show key metrics in Prometheus format
echo "# Key application metrics:"
curl -s http://localhost:20002/metrics | grep -E "^clicklite_" | head -20

echo
echo "# Process metrics:"
curl -s http://localhost:20002/metrics | grep -E "^process_" | head -10

echo
echo "# Go runtime metrics:"
curl -s http://localhost:20002/metrics | grep -E "^go_" | head -10

echo
echo "=== Prometheus Configuration Example ==="
echo
cat << 'EOF'
To scrape these metrics with Prometheus, add this to your prometheus.yml:

scrape_configs:
  - job_name: 'clicklite'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:20002']
    metrics_path: '/metrics'

Then you can:
1. Query metrics in Prometheus: http://localhost:9090
2. Create Grafana dashboards: http://localhost:3000
3. Set up alerts based on these metrics

Example queries:
- rate(clicklite_total_logs_ingested[5m]) - Log ingestion rate
- clicklite_query_duration_ms_p99 - 99th percentile query latency
- clicklite_storage_size_mb - Current storage usage
EOF

echo
echo "✅ Prometheus metrics endpoint is working!"