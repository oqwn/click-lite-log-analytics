package parsing

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// Parser interface for different log parsing strategies
type Parser interface {
	Parse(rawLog string) (*models.Log, error)
	Name() string
	CanParse(rawLog string) bool
}

// ParsingResult contains the parsed log and any parsing metadata
type ParsingResult struct {
	Log        *models.Log `json:"log"`
	Parser     string      `json:"parser"`
	Success    bool        `json:"success"`
	Error      string      `json:"error,omitempty"`
	ParseTime  time.Time   `json:"parse_time"`
	RawMessage string      `json:"raw_message"`
}

// Manager manages multiple parsers and routing
type Manager struct {
	parsers []Parser
	rules   *RuleSet
	stats   *ParseStats
}

// ParseStats tracks parsing statistics
type ParseStats struct {
	TotalParsed    int64            `json:"total_parsed"`
	SuccessCount   int64            `json:"success_count"`
	FailureCount   int64            `json:"failure_count"`
	ParserUsage    map[string]int64 `json:"parser_usage"`
	LastParseTime  time.Time        `json:"last_parse_time"`
}

// NewManager creates a new parsing manager
func NewManager() *Manager {
	return &Manager{
		parsers: []Parser{},
		rules:   NewDefaultRuleSet(),
		stats: &ParseStats{
			ParserUsage: make(map[string]int64),
		},
	}
}

// RegisterParser adds a parser to the manager
func (m *Manager) RegisterParser(parser Parser) {
	m.parsers = append(m.parsers, parser)
	m.stats.ParserUsage[parser.Name()] = 0
	log.Info().Str("parser", parser.Name()).Msg("Parser registered")
}

// Parse attempts to parse a raw log message using available parsers
func (m *Manager) Parse(rawLog string) *ParsingResult {
	startTime := time.Now()
	
	result := &ParsingResult{
		ParseTime:  startTime,
		RawMessage: rawLog,
	}
	
	// Update stats
	m.stats.TotalParsed++
	m.stats.LastParseTime = startTime
	
	// Try each parser in order of preference
	for _, parser := range m.parsers {
		if parser.CanParse(rawLog) {
			log.Debug().Str("parser", parser.Name()).Msg("Attempting to parse with parser")
			
			parsedLog, err := parser.Parse(rawLog)
			if err != nil {
				log.Debug().Err(err).Str("parser", parser.Name()).Msg("Parser failed")
				continue
			}
			
			// Validate parsed log
			if err := m.rules.Validate(parsedLog); err != nil {
				log.Debug().Err(err).Str("parser", parser.Name()).Msg("Validation failed")
				result.Error = fmt.Sprintf("validation failed: %v", err)
				continue
			}
			
			// Apply transformation rules
			if err := m.rules.Transform(parsedLog); err != nil {
				log.Debug().Err(err).Str("parser", parser.Name()).Msg("Transformation failed")
				result.Error = fmt.Sprintf("transformation failed: %v", err)
				continue
			}
			
			// Success
			result.Log = parsedLog
			result.Parser = parser.Name()
			result.Success = true
			m.stats.SuccessCount++
			m.stats.ParserUsage[parser.Name()]++
			
			log.Debug().Str("parser", parser.Name()).Dur("duration", time.Since(startTime)).Msg("Successfully parsed log")
			return result
		}
	}
	
	// No parser could handle the log
	result.Success = false
	result.Error = "no suitable parser found"
	m.stats.FailureCount++
	
	log.Debug().Str("raw_log", rawLog).Msg("Failed to parse log with any parser")
	return result
}

// GetStats returns current parsing statistics
func (m *Manager) GetStats() *ParseStats {
	return m.stats
}

// SetRules sets custom parsing rules
func (m *Manager) SetRules(rules *RuleSet) {
	m.rules = rules
	log.Info().Msg("Custom parsing rules applied")
}

// GetRules returns current parsing rules
func (m *Manager) GetRules() *RuleSet {
	return m.rules
}

// JSONParser handles structured JSON logs
type JSONParser struct {
	name string
}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{
		name: "json",
	}
}

// Name returns the parser name
func (p *JSONParser) Name() string {
	return p.name
}

// CanParse checks if the raw log appears to be JSON
func (p *JSONParser) CanParse(rawLog string) bool {
	trimmed := strings.TrimSpace(rawLog)
	return strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")
}

// Parse parses a JSON log message
func (p *JSONParser) Parse(rawLog string) (*models.Log, error) {
	var logData map[string]interface{}
	if err := json.Unmarshal([]byte(rawLog), &logData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	
	log := &models.Log{
		Attributes: make(map[string]interface{}),
	}
	
	// Extract standard fields
	if timestamp, ok := logData["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			log.Timestamp = t
		} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", timestamp); err == nil {
			log.Timestamp = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", timestamp); err == nil {
			log.Timestamp = t
		} else {
			log.Timestamp = time.Now()
		}
	} else {
		log.Timestamp = time.Now()
	}
	
	if level, ok := logData["level"].(string); ok {
		log.Level = strings.ToLower(level)
	} else if lvl, ok := logData["lvl"].(string); ok {
		log.Level = strings.ToLower(lvl)
	} else if severity, ok := logData["severity"].(string); ok {
		log.Level = mapSeverityToLevel(severity)
	} else {
		log.Level = "info"
	}
	
	if message, ok := logData["message"].(string); ok {
		log.Message = message
	} else if msg, ok := logData["msg"].(string); ok {
		log.Message = msg
	} else if text, ok := logData["text"].(string); ok {
		log.Message = text
	} else {
		log.Message = rawLog
	}
	
	if service, ok := logData["service"].(string); ok {
		log.Service = service
	} else if app, ok := logData["app"].(string); ok {
		log.Service = app
	} else if name, ok := logData["name"].(string); ok {
		log.Service = name
	} else {
		log.Service = "unknown"
	}
	
	if traceID, ok := logData["trace_id"].(string); ok {
		log.TraceID = traceID
	} else if trace, ok := logData["trace"].(string); ok {
		log.TraceID = trace
	}
	
	if spanID, ok := logData["span_id"].(string); ok {
		log.SpanID = spanID
	} else if span, ok := logData["span"].(string); ok {
		log.SpanID = span
	}
	
	// Add remaining fields as attributes
	excludeFields := map[string]bool{
		"timestamp": true, "level": true, "lvl": true, "severity": true,
		"message": true, "msg": true, "text": true,
		"service": true, "app": true, "name": true,
		"trace_id": true, "trace": true,
		"span_id": true, "span": true,
	}
	
	for key, value := range logData {
		if !excludeFields[key] {
			log.Attributes[key] = value
		}
	}
	
	return log, nil
}

// mapSeverityToLevel maps common severity levels to standard levels
func mapSeverityToLevel(severity string) string {
	severity = strings.ToLower(severity)
	switch severity {
	case "emergency", "emerg", "panic":
		return "fatal"
	case "alert", "crit", "critical":
		return "error"
	case "err", "error":
		return "error"
	case "warning", "warn":
		return "warn"
	case "notice", "info", "information":
		return "info"
	case "debug":
		return "debug"
	default:
		return "info"
	}
}