package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// Exporter handles data export in various formats
type Exporter struct {
	db *database.DB
}

// ExportFormat represents supported export formats
type ExportFormat string

const (
	FormatCSV   ExportFormat = "csv"
	FormatJSON  ExportFormat = "json"
	FormatExcel ExportFormat = "xlsx"
)

// ExportOptions defines export parameters
type ExportOptions struct {
	Format      ExportFormat      `json:"format"`
	Query       string            `json:"query"`
	Filters     []models.LogFilter `json:"filters,omitempty"`
	Fields      []string          `json:"fields,omitempty"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Limit       int               `json:"limit"`
	IncludeHeaders bool           `json:"include_headers"`
}

// ExportResult contains export operation results
type ExportResult struct {
	Format     ExportFormat `json:"format"`
	RowCount   int          `json:"row_count"`
	FileSize   int64        `json:"file_size"`
	Duration   time.Duration `json:"duration"`
	FileName   string       `json:"file_name"`
}

// NewExporter creates a new exporter
func NewExporter(db *database.DB) *Exporter {
	return &Exporter{
		db: db,
	}
}

// Export exports data based on options
func (e *Exporter) Export(writer io.Writer, options ExportOptions) (*ExportResult, error) {
	start := time.Now()
	result := &ExportResult{
		Format: options.Format,
	}

	// Execute query to get data
	logs, err := e.fetchLogs(options)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	result.RowCount = len(logs)

	// Export based on format
	switch options.Format {
	case FormatCSV:
		err = e.exportCSV(writer, logs, options)
		result.FileName = fmt.Sprintf("logs_%s.csv", time.Now().Format("20060102_150405"))
	case FormatJSON:
		err = e.exportJSON(writer, logs)
		result.FileName = fmt.Sprintf("logs_%s.json", time.Now().Format("20060102_150405"))
	case FormatExcel:
		err = e.exportExcel(writer, logs, options)
		result.FileName = fmt.Sprintf("logs_%s.xlsx", time.Now().Format("20060102_150405"))
	default:
		return nil, fmt.Errorf("unsupported export format: %s", options.Format)
	}

	if err != nil {
		return nil, err
	}

	result.Duration = time.Since(start)
	return result, nil
}

// fetchLogs retrieves logs based on export options
func (e *Exporter) fetchLogs(options ExportOptions) ([]models.Log, error) {
	// Build query if not provided
	query := options.Query
	if query == "" {
		query = e.buildQuery(options)
	}

	// Execute query using the query engine
	results, err := e.db.ExecuteSQL(query)
	if err != nil {
		return nil, err
	}

	// Convert results to logs
	logs := make([]models.Log, 0, len(results))
	for _, row := range results {
		log, err := e.rowToLog(row)
		if err != nil {
			continue // Skip invalid rows
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// buildQuery builds SQL query from export options
func (e *Exporter) buildQuery(options ExportOptions) string {
	var query strings.Builder
	
	// Select fields
	if len(options.Fields) > 0 {
		query.WriteString("SELECT ")
		query.WriteString(strings.Join(options.Fields, ", "))
	} else {
		query.WriteString("SELECT *")
	}
	
	query.WriteString(" FROM logs WHERE 1=1")

	// Time range
	if !options.StartTime.IsZero() {
		query.WriteString(fmt.Sprintf(" AND timestamp >= '%s'", options.StartTime.Format(time.RFC3339)))
	}
	if !options.EndTime.IsZero() {
		query.WriteString(fmt.Sprintf(" AND timestamp <= '%s'", options.EndTime.Format(time.RFC3339)))
	}

	// Apply filters
	for _, filter := range options.Filters {
		switch filter.Operator {
		case "=":
			query.WriteString(fmt.Sprintf(" AND %s = '%s'", filter.Field, filter.Value))
		case "!=":
			query.WriteString(fmt.Sprintf(" AND %s != '%s'", filter.Field, filter.Value))
		case "contains":
			query.WriteString(fmt.Sprintf(" AND %s LIKE '%%%s%%'", filter.Field, filter.Value))
		case ">":
			query.WriteString(fmt.Sprintf(" AND %s > '%s'", filter.Field, filter.Value))
		case "<":
			query.WriteString(fmt.Sprintf(" AND %s < '%s'", filter.Field, filter.Value))
		}
	}

	// Order and limit
	query.WriteString(" ORDER BY timestamp DESC")
	if options.Limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", options.Limit))
	}

	return query.String()
}

// rowToLog converts database row to log model
func (e *Exporter) rowToLog(row map[string]interface{}) (models.Log, error) {
	log := models.Log{}

	// Parse fields using type assertions with fallbacks
	if id, ok := row["id"]; ok {
		log.ID = fmt.Sprint(id)
	}
	
	// Handle timestamp which might come as string or time.Time
	if ts, ok := row["timestamp"]; ok {
		switch v := ts.(type) {
		case time.Time:
			log.Timestamp = v
		case string:
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				log.Timestamp = parsed
			}
		}
	}
	
	if level, ok := row["level"]; ok {
		log.Level = fmt.Sprint(level)
	}
	if message, ok := row["message"]; ok {
		log.Message = fmt.Sprint(message)
	}
	if service, ok := row["service"]; ok {
		log.Service = fmt.Sprint(service)
	}
	if traceID, ok := row["trace_id"]; ok && fmt.Sprint(traceID) != "" {
		log.TraceID = fmt.Sprint(traceID)
	}
	if spanID, ok := row["span_id"]; ok && fmt.Sprint(spanID) != "" {
		log.SpanID = fmt.Sprint(spanID)
	}

	// Parse attributes
	if attrs, ok := row["attributes"]; ok {
		switch v := attrs.(type) {
		case string:
			var attributes map[string]interface{}
			if err := json.Unmarshal([]byte(v), &attributes); err == nil {
				log.Attributes = attributes
			}
		case map[string]interface{}:
			log.Attributes = v
		}
	}

	return log, nil
}

// exportCSV exports logs to CSV format
func (e *Exporter) exportCSV(writer io.Writer, logs []models.Log, options ExportOptions) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write headers
	if options.IncludeHeaders {
		headers := e.getHeaders(options.Fields)
		if err := csvWriter.Write(headers); err != nil {
			return err
		}
	}

	// Write data rows
	for _, log := range logs {
		row := e.logToCSVRow(log, options.Fields)
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// getHeaders returns CSV headers
func (e *Exporter) getHeaders(fields []string) []string {
	if len(fields) > 0 {
		return fields
	}
	return []string{"id", "timestamp", "level", "service", "message", "trace_id", "span_id", "attributes"}
}

// logToCSVRow converts log to CSV row
func (e *Exporter) logToCSVRow(log models.Log, fields []string) []string {
	row := []string{}

	// Default field order
	if len(fields) == 0 {
		fields = []string{"id", "timestamp", "level", "service", "message", "trace_id", "span_id", "attributes"}
	}

	for _, field := range fields {
		switch field {
		case "id":
			row = append(row, log.ID)
		case "timestamp":
			row = append(row, log.Timestamp.Format(time.RFC3339))
		case "level":
			row = append(row, log.Level)
		case "service":
			row = append(row, log.Service)
		case "message":
			row = append(row, log.Message)
		case "trace_id":
			row = append(row, log.TraceID)
		case "span_id":
			row = append(row, log.SpanID)
		case "attributes":
			attrs, _ := json.Marshal(log.Attributes)
			row = append(row, string(attrs))
		default:
			// Check if it's an attribute field
			if log.Attributes != nil {
				if val, ok := log.Attributes[field]; ok {
					row = append(row, fmt.Sprint(val))
				} else {
					row = append(row, "")
				}
			} else {
				row = append(row, "")
			}
		}
	}

	return row
}

// exportJSON exports logs to JSON format
func (e *Exporter) exportJSON(writer io.Writer, logs []models.Log) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(map[string]interface{}{
		"logs":      logs,
		"count":     len(logs),
		"exported":  time.Now(),
	})
}

// exportExcel exports logs to Excel format
func (e *Exporter) exportExcel(writer io.Writer, logs []models.Log, options ExportOptions) error {
	file := excelize.NewFile()
	sheet := "Logs"
	
	// Create sheet
	index, err := file.NewSheet(sheet)
	if err != nil {
		return err
	}
	file.SetActiveSheet(index)

	// Style for headers
	headerStyle, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	if err != nil {
		return err
	}

	// Write headers
	headers := e.getHeaders(options.Fields)
	for col, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+col)
		file.SetCellValue(sheet, cell, header)
		file.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Auto-fit column widths
	for col := range headers {
		colName := fmt.Sprintf("%c", 'A'+col)
		file.SetColWidth(sheet, colName, colName, 20)
	}

	// Write data
	for row, log := range logs {
		csvRow := e.logToCSVRow(log, options.Fields)
		for col, value := range csvRow {
			cell := fmt.Sprintf("%c%d", 'A'+col, row+2)
			file.SetCellValue(sheet, cell, value)
		}
	}

	// Apply filters
	if len(logs) > 0 {
		lastCol := fmt.Sprintf("%c", 'A'+len(headers)-1)
		file.AutoFilter(sheet, fmt.Sprintf("A1:%s%d", lastCol, len(logs)+1), nil)
	}

	// Write to output
	return file.Write(writer)
}

// ScheduledExport represents a scheduled export job
type ScheduledExport struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Schedule    string        `json:"schedule"` // Cron expression
	Options     ExportOptions `json:"options"`
	Destination string        `json:"destination"` // S3, email, etc.
	Enabled     bool          `json:"enabled"`
	LastRun     time.Time     `json:"last_run"`
	NextRun     time.Time     `json:"next_run"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}