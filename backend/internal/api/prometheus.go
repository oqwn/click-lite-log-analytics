package api

import (
	"net/http"
	"github.com/your-username/click-lite-log-analytics/backend/internal/monitoring"
)

// PrometheusMetrics returns metrics in Prometheus exposition format
func PrometheusMetrics(exporter *monitoring.PrometheusExporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set Prometheus content type
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		
		// Export metrics
		if err := exporter.Export(w); err != nil {
			http.Error(w, "Failed to export metrics", http.StatusInternalServerError)
			return
		}
	}
}