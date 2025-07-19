package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-username/click-lite-log-analytics/backend/pkg/agent"
)

func main() {
	// Create agent with custom configuration
	config := &agent.Config{
		Endpoint:      "http://localhost:20002/api/v1/ingest/logs",
		BatchSize:     50,
		FlushInterval: 3 * time.Second,
		MaxRetries:    3,
		Service:       "example-service",
		Attributes: map[string]interface{}{
			"hostname":    getHostname(),
			"environment": "development",
			"version":     "1.0.0",
		},
		HTTPTimeout: 5 * time.Second,
	}
	
	// Create and start the agent
	logAgent := agent.New(config)
	logAgent.Start()
	defer logAgent.Stop()
	
	fmt.Println("Click-Lite agent started")
	
	// Example 1: Basic logging
	logAgent.Debug("Debug message from agent")
	logAgent.Info("Application started successfully")
	logAgent.Warn("This is a warning message")
	logAgent.Error("This is an error message")
	
	// Example 2: Logging with additional fields
	logAgent.LogWithFields("info", "User logged in", map[string]interface{}{
		"user_id":    "user123",
		"ip_address": "192.168.1.100",
		"user_agent": "Mozilla/5.0...",
	})
	
	// Example 3: Error logging
	err := fmt.Errorf("connection timeout")
	logAgent.LogError(err, "Failed to connect to database")
	
	// Example 4: Simulate application behavior
	fmt.Println("\nSimulating application activity...")
	
	// Startup sequence
	logAgent.Info("Loading configuration")
	time.Sleep(100 * time.Millisecond)
	
	logAgent.Info("Connecting to database")
	time.Sleep(200 * time.Millisecond)
	
	logAgent.Info("Initializing cache")
	time.Sleep(100 * time.Millisecond)
	
	logAgent.Info("Starting HTTP server on port 8080")
	
	// Simulate request processing
	go simulateRequests(logAgent)
	
	// Simulate background jobs
	go simulateBackgroundJobs(logAgent)
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	fmt.Println("\nAgent is running. Press Ctrl+C to stop...")
	<-sigChan
	
	logAgent.Info("Shutting down application")
	fmt.Println("\nStopping agent...")
}

func simulateRequests(logAgent *agent.Agent) {
	endpoints := []string{"/api/users", "/api/orders", "/api/products", "/api/inventory"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	
	for {
		endpoint := endpoints[rand.Intn(len(endpoints))]
		method := methods[rand.Intn(len(methods))]
		requestID := fmt.Sprintf("req-%d", rand.Intn(100000))
		userID := fmt.Sprintf("user-%d", rand.Intn(1000))
		
		// Log request start
		logAgent.LogWithFields("info", "Incoming request", map[string]interface{}{
			"request_id": requestID,
			"method":     method,
			"endpoint":   endpoint,
			"user_id":    userID,
		})
		
		// Simulate processing time
		processingTime := rand.Intn(500)
		time.Sleep(time.Duration(processingTime) * time.Millisecond)
		
		// Randomly simulate success or failure
		if rand.Float32() < 0.95 { // 95% success rate
			statusCode := 200
			if method == "POST" {
				statusCode = 201
			}
			
			logAgent.LogWithFields("info", "Request completed", map[string]interface{}{
				"request_id":  requestID,
				"status_code": statusCode,
				"duration_ms": processingTime,
			})
		} else {
			statusCode := []int{400, 404, 500, 503}[rand.Intn(4)]
			
			logAgent.LogWithFields("error", "Request failed", map[string]interface{}{
				"request_id":  requestID,
				"status_code": statusCode,
				"duration_ms": processingTime,
				"error":       getErrorMessage(statusCode),
			})
		}
		
		// Random delay between requests
		time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)
	}
}

func simulateBackgroundJobs(logAgent *agent.Agent) {
	jobs := []string{"email-sender", "report-generator", "data-sync", "cleanup"}
	
	for {
		job := jobs[rand.Intn(len(jobs))]
		jobID := fmt.Sprintf("job-%d", rand.Intn(10000))
		
		// Log job start
		logAgent.LogWithFields("info", "Background job started", map[string]interface{}{
			"job_id":   jobID,
			"job_type": job,
		})
		
		// Simulate job execution
		executionTime := rand.Intn(10000) + 1000 // 1-11 seconds
		time.Sleep(time.Duration(executionTime) * time.Millisecond)
		
		// Randomly simulate success or failure
		if rand.Float32() < 0.9 { // 90% success rate
			logAgent.LogWithFields("info", "Background job completed", map[string]interface{}{
				"job_id":      jobID,
				"job_type":    job,
				"duration_ms": executionTime,
				"items_processed": rand.Intn(1000),
			})
		} else {
			logAgent.LogWithFields("error", "Background job failed", map[string]interface{}{
				"job_id":      jobID,
				"job_type":    job,
				"duration_ms": executionTime,
				"error":       "Processing error",
			})
		}
		
		// Random delay between jobs
		time.Sleep(time.Duration(rand.Intn(30)) * time.Second)
	}
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getErrorMessage(statusCode int) string {
	switch statusCode {
	case 400:
		return "Bad request"
	case 404:
		return "Resource not found"
	case 500:
		return "Internal server error"
	case 503:
		return "Service unavailable"
	default:
		return "Unknown error"
	}
}