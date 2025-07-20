package api

import (
	"encoding/json"
	"net/http"

	"github.com/your-username/click-lite-log-analytics/backend/internal/monitoring"
)

// GetMetrics returns current system metrics
func GetMetrics(collector *monitoring.MetricsCollector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics := collector.GetMetrics()
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"metrics": metrics,
			"timestamp": metrics[0].Timestamp,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// GetAlerts returns all alerts
func GetAlerts(manager *monitoring.AlertManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alerts := manager.GetAllAlerts()
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"alerts": alerts,
			"total": len(alerts),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// GetActiveAlerts returns only active alerts
func GetActiveAlerts(manager *monitoring.AlertManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alerts := manager.GetActiveAlerts()
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"alerts": alerts,
			"active_count": len(alerts),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}