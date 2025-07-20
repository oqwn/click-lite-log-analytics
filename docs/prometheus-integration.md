# Prometheus Integration Guide

Click-Lite Log Analytics exposes metrics in Prometheus format for monitoring and observability.

## Metrics Endpoint

The metrics are exposed at: `http://localhost:20002/metrics`

## Available Metrics

### Application Metrics

All application metrics are prefixed with `clicklite_` for easy identification.

#### Counters
- `clicklite_total_logs_ingested_total` - Total number of logs ingested since startup
- `clicklite_total_queries_executed_total` - Total number of queries executed
- `clicklite_failed_ingestions_total` - Total number of failed ingestion attempts
- `clicklite_failed_queries_total` - Total number of failed query attempts

#### Gauges
- `clicklite_ingestion_rate_per_second` - Current rate of log ingestion per second
- `clicklite_query_rate_per_second` - Current rate of query execution per second
- `clicklite_storage_size_mb` - Current storage size in megabytes
- `clicklite_storage_size_bytes` - Current storage size in bytes
- `clicklite_websocket_connections` - Current number of WebSocket connections
- `clicklite_active_alerts` - Number of currently active alerts
- `clicklite_table_count` - Number of tables in the database

#### Histograms
- `clicklite_query_duration_ms_*` - Query execution duration metrics
  - `clicklite_query_duration_ms_p50` - 50th percentile (median)
  - `clicklite_query_duration_ms_p90` - 90th percentile
  - `clicklite_query_duration_ms_p99` - 99th percentile
  - `clicklite_query_duration_ms_avg` - Average duration
  - `clicklite_query_duration_ms_min` - Minimum duration
  - `clicklite_query_duration_ms_max` - Maximum duration
- `clicklite_ingestion_request_duration_ms_*` - Ingestion request duration metrics
- `clicklite_batch_write_duration_ms_*` - Batch write operation duration metrics

### Process Metrics

Standard process-level metrics:
- `process_cpu_seconds_total` - Total CPU time spent
- `process_open_fds` - Number of open file descriptors
- `process_resident_memory_bytes` - Resident memory size

### Go Runtime Metrics

Go-specific runtime metrics:
- `go_memstats_alloc_bytes` - Bytes allocated and in use
- `go_goroutines` - Number of goroutines
- `go_gc_duration_seconds` - GC pause duration summary
- `go_info{version="go1.21"}` - Go version information

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'clicklite'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:20002']
    metrics_path: '/metrics'
```

## Grafana Dashboard

A pre-built Grafana dashboard is available at:
`examples/grafana/clicklite-dashboard.json`

To import:
1. Open Grafana (http://localhost:3000)
2. Go to Dashboards → Import
3. Upload the JSON file or paste its contents
4. Select your Prometheus datasource
5. Click Import

The dashboard includes:
- Log ingestion rate graph
- Query latency percentiles
- Total logs and queries counters
- Storage usage gauge
- Active alerts counter
- Memory usage and goroutines graphs

## Example Queries

### PromQL Examples

```promql
# Log ingestion rate over last 5 minutes
rate(clicklite_total_logs_ingested_total[5m])

# Average query latency
clicklite_query_duration_ms_avg

# 99th percentile query latency
clicklite_query_duration_ms_p99

# Storage growth rate (MB per hour)
rate(clicklite_storage_size_mb[1h]) * 3600

# Error rate percentage
rate(clicklite_failed_ingestions_total[5m]) / rate(clicklite_total_logs_ingested_total[5m]) * 100
```

## Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: clicklite
    rules:
      - alert: HighIngestionRate
        expr: clicklite_ingestion_rate_per_second > 10000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High log ingestion rate"
          description: "Ingestion rate is {{ $value }} logs/sec"
      
      - alert: SlowQueries
        expr: clicklite_query_duration_ms_p99 > 5000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Slow query performance"
          description: "P99 query latency is {{ $value }}ms"
      
      - alert: HighStorageUsage
        expr: clicklite_storage_size_mb > 10000
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "High storage usage"
          description: "Storage usage is {{ $value }}MB"
```

## Testing

Run the test script to verify metrics:

```bash
cd examples
./test_prometheus.sh
```

This will:
1. Check if the metrics endpoint is accessible
2. Generate test data
3. Display key metrics
4. Show configuration examples

## Troubleshooting

### No metrics showing
- Ensure the backend is running on port 20002
- Check that `/metrics` returns data: `curl http://localhost:20002/metrics`
- Verify Prometheus can reach the endpoint

### Missing metrics
- Some metrics only appear after activity (e.g., query metrics need queries)
- Histograms show multiple sub-metrics (p50, p90, p99, avg, etc.)
- Counters always end with `_total` suffix

### Performance considerations
- The metrics endpoint is lightweight and can be scraped frequently
- Default scrape interval of 15s is recommended
- Metrics are calculated on-demand, not stored