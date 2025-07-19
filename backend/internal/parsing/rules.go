package parsing

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// RuleSet contains validation and transformation rules
type RuleSet struct {
	ValidationRules   []ValidationRule   `json:"validation_rules"`
	TransformRules    []TransformRule    `json:"transform_rules"`
	FieldMappings     map[string]string  `json:"field_mappings"`
	RequiredFields    []string           `json:"required_fields"`
	DefaultValues     map[string]string  `json:"default_values"`
	FieldConstraints  map[string]FieldConstraint `json:"field_constraints"`
}

// ValidationRule defines a validation rule for parsed logs
type ValidationRule struct {
	Name        string `json:"name"`
	Field       string `json:"field"`
	Type        string `json:"type"` // "required", "regex", "range", "enum"
	Pattern     string `json:"pattern,omitempty"`
	MinValue    *int   `json:"min_value,omitempty"`
	MaxValue    *int   `json:"max_value,omitempty"`
	MinLength   *int   `json:"min_length,omitempty"`
	MaxLength   *int   `json:"max_length,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	Description string `json:"description"`
}

// TransformRule defines a transformation rule for parsed logs
type TransformRule struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "normalize", "extract", "enrich", "filter"
	Field       string            `json:"field"`
	Target      string            `json:"target,omitempty"`
	Pattern     string            `json:"pattern,omitempty"`
	Replacement string            `json:"replacement,omitempty"`
	Mapping     map[string]string `json:"mapping,omitempty"`
	Function    string            `json:"function,omitempty"` // "lowercase", "uppercase", "trim", "extract_json"
	Description string            `json:"description"`
}

// FieldConstraint defines constraints for a specific field
type FieldConstraint struct {
	Type        string   `json:"type"` // "string", "number", "boolean", "timestamp"
	Required    bool     `json:"required"`
	MinLength   *int     `json:"min_length,omitempty"`
	MaxLength   *int     `json:"max_length,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	Description string   `json:"description"`
}

// NewDefaultRuleSet creates a default rule set with common validation and transformation rules
func NewDefaultRuleSet() *RuleSet {
	return &RuleSet{
		ValidationRules: []ValidationRule{
			{
				Name:        "message_required",
				Field:       "message",
				Type:        "required",
				Description: "Message field is required",
			},
			{
				Name:        "level_enum",
				Field:       "level",
				Type:        "enum",
				AllowedValues: []string{"debug", "info", "warn", "error", "fatal", "trace"},
				Description: "Level must be one of the allowed values",
			},
			{
				Name:        "message_length",
				Field:       "message",
				Type:        "range",
				MinLength:   &[]int{1}[0],
				MaxLength:   &[]int{10000}[0],
				Description: "Message length must be between 1 and 10000 characters",
			},
			{
				Name:        "service_format",
				Field:       "service",
				Type:        "regex",
				Pattern:     `^[a-zA-Z0-9_-]+$`,
				Description: "Service name must contain only alphanumeric characters, underscores, and hyphens",
			},
		},
		TransformRules: []TransformRule{
			{
				Name:        "normalize_level",
				Type:        "normalize",
				Field:       "level",
				Function:    "lowercase",
				Description: "Convert log level to lowercase",
			},
			{
				Name:        "trim_message",
				Type:        "normalize",
				Field:       "message",
				Function:    "trim",
				Description: "Trim whitespace from message",
			},
			{
				Name:        "extract_user_id",
				Type:        "extract",
				Field:       "message",
				Target:      "user_id",
				Pattern:     `user_id[=:]\s*([a-zA-Z0-9_-]+)`,
				Description: "Extract user_id from message",
			},
			{
				Name:        "extract_request_id",
				Type:        "extract",
				Field:       "message",
				Target:      "request_id",
				Pattern:     `request_id[=:]\s*([a-zA-Z0-9_-]+)`,
				Description: "Extract request_id from message",
			},
		},
		FieldMappings: map[string]string{
			"msg":       "message",
			"lvl":       "level",
			"app":       "service",
			"component": "service",
			"logger":    "service",
		},
		RequiredFields: []string{"message"},
		DefaultValues: map[string]string{
			"level":   "info",
			"service": "unknown",
		},
		FieldConstraints: map[string]FieldConstraint{
			"level": {
				Type:        "string",
				Required:    false,
				AllowedValues: []string{"debug", "info", "warn", "error", "fatal", "trace"},
				Description: "Log level constraint",
			},
			"message": {
				Type:        "string",
				Required:    true,
				MinLength:   &[]int{1}[0],
				MaxLength:   &[]int{10000}[0],
				Description: "Message field constraint",
			},
			"service": {
				Type:        "string",
				Required:    false,
				Pattern:     `^[a-zA-Z0-9_-]+$`,
				MaxLength:   &[]int{100}[0],
				Description: "Service name constraint",
			},
			"trace_id": {
				Type:        "string",
				Required:    false,
				Pattern:     `^[a-fA-F0-9_-]+$`,
				MaxLength:   &[]int{64}[0],
				Description: "Trace ID constraint",
			},
		},
	}
}

// Validate validates a parsed log against the rule set
func (rs *RuleSet) Validate(log *models.Log) error {
	// Check required fields
	for _, field := range rs.RequiredFields {
		if err := rs.validateRequiredField(log, field); err != nil {
			return err
		}
	}
	
	// Apply field constraints
	for field, constraint := range rs.FieldConstraints {
		if err := rs.validateFieldConstraint(log, field, constraint); err != nil {
			return err
		}
	}
	
	// Apply validation rules
	for _, rule := range rs.ValidationRules {
		if err := rs.validateRule(log, rule); err != nil {
			return err
		}
	}
	
	return nil
}

// Transform applies transformation rules to a parsed log
func (rs *RuleSet) Transform(log *models.Log) error {
	// Apply field mappings
	rs.applyFieldMappings(log)
	
	// Apply default values
	rs.applyDefaultValues(log)
	
	// Apply transformation rules
	for _, rule := range rs.TransformRules {
		if err := rs.applyTransformRule(log, rule); err != nil {
			return fmt.Errorf("transform rule '%s' failed: %w", rule.Name, err)
		}
	}
	
	return nil
}

// validateRequiredField checks if a required field exists and is not empty
func (rs *RuleSet) validateRequiredField(log *models.Log, field string) error {
	var value string
	var exists bool
	
	switch field {
	case "message":
		value = log.Message
		exists = value != ""
	case "level":
		value = log.Level
		exists = value != ""
	case "service":
		value = log.Service
		exists = value != ""
	case "trace_id":
		value = log.TraceID
		exists = value != ""
	case "span_id":
		value = log.SpanID
		exists = value != ""
	default:
		if attr, ok := log.Attributes[field]; ok {
			value = fmt.Sprintf("%v", attr)
			exists = value != ""
		}
	}
	
	if !exists {
		return fmt.Errorf("required field '%s' is missing or empty", field)
	}
	
	return nil
}

// validateFieldConstraint validates a field against its constraint
func (rs *RuleSet) validateFieldConstraint(log *models.Log, field string, constraint FieldConstraint) error {
	var value string
	var exists bool
	
	// Get field value
	switch field {
	case "message":
		value = log.Message
		exists = value != ""
	case "level":
		value = log.Level
		exists = value != ""
	case "service":
		value = log.Service
		exists = value != ""
	case "trace_id":
		value = log.TraceID
		exists = value != ""
	case "span_id":
		value = log.SpanID
		exists = value != ""
	default:
		if attr, ok := log.Attributes[field]; ok {
			value = fmt.Sprintf("%v", attr)
			exists = value != ""
		}
	}
	
	// Check if required
	if constraint.Required && !exists {
		return fmt.Errorf("required field '%s' is missing", field)
	}
	
	// Skip validation if field doesn't exist and isn't required
	if !exists {
		return nil
	}
	
	// Validate length constraints
	if constraint.MinLength != nil && len(value) < *constraint.MinLength {
		return fmt.Errorf("field '%s' length %d is below minimum %d", field, len(value), *constraint.MinLength)
	}
	
	if constraint.MaxLength != nil && len(value) > *constraint.MaxLength {
		return fmt.Errorf("field '%s' length %d exceeds maximum %d", field, len(value), *constraint.MaxLength)
	}
	
	// Validate pattern
	if constraint.Pattern != "" {
		if matched, err := regexp.MatchString(constraint.Pattern, value); err != nil {
			return fmt.Errorf("invalid regex pattern for field '%s': %w", field, err)
		} else if !matched {
			return fmt.Errorf("field '%s' value '%s' does not match pattern '%s'", field, value, constraint.Pattern)
		}
	}
	
	// Validate allowed values
	if len(constraint.AllowedValues) > 0 {
		found := false
		for _, allowed := range constraint.AllowedValues {
			if value == allowed {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field '%s' value '%s' is not in allowed values: %v", field, value, constraint.AllowedValues)
		}
	}
	
	return nil
}

// validateRule validates a log against a specific validation rule
func (rs *RuleSet) validateRule(log *models.Log, rule ValidationRule) error {
	// Get field value
	var value string
	var exists bool
	
	switch rule.Field {
	case "message":
		value = log.Message
		exists = value != ""
	case "level":
		value = log.Level
		exists = value != ""
	case "service":
		value = log.Service
		exists = value != ""
	case "trace_id":
		value = log.TraceID
		exists = value != ""
	case "span_id":
		value = log.SpanID
		exists = value != ""
	default:
		if attr, ok := log.Attributes[rule.Field]; ok {
			value = fmt.Sprintf("%v", attr)
			exists = value != ""
		}
	}
	
	// Apply validation based on rule type
	switch rule.Type {
	case "required":
		if !exists {
			return fmt.Errorf("validation rule '%s': field '%s' is required", rule.Name, rule.Field)
		}
		
	case "regex":
		if exists && rule.Pattern != "" {
			if matched, err := regexp.MatchString(rule.Pattern, value); err != nil {
				return fmt.Errorf("validation rule '%s': invalid regex: %w", rule.Name, err)
			} else if !matched {
				return fmt.Errorf("validation rule '%s': field '%s' does not match pattern", rule.Name, rule.Field)
			}
		}
		
	case "range":
		if exists {
			if rule.MinLength != nil && len(value) < *rule.MinLength {
				return fmt.Errorf("validation rule '%s': field '%s' length below minimum", rule.Name, rule.Field)
			}
			if rule.MaxLength != nil && len(value) > *rule.MaxLength {
				return fmt.Errorf("validation rule '%s': field '%s' length exceeds maximum", rule.Name, rule.Field)
			}
		}
		
	case "enum":
		if exists && len(rule.AllowedValues) > 0 {
			found := false
			for _, allowed := range rule.AllowedValues {
				if value == allowed {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("validation rule '%s': field '%s' value not in allowed list", rule.Name, rule.Field)
			}
		}
	}
	
	return nil
}

// applyFieldMappings applies field mappings to rename fields
func (rs *RuleSet) applyFieldMappings(log *models.Log) {
	for source, target := range rs.FieldMappings {
		if value, exists := log.Attributes[source]; exists {
			switch target {
			case "message":
				if log.Message == "" {
					log.Message = fmt.Sprintf("%v", value)
				}
			case "level":
				if log.Level == "" {
					log.Level = fmt.Sprintf("%v", value)
				}
			case "service":
				if log.Service == "" {
					log.Service = fmt.Sprintf("%v", value)
				}
			case "trace_id":
				if log.TraceID == "" {
					log.TraceID = fmt.Sprintf("%v", value)
				}
			case "span_id":
				if log.SpanID == "" {
					log.SpanID = fmt.Sprintf("%v", value)
				}
			default:
				log.Attributes[target] = value
			}
			delete(log.Attributes, source)
		}
	}
}

// applyDefaultValues applies default values for empty fields
func (rs *RuleSet) applyDefaultValues(log *models.Log) {
	for field, defaultValue := range rs.DefaultValues {
		switch field {
		case "message":
			if log.Message == "" {
				log.Message = defaultValue
			}
		case "level":
			if log.Level == "" {
				log.Level = defaultValue
			}
		case "service":
			if log.Service == "" {
				log.Service = defaultValue
			}
		case "trace_id":
			if log.TraceID == "" {
				log.TraceID = defaultValue
			}
		case "span_id":
			if log.SpanID == "" {
				log.SpanID = defaultValue
			}
		default:
			if _, exists := log.Attributes[field]; !exists {
				log.Attributes[field] = defaultValue
			}
		}
	}
}

// applyTransformRule applies a single transformation rule
func (rs *RuleSet) applyTransformRule(log *models.Log, rule TransformRule) error {
	switch rule.Type {
	case "normalize":
		return rs.applyNormalization(log, rule)
	case "extract":
		return rs.applyExtraction(log, rule)
	case "enrich":
		return rs.applyEnrichment(log, rule)
	case "filter":
		return rs.applyFilter(log, rule)
	default:
		return fmt.Errorf("unknown transform rule type: %s", rule.Type)
	}
}

// applyNormalization applies normalization transformations
func (rs *RuleSet) applyNormalization(log *models.Log, rule TransformRule) error {
	// Get field value
	var value string
	var fieldExists bool
	
	switch rule.Field {
	case "message":
		value = log.Message
		fieldExists = value != ""
	case "level":
		value = log.Level
		fieldExists = value != ""
	case "service":
		value = log.Service
		fieldExists = value != ""
	default:
		if attr, ok := log.Attributes[rule.Field]; ok {
			value = fmt.Sprintf("%v", attr)
			fieldExists = true
		}
	}
	
	if !fieldExists {
		return nil // Skip if field doesn't exist
	}
	
	// Apply function
	var transformedValue string
	switch rule.Function {
	case "lowercase":
		transformedValue = strings.ToLower(value)
	case "uppercase":
		transformedValue = strings.ToUpper(value)
	case "trim":
		transformedValue = strings.TrimSpace(value)
	default:
		return fmt.Errorf("unknown normalization function: %s", rule.Function)
	}
	
	// Set transformed value back
	switch rule.Field {
	case "message":
		log.Message = transformedValue
	case "level":
		log.Level = transformedValue
	case "service":
		log.Service = transformedValue
	default:
		log.Attributes[rule.Field] = transformedValue
	}
	
	return nil
}

// applyExtraction extracts data using regex patterns
func (rs *RuleSet) applyExtraction(log *models.Log, rule TransformRule) error {
	if rule.Pattern == "" || rule.Target == "" {
		return fmt.Errorf("extraction rule requires pattern and target")
	}
	
	// Get source field value
	var sourceValue string
	switch rule.Field {
	case "message":
		sourceValue = log.Message
	case "level":
		sourceValue = log.Level
	case "service":
		sourceValue = log.Service
	default:
		if attr, ok := log.Attributes[rule.Field]; ok {
			sourceValue = fmt.Sprintf("%v", attr)
		}
	}
	
	if sourceValue == "" {
		return nil // Skip if source field is empty
	}
	
	// Extract using regex
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	matches := re.FindStringSubmatch(sourceValue)
	if len(matches) > 1 {
		// Set extracted value to target field
		switch rule.Target {
		case "message":
			log.Message = matches[1]
		case "level":
			log.Level = matches[1]
		case "service":
			log.Service = matches[1]
		case "trace_id":
			log.TraceID = matches[1]
		case "span_id":
			log.SpanID = matches[1]
		default:
			log.Attributes[rule.Target] = matches[1]
		}
	}
	
	return nil
}

// applyEnrichment adds enrichment data
func (rs *RuleSet) applyEnrichment(log *models.Log, rule TransformRule) error {
	// Add current timestamp
	if rule.Target == "parsed_at" {
		log.Attributes["parsed_at"] = time.Now().UTC().Format(time.RFC3339)
	}
	
	// Add environment info
	if rule.Target == "environment" && rule.Replacement != "" {
		log.Attributes["environment"] = rule.Replacement
	}
	
	return nil
}

// applyFilter applies filtering logic (placeholder)
func (rs *RuleSet) applyFilter(log *models.Log, rule TransformRule) error {
	// Filtering could be implemented here
	// For now, this is a placeholder
	return nil
}