package query

import (
	"fmt"
	"regexp"
	"strings"
	
	"github.com/rs/zerolog/log"
)

// Validator validates SQL queries for safety and correctness
type Validator struct {
	allowedStatements []string
	deniedStatements  []string
	maxQueryLength    int
	patterns          map[string]*regexp.Regexp
}

// NewValidator creates a new query validator
func NewValidator() *Validator {
	v := &Validator{
		allowedStatements: []string{"SELECT", "WITH", "SHOW", "DESCRIBE", "DESC", "EXPLAIN"},
		deniedStatements:  []string{"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER", "TRUNCATE", "GRANT", "REVOKE"},
		maxQueryLength:    50000, // 50KB max query size
		patterns:          make(map[string]*regexp.Regexp),
	}
	
	// Compile regex patterns
	v.patterns["comments"] = regexp.MustCompile(`--.*$|/\*[\s\S]*?\*/`)
	v.patterns["semicolon"] = regexp.MustCompile(`;`)
	v.patterns["system_tables"] = regexp.MustCompile(`(?i)\b(system|information_schema)\b`)
	v.patterns["dangerous_functions"] = regexp.MustCompile(`(?i)\b(file|url|jdbc|odbc|mysql|postgresql)\s*\(`)
	
	return v
}

// Validate checks if a query is safe to execute
func (v *Validator) Validate(query string) error {
	if query == "" {
		return fmt.Errorf("empty query")
	}
	
	// Check query length
	if len(query) > v.maxQueryLength {
		return fmt.Errorf("query too long: %d bytes (max %d)", len(query), v.maxQueryLength)
	}
	
	// Remove comments for validation
	cleanQuery := v.removeComments(query)
	
	// Check for multiple statements (SQL injection prevention)
	if v.hasMultipleStatements(cleanQuery) {
		return fmt.Errorf("multiple statements not allowed")
	}
	
	// Get the statement type
	statementType := v.getStatementType(cleanQuery)
	if statementType == "" {
		return fmt.Errorf("unable to determine query type")
	}
	
	// Check if statement is allowed
	if !v.isStatementAllowed(statementType) {
		return fmt.Errorf("statement type '%s' not allowed", statementType)
	}
	
	// Check for dangerous patterns
	if err := v.checkDangerousPatterns(cleanQuery); err != nil {
		return err
	}
	
	// Validate specific to logs table
	if err := v.validateLogsQuery(cleanQuery); err != nil {
		return err
	}
	
	return nil
}

// removeComments removes SQL comments from the query
func (v *Validator) removeComments(query string) string {
	return v.patterns["comments"].ReplaceAllString(query, "")
}

// hasMultipleStatements checks if query contains multiple statements
func (v *Validator) hasMultipleStatements(query string) bool {
	// Count semicolons not within quotes
	inQuote := false
	quoteChar := rune(0)
	semicolonCount := 0
	
	for i, char := range query {
		if !inQuote {
			if char == '\'' || char == '"' {
				inQuote = true
				quoteChar = char
			} else if char == ';' {
				// Check if this is the last character (trailing semicolon is OK)
				if i < len(query)-1 && strings.TrimSpace(query[i+1:]) != "" {
					semicolonCount++
				}
			}
		} else {
			if char == quoteChar && (i == 0 || query[i-1] != '\\') {
				inQuote = false
			}
		}
	}
	
	return semicolonCount > 0
}

// getStatementType extracts the SQL statement type
func (v *Validator) getStatementType(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))
	
	// Handle WITH clause (CTE)
	if strings.HasPrefix(query, "WITH") {
		// Find the main statement after WITH
		parts := strings.Split(query, ")")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			for _, allowed := range v.allowedStatements {
				if strings.Contains(part, allowed) {
					return allowed
				}
			}
		}
		return "WITH"
	}
	
	// Get first word
	fields := strings.Fields(query)
	if len(fields) > 0 {
		return fields[0]
	}
	
	return ""
}

// isStatementAllowed checks if a statement type is allowed
func (v *Validator) isStatementAllowed(statementType string) bool {
	for _, allowed := range v.allowedStatements {
		if statementType == allowed {
			return true
		}
	}
	return false
}

// checkDangerousPatterns checks for potentially dangerous SQL patterns
func (v *Validator) checkDangerousPatterns(query string) error {
	upperQuery := strings.ToUpper(query)
	
	// Check for denied statements anywhere in the query
	for _, denied := range v.deniedStatements {
		if strings.Contains(upperQuery, denied) {
			return fmt.Errorf("statement contains denied operation: %s", denied)
		}
	}
	
	// Check for system table access
	if v.patterns["system_tables"].MatchString(query) {
		// Allow SHOW and DESCRIBE on system tables
		if !strings.Contains(upperQuery, "SHOW") && !strings.Contains(upperQuery, "DESCRIBE") {
			return fmt.Errorf("access to system tables not allowed")
		}
	}
	
	// Check for dangerous functions
	if v.patterns["dangerous_functions"].MatchString(query) {
		return fmt.Errorf("potentially dangerous functions not allowed")
	}
	
	// Check for UNION (can be used for SQL injection)
	if strings.Contains(upperQuery, "UNION") {
		return fmt.Errorf("UNION queries not allowed")
	}
	
	return nil
}

// validateLogsQuery performs specific validation for queries on logs table
func (v *Validator) validateLogsQuery(query string) error {
	upperQuery := strings.ToUpper(query)
	
	// If query references logs table, ensure it has reasonable limits
	if strings.Contains(upperQuery, "FROM LOGS") || strings.Contains(upperQuery, "FROM `LOGS`") {
		// Check for LIMIT clause
		if !strings.Contains(upperQuery, "LIMIT") {
			// Check if it's an aggregation query (which doesn't need LIMIT)
			if !v.isAggregationQuery(upperQuery) {
				return fmt.Errorf("queries on logs table must include a LIMIT clause")
			}
		}
		
		// Check for time range in WHERE clause (recommended)
		if !strings.Contains(upperQuery, "WHERE") || !strings.Contains(upperQuery, "TIMESTAMP") {
			log.Warn().Msg("Query on logs table without timestamp filter may be slow")
		}
	}
	
	return nil
}

// isAggregationQuery checks if query is an aggregation
func (v *Validator) isAggregationQuery(query string) bool {
	aggregateFunctions := []string{
		"COUNT(", "SUM(", "AVG(", "MIN(", "MAX(",
		"GROUP BY", "HAVING",
	}
	
	for _, fn := range aggregateFunctions {
		if strings.Contains(query, fn) {
			return true
		}
	}
	
	return false
}

// ValidateParameterName validates a parameter name
func (v *Validator) ValidateParameterName(name string) error {
	if name == "" {
		return fmt.Errorf("parameter name cannot be empty")
	}
	
	// Only allow alphanumeric and underscore
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(name) {
		return fmt.Errorf("parameter name can only contain letters, numbers, and underscores")
	}
	
	// Check length
	if len(name) > 64 {
		return fmt.Errorf("parameter name too long (max 64 characters)")
	}
	
	return nil
}