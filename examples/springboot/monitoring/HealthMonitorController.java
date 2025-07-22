package com.example.monitoring;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.actuate.health.Health;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.Map;

/**
 * REST controller for health monitoring and manual metric triggers
 */
@RestController
@RequestMapping("/api/monitoring")
public class HealthMonitorController {
    
    @Autowired
    private SpringBootMonitor springBootMonitor;
    
    /**
     * Get current health status
     */
    @GetMapping("/health")
    public ResponseEntity<Map<String, Object>> getHealth() {
        Health health = springBootMonitor.health();
        
        Map<String, Object> response = new HashMap<>();
        response.put("status", health.getStatus().toString());
        response.put("details", health.getDetails());
        response.put("timestamp", System.currentTimeMillis());
        
        return ResponseEntity.ok(response);
    }
    
    /**
     * Get basic metrics
     */
    @GetMapping("/metrics")
    public ResponseEntity<Map<String, Object>> getMetrics() {
        Runtime runtime = Runtime.getRuntime();
        
        Map<String, Object> metrics = new HashMap<>();
        metrics.put("memory", Map.of(
            "max", runtime.maxMemory(),
            "total", runtime.totalMemory(),
            "free", runtime.freeMemory(),
            "used", runtime.totalMemory() - runtime.freeMemory(),
            "usage_percent", (double)(runtime.totalMemory() - runtime.freeMemory()) / runtime.maxMemory() * 100
        ));
        metrics.put("processors", runtime.availableProcessors());
        metrics.put("timestamp", System.currentTimeMillis());
        
        return ResponseEntity.ok(metrics);
    }
    
    /**
     * Force send health metrics (for testing)
     */
    @PostMapping("/force-health-check")
    public ResponseEntity<String> forceHealthCheck() {
        try {
            // This would typically be called by the monitoring component
            return ResponseEntity.ok("Health check triggered manually");
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body("Failed to trigger health check: " + e.getMessage());
        }
    }
    
    /**
     * Simulate error for testing
     */
    @PostMapping("/simulate-error")
    public ResponseEntity<String> simulateError() {
        springBootMonitor.recordError();
        return ResponseEntity.internalServerError().body("Simulated error for testing monitoring");
    }
}