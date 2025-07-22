package com.example.monitoring;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.actuate.health.Health;
import org.springframework.boot.actuate.health.HealthIndicator;
import org.springframework.boot.actuate.metrics.MetricsEndpoint;
import org.springframework.stereotype.Component;

import javax.annotation.PostConstruct;
import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Instant;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * Spring Boot monitoring component that sends metrics and health data 
 * to Click-Lite Log Analytics platform
 */
@Component
public class SpringBootMonitor implements HealthIndicator {
    
    private static final Logger logger = LoggerFactory.getLogger(SpringBootMonitor.class);
    
    private final String analyticsEndpoint = "http://localhost:20002/api/v1/ingest/logs";
    private final String serviceName = "spring-boot-app";
    private final HttpClient httpClient;
    private final ObjectMapper objectMapper;
    private final ScheduledExecutorService scheduler;
    
    // Metrics collection
    private long requestCount = 0;
    private long errorCount = 0;
    private double responseTimeSum = 0;
    private int responseTimeCount = 0;
    
    public SpringBootMonitor() {
        this.httpClient = HttpClient.newHttpClient();
        this.objectMapper = new ObjectMapper();
        this.scheduler = Executors.newScheduledThreadPool(2);
    }
    
    @PostConstruct
    public void startMonitoring() {
        logger.info("Starting Spring Boot monitoring for service: {}", serviceName);
        
        // Send health check every 30 seconds
        scheduler.scheduleAtFixedRate(this::sendHealthMetrics, 0, 30, TimeUnit.SECONDS);
        
        // Send performance metrics every 60 seconds
        scheduler.scheduleAtFixedRate(this::sendPerformanceMetrics, 10, 60, TimeUnit.SECONDS);
        
        // Send system metrics every 2 minutes
        scheduler.scheduleAtFixedRate(this::sendSystemMetrics, 20, 120, TimeUnit.SECONDS);
    }
    
    @Override
    public Health health() {
        try {
            // Basic health check logic
            Runtime runtime = Runtime.getRuntime();
            long maxMemory = runtime.maxMemory();
            long totalMemory = runtime.totalMemory();
            long freeMemory = runtime.freeMemory();
            long usedMemory = totalMemory - freeMemory;
            
            double memoryUsagePercent = (double) usedMemory / maxMemory * 100;
            
            if (memoryUsagePercent > 90) {
                return Health.down()
                    .withDetail("memory_usage_percent", memoryUsagePercent)
                    .withDetail("reason", "High memory usage")
                    .build();
            }
            
            return Health.up()
                .withDetail("memory_usage_percent", memoryUsagePercent)
                .withDetail("used_memory_mb", usedMemory / 1024 / 1024)
                .withDetail("max_memory_mb", maxMemory / 1024 / 1024)
                .build();
                
        } catch (Exception e) {
            return Health.down()
                .withDetail("error", e.getMessage())
                .build();
        }
    }
    
    public void recordRequest(long responseTimeMs) {
        requestCount++;
        responseTimeSum += responseTimeMs;
        responseTimeCount++;
    }
    
    public void recordError() {
        errorCount++;
    }
    
    private void sendHealthMetrics() {
        try {
            Health health = health();
            
            Map<String, Object> logEntry = createBaseLogEntry();
            logEntry.put("message", "Health check: " + health.getStatus());
            logEntry.put("level", health.getStatus().toString().equals("UP") ? "INFO" : "WARN");
            
            Map<String, String> attributes = (Map<String, String>) logEntry.get("attributes");
            attributes.put("metric_type", "health_check");
            attributes.put("health_status", health.getStatus().toString());
            
            // Add health details
            if (health.getDetails() != null) {
                health.getDetails().forEach((key, value) -> 
                    attributes.put("health_" + key, String.valueOf(value)));
            }
            
            sendLogAsync(logEntry);
            
        } catch (Exception e) {
            logger.error("Failed to send health metrics", e);
        }
    }
    
    private void sendPerformanceMetrics() {
        try {
            Map<String, Object> logEntry = createBaseLogEntry();
            logEntry.put("message", String.format("Performance metrics - Requests: %d, Errors: %d", 
                requestCount, errorCount));
            logEntry.put("level", "INFO");
            
            Map<String, String> attributes = (Map<String, String>) logEntry.get("attributes");
            attributes.put("metric_type", "performance");
            attributes.put("request_count", String.valueOf(requestCount));
            attributes.put("error_count", String.valueOf(errorCount));
            attributes.put("error_rate", String.valueOf(requestCount > 0 ? (double) errorCount / requestCount : 0));
            
            if (responseTimeCount > 0) {
                double avgResponseTime = responseTimeSum / responseTimeCount;
                attributes.put("avg_response_time_ms", String.valueOf(avgResponseTime));
            }
            
            sendLogAsync(logEntry);
            
            // Reset counters (optional - you might want to keep cumulative)
            // responseTimeSum = 0;
            // responseTimeCount = 0;
            
        } catch (Exception e) {
            logger.error("Failed to send performance metrics", e);
        }
    }
    
    private void sendSystemMetrics() {
        try {
            Runtime runtime = Runtime.getRuntime();
            long maxMemory = runtime.maxMemory();
            long totalMemory = runtime.totalMemory();
            long freeMemory = runtime.freeMemory();
            long usedMemory = totalMemory - freeMemory;
            
            Map<String, Object> logEntry = createBaseLogEntry();
            logEntry.put("message", String.format("System metrics - Memory usage: %.2f%%", 
                (double) usedMemory / maxMemory * 100));
            logEntry.put("level", "INFO");
            
            Map<String, String> attributes = (Map<String, String>) logEntry.get("attributes");
            attributes.put("metric_type", "system");
            attributes.put("max_memory_bytes", String.valueOf(maxMemory));
            attributes.put("total_memory_bytes", String.valueOf(totalMemory));
            attributes.put("free_memory_bytes", String.valueOf(freeMemory));
            attributes.put("used_memory_bytes", String.valueOf(usedMemory));
            attributes.put("memory_usage_percent", String.valueOf((double) usedMemory / maxMemory * 100));
            attributes.put("available_processors", String.valueOf(runtime.availableProcessors()));
            
            sendLogAsync(logEntry);
            
        } catch (Exception e) {
            logger.error("Failed to send system metrics", e);
        }
    }
    
    private Map<String, Object> createBaseLogEntry() {
        Map<String, Object> logEntry = new HashMap<>();
        logEntry.put("timestamp", DateTimeFormatter.ISO_INSTANT.format(Instant.now()));
        logEntry.put("service", serviceName);
        logEntry.put("trace_id", "");
        logEntry.put("span_id", "");
        
        Map<String, String> attributes = new HashMap<>();
        attributes.put("component", "monitoring");
        attributes.put("port", "20005");
        attributes.put("host", getHostname());
        
        logEntry.put("attributes", attributes);
        
        return logEntry;
    }
    
    private String getHostname() {
        try {
            return java.net.InetAddress.getLocalHost().getHostName();
        } catch (Exception e) {
            return "localhost";
        }
    }
    
    private void sendLogAsync(Map<String, Object> logEntry) {
        scheduler.submit(() -> {
            try {
                String json = objectMapper.writeValueAsString(logEntry);
                
                HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(analyticsEndpoint))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(json))
                    .build();
                
                httpClient.sendAsync(request, HttpResponse.BodyHandlers.ofString())
                    .thenAccept(response -> {
                        if (response.statusCode() != 202) {
                            logger.warn("Failed to send metrics, status: {}, body: {}", 
                                response.statusCode(), response.body());
                        }
                    })
                    .exceptionally(throwable -> {
                        logger.error("Exception sending metrics: {}", throwable.getMessage());
                        return null;
                    });
                    
            } catch (Exception e) {
                logger.error("Failed to serialize metrics: {}", e.getMessage());
            }
        });
    }
}