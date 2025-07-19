# Storage Layer Examples

This directory contains comprehensive examples demonstrating the advanced storage features of Click-Lite Log Analytics.

## 🗜️ Compression Demo (`compression_demo.go`)

Tests the compression efficiency of different log types with ZSTD compression.

**Features Tested:**
- Small repetitive messages (high compression expected)
- Large diverse messages (lower compression expected) 
- JSON-heavy structured messages (medium compression expected)

**Run:**
```bash
go run compression_demo.go
```

**Expected Output:**
- Compression ratios for different data types
- Storage efficiency analysis
- Space savings percentage

## 📅 Partition Demo (`partition_demo.go`)

Demonstrates daily partitioning strategy by creating logs across multiple days.

**Features Tested:**
- Daily table partitioning (PARTITION BY toYYYYMMDD(timestamp))
- Multi-day data distribution
- Partition count validation

**Run:**
```bash
go run partition_demo.go
```

**Expected Output:**
- Logs distributed across 8 daily partitions
- Partition statistics
- Query performance benefits explanation

## ⏰ TTL & Cleanup Demo (`ttl_cleanup_demo.go`)

Tests the Time-To-Live mechanism and automated cleanup features.

**Features Tested:**
- Data lifecycle management (hot → cold → archive → delete)
- TTL behavior with 30-day retention
- Automated cleanup routines
- Data tiering (7 days hot, 23 days cold, 30 days delete)

**Run:**
```bash
go run ttl_cleanup_demo.go
```

**Expected Output:**
- Data accessibility across different ages
- Storage tier distribution
- Cleanup mechanism verification

## 🏁 Storage Benchmark (`storage_benchmark_demo.go`)

Comprehensive performance benchmark testing storage under various load conditions.

**Features Tested:**
- Small load: 10K logs, 5 workers
- Medium load: 50K logs, 10 workers  
- Large load: 100K logs, 20 workers
- Concurrent ingestion performance
- Storage efficiency metrics

**Run:**
```bash
go run storage_benchmark_demo.go
```

**Expected Output:**
- Throughput measurements (logs/second)
- Performance ratings
- Storage efficiency analysis
- Compression statistics

## Prerequisites

1. **Start the backend server:**
```bash
cd backend
go run .
```

2. **Ensure ClickHouse is running locally** on port 8123

## Storage Features Demonstrated

### 🗜️ Advanced Compression
- **ZSTD Level 3** compression for optimal space/speed balance
- **Per-column compression** for different data types
- **Compression ratio monitoring**

### 📅 Intelligent Partitioning  
- **Daily partitions** for efficient TTL and queries
- **Materialized columns** for query optimization
- **Partition-level operations**

### ⏰ Automated TTL Management
- **30-day default retention** with configurable policies
- **Tiered storage**: Hot (7d) → Cold (23d) → Delete (30d)
- **Automated cleanup** every 6 hours
- **Efficient partition dropping**

### 🔍 Query Optimization
- **Specialized indexes**: bloom filters, token filters, set indexes
- **Materialized columns**: date_partition, hour_partition, level_numeric
- **Query-specific optimizations**

### 📊 Monitoring & Statistics
- **Real-time storage metrics** via `/api/v1/storage/stats`
- **Compression efficiency tracking**
- **Partition health monitoring**
- **Performance analytics**

## Expected Performance

Based on the optimized storage layer:

- **Compression**: 70-85% space savings with ZSTD
- **Partitioning**: Sub-second queries on date ranges
- **TTL**: Instant cleanup via partition dropping
- **Throughput**: 50K+ logs/second sustained ingestion
- **Query Speed**: <100ms for typical time-range queries

## Troubleshooting

### Low Compression Ratios
- Check data patterns (repetitive data compresses better)
- Verify ZSTD is being used
- Consider adjusting compression level

### Slow Queries
- Ensure queries use partition key (timestamp)
- Check index usage with EXPLAIN
- Verify materialized columns are working

### TTL Issues
- Check TTL configuration in table schema
- Verify cleanup routine is running
- Monitor system logs for TTL operations

## Storage Configuration

The storage layer uses these optimized defaults:

```go
PartitionType:     "daily"
CompressionCodec:  "ZSTD"  
CompressionLevel:  3
DefaultTTL:        30 * 24 * time.Hour
HotDataTTL:        7 * 24 * time.Hour
ColdDataTTL:       23 * 24 * time.Hour
CleanupInterval:   6 * time.Hour
```

These can be customized based on your specific requirements for retention, performance, and storage costs.