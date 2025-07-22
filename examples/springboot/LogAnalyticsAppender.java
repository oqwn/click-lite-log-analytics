package com.example.loganalytics;

import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.AppenderBase;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Instant;
import java.time.format.DateTimeFormatter;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;

/**
 * Custom Logback appender for sending logs to Click-Lite Log Analytics
 */
public class LogAnalyticsAppender extends AppenderBase<ILoggingEvent> {
    
    private String endpoint = "http://localhost:20002/api/v1/ingest/logs";
    private String serviceName = "spring-boot-app";
    private HttpClient httpClient;
    private ObjectMapper objectMapper;
    private ScheduledExecutorService executor;
    
    @Override
    public void start() {
        this.httpClient = HttpClient.newHttpClient();
        this.objectMapper = new ObjectMapper();
        this.executor = Executors.newScheduledThreadPool(2);
        super.start();
    }
    
    @Override
    public void stop() {
        if (executor != null && !executor.isShutdown()) {
            executor.shutdown();
        }
        super.stop();
    }
    
    @Override
    protected void append(ILoggingEvent event) {
        try {
            // Create log entry map
            Map<String, Object> logEntry = new HashMap<>();
            logEntry.put("timestamp", DateTimeFormatter.ISO_INSTANT.format(Instant.ofEpochMilli(event.getTimeStamp())));
            logEntry.put("level", event.getLevel().toString());
            logEntry.put("message", event.getFormattedMessage());
            logEntry.put("service", serviceName);
            logEntry.put("trace_id", event.getMDCPropertyMap().getOrDefault("traceId", ""));
            logEntry.put("span_id", event.getMDCPropertyMap().getOrDefault("spanId", ""));
            
            // Add attributes
            Map<String, String> attributes = new HashMap<>();
            attributes.put("logger", event.getLoggerName());
            attributes.put("thread", event.getThreadName());
            
            // Add exception if present
            if (event.getThrowableProxy() != null) {
                attributes.put("exception", event.getThrowableProxy().getClassName());
                attributes.put("exception_message", event.getThrowableProxy().getMessage());
            }
            
            // Add MDC properties
            event.getMDCPropertyMap().forEach(attributes::put);
            
            logEntry.put("attributes", attributes);
            
            // Send asynchronously
            sendLogAsync(logEntry);
            
        } catch (Exception e) {
            addError("Failed to send log to analytics platform: " + e.getMessage(), e);
        }
    }
    
    private void sendLogAsync(Map<String, Object> logEntry) {
        CompletableFuture.runAsync(() -> {
            try {
                String json = objectMapper.writeValueAsString(logEntry);
                
                HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(endpoint))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(json))
                    .build();
                
                httpClient.sendAsync(request, HttpResponse.BodyHandlers.ofString())
                    .thenAccept(response -> {
                        if (response.statusCode() != 202) {
                            addError("Failed to send log, status: " + response.statusCode() + 
                                   ", body: " + response.body());
                        }
                    })
                    .exceptionally(throwable -> {
                        addError("Exception sending log: " + throwable.getMessage(), throwable);
                        return null;
                    });
                    
            } catch (Exception e) {
                addError("Failed to serialize log: " + e.getMessage(), e);
            }
        }, executor);
    }
    
    // Getters and setters for configuration
    public void setEndpoint(String endpoint) {
        this.endpoint = endpoint;
    }
    
    public void setServiceName(String serviceName) {
        this.serviceName = serviceName;
    }
}