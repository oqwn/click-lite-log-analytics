package parsing

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// RegexParser handles unstructured logs using configurable regex patterns
type RegexParser struct {
	name     string
	patterns []*RegexPattern
}

// RegexPattern defines a regex pattern with field mappings
type RegexPattern struct {
	Name        string            `json:"name"`
	Pattern     *regexp.Regexp    `json:"-"`
	PatternStr  string            `json:"pattern"`
	FieldMap    map[string]string `json:"field_map"`
	Priority    int               `json:"priority"`
	Description string            `json:"description"`
}

// NewRegexParser creates a new regex parser with default patterns
func NewRegexParser() *RegexParser {
	parser := &RegexParser{
		name:     "regex",
		patterns: []*RegexPattern{},
	}
	
	// Add default patterns
	parser.AddDefaultPatterns()
	return parser
}

// Name returns the parser name
func (p *RegexParser) Name() string {
	return p.name
}

// CanParse checks if any regex pattern matches the log
func (p *RegexParser) CanParse(rawLog string) bool {
	for _, pattern := range p.patterns {
		if pattern.Pattern.MatchString(rawLog) {
			return true
		}
	}
	return false
}

// Parse parses a log using the first matching regex pattern
func (p *RegexParser) Parse(rawLog string) (*models.Log, error) {
	// Try patterns in priority order
	for _, pattern := range p.patterns {
		if matches := pattern.Pattern.FindStringSubmatch(rawLog); matches != nil {
			return p.parseWithPattern(rawLog, pattern, matches)
		}
	}
	
	return nil, fmt.Errorf("no regex pattern matched the log")
}

// parseWithPattern parses a log using a specific pattern and its matches
func (p *RegexParser) parseWithPattern(rawLog string, pattern *RegexPattern, matches []string) (*models.Log, error) {
	log := &models.Log{
		Attributes: make(map[string]interface{}),
	}
	
	// Set default values
	log.Timestamp = time.Now()
	log.Level = "info"
	log.Message = rawLog
	log.Service = "unknown"
	
	// Map captured groups to log fields
	subexpNames := pattern.Pattern.SubexpNames()
	
	for i, match := range matches {
		if i == 0 || match == "" {
			continue // Skip full match and empty matches
		}
		
		fieldName := subexpNames[i]
		if fieldName == "" {
			continue
		}
		
		// Map to standard fields or attributes
		switch fieldName {
		case "timestamp", "time", "date":
			if t, err := parseTimestamp(match); err == nil {
				log.Timestamp = t
			}
		case "level", "severity", "priority":
			log.Level = mapSeverityToLevel(match)
		case "message", "msg", "text":
			log.Message = strings.TrimSpace(match)
		case "service", "app", "component", "logger":
			log.Service = strings.TrimSpace(match)
		case "trace_id", "traceid", "trace":
			log.TraceID = strings.TrimSpace(match)
		case "span_id", "spanid", "span":
			log.SpanID = strings.TrimSpace(match)
		default:
			// Add as attribute
			log.Attributes[fieldName] = strings.TrimSpace(match)
		}
	}
	
	// Apply field mappings if configured
	if pattern.FieldMap != nil {
		for captured, target := range pattern.FieldMap {
			if value, exists := log.Attributes[captured]; exists {
				switch target {
				case "level":
					log.Level = mapSeverityToLevel(fmt.Sprintf("%v", value))
				case "service":
					log.Service = fmt.Sprintf("%v", value)
				case "trace_id":
					log.TraceID = fmt.Sprintf("%v", value)
				case "span_id":
					log.SpanID = fmt.Sprintf("%v", value)
				default:
					log.Attributes[target] = value
				}
				delete(log.Attributes, captured)
			}
		}
	}
	
	// Add parsing metadata
	log.Attributes["_parser"] = "regex"
	log.Attributes["_pattern"] = pattern.Name
	
	return log, nil
}

// AddPattern adds a custom regex pattern
func (p *RegexParser) AddPattern(pattern *RegexPattern) error {
	compiled, err := regexp.Compile(pattern.PatternStr)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	pattern.Pattern = compiled
	p.patterns = append(p.patterns, pattern)
	
	// Sort by priority (higher priority first)
	p.sortPatterns()
	
	return nil
}

// sortPatterns sorts patterns by priority (descending)
func (p *RegexParser) sortPatterns() {
	for i := 0; i < len(p.patterns)-1; i++ {
		for j := 0; j < len(p.patterns)-i-1; j++ {
			if p.patterns[j].Priority < p.patterns[j+1].Priority {
				p.patterns[j], p.patterns[j+1] = p.patterns[j+1], p.patterns[j]
			}
		}
	}
}

// AddDefaultPatterns adds common log format patterns
func (p *RegexParser) AddDefaultPatterns() {
	patterns := []*RegexPattern{
		// Apache Combined Log Format
		{
			Name:        "apache_combined",
			PatternStr:  `^(?P<ip>\S+) \S+ \S+ \[(?P<timestamp>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) (?P<protocol>\S+)" (?P<status>\d+) (?P<size>\d+|-) "(?P<referer>[^"]*)" "(?P<user_agent>[^"]*)"`,
			Priority:    90,
			Description: "Apache Combined Log Format",
		},
		
		// Apache Common Log Format
		{
			Name:        "apache_common",
			PatternStr:  `^(?P<ip>\S+) \S+ \S+ \[(?P<timestamp>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) (?P<protocol>\S+)" (?P<status>\d+) (?P<size>\d+|-)`,
			Priority:    85,
			Description: "Apache Common Log Format",
		},
		
		// Nginx Access Log
		{
			Name:        "nginx_access",
			PatternStr:  `^(?P<ip>\S+) - (?P<user>\S+) \[(?P<timestamp>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) (?P<protocol>\S+)" (?P<status>\d+) (?P<size>\d+) "(?P<referer>[^"]*)" "(?P<user_agent>[^"]*)"`,
			Priority:    88,
			Description: "Nginx Access Log Format",
		},
		
		// Syslog RFC3164
		{
			Name:        "syslog_rfc3164",
			PatternStr:  `^<(?P<priority>\d+)>(?P<timestamp>\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}) (?P<hostname>\S+) (?P<service>\S+)(\[(?P<pid>\d+)\])?: (?P<message>.*)`,
			Priority:    95,
			Description: "Syslog RFC3164 Format",
		},
		
		// Common application log format
		{
			Name:        "app_log",
			PatternStr:  `^(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}[.,]\d{3})\s+(?P<level>\w+)\s+(?P<service>\S+)\s+-\s+(?P<message>.*)`,
			Priority:    80,
			Description: "Common Application Log Format",
		},
		
		// Java/Spring Boot log format
		{
			Name:        "spring_boot",
			PatternStr:  `^(?P<timestamp>\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}[.,]\d{3})\s+(?P<level>\w+)\s+(?P<pid>\d+)\s+---\s+\[(?P<thread>[^\]]+)\]\s+(?P<logger>\S+)\s+:\s+(?P<message>.*)`,
			Priority:    85,
			Description: "Spring Boot Log Format",
		},
		
		// Docker container logs
		{
			Name:        "docker_log",
			PatternStr:  `^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s+(?P<stream>stdout|stderr)\s+F\s+(?P<message>.*)`,
			Priority:    75,
			Description: "Docker Container Log Format",
		},
		
		// Generic timestamp + level + message
		{
			Name:        "generic_timestamped",
			PatternStr:  `^(?P<timestamp>\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?)\s*(?P<level>DEBUG|INFO|WARN|ERROR|FATAL|TRACE)?\s*(?P<message>.+)`,
			Priority:    60,
			Description: "Generic Timestamped Log",
		},
		
		// Simple level + message
		{
			Name:        "level_message",
			PatternStr:  `^(?P<level>DEBUG|INFO|WARN|ERROR|FATAL|TRACE):\s*(?P<message>.+)`,
			Priority:    50,
			Description: "Simple Level + Message",
		},
		
		// Key-value pairs
		{
			Name:        "key_value",
			PatternStr:  `^(?P<message>.*?)(?:\s+(?P<kvpairs>(?:\w+=\S+\s*)+))$`,
			Priority:    40,
			Description: "Message with Key-Value Pairs",
		},
		
		// Fallback: everything as message
		{
			Name:        "fallback",
			PatternStr:  `^(?P<message>.+)$`,
			Priority:    10,
			Description: "Fallback: Treat Everything as Message",
		},
	}
	
	for _, pattern := range patterns {
		if err := p.AddPattern(pattern); err != nil {
			continue // Skip invalid patterns
		}
	}
}

// GetPatterns returns all configured patterns
func (p *RegexParser) GetPatterns() []*RegexPattern {
	return p.patterns
}

// parseTimestamp attempts to parse various timestamp formats
func parseTimestamp(timeStr string) (time.Time, error) {
	// Common timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"Jan 02 15:04:05",
		"Jan _2 15:04:05",
		"02/Jan/2006:15:04:05 -0700",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000-07:00",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	
	// Try Unix timestamp
	if timestamp, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
		if timestamp > 1000000000000 { // Milliseconds
			return time.Unix(0, timestamp*int64(time.Millisecond)), nil
		} else { // Seconds
			return time.Unix(timestamp, 0), nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timeStr)
}