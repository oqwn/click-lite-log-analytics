package api

import (
	"encoding/json"
	"net/http"

	"github.com/your-username/click-lite-log-analytics/backend/internal/errors"
)

// ErrorHandler handles error detection API endpoints
type ErrorHandler struct {
	errorDetector *errors.ErrorDetector
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(errorDetector *errors.ErrorDetector) *ErrorHandler {
	return &ErrorHandler{
		errorDetector: errorDetector,
	}
}

// GetErrorStats returns error statistics
func (h *ErrorHandler) GetErrorStats(w http.ResponseWriter, r *http.Request) {
	stats := h.errorDetector.GetErrorStats()
	
	// Convert to slice for easier frontend handling
	statsList := make([]*errors.ErrorStats, 0, len(stats))
	for _, stat := range stats {
		statsList = append(statsList, stat)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stats": statsList,
		"count": len(statsList),
	})
}

// GetErrorAnomalies returns detected error anomalies
func (h *ErrorHandler) GetErrorAnomalies(w http.ResponseWriter, r *http.Request) {
	anomalies := h.errorDetector.GetAnomalies()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"anomalies": anomalies,
		"count":     len(anomalies),
	})
}

// GetErrorTrends returns error trends over time
func (h *ErrorHandler) GetErrorTrends(w http.ResponseWriter, r *http.Request) {
	stats := h.errorDetector.GetErrorStats()
	
	// Group by category
	trends := make(map[string]interface{})
	categories := make(map[string][]*errors.ErrorStats)
	
	for _, stat := range stats {
		categories[stat.Category] = append(categories[stat.Category], stat)
	}
	
	// Calculate trends per category
	for category, catStats := range categories {
		var totalCount int64
		var totalRate float64
		increasing := 0
		decreasing := 0
		
		for _, stat := range catStats {
			totalCount += stat.Count
			totalRate += stat.Rate
			switch stat.Trend {
			case "increasing":
				increasing++
			case "decreasing":
				decreasing++
			}
		}
		
		trends[category] = map[string]interface{}{
			"total_errors": totalCount,
			"avg_rate":     totalRate / float64(len(catStats)),
			"patterns":     len(catStats),
			"increasing":   increasing,
			"decreasing":   decreasing,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"trends":     trends,
		"categories": len(categories),
	})
}