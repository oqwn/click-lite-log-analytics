package database

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/config"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

type DB struct {
	conn driver.Conn
}

func New(cfg config.DatabaseConfig) (*DB, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	db := &DB{conn: conn}
	
	// Initialize schema
	if err := db.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Info().Msg("Connected to ClickHouse")
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) InitSchema() error {
	ctx := context.Background()
	
	// Create database if not exists
	query := fmt.Sprintf(`CREATE DATABASE IF NOT EXISTS %s`, db.conn.Stats().Addr[0])
	if err := db.conn.Exec(ctx, query); err != nil {
		log.Debug().Err(err).Msg("Database might already exist")
	}

	// Create logs table
	query = `
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
	
	if err := db.conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create logs table: %w", err)
	}

	log.Info().Msg("Database schema initialized")
	return nil
}

func (db *DB) InsertLog(ctx context.Context, log *models.Log) error {
	query := `
		INSERT INTO logs (timestamp, level, message, service, trace_id, span_id, attributes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	// Convert attributes to map[string]string for ClickHouse
	attrs := make(map[string]string)
	for k, v := range log.Attributes {
		attrs[k] = fmt.Sprintf("%v", v)
	}
	
	return db.conn.Exec(ctx, query, 
		log.Timestamp, 
		log.Level, 
		log.Message, 
		log.Service, 
		log.TraceID, 
		log.SpanID, 
		attrs,
	)
}

func (db *DB) QueryLogs(ctx context.Context, query *models.LogQuery) ([]models.Log, error) {
	// Build query
	q := `
		SELECT id, timestamp, level, message, service, trace_id, span_id, attributes
		FROM logs
		WHERE timestamp >= ? AND timestamp <= ?
	`
	args := []interface{}{query.StartTime, query.EndTime}

	if query.Service != "" {
		q += " AND service = ?"
		args = append(args, query.Service)
	}

	if query.Level != "" {
		q += " AND level = ?"
		args = append(args, query.Level)
	}

	if query.TraceID != "" {
		q += " AND trace_id = ?"
		args = append(args, query.TraceID)
	}

	if query.Search != "" {
		q += " AND position(lower(message), lower(?)) > 0"
		args = append(args, query.Search)
	}

	q += " ORDER BY timestamp DESC"
	
	if query.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", query.Limit)
		if query.Offset > 0 {
			q += fmt.Sprintf(" OFFSET %d", query.Offset)
		}
	}

	rows, err := db.conn.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []models.Log
	for rows.Next() {
		var log models.Log
		var attrs map[string]string
		
		if err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Level,
			&log.Message,
			&log.Service,
			&log.TraceID,
			&log.SpanID,
			&attrs,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert string map to interface{} map
		log.Attributes = make(map[string]interface{})
		for k, v := range attrs {
			log.Attributes[k] = v
		}

		logs = append(logs, log)
	}

	return logs, nil
}

func (db *DB) Health(ctx context.Context) error {
	return db.conn.Ping(ctx)
}