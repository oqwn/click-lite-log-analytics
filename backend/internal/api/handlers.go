package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/parsing"
	"github.com/your-username/click-lite-log-analytics/backend/internal/websocket"
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

// IngestLogs handles log ingestion with parsing support
func IngestLogs(db *database.DB) http.HandlerFunc {
	// Initialize parsing manager with parsers
	parseManager := parsing.NewManager()
	parseManager.RegisterParser(parsing.NewJSONParser())
	parseManager.RegisterParser(parsing.NewRegexParser())
	
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle both bulk and single log requests
		var requestBody struct {
			Logs    []models.Log       `json:"logs,omitempty"`
			Log     *models.Log        `json:"log,omitempty"`
			Options map[string]bool    `json:"options,omitempty"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		var logs []models.Log
		if len(requestBody.Logs) > 0 {
			logs = requestBody.Logs
		} else if requestBody.Log != nil {
			logs = []models.Log{*requestBody.Log}
		} else {
			http.Error(w, "No logs provided", http.StatusBadRequest)
			return
		}
		
		ctx := r.Context()
		successCount := 0
		parseFailures := 0
		validationFailures := 0
		
		// Check if parsing is enabled
		enableParsing := requestBody.Options["enable_parsing"]
		enableValidation := requestBody.Options["enable_validation"]

		for _, logEntry := range logs {
			processedLog := &logEntry
			
			// Apply parsing if enabled and message looks like it needs parsing
			if enableParsing && (logEntry.Message != "" && (isJSONLike(logEntry.Message) || needsRegexParsing(logEntry.Message))) {
				parseResult := parseManager.Parse(logEntry.Message)
				if parseResult.Success {
					// Use parsed log instead
					processedLog = parseResult.Log
					// Preserve original metadata
					if processedLog.Service == "unknown" && logEntry.Service != "" {
						processedLog.Service = logEntry.Service
					}
					if processedLog.TraceID == "" && logEntry.TraceID != "" {
						processedLog.TraceID = logEntry.TraceID
					}
					if processedLog.SpanID == "" && logEntry.SpanID != "" {
						processedLog.SpanID = logEntry.SpanID
					}
					// Merge attributes
					if processedLog.Attributes == nil {
						processedLog.Attributes = make(map[string]interface{})
					}
					for k, v := range logEntry.Attributes {
						if _, exists := processedLog.Attributes[k]; !exists {
							processedLog.Attributes[k] = v
						}
					}
				} else {
					parseFailures++
					log.Debug().Str("error", parseResult.Error).Msg("Failed to parse log")
					// Continue with original log
				}
			}
			
			// Set timestamp if not provided
			if processedLog.Timestamp.IsZero() {
				processedLog.Timestamp = time.Now()
			}

			// Set default level if not provided
			if processedLog.Level == "" {
				processedLog.Level = "info"
			}

			// Set default service if not provided
			if processedLog.Service == "" {
				processedLog.Service = "unknown"
			}
			
			// Validate if enabled
			if enableValidation {
				rules := parseManager.GetRules()
				if err := rules.Validate(processedLog); err != nil {
					validationFailures++
					log.Debug().Err(err).Msg("Log validation failed")
					continue // Skip invalid logs
				}
			}

			if err := db.InsertLog(ctx, processedLog); err != nil {
				log.Error().Err(err).Msg("Failed to insert log")
				continue
			}
			successCount++
		}

		response := map[string]interface{}{
			"success": successCount,
			"total":   len(logs),
		}
		
		if parseFailures > 0 {
			response["parse_failures"] = parseFailures
		}
		if validationFailures > 0 {
			response["validation_failures"] = validationFailures
		}
		
		// Add parsing stats if parsing was used
		if enableParsing {
			stats := parseManager.GetStats()
			response["parsing_stats"] = map[string]interface{}{
				"total_parsed":  stats.TotalParsed,
				"success_count": stats.SuccessCount,
				"failure_count": stats.FailureCount,
				"parser_usage":  stats.ParserUsage,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// isJSONLike checks if a string looks like JSON
func isJSONLike(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
}

// needsRegexParsing checks if a string looks like it needs regex parsing
func needsRegexParsing(s string) bool {
	// Basic heuristics for unstructured logs
	return strings.Contains(s, "[") || // Syslog or timestamp brackets
		   strings.Contains(s, " - ") || // Common separator
		   strings.Contains(s, "HTTP/") || // Web logs
		   strings.Contains(s, "INFO") || strings.Contains(s, "ERROR") || // Log levels
		   strings.Contains(s, "WARN") || strings.Contains(s, "DEBUG")
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
// StorageStats returns detailed storage statistics
func StorageStats(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := db.GetStorageStats()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get storage statistics")
			http.Error(w, "Failed to get storage statistics", http.StatusInternalServerError)
			return
		}
		
		response := map[string]interface{}{
			"storage_stats": stats,
			"timestamp":     time.Now().UTC(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// WebSocketStats returns WebSocket connection statistics
func WebSocketStats(hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := map[string]interface{}{
			"active_clients": hub.GetConnectedClients(),
			"timestamp":      time.Now(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}