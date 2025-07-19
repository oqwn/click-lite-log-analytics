package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// HealthCheck returns the health status of the service
func HealthCheck(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		status := map[string]interface{}{
			"status": "ok",
			"time":   time.Now().UTC(),
		}

		// Check database health
		if err := db.Health(ctx); err != nil {
			status["status"] = "error"
			status["database"] = "unhealthy"
			status["error"] = err.Error()
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			status["database"] = "healthy"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}

// IngestLogs handles log ingestion
func IngestLogs(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var logs []models.Log
		if err := json.NewDecoder(r.Body).Decode(&logs); err != nil {
			// Try single log format
			var singleLog models.Log
			if err := json.NewDecoder(r.Body).Decode(&singleLog); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			logs = []models.Log{singleLog}
		}

		ctx := r.Context()
		successCount := 0

		for _, logEntry := range logs {
			// Set timestamp if not provided
			if logEntry.Timestamp.IsZero() {
				logEntry.Timestamp = time.Now()
			}

			// Set default level if not provided
			if logEntry.Level == "" {
				logEntry.Level = "info"
			}

			// Set default service if not provided
			if logEntry.Service == "" {
				logEntry.Service = "unknown"
			}

			if err := db.InsertLog(ctx, &logEntry); err != nil {
				log.Error().Err(err).Msg("Failed to insert log")
				continue
			}
			successCount++
		}

		response := map[string]interface{}{
			"success": successCount,
			"total":   len(logs),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// QueryLogs handles log queries
func QueryLogs(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := &models.LogQuery{
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
			Limit:     100,
		}

		// Parse query parameters
		if start := r.URL.Query().Get("start_time"); start != "" {
			if t, err := time.Parse(time.RFC3339, start); err == nil {
				query.StartTime = t
			}
		}

		if end := r.URL.Query().Get("end_time"); end != "" {
			if t, err := time.Parse(time.RFC3339, end); err == nil {
				query.EndTime = t
			}
		}

		if service := r.URL.Query().Get("service"); service != "" {
			query.Service = service
		}

		if level := r.URL.Query().Get("level"); level != "" {
			query.Level = level
		}

		if traceID := r.URL.Query().Get("trace_id"); traceID != "" {
			query.TraceID = traceID
		}

		if search := r.URL.Query().Get("search"); search != "" {
			query.Search = search
		}

		if limit := r.URL.Query().Get("limit"); limit != "" {
			if l, err := strconv.Atoi(limit); err == nil && l > 0 {
				query.Limit = l
			}
		}

		if offset := r.URL.Query().Get("offset"); offset != "" {
			if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
				query.Offset = o
			}
		}

		ctx := r.Context()
		logs, err := db.QueryLogs(ctx, query)
		if err != nil {
			log.Error().Err(err).Msg("Failed to query logs")
			http.Error(w, "Failed to query logs", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"logs":  logs,
			"count": len(logs),
			"query": query,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}