package query

import (
	"regexp"
	"strings"
)

// Optimizer optimizes SQL queries for ClickHouse
type Optimizer struct {
	rules []OptimizationRule
}

// OptimizationRule defines a query optimization rule
type OptimizationRule struct {
	Name        string
	Description string
	Pattern     *regexp.Regexp
	Apply       func(query string) string
}

// NewOptimizer creates a new query optimizer
func NewOptimizer() *Optimizer {
	o := &Optimizer{
		rules: []OptimizationRule{},
	}
	
	// Add optimization rules
	o.addOptimizationRules()
	
	return o
}

// Optimize applies optimization rules to a query
func (o *Optimizer) Optimize(query string) string {
	optimized := query
	
	// Apply each optimization rule
	for _, rule := range o.rules {
		if rule.Pattern.MatchString(optimized) {
			optimized = rule.Apply(optimized)
		}
	}
	
	return optimized
}

// addOptimizationRules adds ClickHouse-specific optimization rules
func (o *Optimizer) addOptimizationRules() {
	// Rule 1: Use PREWHERE for timestamp filters
	o.rules = append(o.rules, OptimizationRule{
		Name:        "timestamp_prewhere",
		Description: "Move timestamp conditions to PREWHERE for better performance",
		Pattern:     regexp.MustCompile(`(?i)FROM\s+logs\s+WHERE\s+.*timestamp`),
		Apply: func(query string) string {
			// Only apply if no PREWHERE exists
			if strings.Contains(strings.ToUpper(query), "PREWHERE") {
				return query
			}
			
			// Extract WHERE clause
			re := regexp.MustCompile(`(?i)(FROM\s+logs)\s+WHERE\s+(.*?)(\s+(?:GROUP|ORDER|LIMIT|$))`)
			matches := re.FindStringSubmatch(query)
			if len(matches) < 3 {
				return query
			}
			
			whereClause := matches[2]
			
			// Find timestamp conditions
			timestampRe := regexp.MustCompile(`(?i)(timestamp\s*[><=]+\s*'[^']+')`)
			timestampMatches := timestampRe.FindAllString(whereClause, -1)
			
			if len(timestampMatches) == 0 {
				return query
			}
			
			// Build PREWHERE clause
			prewhereConditions := strings.Join(timestampMatches, " AND ")
			
			// Remove timestamp conditions from WHERE
			remainingWhere := whereClause
			for _, cond := range timestampMatches {
				remainingWhere = strings.Replace(remainingWhere, cond, "", 1)
			}
			remainingWhere = strings.TrimSpace(strings.ReplaceAll(remainingWhere, "AND  AND", "AND"))
			remainingWhere = strings.Trim(remainingWhere, "AND ")
			
			// Rebuild query
			newQuery := matches[1] + " PREWHERE " + prewhereConditions
			if remainingWhere != "" {
				newQuery += " WHERE " + remainingWhere
			}
			newQuery += matches[3]
			
			return re.ReplaceAllString(query, newQuery)
		},
	})
	
	// Rule 2: Use any() for non-grouped columns in aggregations
	o.rules = append(o.rules, OptimizationRule{
		Name:        "any_aggregation",
		Description: "Use any() for non-grouped columns to improve performance",
		Pattern:     regexp.MustCompile(`(?i)SELECT.*GROUP\s+BY`),
		Apply: func(query string) string {
			// This is a complex optimization that would need parsing
			// For now, just return the original query
			return query
		},
	})
	
	// Rule 3: Add FINAL for queries on replicated tables if needed
	o.rules = append(o.rules, OptimizationRule{
		Name:        "final_modifier",
		Description: "Add FINAL modifier for deduplicated results",
		Pattern:     regexp.MustCompile(`(?i)FROM\s+logs\s+`),
		Apply: func(query string) string {
			// Only add if user is querying recent data and no FINAL exists
			if strings.Contains(strings.ToUpper(query), "FINAL") {
				return query
			}
			
			// Check if query has recent timestamp filter
			if strings.Contains(query, "toStartOfDay(now())") || 
			   strings.Contains(query, "today()") ||
			   strings.Contains(query, "yesterday()") {
				// Don't add FINAL for today's data as it's likely still being written
				return query
			}
			
			return query
		},
	})
	
	// Rule 4: Optimize COUNT(*) to COUNT()
	o.rules = append(o.rules, OptimizationRule{
		Name:        "count_optimization",
		Description: "Optimize COUNT(*) to COUNT() for better performance",
		Pattern:     regexp.MustCompile(`(?i)COUNT\s*\(\s*\*\s*\)`),
		Apply: func(query string) string {
			return regexp.MustCompile(`(?i)COUNT\s*\(\s*\*\s*\)`).ReplaceAllString(query, "COUNT()")
		},
	})
	
	// Rule 5: Use materialized columns for common calculations
	o.rules = append(o.rules, OptimizationRule{
		Name:        "materialized_columns",
		Description: "Use materialized columns when available",
		Pattern:     regexp.MustCompile(`(?i)toDate\s*\(\s*timestamp\s*\)`),
		Apply: func(query string) string {
			// Replace toDate(timestamp) with date_partition (materialized column)
			return regexp.MustCompile(`(?i)toDate\s*\(\s*timestamp\s*\)`).ReplaceAllString(query, "date_partition")
		},
	})
	
	// Rule 6: Optimize LIKE patterns
	o.rules = append(o.rules, OptimizationRule{
		Name:        "like_optimization",
		Description: "Optimize LIKE patterns for better index usage",
		Pattern:     regexp.MustCompile(`(?i)LIKE\s+'%([^%]+)%'`),
		Apply: func(query string) string {
			// Convert LIKE '%text%' to position(message, 'text') > 0 for better performance
			re := regexp.MustCompile(`(?i)(\w+)\s+LIKE\s+'%([^%]+)%'`)
			return re.ReplaceAllString(query, "position($1, '$2') > 0")
		},
	})
	
	// Rule 7: Add FORMAT clause if missing
	o.rules = append(o.rules, OptimizationRule{
		Name:        "format_clause",
		Description: "Add FORMAT JSONEachRow for consistent output",
		Pattern:     regexp.MustCompile(`(?i)^SELECT.*`),
		Apply: func(query string) string {
			upperQuery := strings.ToUpper(query)
			if !strings.Contains(upperQuery, "FORMAT") {
				query = strings.TrimRight(query, "; \n\r\t") + " FORMAT JSONEachRow"
			}
			return query
		},
	})
	
	// Rule 8: Optimize time range queries
	o.rules = append(o.rules, OptimizationRule{
		Name:        "time_range_optimization",
		Description: "Optimize time range queries using ClickHouse functions",
		Pattern:     regexp.MustCompile(`(?i)timestamp\s*>=\s*now\(\)\s*-\s*INTERVAL\s+(\d+)\s+(HOUR|DAY|WEEK|MONTH)`),
		Apply: func(query string) string {
			// Already optimized format
			return query
		},
	})
}

// ExplainOptimizations returns explanations of optimizations that would be applied
func (o *Optimizer) ExplainOptimizations(query string) []string {
	explanations := []string{}
	
	for _, rule := range o.rules {
		if rule.Pattern.MatchString(query) {
			explanations = append(explanations, 
				rule.Name + ": " + rule.Description)
		}
	}
	
	return explanations
}