#!/bin/bash

# Test monitoring endpoints for Click-Lite Log Analytics
# This script tests health checks, metrics, and alerts

API_URL="http://localhost:20002/api/v1"

echo "=== Testing Monitoring System ==="
echo

# Test health endpoint
echo "1. Testing Health Check Endpoint..."
curl -s "${API_URL}/monitoring/health" | jq '.'
echo

# Test liveness endpoint
echo "2. Testing Liveness Endpoint..."
curl -s "${API_URL}/monitoring/health/live" | jq '.'
echo

# Test readiness endpoint
echo "3. Testing Readiness Endpoint..."
curl -s "${API_URL}/monitoring/health/ready" | jq '.'
echo

# Test metrics endpoint
echo "4. Testing Metrics Endpoint..."
curl -s "${API_URL}/monitoring/metrics" | jq '.'
echo

# Test alerts endpoint
echo "5. Testing Active Alerts Endpoint..."
curl -s "${API_URL}/monitoring/alerts/active" | jq '.'
echo

# Generate some load to see metrics change
echo "6. Generating Load for Metrics..."
echo "   Ingesting 1000 test logs..."

# Generate test logs
for i in {1..10}; do
  logs='['
  for j in {1..100}; do
    if [ $j -gt 1 ]; then logs+=','; fi
    logs+='{
      "level": "info",
      "message": "Test log message '$((i*100+j))'",
      "service": "monitoring-test",
      "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'",
      "attributes": {
        "batch": '$i',
        "index": '$j',
        "test": true
      }
    }'
  done
  logs+=']'
  
  curl -s -X POST "${API_URL}/ingest/logs" \
    -H "Content-Type: application/json" \
    -d "$logs" > /dev/null
  
  echo -n "."
done
echo " Done!"
echo

# Execute some queries to generate query metrics
echo "7. Executing Test Queries..."
for i in {1..5}; do
  curl -s -X POST "${API_URL}/query/execute" \
    -H "Content-Type: application/json" \
    -d '{
      "sql": "SELECT level, COUNT(*) as count FROM logs WHERE service = \"monitoring-test\" GROUP BY level"
    }' > /dev/null
  echo -n "."
done
echo " Done!"
echo

# Wait a bit for metrics to update
sleep 2

# Check metrics again to see changes
echo "8. Checking Updated Metrics..."
curl -s "${API_URL}/monitoring/metrics" | jq '.metrics[] | select(.name | contains("ingestion_rate") or contains("query") or contains("total"))'
echo

# Test bulk ingestion to trigger potential alerts
echo "9. Testing Bulk Ingestion (10k logs)..."
logs='['
for i in {1..10000}; do
  if [ $i -gt 1 ]; then logs+=','; fi
  logs+='{
    "level": "'$([ $((i % 10)) -eq 0 ] && echo "error" || echo "info")'",
    "message": "Bulk test log '$i'",
    "service": "bulk-test",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'"
  }'
done
logs+=']'

curl -s -X POST "${API_URL}/ingest/bulk" \
  -H "Content-Type: application/json" \
  -d "$logs" | jq '.'
echo

# Check for any alerts that might have been triggered
echo "10. Checking for Alerts After Load..."
curl -s "${API_URL}/monitoring/alerts/active" | jq '.'
echo

echo "=== Monitoring Test Complete ==="
echo
echo "To view the monitoring dashboard, open: http://localhost:5173/monitoring"