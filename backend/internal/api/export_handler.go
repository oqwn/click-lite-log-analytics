package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/your-username/click-lite-log-analytics/backend/internal/export"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// ExportHandler handles data export API endpoints
type ExportHandler struct {
	exporter *export.Exporter
}

// NewExportHandler creates a new export handler
func NewExportHandler(exporter *export.Exporter) *ExportHandler {
	return &ExportHandler{
		exporter: exporter,
	}
}

// ExportLogs exports logs in the requested format
func (h *ExportHandler) ExportLogs(w http.ResponseWriter, r *http.Request) {
	var options export.ExportOptions
	
	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
		// Try to get options from query params for simple exports
		options = h.parseQueryOptions(r)
	}

	// Validate format
	if options.Format == "" {
		options.Format = export.FormatCSV // Default to CSV
	}

	// Set appropriate content type
	switch options.Format {
	case export.FormatCSV:
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=logs_%s.csv", time.Now().Format("20060102_150405")))
	case export.FormatJSON:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=logs_%s.json", time.Now().Format("20060102_150405")))
	case export.FormatExcel:
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=logs_%s.xlsx", time.Now().Format("20060102_150405")))
	default:
		http.Error(w, "Unsupported export format", http.StatusBadRequest)
		return
	}

	// Perform export
	result, err := h.exporter.Export(w, options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log export info
	w.Header().Set("X-Export-Rows", fmt.Sprintf("%d", result.RowCount))
	w.Header().Set("X-Export-Duration", result.Duration.String())
}

// parseQueryOptions parses export options from query parameters
func (h *ExportHandler) parseQueryOptions(r *http.Request) export.ExportOptions {
	options := export.ExportOptions{
		Format:         export.ExportFormat(r.URL.Query().Get("format")),
		Query:          r.URL.Query().Get("query"),
		IncludeHeaders: r.URL.Query().Get("headers") != "false",
	}

	// Parse time range
	if start := r.URL.Query().Get("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			options.StartTime = t
		}
	}
	if end := r.URL.Query().Get("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			options.EndTime = t
		}
	}

	// Parse limit
	if limit := r.URL.Query().Get("limit"); limit != "" {
		fmt.Sscanf(limit, "%d", &options.Limit)
	}

	// Parse fields
	if fields := r.URL.Query().Get("fields"); fields != "" {
		options.Fields = splitFields(fields)
	}

	// Parse filters
	if level := r.URL.Query().Get("level"); level != "" {
		options.Filters = append(options.Filters, models.LogFilter{
			Field:    "level",
			Operator: "=",
			Value:    level,
		})
	}
	if service := r.URL.Query().Get("service"); service != "" {
		options.Filters = append(options.Filters, models.LogFilter{
			Field:    "service",
			Operator: "=",
			Value:    service,
		})
	}

	return options
}

// splitFields splits comma-separated field list
func splitFields(fields string) []string {
	result := []string{}
	for _, field := range strings.Split(fields, ",") {
		field = strings.TrimSpace(field)
		if field != "" {
			result = append(result, field)
		}
	}
	return result
}

// GetExportFormats returns supported export formats
func (h *ExportHandler) GetExportFormats(w http.ResponseWriter, r *http.Request) {
	formats := []map[string]string{
		{
			"format":      string(export.FormatCSV),
			"name":        "CSV",
			"description": "Comma-separated values, compatible with Excel and other tools",
			"mime_type":   "text/csv",
			"extension":   ".csv",
		},
		{
			"format":      string(export.FormatJSON),
			"name":        "JSON",
			"description": "JavaScript Object Notation, structured data format",
			"mime_type":   "application/json",
			"extension":   ".json",
		},
		{
			"format":      string(export.FormatExcel),
			"name":        "Excel",
			"description": "Microsoft Excel format with formatting and filters",
			"mime_type":   "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"extension":   ".xlsx",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"formats": formats,
	})
}