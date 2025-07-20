package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// StorageOptimizer optimizes storage layout and performance
type StorageOptimizer struct {
	executor      QueryExecutor
	config        OptimizationConfig
	lastOptimized time.Time
}

// OptimizationConfig configures storage optimization
type OptimizationConfig struct {
	EnablePartitioning   bool
	PartitionInterval    string // hourly, daily, weekly, monthly
	EnableCompression    bool
	CompressionCodec     string // LZ4, ZSTD, LZ4HC
	EnableDeduplication  bool
	MergeTreeGranularity int
	TTLDays              int
	OptimizeInterval     time.Duration
}

// QueryExecutor interface for executing optimization queries
type QueryExecutor interface {
	Execute(ctx context.Context, query string) error
	Query(ctx context.Context, query string) ([]map[string]interface{}, error)
}

// DefaultOptimizationConfig returns default optimization settings
func DefaultOptimizationConfig() OptimizationConfig {
	return OptimizationConfig{
		EnablePartitioning:   true,
		PartitionInterval:    "daily",
		EnableCompression:    true,
		CompressionCodec:     "ZSTD",
		EnableDeduplication:  true,
		MergeTreeGranularity: 8192,
		TTLDays:              30,
		OptimizeInterval:     6 * time.Hour,
	}
}

// NewStorageOptimizer creates a new storage optimizer
func NewStorageOptimizer(executor QueryExecutor, config OptimizationConfig) *StorageOptimizer {
	return &StorageOptimizer{
		executor: executor,
		config:   config,
	}
}

// OptimizeSchema creates optimized table schema
func (so *StorageOptimizer) OptimizeSchema(ctx context.Context) error {
	// Drop existing table if needed
	dropQuery := "DROP TABLE IF EXISTS logs_optimized"
	if err := so.executor.Execute(ctx, dropQuery); err != nil {
		return fmt.Errorf("failed to drop existing table: %w", err)
	}
	
	// Build optimized schema
	schema := so.buildOptimizedSchema()
	
	// Create new table
	if err := so.executor.Execute(ctx, schema); err != nil {
		return fmt.Errorf("failed to create optimized table: %w", err)
	}
	
	log.Info().Msg("Optimized storage schema created")
	return nil
}

// buildOptimizedSchema builds the optimized table schema
func (so *StorageOptimizer) buildOptimizedSchema() string {
	partitionKey := so.getPartitionKey()
	orderBy := "(service, toStartOfHour(timestamp), level)"
	codec := so.getCompressionCodec()
	
	schema := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS logs_optimized (
    id UUID DEFAULT generateUUIDv4(),
    timestamp DateTime64(3) CODEC(%s),
    level LowCardinality(String) CODEC(%s),
    message String CODEC(%s),
    service LowCardinality(String) CODEC(%s),
    trace_id String CODEC(%s),
    span_id String CODEC(%s),
    attributes Map(String, String) CODEC(%s),
    
    -- Materialized columns for better performance
    hour DateTime MATERIALIZED toStartOfHour(timestamp),
    day Date MATERIALIZED toDate(timestamp),
    
    -- Indexes for common queries
    INDEX idx_timestamp timestamp TYPE minmax GRANULARITY 1,
    INDEX idx_hour hour TYPE minmax GRANULARITY 1,
    INDEX idx_service service TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_level level TYPE set(100) GRANULARITY 1,
    INDEX idx_trace_id trace_id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_message message TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
) ENGINE = MergeTree()
PARTITION BY %s
ORDER BY %s
PRIMARY KEY (service, hour)
TTL timestamp + INTERVAL %d DAY DELETE
SETTINGS 
    index_granularity = %d,
    enable_mixed_granularity_parts = 1,
    min_rows_for_compact_part = 100000,
    min_bytes_for_compact_part = 10485760,
    merge_with_ttl_timeout = 3600,
    ttl_only_drop_parts = 1`,
		codec, // timestamp
		codec, // level
		codec, // message
		codec, // service
		codec, // trace_id
		codec, // span_id
		codec, // attributes
		partitionKey,
		orderBy,
		so.config.TTLDays,
		so.config.MergeTreeGranularity,
	)
	
	return schema
}

// getPartitionKey returns partition key based on interval
func (so *StorageOptimizer) getPartitionKey() string {
	switch so.config.PartitionInterval {
	case "hourly":
		return "toYYYYMMDDHH(timestamp)"
	case "weekly":
		return "toYearWeek(timestamp)"
	case "monthly":
		return "toYYYYMM(timestamp)"
	default: // daily
		return "toYYYYMMDD(timestamp)"
	}
}

// getCompressionCodec returns compression codec
func (so *StorageOptimizer) getCompressionCodec() string {
	switch so.config.CompressionCodec {
	case "LZ4":
		return "LZ4"
	case "LZ4HC":
		return "LZ4HC(9)"
	case "ZSTD":
		return "ZSTD(3)"
	default:
		return "LZ4"
	}
}

// OptimizePartitions optimizes table partitions
func (so *StorageOptimizer) OptimizePartitions(ctx context.Context, tableName string) error {
	// Get partition information
	partitions, err := so.getPartitions(ctx, tableName)
	if err != nil {
		return err
	}
	
	// Optimize each partition
	for _, partition := range partitions {
		// Skip recent partitions
		if partition.Age < 24*time.Hour {
			continue
		}
		
		// Optimize partition
		query := fmt.Sprintf("OPTIMIZE TABLE %s PARTITION '%s' FINAL", tableName, partition.Name)
		if err := so.executor.Execute(ctx, query); err != nil {
			log.Error().Err(err).Str("partition", partition.Name).Msg("Failed to optimize partition")
			continue
		}
		
		log.Info().Str("partition", partition.Name).Msg("Optimized partition")
	}
	
	return nil
}

// PartitionInfo contains partition information
type PartitionInfo struct {
	Name      string
	Rows      int64
	Bytes     int64
	Age       time.Duration
	DataParts int
}

// getPartitions retrieves partition information
func (so *StorageOptimizer) getPartitions(ctx context.Context, tableName string) ([]PartitionInfo, error) {
	query := fmt.Sprintf(`
SELECT 
    partition,
    sum(rows) as rows,
    sum(bytes_on_disk) as bytes,
    count() as parts,
    max(modification_time) as last_modified
FROM system.parts
WHERE table = '%s' AND active
GROUP BY partition
ORDER BY partition DESC`, tableName)
	
	results, err := so.executor.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	
	partitions := []PartitionInfo{}
	for _, row := range results {
		partition := PartitionInfo{
			Name:      fmt.Sprint(row["partition"]),
			Rows:      row["rows"].(int64),
			Bytes:     row["bytes"].(int64),
			DataParts: row["parts"].(int),
		}
		
		if lastMod, ok := row["last_modified"].(time.Time); ok {
			partition.Age = time.Since(lastMod)
		}
		
		partitions = append(partitions, partition)
	}
	
	return partitions, nil
}

// CreateMaterializedViews creates materialized views for common queries
func (so *StorageOptimizer) CreateMaterializedViews(ctx context.Context) error {
	views := []struct {
		name  string
		query string
	}{
		{
			name: "logs_by_service_hourly",
			query: `
CREATE MATERIALIZED VIEW IF NOT EXISTS logs_by_service_hourly
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMMDD(hour)
ORDER BY (service, hour, level)
AS SELECT
    toStartOfHour(timestamp) as hour,
    service,
    level,
    count() as count,
    uniqExact(trace_id) as unique_traces
FROM logs
GROUP BY hour, service, level`,
		},
		{
			name: "logs_errors_daily",
			query: `
CREATE MATERIALIZED VIEW IF NOT EXISTS logs_errors_daily
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(day)
ORDER BY (day, service)
AS SELECT
    toDate(timestamp) as day,
    service,
    countIf(level IN ('error', 'fatal')) as error_count,
    countIf(level = 'warn') as warn_count,
    count() as total_count
FROM logs
GROUP BY day, service`,
		},
		{
			name: "logs_trace_summary",
			query: `
CREATE MATERIALIZED VIEW IF NOT EXISTS logs_trace_summary
ENGINE = ReplacingMergeTree()
ORDER BY trace_id
AS SELECT
    trace_id,
    min(timestamp) as start_time,
    max(timestamp) as end_time,
    count() as span_count,
    uniqExact(service) as service_count,
    countIf(level IN ('error', 'fatal')) as error_count,
    groupArray(service) as services
FROM logs
WHERE trace_id != ''
GROUP BY trace_id`,
		},
	}
	
	for _, view := range views {
		if err := so.executor.Execute(ctx, view.query); err != nil {
			log.Error().Err(err).Str("view", view.name).Msg("Failed to create materialized view")
			continue
		}
		log.Info().Str("view", view.name).Msg("Created materialized view")
	}
	
	return nil
}

// AnalyzeStorageUsage analyzes storage usage and provides recommendations
func (so *StorageOptimizer) AnalyzeStorageUsage(ctx context.Context, tableName string) (*StorageAnalysis, error) {
	analysis := &StorageAnalysis{
		TableName:        tableName,
		AnalysisTime:     time.Now(),
		Recommendations:  []string{},
	}
	
	// Get table size
	sizeQuery := fmt.Sprintf(`
SELECT 
    sum(rows) as total_rows,
    sum(bytes_on_disk) as total_bytes,
    sum(data_compressed_bytes) as compressed_bytes,
    sum(data_uncompressed_bytes) as uncompressed_bytes,
    count() as total_parts,
    avg(rows) as avg_rows_per_part
FROM system.parts
WHERE table = '%s' AND active`, tableName)
	
	results, err := so.executor.Query(ctx, sizeQuery)
	if err != nil {
		return nil, err
	}
	
	if len(results) > 0 {
		row := results[0]
		analysis.TotalRows = row["total_rows"].(int64)
		analysis.TotalBytes = row["total_bytes"].(int64)
		analysis.CompressedBytes = row["compressed_bytes"].(int64)
		analysis.UncompressedBytes = row["uncompressed_bytes"].(int64)
		analysis.TotalParts = row["total_parts"].(int)
		
		// Calculate compression ratio
		if analysis.UncompressedBytes > 0 {
			analysis.CompressionRatio = float64(analysis.UncompressedBytes) / float64(analysis.CompressedBytes)
		}
	}
	
	// Generate recommendations
	so.generateRecommendations(analysis)
	
	return analysis, nil
}

// StorageAnalysis contains storage analysis results
type StorageAnalysis struct {
	TableName         string
	AnalysisTime      time.Time
	TotalRows         int64
	TotalBytes        int64
	CompressedBytes   int64
	UncompressedBytes int64
	CompressionRatio  float64
	TotalParts        int
	Recommendations   []string
}

// generateRecommendations generates optimization recommendations
func (so *StorageOptimizer) generateRecommendations(analysis *StorageAnalysis) {
	// Check compression ratio
	if analysis.CompressionRatio < 2.0 {
		analysis.Recommendations = append(analysis.Recommendations,
			"Consider using ZSTD compression for better compression ratio")
	}
	
	// Check part count
	avgBytesPerPart := analysis.TotalBytes / int64(analysis.TotalParts)
	if avgBytesPerPart < 100*1024*1024 { // Less than 100MB per part
		analysis.Recommendations = append(analysis.Recommendations,
			"Too many small parts, consider adjusting merge settings")
	}
	
	// Check table size
	if analysis.TotalBytes > 1024*1024*1024*1024 { // Over 1TB
		analysis.Recommendations = append(analysis.Recommendations,
			"Large table size, consider implementing data archival strategy")
	}
}

// RunPeriodicOptimization runs periodic storage optimization
func (so *StorageOptimizer) RunPeriodicOptimization(ctx context.Context) {
	ticker := time.NewTicker(so.config.OptimizeInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if err := so.OptimizePartitions(ctx, "logs"); err != nil {
				log.Error().Err(err).Msg("Failed to optimize partitions")
			}
			so.lastOptimized = time.Now()
		case <-ctx.Done():
			return
		}
	}
}