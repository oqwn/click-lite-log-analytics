# Spring Boot Monitoring Setup Guide

Complete step-by-step guide to monitor your Spring Boot application (port 20005) using Click-Lite Log Analytics.

## 📋 Prerequisites

- Spring Boot application running on port 20005
- Click-Lite Log Analytics backend running on port 20002
- Click-Lite frontend running on port 3000

## 🔧 Step 1: Add Dependencies

Add these dependencies to your Spring Boot `pom.xml`:

```xml
<dependencies>
    <!-- Spring Boot Actuator for health checks -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-actuator</artifactId>
    </dependency>
    
    <!-- Jackson for JSON processing -->
    <dependency>
        <groupId>com.fasterxml.jackson.core</groupId>
        <artifactId>jackson-databind</artifactId>
    </dependency>
    
    <!-- Logback for logging -->
    <dependency>
        <groupId>ch.qos.logback</groupId>
        <artifactId>logback-classic</artifactId>
    </dependency>
</dependencies>
```

## 📂 Step 2: Copy Monitoring Files

Copy these files to your Spring Boot project:

```bash
# Create monitoring package
mkdir -p src/main/java/com/example/monitoring

# Copy monitoring files
cp examples/springboot/monitoring/*.java src/main/java/com/example/monitoring/
cp examples/springboot/logback-spring.xml src/main/resources/
```

## ⚙️ Step 3: Configuration

### 3.1 Update `application.yml` or `application.properties`:

```yaml
# application.yml
management:
  endpoints:
    web:
      exposure:
        include: health,metrics,info
  endpoint:
    health:
      show-details: always
      
logging:
  level:
    com.example.monitoring: DEBUG
    org.springframework.web: INFO
```

### 3.2 Update package names in Java files:

Replace `com.example` with your actual package name in all monitoring files.

## 🚀 Step 4: Start Monitoring

### 4.1 Restart your Spring Boot application

```bash
# Your Spring Boot application should now have monitoring enabled
# Check that it's still running on port 20005
```

### 4.2 Verify monitoring endpoints:

```bash
# Test health endpoint
curl http://localhost:20005/api/monitoring/health

# Test metrics endpoint  
curl http://localhost:20005/api/monitoring/metrics

# Force a health check
curl -X POST http://localhost:20005/api/monitoring/force-health-check
```

## 📊 Step 5: Create Dashboard

### 5.1 Import dashboard configuration:

1. Go to your Click-Lite frontend: `http://localhost:3000`
2. Navigate to **Dashboards** → **Create New Dashboard**
3. Import the configuration from `examples/springboot/dashboard-config.json`

### 5.2 Or create dashboard manually:

1. Go to `http://localhost:3000/dashboards`
2. Click **"Create New Dashboard"**
3. Add these widgets:

#### Health Status Widget (Metric):
```sql
SELECT COUNT(*) as healthy_checks
FROM logs 
WHERE service = 'spring-boot-app' 
  AND attributes['metric_type'] = 'health_check'
  AND attributes['health_status'] = 'UP'
  AND timestamp >= now() - INTERVAL 5 MINUTE
```

#### Memory Usage Chart:
```sql
SELECT 
  toStartOfMinute(timestamp) as time,
  AVG(toFloat64(attributes['memory_usage_percent'])) as memory_percent
FROM logs 
WHERE service = 'spring-boot-app'
  AND attributes['metric_type'] = 'system'
  AND timestamp >= now() - INTERVAL 1 HOUR
GROUP BY time 
ORDER BY time
```

#### Request Count Metric:
```sql
SELECT MAX(toInt64(attributes['request_count'])) as total_requests
FROM logs 
WHERE service = 'spring-boot-app'
  AND attributes['metric_type'] = 'performance'
  AND timestamp >= now() - INTERVAL 1 HOUR
```

## 🧪 Step 6: Test the Setup

### 6.1 Generate some test traffic:

```bash
# Make some requests to your Spring Boot app
curl http://localhost:20005/api/monitoring/health
curl http://localhost:20005/api/monitoring/metrics

# Generate an error for testing
curl -X POST http://localhost:20005/api/monitoring/simulate-error

# Generate normal traffic (replace with your actual endpoints)
for i in {1..10}; do
  curl http://localhost:20005/actuator/health
  sleep 1
done
```

### 6.2 Check logs are arriving:

```bash
# Check that logs are being received by analytics platform
curl -X POST http://localhost:20002/api/v1/query/execute \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT COUNT(*) FROM logs WHERE service = '\''spring-boot-app'\'' AND timestamp >= now() - INTERVAL 10 MINUTE"}'
```

### 6.3 View in frontend:

1. **Live Logs**: Go to `http://localhost:3000/logs` 
   - Filter by service: `spring-boot-app`
   - You should see health checks, performance metrics, and request logs

2. **Dashboard**: Go to `http://localhost:3000/dashboards`
   - Open your Spring Boot monitoring dashboard
   - Verify all widgets are showing data

3. **Query Builder**: Go to `http://localhost:3000/query-builder`
   - Try custom queries to explore your data

## 📈 What You'll Monitor

### Automatic Monitoring (every 30-120 seconds):
- **Health Status**: Application health, memory usage
- **Performance Metrics**: Request count, error count, response times  
- **System Metrics**: Memory usage, CPU info, JVM stats

### Request-Level Monitoring:
- **Request Logs**: All HTTP requests with response times
- **Error Tracking**: Automatic error detection and logging
- **Custom Metrics**: Any additional metrics you log

### Dashboard Views:
- Real-time health status
- Memory usage trends
- Request volume over time
- Error rates and recent errors
- System resource utilization

## 🔍 Advanced Queries

Once data is flowing, try these queries in the Query Builder:

```sql
-- Find high memory usage periods
SELECT timestamp, attributes['memory_usage_percent'] as memory_pct
FROM logs 
WHERE service = 'spring-boot-app' 
  AND toFloat64(attributes['memory_usage_percent']) > 80
ORDER BY timestamp DESC;

-- Request performance analysis
SELECT 
  toStartOfHour(timestamp) as hour,
  AVG(toFloat64(attributes['avg_response_time_ms'])) as avg_response_time,
  COUNT(*) as request_count
FROM logs 
WHERE service = 'spring-boot-app' 
  AND attributes['metric_type'] = 'performance'
GROUP BY hour 
ORDER BY hour DESC;

-- Error pattern analysis
SELECT 
  level,
  COUNT(*) as error_count,
  attributes['exception'] as exception_type
FROM logs 
WHERE service = 'spring-boot-app' 
  AND level IN ('ERROR', 'WARN')
  AND timestamp >= now() - INTERVAL 24 HOUR
GROUP BY level, exception_type
ORDER BY error_count DESC;
```

## 🚨 Troubleshooting

### Issue: No logs appearing
```bash
# Check Spring Boot app is sending logs
curl http://localhost:20005/api/monitoring/health

# Check analytics platform is receiving
curl -X POST http://localhost:20002/api/v1/ingest/logs \
  -H "Content-Type: application/json" \
  -d '{"level":"INFO","message":"test","service":"spring-boot-app"}'
```

### Issue: Dashboard not showing data
- Wait 1-2 minutes for initial data collection
- Check query syntax in dashboard widgets
- Verify service name matches (`spring-boot-app`)

### Issue: High resource usage
- Adjust monitoring intervals in `SpringBootMonitor.java`
- Reduce log retention in ClickHouse
- Disable WebSocket broadcasting for bulk operations

## ✅ Success Indicators

You'll know monitoring is working when you see:

1. **Backend logs**: Spring Boot app logs in Click-Lite analytics
2. **Health metrics**: Regular health check entries every 30 seconds
3. **Performance data**: Request counts and response times every minute  
4. **Dashboard data**: All dashboard widgets showing current data
5. **Real-time updates**: Live log streaming in the frontend

Your Spring Boot application is now fully monitored! 🎉