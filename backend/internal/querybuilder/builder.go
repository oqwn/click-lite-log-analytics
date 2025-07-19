package querybuilder

import (
	"fmt"
	"strings"
	"time"

	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// Service handles query builder operations
type Service struct {
	availableFields []models.QueryField
}

// NewService creates a new query builder service
func NewService() *Service {
	return &Service{
		availableFields: getAvailableFields(),
	}
}

// GetAvailableFields returns the available fields for query building
func (s *Service) GetAvailableFields() []models.QueryField {
	return s.availableFields
}

// GenerateSQL converts a QueryBuilder configuration to SQL
func (s *Service) GenerateSQL(qb *models.QueryBuilder) (string, error) {
	var parts []string

	// SELECT clause
	selectClause, err := s.buildSelectClause(qb)
	if err != nil {
		return "", fmt.Errorf("failed to build SELECT clause: %w", err)
	}
	parts = append(parts, selectClause)

	// FROM clause
	parts = append(parts, "FROM logs")

	// WHERE clause
	if len(qb.Filters) > 0 || qb.TimeRange != nil {
		whereClause, err := s.buildWhereClause(qb)
		if err != nil {
			return "", fmt.Errorf("failed to build WHERE clause: %w", err)
		}
		if whereClause != "" {
			parts = append(parts, "WHERE "+whereClause)
		}
	}

	// GROUP BY clause
	if len(qb.GroupBy) > 0 {
		groupByClause := s.buildGroupByClause(qb.GroupBy)
		parts = append(parts, "GROUP BY "+groupByClause)
	}

	// ORDER BY clause
	if len(qb.OrderBy) > 0 {
		orderByClause := s.buildOrderByClause(qb.OrderBy)
		parts = append(parts, "ORDER BY "+orderByClause)
	}

	// LIMIT clause
	if qb.Limit > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT %d", qb.Limit))
	}

	sql := strings.Join(parts, "\n")
	return sql, nil
}

// ValidateQueryBuilder validates a query builder configuration
func (s *Service) ValidateQueryBuilder(qb *models.QueryBuilder) error {
	if qb.Name == "" {
		return fmt.Errorf("query name is required")
	}

	// Validate fields
	availableFieldMap := make(map[string]bool)
	for _, field := range s.availableFields {
		availableFieldMap[field.Name] = true
	}

	for _, field := range qb.Fields {
		if field.Selected && !availableFieldMap[field.Name] {
			return fmt.Errorf("unknown field: %s", field.Name)
		}
	}

	// Validate filters
	for _, filter := range qb.Filters {
		if !availableFieldMap[filter.Field] {
			return fmt.Errorf("unknown field in filter: %s", filter.Field)
		}
		if err := s.validateFilterOperator(filter.Operator); err != nil {
			return err
		}
	}

	// Validate aggregations
	for _, agg := range qb.Aggregations {
		if err := s.validateAggregationFunction(agg.Function); err != nil {
			return err
		}
		if agg.Field != "" && !availableFieldMap[agg.Field] {
			return fmt.Errorf("unknown field in aggregation: %s", agg.Field)
		}
	}

	return nil
}

// buildSelectClause builds the SELECT part of the SQL query
func (s *Service) buildSelectClause(qb *models.QueryBuilder) (string, error) {
	var columns []string

	// Add selected fields
	for _, field := range qb.Fields {
		if field.Selected {
			columns = append(columns, field.Name)
		}
	}

	// Add aggregations
	for _, agg := range qb.Aggregations {
		aggSQL, err := s.buildAggregationSQL(agg)
		if err != nil {
			return "", err
		}
		columns = append(columns, aggSQL)
	}

	if len(columns) == 0 {
		columns = append(columns, "*")
	}

	return "SELECT " + strings.Join(columns, ", "), nil
}

// buildWhereClause builds the WHERE part of the SQL query
func (s *Service) buildWhereClause(qb *models.QueryBuilder) (string, error) {
	var conditions []string

	// Add time range filter
	if qb.TimeRange != nil {
		timeCondition, err := s.buildTimeRangeCondition(qb.TimeRange)
		if err != nil {
			return "", err
		}
		if timeCondition != "" {
			conditions = append(conditions, timeCondition)
		}
	}

	// Add custom filters
	for i, filter := range qb.Filters {
		condition, err := s.buildFilterCondition(filter)
		if err != nil {
			return "", err
		}

		if i > 0 && filter.LogicalOp != "" {
			condition = filter.LogicalOp + " " + condition
		}
		conditions = append(conditions, condition)
	}

	return strings.Join(conditions, " "), nil
}

// buildFilterCondition builds a single filter condition
func (s *Service) buildFilterCondition(filter models.QueryBuilderFilter) (string, error) {
	field := filter.Field
	operator := filter.Operator
	value := filter.Value

	switch operator {
	case "equals":
		return fmt.Sprintf("%s = %s", field, s.formatValue(value)), nil
	case "not_equals":
		return fmt.Sprintf("%s != %s", field, s.formatValue(value)), nil
	case "contains":
		return fmt.Sprintf("%s LIKE %s", field, s.formatValue("%"+fmt.Sprintf("%v", value)+"%")), nil
	case "not_contains":
		return fmt.Sprintf("%s NOT LIKE %s", field, s.formatValue("%"+fmt.Sprintf("%v", value)+"%")), nil
	case "greater_than":
		return fmt.Sprintf("%s > %s", field, s.formatValue(value)), nil
	case "less_than":
		return fmt.Sprintf("%s < %s", field, s.formatValue(value)), nil
	case "greater_equal":
		return fmt.Sprintf("%s >= %s", field, s.formatValue(value)), nil
	case "less_equal":
		return fmt.Sprintf("%s <= %s", field, s.formatValue(value)), nil
	case "between":
		if len(filter.Values) != 2 {
			return "", fmt.Errorf("between operator requires exactly 2 values")
		}
		return fmt.Sprintf("%s BETWEEN %s AND %s", field, 
			s.formatValue(filter.Values[0]), s.formatValue(filter.Values[1])), nil
	case "in":
		if len(filter.Values) == 0 {
			return "", fmt.Errorf("in operator requires at least 1 value")
		}
		valueList := make([]string, len(filter.Values))
		for i, v := range filter.Values {
			valueList[i] = s.formatValue(v)
		}
		return fmt.Sprintf("%s IN (%s)", field, strings.Join(valueList, ", ")), nil
	case "not_in":
		if len(filter.Values) == 0 {
			return "", fmt.Errorf("not_in operator requires at least 1 value")
		}
		valueList := make([]string, len(filter.Values))
		for i, v := range filter.Values {
			valueList[i] = s.formatValue(v)
		}
		return fmt.Sprintf("%s NOT IN (%s)", field, strings.Join(valueList, ", ")), nil
	case "is_null":
		return fmt.Sprintf("%s IS NULL", field), nil
	case "is_not_null":
		return fmt.Sprintf("%s IS NOT NULL", field), nil
	default:
		return "", fmt.Errorf("unsupported operator: %s", operator)
	}
}

// buildTimeRangeCondition builds time range filter condition
func (s *Service) buildTimeRangeCondition(timeRange *models.QueryTimeRange) (string, error) {
	var start, end time.Time

	if timeRange.Relative != "" {
		var err error
		start, end, err = s.parseRelativeTimeRange(timeRange.Relative)
		if err != nil {
			return "", err
		}
	} else {
		start = timeRange.Start
		end = timeRange.End
	}

	if start.IsZero() && end.IsZero() {
		return "", nil
	}

	var conditions []string
	if !start.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= '%s'", start.Format("2006-01-02 15:04:05")))
	}
	if !end.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= '%s'", end.Format("2006-01-02 15:04:05")))
	}

	return strings.Join(conditions, " AND "), nil
}

// parseRelativeTimeRange converts relative time range to absolute times
func (s *Service) parseRelativeTimeRange(relative string) (time.Time, time.Time, error) {
	now := time.Now()
	var start time.Time

	switch relative {
	case "last_1h":
		start = now.Add(-1 * time.Hour)
	case "last_24h":
		start = now.Add(-24 * time.Hour)
	case "last_7d":
		start = now.Add(-7 * 24 * time.Hour)
	case "last_30d":
		start = now.Add(-30 * 24 * time.Hour)
	case "last_1m":
		start = now.Add(-1 * time.Minute)
	case "last_5m":
		start = now.Add(-5 * time.Minute)
	case "last_15m":
		start = now.Add(-15 * time.Minute)
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported relative time range: %s", relative)
	}

	return start, now, nil
}

// buildAggregationSQL builds SQL for aggregation functions
func (s *Service) buildAggregationSQL(agg models.QueryAggregation) (string, error) {
	alias := agg.Alias
	if alias == "" {
		alias = fmt.Sprintf("%s_%s", strings.ToLower(agg.Function), agg.Field)
	}

	switch agg.Function {
	case "COUNT":
		if agg.Field == "" {
			return fmt.Sprintf("COUNT(*) AS %s", alias), nil
		}
		return fmt.Sprintf("COUNT(%s) AS %s", agg.Field, alias), nil
	case "COUNT_DISTINCT":
		if agg.Field == "" {
			return "", fmt.Errorf("COUNT_DISTINCT requires a field")
		}
		return fmt.Sprintf("COUNT(DISTINCT %s) AS %s", agg.Field, alias), nil
	case "SUM", "AVG", "MIN", "MAX":
		if agg.Field == "" {
			return "", fmt.Errorf("%s requires a field", agg.Function)
		}
		return fmt.Sprintf("%s(%s) AS %s", agg.Function, agg.Field, alias), nil
	default:
		return "", fmt.Errorf("unsupported aggregation function: %s", agg.Function)
	}
}

// buildGroupByClause builds GROUP BY clause
func (s *Service) buildGroupByClause(groupBy []string) string {
	return strings.Join(groupBy, ", ")
}

// buildOrderByClause builds ORDER BY clause
func (s *Service) buildOrderByClause(orderBy []models.QueryOrderBy) string {
	var parts []string
	for _, order := range orderBy {
		parts = append(parts, fmt.Sprintf("%s %s", order.Field, order.Direction))
	}
	return strings.Join(parts, ", ")
}

// formatValue formats a value for SQL
func (s *Service) formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Escape single quotes
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case time.Time:
		return fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05"))
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// validateFilterOperator validates filter operators
func (s *Service) validateFilterOperator(operator string) error {
	validOperators := []string{
		"equals", "not_equals", "contains", "not_contains",
		"greater_than", "less_than", "greater_equal", "less_equal",
		"between", "in", "not_in", "is_null", "is_not_null",
	}

	for _, valid := range validOperators {
		if operator == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid operator: %s", operator)
}

// validateAggregationFunction validates aggregation functions
func (s *Service) validateAggregationFunction(function string) error {
	validFunctions := []string{"COUNT", "COUNT_DISTINCT", "SUM", "AVG", "MIN", "MAX"}

	for _, valid := range validFunctions {
		if function == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid aggregation function: %s", function)
}

// getAvailableFields returns the schema fields available for query building
func getAvailableFields() []models.QueryField {
	return []models.QueryField{
		{Name: "id", Type: "string", Label: "ID"},
		{Name: "timestamp", Type: "date", Label: "Timestamp"},
		{Name: "level", Type: "string", Label: "Log Level"},
		{Name: "message", Type: "string", Label: "Message"},
		{Name: "service", Type: "string", Label: "Service"},
		{Name: "trace_id", Type: "string", Label: "Trace ID"},
		{Name: "span_id", Type: "string", Label: "Span ID"},
		{Name: "raw_log", Type: "string", Label: "Raw Log"},
	}
}