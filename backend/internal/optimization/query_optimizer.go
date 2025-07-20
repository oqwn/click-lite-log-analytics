package optimization

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// QueryOptimizer optimizes SQL queries for ClickHouse
type QueryOptimizer struct {
	indexHints    map[string][]string
	queryPatterns []QueryPattern
	rewriteRules  []RewriteRule
}

// QueryPattern represents a query pattern for optimization
type QueryPattern struct {
	Pattern    *regexp.Regexp
	Optimizer  func(string) string
	Priority   int
}

// RewriteRule represents a query rewrite rule
type RewriteRule struct {
	Name        string
	Condition   func(string) bool
	Rewrite     func(string) string
	Description string
}

// QueryPlan represents an optimized query execution plan
type QueryPlan struct {
	OriginalQuery   string
	OptimizedQuery  string
	Optimizations   []string
	EstimatedCost   float64
	IndexesUsed     []string
	PartitionPruning bool
	Parallelism     int
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer() *QueryOptimizer {
	optimizer := &QueryOptimizer{
		indexHints: map[string][]string{
			"timestamp": {"idx_timestamp"},
			"service":   {"idx_service"},
			"level":     {"idx_level"},
			"trace_id":  {"idx_trace_id"},
		},
		queryPatterns: []QueryPattern{},
		rewriteRules:  []RewriteRule{},
	}
	
	optimizer.initializePatterns()
	optimizer.initializeRules()
	
	return optimizer
}

// initializePatterns sets up query optimization patterns
func (o *QueryOptimizer) initializePatterns() {
	o.queryPatterns = []QueryPattern{
		{
			// Optimize time range queries
			Pattern: regexp.MustCompile(`WHERE\s+timestamp\s*>=\s*'([^']+)'\s+AND\s+timestamp\s*<=\s*'([^']+)'`),
			Optimizer: func(query string) string {
				// Use BETWEEN for better optimization
				return regexp.MustCompile(`timestamp\s*>=\s*'([^']+)'\s+AND\s+timestamp\s*<=\s*'([^']+)'`).
					ReplaceAllString(query, "timestamp BETWEEN '$1' AND '$2'")
			},
			Priority: 10,
		},
		{
			// Optimize COUNT(*) queries
			Pattern: regexp.MustCompile(`SELECT\s+COUNT\(\*\)\s+FROM\s+logs`),
			Optimizer: func(query string) string {
				// Use system table for approximate count
				if !strings.Contains(query, "WHERE") {
					return "SELECT sum(rows) FROM system.parts WHERE table = 'logs' AND active"
				}
				return query
			},
			Priority: 20,
		},
		{
			// Optimize DISTINCT queries
			Pattern: regexp.MustCompile(`SELECT\s+DISTINCT\s+(\w+)`),
			Optimizer: func(query string) string {
				// Use GROUP BY for better performance
				matches := regexp.MustCompile(`SELECT\s+DISTINCT\s+(\w+)(.*)FROM`).FindStringSubmatch(query)
				if len(matches) > 1 {
					field := matches[1]
					return strings.Replace(query, "SELECT DISTINCT "+field, "SELECT "+field+" GROUP BY "+field, 1)
				}
				return query
			},
			Priority: 15,
		},
		{
			// Optimize ORDER BY with LIMIT
			Pattern: regexp.MustCompile(`ORDER\s+BY\s+timestamp\s+DESC\s+LIMIT\s+(\d+)`),
			Optimizer: func(query string) string {
				// Add optimization hint for recent data
				if strings.Contains(query, "WHERE") && !strings.Contains(query, "timestamp") {
					// Add time constraint for better performance
					return strings.Replace(query, "WHERE", 
						fmt.Sprintf("WHERE timestamp > now() - INTERVAL 7 DAY AND"), 1)
				}
				return query
			},
			Priority: 5,
		},
	}
}

// initializeRules sets up query rewrite rules
func (o *QueryOptimizer) initializeRules() {
	o.rewriteRules = []RewriteRule{
		{
			Name: "UsePrewhere",
			Condition: func(query string) bool {
				return strings.Contains(query, "WHERE") && !strings.Contains(query, "PREWHERE")
			},
			Rewrite: func(query string) string {
				// Move simple conditions to PREWHERE for better performance
				whereIdx := strings.Index(query, "WHERE")
				if whereIdx == -1 {
					return query
				}
				
				conditions := extractWhereConditions(query)
				prewhere := []string{}
				where := []string{}
				
				for _, cond := range conditions {
					if isSimpleCondition(cond) {
						prewhere = append(prewhere, cond)
					} else {
						where = append(where, cond)
					}
				}
				
				if len(prewhere) == 0 {
					return query
				}
				
				result := query[:whereIdx]
				if len(prewhere) > 0 {
					result += "PREWHERE " + strings.Join(prewhere, " AND ")
				}
				if len(where) > 0 {
					result += " WHERE " + strings.Join(where, " AND ")
				}
				result += query[strings.Index(query, "WHERE")+5+len(strings.Join(conditions, " AND ")):]
				
				return result
			},
			Description: "Move simple filtering conditions to PREWHERE",
		},
		{
			Name: "OptimizeSubqueries",
			Condition: func(query string) bool {
				return strings.Contains(query, "IN (SELECT")
			},
			Rewrite: func(query string) string {
				// Convert IN (SELECT) to JOIN for better performance
				inSelectRegex := regexp.MustCompile(`(\w+)\s+IN\s+\(SELECT\s+(\w+)\s+FROM\s+(\w+)\s+WHERE\s+([^)]+)\)`)
				return inSelectRegex.ReplaceAllString(query, "EXISTS (SELECT 1 FROM $3 WHERE $4 AND $3.$2 = logs.$1)")
			},
			Description: "Convert IN subqueries to EXISTS for better performance",
		},
		{
			Name: "UsePartitionPruning",
			Condition: func(query string) bool {
				return strings.Contains(query, "timestamp") && !strings.Contains(query, "toYYYYMMDD")
			},
			Rewrite: func(query string) string {
				// Add partition key to WHERE clause for pruning
				timestampRegex := regexp.MustCompile(`timestamp\s*(?:>|>=|BETWEEN)\s*'([^']+)'`)
				matches := timestampRegex.FindStringSubmatch(query)
				if len(matches) > 1 {
					date, _ := time.Parse(time.RFC3339, matches[1])
					partitionKey := fmt.Sprintf("toYYYYMMDD(timestamp) >= %d", 
						date.Year()*10000+int(date.Month())*100+date.Day())
					
					if strings.Contains(query, "WHERE") {
						return strings.Replace(query, "WHERE", "WHERE "+partitionKey+" AND", 1)
					}
				}
				return query
			},
			Description: "Add partition pruning hints",
		},
		{
			Name: "OptimizeAggregations",
			Condition: func(query string) bool {
				return strings.Contains(query, "GROUP BY") && strings.Contains(query, "COUNT")
			},
			Rewrite: func(query string) string {
				// Use -State and -Merge functions for distributed aggregations
				if strings.Contains(query, "COUNT(*)") {
					query = strings.Replace(query, "COUNT(*)", "count()", -1)
				}
				if strings.Contains(query, "COUNT(DISTINCT") {
					query = strings.Replace(query, "COUNT(DISTINCT", "uniqExact(", -1)
				}
				return query
			},
			Description: "Optimize aggregation functions",
		},
	}
}

// Optimize applies optimizations to a query
func (o *QueryOptimizer) Optimize(query string) *QueryPlan {
	plan := &QueryPlan{
		OriginalQuery:    query,
		OptimizedQuery:   query,
		Optimizations:    []string{},
		EstimatedCost:    100.0,
		IndexesUsed:      []string{},
		PartitionPruning: false,
		Parallelism:      1,
	}
	
	// Apply query patterns
	for _, pattern := range o.queryPatterns {
		if pattern.Pattern.MatchString(plan.OptimizedQuery) {
			newQuery := pattern.Optimizer(plan.OptimizedQuery)
			if newQuery != plan.OptimizedQuery {
				plan.OptimizedQuery = newQuery
				plan.Optimizations = append(plan.Optimizations, 
					fmt.Sprintf("Applied pattern optimization (priority %d)", pattern.Priority))
			}
		}
	}
	
	// Apply rewrite rules
	for _, rule := range o.rewriteRules {
		if rule.Condition(plan.OptimizedQuery) {
			newQuery := rule.Rewrite(plan.OptimizedQuery)
			if newQuery != plan.OptimizedQuery {
				plan.OptimizedQuery = newQuery
				plan.Optimizations = append(plan.Optimizations, rule.Description)
			}
		}
	}
	
	// Analyze indexes
	plan.IndexesUsed = o.analyzeIndexUsage(plan.OptimizedQuery)
	
	// Check partition pruning
	plan.PartitionPruning = strings.Contains(plan.OptimizedQuery, "toYYYYMMDD")
	
	// Estimate parallelism
	plan.Parallelism = o.estimateParallelism(plan.OptimizedQuery)
	
	// Calculate estimated cost
	plan.EstimatedCost = o.estimateCost(plan)
	
	return plan
}

// analyzeIndexUsage identifies which indexes will be used
func (o *QueryOptimizer) analyzeIndexUsage(query string) []string {
	indexes := []string{}
	
	for field, indexNames := range o.indexHints {
		if strings.Contains(query, field) {
			indexes = append(indexes, indexNames...)
		}
	}
	
	return indexes
}

// estimateParallelism estimates query parallelism level
func (o *QueryOptimizer) estimateParallelism(query string) int {
	// Base parallelism
	parallelism := 1
	
	// Increase for aggregations
	if strings.Contains(query, "GROUP BY") {
		parallelism *= 4
	}
	
	// Increase for large scans
	if !strings.Contains(query, "LIMIT") || strings.Contains(query, "LIMIT 1000") {
		parallelism *= 2
	}
	
	// Cap at reasonable level
	if parallelism > 16 {
		parallelism = 16
	}
	
	return parallelism
}

// estimateCost estimates query execution cost
func (o *QueryOptimizer) estimateCost(plan *QueryPlan) float64 {
	cost := 100.0
	
	// Reduce cost for index usage
	cost -= float64(len(plan.IndexesUsed)) * 10
	
	// Reduce cost for partition pruning
	if plan.PartitionPruning {
		cost *= 0.5
	}
	
	// Reduce cost for optimizations
	cost -= float64(len(plan.Optimizations)) * 5
	
	// Increase cost for complex operations
	if strings.Contains(plan.OptimizedQuery, "JOIN") {
		cost *= 2
	}
	if strings.Contains(plan.OptimizedQuery, "DISTINCT") {
		cost *= 1.5
	}
	
	if cost < 1 {
		cost = 1
	}
	
	return cost
}

// extractWhereConditions extracts individual conditions from WHERE clause
func extractWhereConditions(query string) []string {
	whereIdx := strings.Index(query, "WHERE")
	if whereIdx == -1 {
		return []string{}
	}
	
	// Find the end of WHERE clause
	endIdx := len(query)
	for _, keyword := range []string{"GROUP BY", "ORDER BY", "LIMIT", "HAVING"} {
		if idx := strings.Index(query[whereIdx:], keyword); idx != -1 {
			if whereIdx+idx < endIdx {
				endIdx = whereIdx + idx
			}
		}
	}
	
	whereClause := query[whereIdx+5 : endIdx]
	// Simple split by AND (this is a simplification)
	conditions := strings.Split(whereClause, " AND ")
	
	result := []string{}
	for _, cond := range conditions {
		cond = strings.TrimSpace(cond)
		if cond != "" {
			result = append(result, cond)
		}
	}
	
	return result
}

// isSimpleCondition checks if a condition is simple enough for PREWHERE
func isSimpleCondition(condition string) bool {
	// Simple equality or comparison conditions
	simplePatterns := []string{
		`^\w+\s*=\s*'[^']+'$`,
		`^\w+\s*=\s*\d+$`,
		`^\w+\s*(?:>|<|>=|<=)\s*\d+$`,
		`^\w+\s*IN\s*\([^)]+\)$`,
	}
	
	for _, pattern := range simplePatterns {
		if matched, _ := regexp.MatchString(pattern, strings.TrimSpace(condition)); matched {
			return true
		}
	}
	
	return false
}

// SuggestIndexes suggests indexes based on query patterns
func (o *QueryOptimizer) SuggestIndexes(queries []string) []IndexSuggestion {
	fieldUsage := make(map[string]int)
	suggestions := []IndexSuggestion{}
	
	// Analyze field usage in WHERE clauses
	for _, query := range queries {
		fields := extractFieldsFromWhere(query)
		for _, field := range fields {
			fieldUsage[field]++
		}
	}
	
	// Generate suggestions
	for field, count := range fieldUsage {
		if count > len(queries)/10 { // Used in more than 10% of queries
			if _, hasIndex := o.indexHints[field]; !hasIndex {
				suggestions = append(suggestions, IndexSuggestion{
					Field:       field,
					IndexType:   "bloom_filter",
					Reason:      fmt.Sprintf("Field used in %d queries", count),
					Impact:      float64(count) / float64(len(queries)),
				})
			}
		}
	}
	
	return suggestions
}

// IndexSuggestion represents a suggested index
type IndexSuggestion struct {
	Field     string
	IndexType string
	Reason    string
	Impact    float64
}

// extractFieldsFromWhere extracts field names from WHERE clause
func extractFieldsFromWhere(query string) []string {
	fields := []string{}
	
	// Simple regex to find field names in WHERE clause
	fieldRegex := regexp.MustCompile(`\b(\w+)\s*(?:=|>|<|>=|<=|IN|LIKE)`)
	matches := fieldRegex.FindAllStringSubmatch(query, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			fields = append(fields, match[1])
		}
	}
	
	return fields
}