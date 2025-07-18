# Click-Lite Log Analytics - Architecture

## Overview

Click-Lite is a lightweight, high-performance log analytics platform built on ClickHouse. It provides real-time log ingestion, storage, querying, and visualization capabilities with enterprise-grade features like RBAC, monitoring, and trace correlation.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          Frontend Layer                          │
├─────────────────┬────────────────┬──────────────┬──────────────┤
│   Dashboard UI  │  Query Builder │ Real-time UI │  Export UI   │
└────────┬────────┴───────┬────────┴──────┬───────┴──────┬───────┘
         │                │                │              │
┌────────┴────────────────┴────────────────┴──────────────┴───────┐
│                          API Gateway                             │
│                    (Authentication & Routing)                    │
└────────┬────────────────┬────────────────┬──────────────────────┘
         │                │                │
┌────────┴──────┐ ┌───────┴──────┐ ┌──────┴────────┐
│  Query Engine │ │ Ingestion API│ │ Export Service│
└───────┬───────┘ └───────┬──────┘ └───────┬───────┘
        │                 │                 │
┌───────┴─────────────────┴─────────────────┴─────────┐
│                 ClickHouse Cluster                   │
│          (Distributed Tables & Sharding)             │
└──────────────────────────────────────────────────────┘
```

## Core Components

### 1. Log Ingestion System

**Multi-Protocol Receiver**
- **HTTP Receiver**: RESTful API for log submission
  - Endpoint: `/api/v1/logs`
  - Supports JSON and plain text
  - Bulk ingestion support
- **TCP Receiver**: Raw socket connection for high-throughput
  - Port: 5514
  - Binary protocol for efficiency
- **Syslog Receiver**: RFC 5424 compliant
  - Port: 514 (UDP/TCP)
  - Automatic parsing of syslog format

**Go Agent**
- Lightweight daemon for log collection
- Features:
  - File tailing
  - Log rotation handling
  - Buffering and batching
  - Compression before transmission
  - Automatic retry with exponential backoff

**Batch Processing**
- Configurable batch size (default: 1000 logs)
- Time-based flushing (default: 5 seconds)
- Memory-efficient circular buffer
- At-least-once delivery guarantee

### 2. Storage Layer

**Table Structure**
```sql
CREATE TABLE logs_YYYYMMDD (
    timestamp DateTime64(3),
    level String,
    message String,
    service String,
    trace_id String,
    span_id String,
    attributes Map(String, String),
    INDEX idx_timestamp timestamp TYPE minmax GRANULARITY 1,
    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (service, timestamp)
TTL timestamp + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;
```

**Compression Strategy**
- CODEC: LZ4 for hot data, ZSTD for cold data
- Compression ratio: ~10:1 for typical logs
- Background compression jobs for older partitions

**Data Lifecycle**
- Hot tier: Last 7 days (SSD storage)
- Warm tier: 7-30 days (HDD storage)
- Cold tier: 30-90 days (Object storage)
- Automatic archival after TTL

### 3. Parsing Engine

**JSON Parser**
- Automatic schema detection
- Nested object support
- Array handling
- Type inference

**Regex Parser**
- Pre-built patterns for common formats
- Custom pattern definition
- Named capture groups
- Performance optimization with RE2

**Parser Configuration**
```yaml
parsers:
  - name: nginx_access
    type: regex
    pattern: '^(?P<ip>\S+) .* \[(?P<timestamp>.*)\] "(?P<method>\S+) (?P<path>\S+).*" (?P<status>\d+)'
  - name: json_logs
    type: json
    fields:
      - name: timestamp
        path: $.timestamp
        type: datetime
```

### 4. Query Engine

**SQL Interface**
- Full ClickHouse SQL support
- Query optimization
- Prepared statements
- Query caching

**Query Builder**
- Visual query construction
- Auto-completion
- Syntax highlighting
- Query validation

**Aggregation Functions**
- Time-based: rate(), increase()
- Statistical: percentile(), stddev()
- Custom UDFs support

### 5. Real-time Streaming

**WebSocket Architecture**
```
Client <---> WebSocket Server <---> Kafka <---> Log Ingestion
                    |
                    +---> Filtering Engine
                    |
                    +---> Rate Limiter
```

**Features**
- Live tail with <100ms latency
- Client-side filtering
- Regex and field-based filters
- Pause/resume capability
- Backpressure handling

### 6. Dashboard System

**Widget Types**
- Time series charts (Line, Area)
- Categorical charts (Bar, Pie)
- Stat panels
- Log tables
- Heatmaps

**Dashboard Storage**
```json
{
  "id": "dashboard-123",
  "title": "Application Metrics",
  "widgets": [
    {
      "type": "line_chart",
      "query": "SELECT toStartOfMinute(timestamp) as t, count() FROM logs GROUP BY t",
      "position": {"x": 0, "y": 0, "w": 6, "h": 4}
    }
  ],
  "refresh_interval": 30
}
```

### 7. Monitoring System

**Metrics Collection**
- Prometheus-compatible metrics
- Key metrics:
  - Ingestion rate (logs/sec)
  - Query latency (p50, p95, p99)
  - Storage utilization
  - Error rates

**Health Checks**
- Component health endpoints
- Dependency checks
- Automated failover triggers

### 8. Security & RBAC

**Authentication**
- JWT-based authentication
- OAuth2/OIDC support
- API key authentication

**Authorization Model**
```yaml
roles:
  - name: admin
    permissions:
      - logs:*
      - dashboards:*
      - users:*
  - name: analyst
    permissions:
      - logs:read
      - dashboards:read
      - dashboards:create
  - name: viewer
    permissions:
      - logs:read
      - dashboards:read
```

**Audit Logging**
- All API calls logged
- Query history tracking
- Configuration changes

### 9. Trace Correlation

**Trace ID Extraction**
- Automatic detection in logs
- OpenTelemetry format support
- Custom trace ID patterns

**Correlation Features**
- Cross-service trace viewing
- Latency analysis
- Error propagation tracking

### 10. Export System

**Supported Formats**
- CSV with custom delimiters
- Excel with formatting
- JSON (newline-delimited)
- Parquet for big data

**Export Pipeline**
```
Query Engine --> Result Set --> Format Converter --> Compression --> S3/Download
```

## Deployment Architecture

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: click-lite-api
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: api
        image: click-lite/api:latest
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
```

### High Availability

**API Layer**
- Multiple replicas behind load balancer
- Health check-based routing
- Circuit breaker pattern

**ClickHouse Cluster**
- ReplicatedMergeTree for data redundancy
- Distributed tables for sharding
- ZooKeeper for coordination

## Performance Characteristics

### Ingestion Performance
- Target: 1M logs/second per node
- Batching: 1000 logs per batch
- Latency: <100ms end-to-end

### Query Performance
- Simple queries: <100ms
- Aggregation queries: <1s for 1B records
- Full-text search: <500ms

### Storage Efficiency
- Raw to compressed: 10:1 ratio
- Daily partition size: ~100GB compressed
- Query granularity: 8192 rows

## Scalability Considerations

### Horizontal Scaling
- Ingestion: Add more receiver nodes
- Storage: Add ClickHouse shards
- Query: Add reader replicas

### Vertical Scaling
- CPU: Query complexity bound
- Memory: Working set size
- Storage: Retention period

## Future Architecture Enhancements

1. **Multi-Region Support**
   - Cross-region replication
   - Geo-distributed queries
   - Regional data residency

2. **Machine Learning Pipeline**
   - Anomaly detection models
   - Pattern recognition
   - Predictive analytics

3. **Plugin Architecture**
   - Custom parsers
   - Output integrations
   - Authentication providers

4. **Stream Processing**
   - Apache Flink integration
   - Real-time alerting
   - Complex event processing