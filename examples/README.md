# Click-Lite Log Analytics - Example Programs

This directory contains example programs demonstrating how to send logs to Click-Lite using various ingestion methods.

## Available Examples

### 1. HTTP Ingestion (`http/`)
- `simple.go` - Basic HTTP log ingestion
- `bulk.go` - Bulk log ingestion for high volume
- `stress_test.go` - Stress testing the HTTP endpoint

### 2. TCP Ingestion (`tcp/`)
- `client.go` - TCP client for streaming logs
- `json_client.go` - TCP client sending JSON formatted logs

### 3. Syslog Ingestion (`syslog/`)
- `logger.go` - Send logs using syslog protocol
- `test_syslog.sh` - Shell script using system logger command

### 4. Go Agent (`agent/`)
- `example.go` - Using the Click-Lite Go agent library
- `app_integration.go` - Integrating the agent into an application

## Prerequisites

1. Start the Click-Lite backend server:
```bash
cd backend
go run .
```

2. The server will start with:
   - HTTP API on port 20002
   - TCP receiver on port 20003
   - Syslog receiver on port 20004 (UDP)

## Running the Examples

Each example can be run independently. Navigate to the specific directory and run:

```bash
go run <example_file>.go
```

For shell scripts:
```bash
chmod +x <script>.sh
./<script>.sh
```

## Monitoring

You can monitor incoming logs through:
1. The web UI at http://localhost:3000
2. Direct API queries to http://localhost:20002/api/v1/logs
3. WebSocket connection for real-time streaming