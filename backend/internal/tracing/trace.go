package tracing

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// TraceManager manages trace ID correlations and trace analysis
type TraceManager struct {
	mu              sync.RWMutex
	tracePatterns   []TracePattern
	traceCache      map[string]*Trace
	cacheExpiration time.Duration
}

// TracePattern defines patterns for extracting trace IDs from logs
type TracePattern struct {
	Name        string
	Pattern     *regexp.Regexp
	TraceField  string
	SpanField   string
	ParentField string
}

// Trace represents a distributed trace
type Trace struct {
	TraceID      string            `json:"trace_id"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	ServiceCount int               `json:"service_count"`
	SpanCount    int               `json:"span_count"`
	ErrorCount   int               `json:"error_count"`
	Services     map[string]bool   `json:"services"`
	Spans        []*Span           `json:"spans"`
	RootSpan     *Span             `json:"root_span"`
	LastUpdated  time.Time         `json:"last_updated"`
}

// Span represents a span within a trace
type Span struct {
	SpanID      string              `json:"span_id"`
	TraceID     string              `json:"trace_id"`
	ParentID    string              `json:"parent_id,omitempty"`
	Service     string              `json:"service"`
	Operation   string              `json:"operation"`
	StartTime   time.Time           `json:"start_time"`
	EndTime     time.Time           `json:"end_time"`
	Duration    time.Duration       `json:"duration"`
	Status      string              `json:"status"`
	Logs        []models.Log        `json:"logs"`
	Children    []*Span             `json:"children,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// NewTraceManager creates a new trace manager
func NewTraceManager() *TraceManager {
	tm := &TraceManager{
		traceCache:      make(map[string]*Trace),
		cacheExpiration: 1 * time.Hour,
		tracePatterns: []TracePattern{
			{
				Name:       "OpenTelemetry",
				Pattern:    regexp.MustCompile(`trace[_-]?id["\s:=]+([a-fA-F0-9]{32})`),
				TraceField: "trace_id",
				SpanField:  "span_id",
				ParentField: "parent_span_id",
			},
			{
				Name:       "Jaeger",
				Pattern:    regexp.MustCompile(`"?traceID"?\s*:\s*"?([a-fA-F0-9]{32})"?`),
				TraceField: "traceID",
				SpanField:  "spanID",
				ParentField: "parentSpanID",
			},
			{
				Name:       "Zipkin",
				Pattern:    regexp.MustCompile(`"?traceId"?\s*:\s*"?([a-fA-F0-9]{16,32})"?`),
				TraceField: "traceId",
				SpanField:  "id",
				ParentField: "parentId",
			},
			{
				Name:       "X-Ray",
				Pattern:    regexp.MustCompile(`Root=([0-9a-fA-F-]+);`),
				TraceField: "trace_id",
				SpanField:  "id",
				ParentField: "parent_id",
			},
		},
	}

	// Start cleanup routine
	go tm.cleanupExpiredTraces()

	return tm
}

// ExtractTraceID extracts trace ID from a log entry
func (tm *TraceManager) ExtractTraceID(log *models.Log) string {
	// First check if trace_id is already in the log
	if log.TraceID != "" {
		return log.TraceID
	}

	// Check attributes for trace ID fields
	if log.Attributes != nil {
		for _, pattern := range tm.tracePatterns {
			if traceID, ok := log.Attributes[pattern.TraceField].(string); ok && traceID != "" {
				return traceID
			}
		}
	}

	// Try to extract from message using patterns
	for _, pattern := range tm.tracePatterns {
		matches := pattern.Pattern.FindStringSubmatch(log.Message)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// ExtractSpanInfo extracts span information from a log
func (tm *TraceManager) ExtractSpanInfo(log *models.Log) (spanID, parentID string) {
	// Check if span_id is already in the log
	if log.SpanID != "" {
		spanID = log.SpanID
	}

	// Check attributes for span ID fields
	if log.Attributes != nil {
		for _, pattern := range tm.tracePatterns {
			if sid, ok := log.Attributes[pattern.SpanField].(string); ok && sid != "" {
				spanID = sid
			}
			if pid, ok := log.Attributes[pattern.ParentField].(string); ok && pid != "" {
				parentID = pid
			}
		}
	}

	return spanID, parentID
}

// ProcessLog processes a log entry for trace correlation
func (tm *TraceManager) ProcessLog(log *models.Log) {
	traceID := tm.ExtractTraceID(log)
	if traceID == "" {
		return
	}

	// Update log with trace ID
	log.TraceID = traceID

	spanID, parentID := tm.ExtractSpanInfo(log)
	if spanID != "" {
		log.SpanID = spanID
	}

	// Update trace information
	tm.mu.Lock()
	defer tm.mu.Unlock()

	trace, exists := tm.traceCache[traceID]
	if !exists {
		trace = &Trace{
			TraceID:   traceID,
			StartTime: log.Timestamp,
			EndTime:   log.Timestamp,
			Services:  make(map[string]bool),
			Spans:     make([]*Span, 0),
		}
		tm.traceCache[traceID] = trace
	}

	// Update trace metadata
	trace.LastUpdated = time.Now()
	if log.Timestamp.Before(trace.StartTime) {
		trace.StartTime = log.Timestamp
	}
	if log.Timestamp.After(trace.EndTime) {
		trace.EndTime = log.Timestamp
	}
	trace.Duration = trace.EndTime.Sub(trace.StartTime)

	// Track service
	if log.Service != "" {
		trace.Services[log.Service] = true
		trace.ServiceCount = len(trace.Services)
	}

	// Track errors
	if strings.ToLower(log.Level) == "error" || strings.ToLower(log.Level) == "fatal" {
		trace.ErrorCount++
	}

	// Create or update span
	if spanID != "" {
		span := tm.findOrCreateSpan(trace, spanID, parentID, log)
		span.Logs = append(span.Logs, *log)
		trace.SpanCount = len(trace.Spans)
	}
}

// findOrCreateSpan finds or creates a span
func (tm *TraceManager) findOrCreateSpan(trace *Trace, spanID, parentID string, log *models.Log) *Span {
	// Find existing span
	for _, span := range trace.Spans {
		if span.SpanID == spanID {
			tm.updateSpan(span, log)
			return span
		}
	}

	// Create new span
	span := &Span{
		SpanID:    spanID,
		TraceID:   trace.TraceID,
		ParentID:  parentID,
		Service:   log.Service,
		StartTime: log.Timestamp,
		EndTime:   log.Timestamp,
		Logs:      make([]models.Log, 0),
		Children:  make([]*Span, 0),
	}

	// Extract operation name from message or attributes
	if op, ok := log.Attributes["operation"].(string); ok {
		span.Operation = op
	} else if op, ok := log.Attributes["span.name"].(string); ok {
		span.Operation = op
	} else {
		span.Operation = tm.extractOperation(log.Message)
	}

	// Set status based on log level
	span.Status = tm.getSpanStatus(log.Level)
	span.Attributes = log.Attributes

	trace.Spans = append(trace.Spans, span)

	// Update root span if this is it
	if parentID == "" && trace.RootSpan == nil {
		trace.RootSpan = span
	}

	return span
}

// updateSpan updates span information
func (tm *TraceManager) updateSpan(span *Span, log *models.Log) {
	if log.Timestamp.Before(span.StartTime) {
		span.StartTime = log.Timestamp
	}
	if log.Timestamp.After(span.EndTime) {
		span.EndTime = log.Timestamp
	}
	span.Duration = span.EndTime.Sub(span.StartTime)

	// Update status if error
	if strings.ToLower(log.Level) == "error" || strings.ToLower(log.Level) == "fatal" {
		span.Status = "error"
	}
}

// extractOperation extracts operation name from log message
func (tm *TraceManager) extractOperation(message string) string {
	// Common patterns for operation names
	patterns := []struct {
		pattern *regexp.Regexp
		index   int
	}{
		{regexp.MustCompile(`(?i)(GET|POST|PUT|DELETE|PATCH)\s+([^\s]+)`), 0},
		{regexp.MustCompile(`(?i)operation[:\s]+([^\s,]+)`), 1},
		{regexp.MustCompile(`(?i)method[:\s]+([^\s,]+)`), 1},
		{regexp.MustCompile(`(?i)calling\s+([^\s]+)`), 1},
		{regexp.MustCompile(`(?i)executing\s+([^\s]+)`), 1},
	}

	for _, p := range patterns {
		matches := p.pattern.FindStringSubmatch(message)
		if len(matches) > p.index {
			return matches[p.index]
		}
	}

	// Default to first few words
	words := strings.Fields(message)
	if len(words) > 0 {
		return strings.Join(words[:min(3, len(words))], " ")
	}

	return "unknown"
}

// getSpanStatus determines span status from log level
func (tm *TraceManager) getSpanStatus(level string) string {
	switch strings.ToLower(level) {
	case "error", "fatal", "panic":
		return "error"
	case "warn", "warning":
		return "warning"
	default:
		return "ok"
	}
}

// GetTrace retrieves a trace by ID
func (tm *TraceManager) GetTrace(traceID string) (*Trace, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	trace, exists := tm.traceCache[traceID]
	if !exists {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	// Build span hierarchy
	tm.buildSpanHierarchy(trace)

	return trace, nil
}

// buildSpanHierarchy builds parent-child relationships between spans
func (tm *TraceManager) buildSpanHierarchy(trace *Trace) {
	spanMap := make(map[string]*Span)
	for _, span := range trace.Spans {
		spanMap[span.SpanID] = span
		span.Children = make([]*Span, 0) // Reset children
	}

	// Build hierarchy
	for _, span := range trace.Spans {
		if span.ParentID != "" {
			if parent, ok := spanMap[span.ParentID]; ok {
				parent.Children = append(parent.Children, span)
			}
		}
	}
}

// GetTraces returns all active traces
func (tm *TraceManager) GetTraces(limit int) []*Trace {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	traces := make([]*Trace, 0, len(tm.traceCache))
	for _, trace := range tm.traceCache {
		traces = append(traces, trace)
		if len(traces) >= limit {
			break
		}
	}

	return traces
}

// cleanupExpiredTraces removes old traces from cache
func (tm *TraceManager) cleanupExpiredTraces() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		tm.mu.Lock()
		now := time.Now()
		for traceID, trace := range tm.traceCache {
			if now.Sub(trace.LastUpdated) > tm.cacheExpiration {
				delete(tm.traceCache, traceID)
			}
		}
		tm.mu.Unlock()
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}