package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/your-username/click-lite-log-analytics/backend/internal/tracing"
)

// TraceHandler handles trace-related API endpoints
type TraceHandler struct {
	traceManager *tracing.TraceManager
}

// NewTraceHandler creates a new trace handler
func NewTraceHandler(traceManager *tracing.TraceManager) *TraceHandler {
	return &TraceHandler{
		traceManager: traceManager,
	}
}

// GetTrace retrieves a specific trace by ID
func (h *TraceHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceID")
	if traceID == "" {
		http.Error(w, "Trace ID is required", http.StatusBadRequest)
		return
	}

	trace, err := h.traceManager.GetTrace(traceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}

// GetTraces retrieves all active traces
func (h *TraceHandler) GetTraces(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	traces := h.traceManager.GetTraces(limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"traces": traces,
		"count":  len(traces),
	})
}

// GetTraceTimeline retrieves trace timeline visualization data
func (h *TraceHandler) GetTraceTimeline(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceID")
	if traceID == "" {
		http.Error(w, "Trace ID is required", http.StatusBadRequest)
		return
	}

	trace, err := h.traceManager.GetTrace(traceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Convert to timeline format
	timeline := h.buildTimeline(trace)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeline)
}

// buildTimeline builds timeline visualization data
func (h *TraceHandler) buildTimeline(trace *tracing.Trace) map[string]interface{} {
	events := []map[string]interface{}{}

	// Add spans as timeline events
	for _, span := range trace.Spans {
		events = append(events, map[string]interface{}{
			"id":        span.SpanID,
			"parent_id": span.ParentID,
			"service":   span.Service,
			"operation": span.Operation,
			"start":     span.StartTime.UnixMilli(),
			"end":       span.EndTime.UnixMilli(),
			"duration":  span.Duration.Milliseconds(),
			"status":    span.Status,
			"logs":      len(span.Logs),
		})
	}

	return map[string]interface{}{
		"trace_id":      trace.TraceID,
		"start_time":    trace.StartTime.UnixMilli(),
		"end_time":      trace.EndTime.UnixMilli(),
		"duration":      trace.Duration.Milliseconds(),
		"service_count": trace.ServiceCount,
		"span_count":    trace.SpanCount,
		"error_count":   trace.ErrorCount,
		"events":        events,
	}
}