package ingestion

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/monitoring"
	"github.com/your-username/click-lite-log-analytics/backend/internal/websocket"
)

// HTTPHandlerWithMetrics handles HTTP log ingestion with batching and metrics
type HTTPHandlerWithMetrics struct {
	batchProcessor *BatchProcessor
	wsHub          *websocket.Hub
	metrics        *monitoring.MetricsCollector
}

// NewHTTPHandlerWithMetrics creates a new HTTP ingestion handler with metrics
func NewHTTPHandlerWithMetrics(batchProcessor *BatchProcessor, wsHub *websocket.Hub, metrics *monitoring.MetricsCollector) *HTTPHandlerWithMetrics {
	return &HTTPHandlerWithMetrics{
		batchProcessor: batchProcessor,
		wsHub:          wsHub,
		metrics:        metrics,
	}
}

// IngestLogs handles POST /api/v1/ingest/logs endpoint
func (h *HTTPHandlerWithMetrics) IngestLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var logs []models.Log
		
		// Read body into bytes first
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		
		// Try to decode as array first
		if err := json.Unmarshal(body, &logs); err != nil {
			// Try single log format
			var singleLog models.Log
			if err2 := json.Unmarshal(body, &singleLog); err2 != nil {
				log.Error().Err(err).Err(err2).Str("body", string(body)).Msg("Failed to parse log request")
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			logs = []models.Log{singleLog}
		}
		
		// Set timestamps and IDs
		now := time.Now()
		for i := range logs {
			if logs[i].ID == "" {
				logs[i].ID = uuid.New().String()
			}
			if logs[i].Timestamp.IsZero() {
				logs[i].Timestamp = now
			}
		}
		
		// Add logs to batch processor
		for _, log := range logs {
			h.batchProcessor.Add(log)
		}
		
		// Broadcast logs via WebSocket
		for i := range logs {
			h.wsHub.BroadcastLog(&logs[i])
		}
		
		// Record metrics
		h.metrics.RecordIngestion(len(logs))
		h.metrics.RecordHistogram("ingestion_request_duration_ms", float64(time.Since(start).Milliseconds()))
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "accepted",
			"count":  len(logs),
		})
	}
}

// BulkIngestLogs handles POST /api/v1/ingest/bulk endpoint for large batches
func (h *HTTPHandlerWithMetrics) BulkIngestLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Check content length
		if r.ContentLength > 10*1024*1024 { // 10MB limit
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		
		var logs []models.Log
		decoder := json.NewDecoder(r.Body)
		
		if err := decoder.Decode(&logs); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// Set timestamps and IDs
		now := time.Now()
		for i := range logs {
			if logs[i].ID == "" {
				logs[i].ID = uuid.New().String()
			}
			if logs[i].Timestamp.IsZero() {
				logs[i].Timestamp = now
			}
		}
		
		// Add logs to batch processor
		for _, log := range logs {
			h.batchProcessor.Add(log)
		}
		
		// For bulk ingestion, only broadcast a summary to avoid overwhelming WebSocket
		if len(logs) > 0 {
			summaryLog := models.Log{
				ID:        uuid.New().String(),
				Timestamp: now,
				Level:     "info",
				Message:   "Bulk ingestion",
				Service:   "ingestion",
				Attributes: map[string]interface{}{
					"count": len(logs),
					"type":  "bulk_ingestion",
				},
			}
			h.wsHub.BroadcastLog(&summaryLog)
		}
		
		// Record metrics
		h.metrics.RecordIngestion(len(logs))
		h.metrics.RecordHistogram("bulk_ingestion_duration_ms", float64(time.Since(start).Milliseconds()))
		h.metrics.RecordHistogram("bulk_ingestion_size", float64(len(logs)))
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "accepted",
			"count":  len(logs),
		})
	}
}

// HealthCheck returns the health status of the ingestion service
func (h *HTTPHandlerWithMetrics) HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"service": "ingestion",
		})
	}
}