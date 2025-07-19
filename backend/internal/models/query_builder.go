package models

import (
	"time"
)

// QueryBuilder represents a visual query builder configuration
type QueryBuilder struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Fields      []QueryField          `json:"fields"`
	Filters     []QueryBuilderFilter  `json:"filters"`
	Aggregations []QueryAggregation   `json:"aggregations"`
	GroupBy     []string              `json:"group_by"`
	OrderBy     []QueryOrderBy        `json:"order_by"`
	Limit       int                   `json:"limit,omitempty"`
	TimeRange   *QueryTimeRange       `json:"time_range,omitempty"`
	GeneratedSQL string               `json:"generated_sql,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	CreatedBy   string                `json:"created_by"`
}

// QueryField represents a selected field in the query
type QueryField struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // string, number, date, boolean
	Label    string `json:"label,omitempty"`
	Selected bool   `json:"selected"`
}

// QueryBuilderFilter represents a filter condition
type QueryBuilderFilter struct {
	ID       string      `json:"id"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // equals, not_equals, contains, not_contains, greater_than, less_than, between, in, not_in
	Value    interface{} `json:"value"`
	Values   []interface{} `json:"values,omitempty"` // for 'in', 'not_in', 'between'
	LogicalOp string     `json:"logical_op,omitempty"` // AND, OR
}

// QueryAggregation represents an aggregation function
type QueryAggregation struct {
	ID       string `json:"id"`
	Function string `json:"function"` // COUNT, SUM, AVG, MIN, MAX, COUNT_DISTINCT
	Field    string `json:"field,omitempty"`
	Alias    string `json:"alias,omitempty"`
}

// QueryOrderBy represents ordering
type QueryOrderBy struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // ASC, DESC
}

// QueryTimeRange represents time range filtering
type QueryTimeRange struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Relative string    `json:"relative,omitempty"` // last_1h, last_24h, last_7d, last_30d
}

// QueryBuilderResponse represents the result of executing a query builder
type QueryBuilderResponse struct {
	SQL          string                   `json:"sql"`
	Columns      []QueryResultColumn      `json:"columns"`
	Rows         []map[string]interface{} `json:"rows"`
	RowCount     int                      `json:"row_count"`
	ExecutionTime int64                   `json:"execution_time_ms"`
	Error        string                   `json:"error,omitempty"`
}

// QueryResultColumn represents metadata about result columns
type QueryResultColumn struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	IsAggregated bool   `json:"is_aggregated"`
}

// AvailableFields represents the schema information for query building
type AvailableFields struct {
	Fields []QueryField `json:"fields"`
}