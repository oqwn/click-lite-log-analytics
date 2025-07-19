# SQL Query Engine Examples

This directory contains examples demonstrating the SQL query capabilities of Click-Lite Log Analytics.

## Features

### SQL Query Engine
- Direct SQL access to ClickHouse
- Query validation and safety checks
- Query optimization for performance
- Parameter substitution
- Result formatting (JSON, CSV, TSV)

### Query Management
- Save and organize queries
- Query templates with parameters
- CRUD operations for saved queries
- Query categorization and tagging

### Built-in Query Templates
1. **Errors by Service** - Count errors grouped by service
2. **Log Level Distribution** - Log levels over time
3. **Slow Requests Analysis** - Find high response time requests
4. **Search by Trace ID** - Find all logs for a trace

## API Endpoints

### Execute SQL Query
```
POST /api/v1/query/execute
{
  "query": "SELECT * FROM logs WHERE level = 'error' LIMIT 10",
  "timeout": 30,
  "max_rows": 1000
}
```

### Save Query
```
POST /api/v1/query/saved
{
  "name": "My Query",
  "description": "Description",
  "query": "SELECT ...",
  "parameters": [...],
  "tags": ["tag1", "tag2"]
}
```

### Execute Saved Query
```
POST /api/v1/query/saved/{id}/execute
{
  "param1": "value1",
  "param2": "value2"
}
```

## Examples Included

- `sql_query_demo.go` - Basic SQL query execution
- `saved_queries_demo.go` - Managing saved queries
- `query_templates_demo.go` - Using query templates
- `advanced_queries_demo.go` - Complex analytical queries

## Running Examples

```bash
# Start the backend server
cd backend && go run main.go

# Run query examples
go run examples/query/sql_query_demo.go
go run examples/query/saved_queries_demo.go
go run examples/query/query_templates_demo.go
go run examples/query/advanced_queries_demo.go
```