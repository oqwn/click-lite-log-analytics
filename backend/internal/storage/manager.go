package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Config holds storage configuration
type Config struct {
	// Partitioning settings
	PartitionType     string        // "daily", "weekly", "monthly"
	PartitionStrategy string        // "date", "hash", "custom"
	
	// Compression settings
	CompressionCodec  string        // "LZ4", "ZSTD", "LZ4HC"
	CompressionLevel  int           // 1-22 for ZSTD, 1-12 for LZ4HC
	
	// TTL settings
	DefaultTTL        time.Duration // Default retention period
	HotDataTTL        time.Duration // Keep in fast storage
	ColdDataTTL       time.Duration // Move to slow storage
	ArchiveTTL        time.Duration // Final deletion
	
	// Cleanup settings
	CleanupInterval   time.Duration // How often to run cleanup
	BatchSize         int           // Number of partitions to clean at once
}

// DefaultConfig returns optimized default storage configuration
func DefaultConfig() *Config {
	return &Config{
		PartitionType:     "daily",
		PartitionStrategy: "date",
		CompressionCodec:  "ZSTD",
		CompressionLevel:  3,
		DefaultTTL:        30 * 24 * time.Hour,  // 30 days
		HotDataTTL:        7 * 24 * time.Hour,   // 7 days in fast storage
		ColdDataTTL:       23 * 24 * time.Hour,  // 23 days in slow storage  
		ArchiveTTL:        30 * 24 * time.Hour,  // Delete after 30 days
		CleanupInterval:   6 * time.Hour,        // Cleanup every 6 hours
		BatchSize:         10,                   // Clean 10 partitions per batch
	}
}

// Manager handles advanced storage operations
type Manager struct {
	config     *Config
	db         DatabaseInterface
	stopChan   chan struct{}
}

// DatabaseInterface defines the required database operations
type DatabaseInterface interface {
	Exec(query string) error
	Query(query string) ([]map[string]interface{}, error)
}

// NewManager creates a new storage manager
func NewManager(config *Config, db DatabaseInterface) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &Manager{
		config:   config,
		db:       db,
		stopChan: make(chan struct{}),
	}
}

// InitializeSchema creates optimized table schema with partitioning, compression, and TTL
func (m *Manager) InitializeSchema() error {
	// Drop existing table if it exists (for schema updates)
	dropQuery := `DROP TABLE IF EXISTS logs`
	if err := m.db.Exec(dropQuery); err != nil {
		log.Warn().Err(err).Msg("Failed to drop existing table")
	}
	
	// Create optimized logs table with advanced features
	query := m.buildTableSchema()
	
	if err := m.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create optimized logs table: %w", err)
	}
	
	log.Info().Str("compression", m.config.CompressionCodec).
		Str("partition", m.config.PartitionType).
		Dur("ttl", m.config.DefaultTTL).
		Msg("Optimized schema initialized")
	
	return nil
}

// buildTableSchema constructs the CREATE TABLE query with all optimizations
func (m *Manager) buildTableSchema() string {
	compressionClause := m.buildCompressionClause()
	partitionClause := m.buildPartitionClause()
	ttlClause := m.buildTTLClause()
	
	return fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS logs (
		id UUID DEFAULT generateUUIDv4(),
		timestamp DateTime64(3) CODEC(%s),
		level LowCardinality(String) CODEC(%s),
		message String CODEC(%s),
		service LowCardinality(String) CODEC(%s),
		trace_id String CODEC(%s),
		span_id String CODEC(%s),
		attributes Map(String, String) CODEC(%s),
		
		-- Materialized columns for faster queries
		date_partition Date MATERIALIZED toDate(timestamp),
		hour_partition UInt8 MATERIALIZED toHour(timestamp),
		level_numeric UInt8 MATERIALIZED multiIf(
			level = 'debug', 1,
			level = 'info', 2, 
			level = 'warn', 3,
			level = 'error', 4,
			level = 'fatal', 5,
			0
		),
		
		-- Indexes for common query patterns
		INDEX idx_service service TYPE set(1000) GRANULARITY 1,
		INDEX idx_level level TYPE set(10) GRANULARITY 1,
		INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 1,
		INDEX idx_message message TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1,
		INDEX idx_hour hour_partition TYPE set(24) GRANULARITY 1
	) ENGINE = MergeTree()
	%s
	ORDER BY (service, level_numeric, timestamp)
	%s
	SETTINGS 
		index_granularity = 8192,
		merge_with_ttl_timeout = 3600,
		merge_with_recompression_ttl_timeout = 7200,
		max_compress_block_size = 1048576
	`, 
		compressionClause, compressionClause, compressionClause, 
		compressionClause, compressionClause, compressionClause, compressionClause,
		partitionClause, ttlClause)
}

// buildCompressionClause creates the compression specification
func (m *Manager) buildCompressionClause() string {
	switch strings.ToUpper(m.config.CompressionCodec) {
	case "ZSTD":
		if m.config.CompressionLevel > 0 {
			return fmt.Sprintf("ZSTD(%d)", m.config.CompressionLevel)
		}
		return "ZSTD(3)" // Default level
	case "LZ4HC":
		if m.config.CompressionLevel > 0 {
			return fmt.Sprintf("LZ4HC(%d)", m.config.CompressionLevel)
		}
		return "LZ4HC(9)" // Default level
	case "LZ4":
		return "LZ4"
	default:
		return "ZSTD(3)" // Safe default
	}
}

// buildPartitionClause creates the partitioning specification
func (m *Manager) buildPartitionClause() string {
	switch m.config.PartitionType {
	case "daily":
		return "PARTITION BY toYYYYMMDD(timestamp)"
	case "weekly":
		return "PARTITION BY toYYYYWW(timestamp)"
	case "monthly":
		return "PARTITION BY toYYYYMM(timestamp)"
	case "hourly":
		return "PARTITION BY (toYYYYMMDD(timestamp), toHour(timestamp))"
	default:
		return "PARTITION BY toYYYYMMDD(timestamp)"
	}
}

// buildTTLClause creates the TTL specification with tiered storage
func (m *Manager) buildTTLClause() string {
	hotDays := int(m.config.HotDataTTL.Hours() / 24)
	coldDays := int(m.config.ColdDataTTL.Hours() / 24)
	archiveDays := int(m.config.ArchiveTTL.Hours() / 24)
	
	return fmt.Sprintf(`TTL 
		timestamp + INTERVAL %d DAY TO DISK 'hot',
		timestamp + INTERVAL %d DAY TO DISK 'cold',
		timestamp + INTERVAL %d DAY DELETE`,
		hotDays, coldDays, archiveDays)
}

// StartCleanupRoutine starts the automated cleanup process
func (m *Manager) StartCleanupRoutine() {
	go m.cleanupRoutine()
	log.Info().Dur("interval", m.config.CleanupInterval).Msg("Storage cleanup routine started")
}

// StopCleanupRoutine stops the cleanup process
func (m *Manager) StopCleanupRoutine() {
	close(m.stopChan)
}

// cleanupRoutine runs periodic cleanup tasks
func (m *Manager) cleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			if err := m.runCleanup(); err != nil {
				log.Error().Err(err).Msg("Cleanup routine failed")
			}
		}
	}
}

// runCleanup performs cleanup operations
func (m *Manager) runCleanup() error {
	start := time.Now()
	
	// Force merge of old partitions
	if err := m.optimizeOldPartitions(); err != nil {
		log.Error().Err(err).Msg("Failed to optimize old partitions")
	}
	
	// Clean up orphaned temporary files
	if err := m.cleanupTempFiles(); err != nil {
		log.Error().Err(err).Msg("Failed to cleanup temp files")
	}
	
	// Update statistics
	if err := m.updateTableStatistics(); err != nil {
		log.Error().Err(err).Msg("Failed to update statistics")
	}
	
	duration := time.Since(start)
	log.Info().Dur("duration", duration).Msg("Cleanup routine completed")
	
	return nil
}

// optimizeOldPartitions forces merge operations on old partitions
func (m *Manager) optimizeOldPartitions() error {
	// Get partitions older than hot data threshold
	cutoffDate := time.Now().Add(-m.config.HotDataTTL).Format("2006-01-02")
	
	query := fmt.Sprintf(`
		SELECT DISTINCT partition 
		FROM system.parts 
		WHERE table = 'logs' 
		AND database = 'click_lite'
		AND partition < '%s'
		AND active = 1
		ORDER BY partition
		LIMIT %d
	`, cutoffDate, m.config.BatchSize)
	
	results, err := m.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get old partitions: %w", err)
	}
	
	for _, row := range results {
		if partition, ok := row["partition"].(string); ok {
			optimizeQuery := fmt.Sprintf("OPTIMIZE TABLE logs PARTITION '%s' FINAL", partition)
			if err := m.db.Exec(optimizeQuery); err != nil {
				log.Error().Err(err).Str("partition", partition).Msg("Failed to optimize partition")
			} else {
				log.Debug().Str("partition", partition).Msg("Optimized partition")
			}
		}
	}
	
	return nil
}

// cleanupTempFiles removes orphaned temporary files
func (m *Manager) cleanupTempFiles() error {
	query := `
		SELECT count() as temp_files
		FROM system.parts 
		WHERE table = 'logs' 
		AND database = 'click_lite'
		AND name LIKE '%tmp%'
	`
	
	results, err := m.db.Query(query)
	if err != nil {
		return err
	}
	
	if len(results) > 0 {
		if count, ok := results[0]["temp_files"].(int64); ok && count > 0 {
			log.Warn().Int64("count", count).Msg("Found temporary files, attempting cleanup")
			
			// Force cleanup of temporary parts
			cleanupQuery := "SYSTEM FLUSH LOGS; SYSTEM RELOAD DICTIONARIES"
			return m.db.Exec(cleanupQuery)
		}
	}
	
	return nil
}

// updateTableStatistics updates table statistics for query optimization
func (m *Manager) updateTableStatistics() error {
	query := "ANALYZE TABLE logs"
	return m.db.Exec(query)
}

// GetStorageStats returns detailed storage statistics
func (m *Manager) GetStorageStats() (*StorageStats, error) {
	query := `
		SELECT 
			count() as total_rows,
			formatReadableSize(sum(data_compressed_bytes)) as compressed_size,
			formatReadableSize(sum(data_uncompressed_bytes)) as uncompressed_size,
			round(sum(data_compressed_bytes) / sum(data_uncompressed_bytes), 4) as compression_ratio,
			uniqExact(partition) as partition_count,
			min(min_date) as oldest_date,
			max(max_date) as newest_date
		FROM system.parts 
		WHERE table = 'logs' 
		AND database = 'click_lite'
		AND active = 1
	`
	
	results, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return &StorageStats{}, nil
	}
	
	row := results[0]
	stats := &StorageStats{
		TotalRows:        getInt64(row, "total_rows"),
		CompressedSize:   getString(row, "compressed_size"),
		UncompressedSize: getString(row, "uncompressed_size"),
		CompressionRatio: getFloat64(row, "compression_ratio"),
		PartitionCount:   getInt64(row, "partition_count"),
		OldestDate:       getString(row, "oldest_date"),
		NewestDate:       getString(row, "newest_date"),
	}
	
	return stats, nil
}

// StorageStats holds storage statistics
type StorageStats struct {
	TotalRows        int64   `json:"total_rows"`
	CompressedSize   string  `json:"compressed_size"`
	UncompressedSize string  `json:"uncompressed_size"`
	CompressionRatio float64 `json:"compression_ratio"`
	PartitionCount   int64   `json:"partition_count"`
	OldestDate       string  `json:"oldest_date"`
	NewestDate       string  `json:"newest_date"`
}

// Helper functions for type conversion
func getInt64(row map[string]interface{}, key string) int64 {
	if val, ok := row[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}

func getFloat64(row map[string]interface{}, key string) float64 {
	if val, ok := row[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return 0.0
}

func getString(row map[string]interface{}, key string) string {
	if val, ok := row[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}