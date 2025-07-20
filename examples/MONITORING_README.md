# Click-Lite Monitoring System

## Overview

The Click-Lite monitoring system provides comprehensive observability for your log analytics platform, including:

- **Health Checks**: Monitor the status of all system components
- **Metrics Collection**: Track ingestion rates, query performance, and resource usage
- **Alerting System**: Get notified of critical issues and performance degradation
- **Web Dashboard**: Visualize system health and metrics in real-time

## Quick Start

### 1. Start the Services

```bash
# Terminal 1: Start the backend
cd backend
go run main.go

# Terminal 2: Start the frontend
cd frontend
pnpm dev
```

### 2. Test Monitoring Endpoints

Run the test script to verify all monitoring endpoints are working:

```bash
cd examples
./test_monitoring.sh
```

### 3. Run the Interactive Demo

Experience the full monitoring capabilities with the interactive demo:

```bash
cd examples
python3 monitoring_demo.py
```

This demo will:
- Display a real-time console dashboard
- Generate various load patterns
- Trigger alerts
- Show system behavior under different conditions

### 4. View the Web Dashboard

Open http://localhost:5173/monitoring in your browser to see:
- System health status
- Real-time metrics
- Active alerts
- Component health checks

## API Endpoints

### Health Checks

- `GET /api/v1/monitoring/health` - Comprehensive health status
- `GET /api/v1/monitoring/health/live` - Simple liveness check
- `GET /api/v1/monitoring/health/ready` - Readiness check

### Metrics

- `GET /api/v1/monitoring/metrics` - All system metrics

### Alerts

- `GET /api/v1/monitoring/alerts` - All alerts (active and resolved)
- `GET /api/v1/monitoring/alerts/active` - Only active alerts

## Key Metrics

### Ingestion Metrics
- `ingestion_rate_per_second` - Current log ingestion rate
- `total_logs_ingested` - Total number of logs ingested
- `ingestion_request_duration_ms` - HTTP request processing time
- `bulk_ingestion_size` - Size of bulk ingestion batches

### Query Metrics
- `query_rate_per_second` - Current query execution rate
- `total_queries_executed` - Total number of queries executed
- `query_duration_ms_*` - Query execution time statistics (avg, p50, p90, p99)
- `query_result_size` - Number of rows returned by queries

### Storage Metrics
- `storage_size_bytes` - Total storage size in bytes
- `storage_size_mb` - Total storage size in MB
- `storage_log_count` - Number of stored logs
- `storage_utilization_percent` - Storage capacity utilization

### System Metrics
- `memory_alloc_mb` - Allocated memory in MB
- `memory_total_mb` - Total memory usage in MB
- `num_goroutines` - Number of active goroutines

## Alert Rules

The system includes pre-configured alert rules:

1. **High Ingestion Rate**
   - Triggers when ingestion exceeds 10,000 logs/sec
   - Severity: Warning

2. **Slow Queries**
   - Triggers when P99 query duration exceeds 5 seconds
   - Severity: Warning

3. **High Memory Usage**
   - Triggers when memory usage exceeds 1GB
   - Severity: Critical

4. **Low Storage Space**
   - Triggers when free storage drops below 10%
   - Severity: Critical

5. **No Recent Logs**
   - Triggers when no logs are received for 1 minute
   - Severity: Info

## Integration Examples

### Spring Boot Integration

Add this to your `logback-spring.xml`:

```xml
<appender name="CLICK_LITE_SYSLOG" class="ch.qos.logback.classic.net.SyslogAppender">
    <syslogHost>localhost</syslogHost>
    <port>20004</port>
    <facility>USER</facility>
    <suffixPattern>%thread %logger %msg</suffixPattern>
</appender>

<root level="INFO">
    <appender-ref ref="CLICK_LITE_SYSLOG" />
</root>
```

### Python Application Monitoring

```python
import requests
import json
from datetime import datetime

def send_log(level, message, **attributes):
    log_entry = {
        "level": level,
        "message": message,
        "service": "my-python-app",
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "attributes": attributes
    }
    
    response = requests.post(
        "http://localhost:20002/api/v1/ingest/logs",
        json=[log_entry],
        headers={"Content-Type": "application/json"}
    )
    return response.status_code == 200

# Example usage
send_log("info", "Application started", version="1.0.0")
send_log("error", "Database connection failed", error_code="DB_CONN_001")
```

### Node.js Monitoring Client

```javascript
const axios = require('axios');

class ClickLiteLogger {
    constructor(service, apiUrl = 'http://localhost:20002/api/v1') {
        this.service = service;
        this.apiUrl = apiUrl;
        this.buffer = [];
        this.flushInterval = setInterval(() => this.flush(), 5000);
    }

    log(level, message, attributes = {}) {
        this.buffer.push({
            level,
            message,
            service: this.service,
            timestamp: new Date().toISOString(),
            attributes
        });

        if (this.buffer.length >= 100) {
            this.flush();
        }
    }

    async flush() {
        if (this.buffer.length === 0) return;

        const logs = [...this.buffer];
        this.buffer = [];

        try {
            await axios.post(`${this.apiUrl}/ingest/logs`, logs);
        } catch (error) {
            console.error('Failed to send logs:', error.message);
            // Re-add logs to buffer for retry
            this.buffer.unshift(...logs);
        }
    }

    close() {
        clearInterval(this.flushInterval);
        this.flush();
    }
}

// Usage
const logger = new ClickLiteLogger('my-node-app');
logger.log('info', 'Application started');
logger.log('error', 'Failed to process request', { user_id: 123, error: 'timeout' });
```

## Troubleshooting

### No Metrics Showing

1. Ensure logs are being ingested - check `/api/v1/ingest/health`
2. Wait 30-60 seconds for metrics to accumulate
3. Check browser console for API errors

### Alerts Not Triggering

1. Alert checks run every 30 seconds
2. Ensure thresholds are being exceeded
3. Check alert cooldown periods (5-30 minutes)

### High Memory Usage

1. Check batch processor settings
2. Reduce ingestion rate
3. Increase batch flush frequency

## Performance Tuning

### Optimize Ingestion

```go
// In main.go, adjust batch processor settings
batchProcessor := ingestion.NewBatchProcessor(
    db, 
    1000,              // Increase batch size
    2*time.Second      // Decrease flush interval
)
```

### Optimize Query Performance

1. Add appropriate indexes
2. Limit query result sizes
3. Use time-based filters

### Reduce Memory Usage

1. Decrease batch sizes
2. Implement log rotation
3. Archive old logs