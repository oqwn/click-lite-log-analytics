package websocket

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/query"
)

// LogTailer continuously polls for new logs and broadcasts them
type LogTailer struct {
	db          *database.DB
	hub         *Hub
	pollInterval time.Duration
	batchSize    int
}

// NewLogTailer creates a new log tailer
func NewLogTailer(db *database.DB, hub *Hub) *LogTailer {
	return &LogTailer{
		db:           db,
		hub:          hub,
		pollInterval: 1 * time.Second, // Poll every second
		batchSize:    100,              // Fetch up to 100 logs per poll
	}
}

// Start begins tailing logs
func (lt *LogTailer) Start(ctx context.Context) {
	ticker := time.NewTicker(lt.pollInterval)
	defer ticker.Stop()

	// Track the last seen timestamp to avoid duplicates
	lastTimestamp := time.Now().Add(-5 * time.Second) // Start from 5 seconds ago

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Log tailer stopping")
			return
		case <-ticker.C:
			// Only fetch logs if there are active clients
			if lt.hub.GetConnectedClients() == 0 {
				continue
			}

			// Fetch new logs
			logs, err := lt.fetchNewLogs(ctx, lastTimestamp)
			if err != nil {
				log.Error().Err(err).Msg("Failed to fetch new logs")
				continue
			}

			// Broadcast logs to clients
			for _, logEntry := range logs {
				lt.hub.BroadcastToClients(logEntry)
				
				// Update last timestamp
				if logEntry.Timestamp.After(lastTimestamp) {
					lastTimestamp = logEntry.Timestamp
				}
			}

			if len(logs) > 0 {
				log.Debug().
					Int("count", len(logs)).
					Time("last_timestamp", lastTimestamp).
					Msg("Broadcasted new logs")
			}
		}
	}
}

// fetchNewLogs fetches logs newer than the given timestamp using the query engine
func (lt *LogTailer) fetchNewLogs(ctx context.Context, since time.Time) ([]*models.Log, error) {
	// Create query request
	queryText := fmt.Sprintf(`
		SELECT 
			toString(timestamp) as id,
			timestamp,
			level,
			service,
			message,
			trace_id
		FROM logs
		WHERE timestamp > '%s'
		ORDER BY timestamp ASC
		LIMIT %d
	`, since.Format("2006-01-02 15:04:05.999999"), lt.batchSize)

	// Get query engine and execute query
	queryEngine := lt.db.GetQueryEngine()
	if queryEngine == nil {
		return nil, fmt.Errorf("query engine not available")
	}

	req := &query.QueryRequest{
		Query:   queryText,
		Timeout: 10, // 10 second timeout
	}

	response, err := queryEngine.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("query error: %s", response.Error)
	}

	// Convert response rows to models.Log
	var logs []*models.Log
	for _, row := range response.Rows {
		entry := &models.Log{}

		if id, ok := row["id"].(string); ok {
			entry.ID = id
		}
		
		if timestampStr, ok := row["timestamp"].(string); ok {
			if ts, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
				entry.Timestamp = ts
			}
		}
		
		if level, ok := row["level"].(string); ok {
			entry.Level = level
		}
		
		if service, ok := row["service"].(string); ok {
			entry.Service = service
		}
		
		if message, ok := row["message"].(string); ok {
			entry.Message = message
		}
		
		if traceID, ok := row["trace_id"].(string); ok {
			entry.TraceID = traceID
		}

		// Set empty attributes if none exist
		if entry.Attributes == nil {
			entry.Attributes = make(map[string]interface{})
		}

		logs = append(logs, entry)
	}

	return logs, nil
}

// SetPollInterval updates the polling interval
func (lt *LogTailer) SetPollInterval(interval time.Duration) {
	lt.pollInterval = interval
}

// SetBatchSize updates the batch size
func (lt *LogTailer) SetBatchSize(size int) {
	lt.batchSize = size
}