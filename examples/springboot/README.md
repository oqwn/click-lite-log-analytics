# Spring Boot Integration with Click-Lite Log Analytics

This directory contains the integration components to send Spring Boot application logs to your Click-Lite Log Analytics platform.

## Setup Instructions

### 1. Add Dependencies to your Spring Boot project

Add these dependencies to your `pom.xml`:

```xml
<dependencies>
    <!-- Required for custom appender -->
    <dependency>
        <groupId>ch.qos.logback</groupId>
        <artifactId>logback-classic</artifactId>
    </dependency>
    <dependency>
        <groupId>com.fasterxml.jackson.core</groupId>
        <artifactId>jackson-databind</artifactId>
    </dependency>
</dependencies>
```

### 2. Add the Custom Appender

1. Copy `LogAnalyticsAppender.java` to your Spring Boot project:
   ```
   src/main/java/com/example/loganalytics/LogAnalyticsAppender.java
   ```

2. Copy `logback-spring.xml` to your resources directory:
   ```
   src/main/resources/logback-spring.xml
   ```

### 3. Configuration

Update the `logback-spring.xml` configuration:

```xml
<appender name="LOG_ANALYTICS" class="com.example.loganalytics.LogAnalyticsAppender">
    <endpoint>http://localhost:20002/api/v1/ingest/logs</endpoint>
    <serviceName>your-spring-app-name</serviceName>
</appender>
```

### 4. Test the Integration

Start your Spring Boot application and check that logs are being sent:

```bash
# Test by making a request to your Spring Boot app
curl -X GET http://localhost:20005/actuator/health

# Check logs in Click-Lite Analytics frontend
# Navigate to: http://localhost:3000/logs
```

### 5. Query Your Spring Boot Logs

Use the frontend query builder or API to query your logs:

```sql
-- Find all ERROR logs from your Spring Boot app
SELECT timestamp, level, message, attributes 
FROM logs 
WHERE service = 'spring-boot-app' 
  AND level = 'ERROR'
  AND timestamp >= now() - INTERVAL 1 HOUR
ORDER BY timestamp DESC;

-- Find logs by specific logger
SELECT timestamp, message, attributes
FROM logs 
WHERE service = 'spring-boot-app'
  AND attributes['logger'] LIKE '%Controller%'
ORDER BY timestamp DESC
LIMIT 100;

-- Trace-based queries (if using distributed tracing)
SELECT timestamp, level, message, span_id
FROM logs 
WHERE trace_id = 'your-trace-id'
ORDER BY timestamp ASC;
```

### 6. Dashboard Integration

Create dashboards with widgets for:
- Error rate over time
- Log level distribution  
- Request patterns
- Performance metrics

## Features

- **Asynchronous logging**: Won't block your application
- **Structured data**: Includes MDC properties, thread info, exceptions
- **Trace correlation**: Supports distributed tracing with trace/span IDs
- **Error handling**: Robust error handling with fallback logging
- **Configurable**: Easy to customize endpoint and service name

## Troubleshooting

1. **Logs not appearing**: Check that your analytics backend is running on port 20002
2. **Performance issues**: Adjust queue size in async appender configuration
3. **Connection errors**: Verify endpoint URL and network connectivity

```bash
# Test connectivity to analytics platform
curl -X POST http://localhost:20002/api/v1/ingest/logs \
  -H "Content-Type: application/json" \
  -d '{"level":"INFO","message":"test","service":"spring-boot-app"}'
```