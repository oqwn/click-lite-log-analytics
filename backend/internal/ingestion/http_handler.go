package ingestion

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/websocket"
)

// HTTPHandler handles HTTP log ingestion with batching
type HTTPHandler struct {
	batchProcessor *BatchProcessor
	wsHub          *websocket.Hub
}

// NewHTTPHandler creates a new HTTP ingestion handler
func NewHTTPHandler(batchProcessor *BatchProcessor, wsHub *websocket.Hub) *HTTPHandler {
	return &HTTPHandler{
		batchProcessor: batchProcessor,
		wsHub:          wsHub,
	}
}

// IngestLogs handles POST /api/v1/ingest/logs endpoint
func (h *HTTPHandler) IngestLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		
		// Process logs
		now := time.Now()
		for i := range logs {
			// Generate ID if not provided
			if logs[i].ID == "" {
				logs[i].ID = uuid.New().String()
			}
			
			// Set timestamp if not provided
			if logs[i].Timestamp.IsZero() {
				logs[i].Timestamp = now
			}
			
			// Set defaults
			if logs[i].Level == "" {
				logs[i].Level = "info"
			}
			if logs[i].Service == "" {
				logs[i].Service = "unknown"
			}
			
			// Broadcast to WebSocket clients
			h.wsHub.BroadcastLog(&logs[i])
		}
		
		// Add to batch processor
		h.batchProcessor.AddBatch(logs)
		
		// Return acknowledgment
		response := map[string]interface{}{
			"status":   "accepted",
			"received": len(logs),
			"message":  "Logs queued for processing",
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
	}
}

// BulkIngestLogs handles POST /api/v1/ingest/bulk endpoint for large batches
func (h *HTTPHandler) BulkIngestLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set max body size to 10MB
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		
		var request struct {
			Logs []models.Log `json:"logs"`
			Options struct {
				SkipBroadcast bool `json:"skip_broadcast"`
			} `json:"options"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		if len(request.Logs) == 0 {
			http.Error(w, "No logs provided", http.StatusBadRequest)
			return
		}
		
		// Process logs
		now := time.Now()
		for i := range request.Logs {
			// Generate ID if not provided
			if request.Logs[i].ID == "" {
				request.Logs[i].ID = uuid.New().String()
			}
			
			// Set timestamp if not provided
			if request.Logs[i].Timestamp.IsZero() {
				request.Logs[i].Timestamp = now
			}
			
			// Set defaults
			if request.Logs[i].Level == "" {
				request.Logs[i].Level = "info"
			}
			if request.Logs[i].Service == "" {
				request.Logs[i].Service = "unknown"
			}
			
			// Optionally broadcast to WebSocket clients (disabled for bulk by default)
			if !request.Options.SkipBroadcast && i < 100 { // Limit broadcasts for bulk
				h.wsHub.BroadcastLog(&request.Logs[i])
			}
		}
		
		// Add to batch processor
		h.batchProcessor.AddBatch(request.Logs)
		
		// Return acknowledgment
		response := map[string]interface{}{
			"status":   "accepted",
			"received": len(request.Logs),
			"message":  "Bulk logs queued for processing",
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
	}
}

// HealthCheck returns the health status of the ingestion service
func (h *HTTPHandler) HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status": "healthy",
			"time":   time.Now().UTC(),
			"service": "log-ingestion",
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}