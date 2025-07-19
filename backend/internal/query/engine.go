package query

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Engine manages SQL query execution and optimization
type Engine struct {
	db         QueryExecutor
	validator  *Validator
	optimizer  *Optimizer
	queryStore *QueryStore
}

// QueryExecutor interface for database operations
type QueryExecutor interface {
	ExecuteQuery(ctx context.Context, query string) ([]map[string]interface{}, error)
}

// QueryRequest represents a SQL query request
type QueryRequest struct {
	Query      string                 `json:"query"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Timeout    int                    `json:"timeout,omitempty"` // seconds
	MaxRows    int                    `json:"max_rows,omitempty"`
	Format     string                 `json:"format,omitempty"` // json, csv, tsv
}

// QueryResponse represents a SQL query response
type QueryResponse struct {
	Columns      []ColumnInfo           `json:"columns"`
	Rows         []map[string]interface{} `json:"rows"`
	RowCount     int                    `json:"row_count"`
	ExecutionTime int64                  `json:"execution_time_ms"`
	Query        string                 `json:"query"`
	Error        string                 `json:"error,omitempty"`
}

// ColumnInfo represents column metadata
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// NewEngine creates a new query engine
func NewEngine(db QueryExecutor) *Engine {
	return &Engine{
		db:         db,
		validator:  NewValidator(),
		optimizer:  NewOptimizer(),
		queryStore: NewQueryStore(),
	}
}

// Execute executes a SQL query with validation and optimization
func (e *Engine) Execute(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	start := time.Now()
	response := &QueryResponse{
		Query: req.Query,
	}

	// Apply default timeout if not specified
	if req.Timeout <= 0 {
		req.Timeout = 30 // 30 seconds default
	}
	
	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
	defer cancel()

	// Validate query
	if err := e.validator.Validate(req.Query); err != nil {
		response.Error = fmt.Sprintf("validation error: %v", err)
		return response, err
	}

	// Parameter substitution
	query, err := e.substituteParameters(req.Query, req.Parameters)
	if err != nil {
		response.Error = fmt.Sprintf("parameter error: %v", err)
		return response, err
	}

	// Optimize query
	query = e.optimizer.Optimize(query)

	// Apply row limit
	if req.MaxRows > 0 && !strings.Contains(strings.ToUpper(query), "LIMIT") {
		query = fmt.Sprintf("%s LIMIT %d", query, req.MaxRows)
	}

	// Execute query
	rows, err := e.db.ExecuteQuery(ctx, query)
	if err != nil {
		response.Error = fmt.Sprintf("execution error: %v", err)
		return response, err
	}

	// Process results
	response.Rows = rows
	response.RowCount = len(rows)
	
	// Extract column info from first row
	if len(rows) > 0 {
		response.Columns = make([]ColumnInfo, 0)
		for colName := range rows[0] {
			response.Columns = append(response.Columns, ColumnInfo{
				Name:     colName,
				Type:     "String", // Simplified - ClickHouse can provide actual types
				Nullable: false,
			})
		}
	}

	response.ExecutionTime = time.Since(start).Milliseconds()
	return response, nil
}


// substituteParameters replaces named parameters in the query
func (e *Engine) substituteParameters(query string, params map[string]interface{}) (string, error) {
	if params == nil || len(params) == 0 {
		return query, nil
	}

	// Find all parameter placeholders like :param_name or ${param_name}
	re := regexp.MustCompile(`:(\w+)|\$\{(\w+)\}`)
	
	result := re.ReplaceAllStringFunc(query, func(match string) string {
		// Extract parameter name
		paramName := ""
		if strings.HasPrefix(match, ":") {
			paramName = match[1:]
		} else {
			paramName = match[2 : len(match)-1]
		}
		
		// Get parameter value
		if value, exists := params[paramName]; exists {
			return e.formatParameterValue(value)
		}
		
		// Return original if parameter not found
		return match
	})
	
	return result, nil
}

// formatParameterValue formats a parameter value for SQL
func (e *Engine) formatParameterValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Escape single quotes and wrap in quotes
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
	case nil:
		return "NULL"
	default:
		// Try JSON encoding for complex types
		if data, err := json.Marshal(v); err == nil {
			return fmt.Sprintf("'%s'", strings.ReplaceAll(string(data), "'", "''"))
		}
		return fmt.Sprintf("%v", v)
	}
}

// convertValue converts database values to appropriate Go types
func convertValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	case nil:
		return nil
	default:
		return val
	}
}

// GetQueryStore returns the query store
func (e *Engine) GetQueryStore() *QueryStore {
	return e.queryStore
}