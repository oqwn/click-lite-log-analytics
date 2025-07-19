# Log Parsing Examples

This directory contains examples demonstrating the log parsing capabilities of the Click-Lite Log Analytics system.

## Parsers Available

### 1. JSON Parser
- Handles structured JSON logs
- Automatically extracts standard fields (timestamp, level, message, service, etc.)
- Maps remaining fields to attributes
- Supports multiple timestamp formats and field name variations

### 2. Regex Parser
- Handles unstructured logs using configurable regex patterns
- Includes built-in patterns for common log formats:
  - Apache Combined/Common Log Format
  - Nginx Access Logs
  - Syslog RFC3164
  - Application logs (Spring Boot, Docker, etc.)
  - Generic timestamped logs
- Supports custom regex patterns with field mappings

### 3. Configurable Rules
- Validation rules for data quality
- Transformation rules for data normalization
- Field mappings for standardization
- Default values for missing fields
- Field constraints with type checking

## Examples Included

- `json_parsing_demo.go` - Demonstrates JSON log parsing
- `regex_parsing_demo.go` - Shows regex pattern matching for various log formats
- `custom_rules_demo.go` - Illustrates custom parsing rules and validation
- `parsing_benchmark_demo.go` - Performance testing for different parsers

## Running Examples

```bash
# Start the backend server first
go run ../backend/main.go

# Run parsing examples
go run examples/parsing/json_parsing_demo.go
go run examples/parsing/regex_parsing_demo.go
go run examples/parsing/custom_rules_demo.go
go run examples/parsing/parsing_benchmark_demo.go
```

## Features Demonstrated

- ✅ Automatic parser selection based on log format
- ✅ Data validation and transformation
- ✅ Field mapping and normalization
- ✅ Error handling and fallback mechanisms
- ✅ Performance metrics and statistics
- ✅ Custom rule configuration