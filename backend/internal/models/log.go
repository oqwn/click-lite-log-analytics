package models

import (
	"time"
)

type Log struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Service    string                 `json:"service"`
	TraceID    string                 `json:"trace_id,omitempty"`
	SpanID     string                 `json:"span_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type LogQuery struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Service   string    `json:"service,omitempty"`
	Level     string    `json:"level,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
	Search    string    `json:"search,omitempty"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

type LogFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Action  string      `json:"action,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Filters []LogFilter `json:"filters,omitempty"`
}