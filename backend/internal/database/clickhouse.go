package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/config"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/storage"
)

type DB struct {
	baseURL        string
	client         *http.Client
	storageManager *storage.Manager
}

func New(cfg config.DatabaseConfig) (*DB, error) {
	// Use HTTP connection to ClickHouse on port 8123
	port := "8123" // Always use HTTP port
	baseURL := fmt.Sprintf("http://%s:%s", cfg.Host, port)
	
	log.Info().Str("url", baseURL).Str("database", cfg.Database).Str("username", cfg.Username).Msg("Connecting to ClickHouse")
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Create ClickHouse adapter for storage manager
	adapter := storage.NewClickHouseAdapter(baseURL)
	
	// Initialize storage manager with optimized configuration
	storageConfig := storage.DefaultConfig()
	storageManager := storage.NewManager(storageConfig, adapter)
	
	db := &DB{
		baseURL:        baseURL,
		client:         client,
		storageManager: storageManager,
	}
	
	// Test connection
	ctx := context.Background()
	if err := db.ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to test ClickHouse connection: %w", err)
	}
	
	// Initialize optimized schema with partitioning, compression, and TTL
	if err := storageManager.InitializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize optimized schema: %w", err)
	}
	
	// Start automated cleanup routines
	storageManager.StartCleanupRoutine()
	
	log.Info().Msg("Connected to ClickHouse with optimized storage")
	return db, nil
}

func (db *DB) ping(ctx context.Context) error {
	query := "SELECT 1"
	resp, err := db.client.Post(db.baseURL, "text/plain", strings.NewReader(query))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ClickHouse error: %s", string(body))
	}
	
	return nil
}

func (db *DB) Close() error {
	// Stop storage manager cleanup routines
	if db.storageManager != nil {
		db.storageManager.StopCleanupRoutine()
	}
	
	// HTTP client doesn't need explicit closing
	return nil
}

func (db *DB) InitSchema() error {
	// Create logs table
	query := `
	CREATE TABLE IF NOT EXISTS logs (
		id UUID DEFAULT generateUUIDv4(),
		timestamp DateTime64(3),
		level String,
		message String,
		service String,
		trace_id String,
		span_id String,
		attributes Map(String, String),
		INDEX idx_timestamp timestamp TYPE minmax GRANULARITY 1,
		INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 1,
		INDEX idx_service service TYPE bloom_filter GRANULARITY 1,
		INDEX idx_level level TYPE set(100) GRANULARITY 1
	) ENGINE = MergeTree()
	PARTITION BY toYYYYMMDD(timestamp)
	ORDER BY (service, timestamp)
	TTL timestamp + INTERVAL 30 DAY
	SETTINGS index_granularity = 8192
	`
	
	if err := db.exec(query); err != nil {
		return fmt.Errorf("failed to create logs table: %w", err)
	}

	log.Info().Msg("Database schema initialized")
	return nil
}

func (db *DB) exec(query string) error {
	resp, err := db.client.Post(db.baseURL, "text/plain", strings.NewReader(query))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ClickHouse error: %s", string(body))
	}
	
	return nil
}

func (db *DB) InsertLog(ctx context.Context, logEntry *models.Log) error {
	// Convert attributes to JSON format for ClickHouse
	attrs := make(map[string]string)
	for k, v := range logEntry.Attributes {
		attrs[k] = fmt.Sprintf("%v", v)
	}
	
	// Build INSERT query with VALUES format
	query := fmt.Sprintf(`
		INSERT INTO logs (timestamp, level, message, service, trace_id, span_id, attributes)
		VALUES ('%s', '%s', '%s', '%s', '%s', '%s', %s)
	`,
		logEntry.Timestamp.Format("2006-01-02 15:04:05.000"),
		strings.ReplaceAll(logEntry.Level, "'", "\\'"),
		strings.ReplaceAll(logEntry.Message, "'", "\\'"),
		strings.ReplaceAll(logEntry.Service, "'", "\\'"),
		strings.ReplaceAll(logEntry.TraceID, "'", "\\'"),
		strings.ReplaceAll(logEntry.SpanID, "'", "\\'"),
		formatMapForClickHouse(attrs),
	)
	
	return db.exec(query)
}

func formatMapForClickHouse(m map[string]string) string {
	if len(m) == 0 {
		return "map()"
	}
	
	var pairs []string
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("'%s', '%s'", 
			strings.ReplaceAll(k, "'", "\\'"),
			strings.ReplaceAll(v, "'", "\\'")))
	}
	
	return fmt.Sprintf("map(%s)", strings.Join(pairs, ", "))
}

func (db *DB) QueryLogs(ctx context.Context, query *models.LogQuery) ([]models.Log, error) {
	// Build query
	q := fmt.Sprintf(`
		SELECT id, timestamp, level, message, service, trace_id, span_id, attributes
		FROM logs
		WHERE timestamp >= '%s' AND timestamp <= '%s'
	`, query.StartTime.Format("2006-01-02 15:04:05"), query.EndTime.Format("2006-01-02 15:04:05"))

	if query.Service != "" {
		q += fmt.Sprintf(" AND service = '%s'", strings.ReplaceAll(query.Service, "'", "\\'"))
	}

	if query.Level != "" {
		q += fmt.Sprintf(" AND level = '%s'", strings.ReplaceAll(query.Level, "'", "\\'"))
	}

	if query.TraceID != "" {
		q += fmt.Sprintf(" AND trace_id = '%s'", strings.ReplaceAll(query.TraceID, "'", "\\'"))
	}

	if query.Search != "" {
		q += fmt.Sprintf(" AND position(lower(message), lower('%s')) > 0", strings.ReplaceAll(query.Search, "'", "\\'"))
	}

	q += " ORDER BY timestamp DESC"
	
	if query.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", query.Limit)
		if query.Offset > 0 {
			q += fmt.Sprintf(" OFFSET %d", query.Offset)
		}
	}

	// Add FORMAT JSONEachRow for easier parsing
	q += " FORMAT JSONEachRow"

	resp, err := db.client.Post(db.baseURL, "text/plain", strings.NewReader(q))
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ClickHouse error: %s", string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var logs []models.Log
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue // Skip invalid rows
		}

		log := models.Log{
			ID:      row["id"].(string),
			Level:   row["level"].(string),
			Message: row["message"].(string),
			Service: row["service"].(string),
			TraceID: row["trace_id"].(string),
			SpanID:  row["span_id"].(string),
		}

		// Parse timestamp
		if timestampStr, ok := row["timestamp"].(string); ok {
			if timestamp, err := time.Parse("2006-01-02 15:04:05.000", timestampStr); err == nil {
				log.Timestamp = timestamp
			}
		}

		// Parse attributes
		log.Attributes = make(map[string]interface{})
		if attrs, ok := row["attributes"].(map[string]interface{}); ok {
			log.Attributes = attrs
		}

		logs = append(logs, log)
	}

	return logs, nil
}

func (db *DB) Health(ctx context.Context) error {
	return db.ping(ctx)
}

// GetStorageStats returns detailed storage statistics
func (db *DB) GetStorageStats() (*storage.StorageStats, error) {
	if db.storageManager == nil {
		return nil, fmt.Errorf("storage manager not initialized")
	}
	return db.storageManager.GetStorageStats()
}