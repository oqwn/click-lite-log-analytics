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

# Test bulk ingestion to trigger high ingestion rate alert
echo "9. Testing High Ingestion Rate (15k logs in rapid succession)..."
echo "   This should trigger 'high_ingestion_rate' alert (threshold: 10,000 logs/sec)"

# Send multiple batches rapidly to exceed 10k logs/sec threshold
for batch in {1..3}; do
  logs='['
  for i in {1..5000}; do
    if [ $i -gt 1 ]; then logs+=','; fi
    logs+='{
      "level": "'$([ $((i % 10)) -eq 0 ] && echo "error" || echo "info")'",
      "message": "High rate test log batch'$batch' item'$i'",
      "service": "load-test",
      "attributes": {
        "batch": '$batch',
        "test_type": "high_ingestion_rate"
      }
    }'
  done
  logs+=']'
  
  curl -s -X POST "${API_URL}/ingest/bulk" \
    -H "Content-Type: application/json" \
    -d "$logs" > /dev/null &
done

# Wait for background jobs
wait
echo "   ✓ Sent 15,000 logs rapidly"
echo

# Wait for metrics to update
sleep 3

# Check current ingestion rate
echo "10. Current Ingestion Rate:"
curl -s "${API_URL}/monitoring/metrics" | jq '.metrics[] | select(.name == "ingestion_rate_per_second")'
echo

# Generate slow queries to trigger slow query alert
echo "11. Testing Slow Queries (complex aggregations)..."
echo "   This should trigger 'slow_queries' alert (P99 > 5000ms threshold)"

# Create complex queries that will be slow
for i in {1..20}; do
  # Complex aggregation query that should be slow
  complex_query='{
    "sql": "SELECT service, level, COUNT(*) as count, AVG(CAST(json_extract(attributes, \"$.batch\") as FLOAT)) as avg_batch FROM logs WHERE timestamp > datetime(\"now\", \"-1 hour\") GROUP BY service, level HAVING count > 10 ORDER BY count DESC"
  }'
  
  curl -s -X POST "${API_URL}/query/execute" \
    -H "Content-Type: application/json" \
    -d "$complex_query" > /dev/null &
  
  # Also run multiple simultaneous queries to increase load
  if [ $((i % 5)) -eq 0 ]; then
    for j in {1..10}; do
      curl -s -X POST "${API_URL}/query/execute" \
        -H "Content-Type: application/json" \
        -d '{"sql": "SELECT * FROM logs ORDER BY timestamp DESC LIMIT 1000"}' > /dev/null &
    done
  fi
done

# Wait for queries to complete
wait
echo "   ✓ Executed 70+ complex queries"
echo

# Wait for metrics and alerts to update
sleep 5

# Check query performance metrics
echo "12. Query Performance Metrics:"
curl -s "${API_URL}/monitoring/metrics" | jq '.metrics[] | select(.name | contains("query_duration_ms"))'
echo

# Check for active alerts
echo "13. Active Alerts After Load Tests:"
curl -s "${API_URL}/monitoring/alerts/active" | jq '.alerts[] | {name: .name, severity: .severity, message: .message}'
echo

# Test memory spike (if possible)
echo "14. Simulating Memory Pressure..."
echo "   Sending very large log entries to increase memory usage"

# Create logs with large attributes
large_logs='['
for i in {1..100}; do
  if [ $i -gt 1 ]; then large_logs+=','; fi
  # Create a large attribute payload (about 10KB per log)
  large_data=$(printf '%.0s-' {1..10000})
  large_logs+='{
    "level": "debug",
    "message": "Large payload test '$i'",
    "service": "memory-test",
    "attributes": {
      "large_data": "'${large_data:0:10000}'",
      "size": 10000
    }
  }'
done
large_logs+=']'

curl -s -X POST "${API_URL}/ingest/bulk" \
  -H "Content-Type: application/json" \
  -d "$large_logs" > /dev/null
echo "   ✓ Sent logs with large payloads"
echo

# Final wait for all metrics to settle
sleep 5

# Show final system state
echo "15. Final System State:"
echo "   Health Status:"
curl -s "${API_URL}/monitoring/health" | jq '{status: .status, components: .components | to_entries | map({name: .key, status: .value.status})}'
echo
echo "   Active Alerts:"
curl -s "${API_URL}/monitoring/alerts/active" | jq '.alerts | length as $count | if $count == 0 then "No active alerts" else .[] | {name: .name, severity: .severity, message: .message} end'
echo

# Provide summary
echo "16. Summary of Alert Triggers:"
echo "   • High Ingestion Rate: Sent 15k logs rapidly (threshold: 10k/sec)"
echo "   • Slow Queries: Executed complex aggregations (threshold: P99 > 5000ms)"
echo "   • Memory Usage: Sent large payloads to increase memory"
echo "   • No Recent Logs: Will trigger after 1 minute of inactivity"
echo
echo "Check http://localhost:5173/monitoring to see:"
echo "  - Multiple active alerts with different severities"
echo "  - High ingestion rate metrics"
echo "  - Degraded query performance metrics"
echo "  - System under load conditions"

echo "=== Monitoring Test Complete ==="
echo
echo "To view the monitoring dashboard, open: http://localhost:5173/monitoring"