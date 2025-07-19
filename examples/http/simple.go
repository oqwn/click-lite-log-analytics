package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Service    string                 `json:"service"`
	TraceID    string                 `json:"trace_id,omitempty"`
	SpanID     string                 `json:"span_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

func main() {
	endpoint := "http://localhost:20002/api/v1/ingest/logs"
	
	// Example 1: Send a single log
	log1 := LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "Application started successfully",
		Service:   "example-app",
		Attributes: map[string]interface{}{
			"version": "1.0.0",
			"env":     "development",
		},
	}
	
	if err := sendLog(endpoint, log1); err != nil {
		log.Printf("Failed to send log: %v", err)
	} else {
		fmt.Println("Successfully sent single log")
	}
	
	// Example 2: Send multiple logs
	logs := []LogEntry{
		{
			Timestamp: time.Now(),
			Level:     "debug",
			Message:   "Database connection established",
			Service:   "example-app",
			TraceID:   "abc123",
			Attributes: map[string]interface{}{
				"db_host": "localhost",
				"db_name": "myapp",
			},
		},
		{
			Timestamp: time.Now(),
			Level:     "warn",
			Message:   "Cache miss for key: user:123",
			Service:   "example-app",
			TraceID:   "abc123",
			SpanID:    "span456",
			Attributes: map[string]interface{}{
				"cache_key": "user:123",
				"latency_ms": 45,
			},
		},
		{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "Failed to process payment",
			Service:   "example-app",
			TraceID:   "xyz789",
			Attributes: map[string]interface{}{
				"error_code": "PAYMENT_DECLINED",
				"amount": 99.99,
				"currency": "USD",
			},
		},
	}
	
	if err := sendLogs(endpoint, logs); err != nil {
		log.Printf("Failed to send logs: %v", err)
	} else {
		fmt.Printf("Successfully sent %d logs\n", len(logs))
	}
	
	// Example 3: Send logs with different levels
	levels := []string{"debug", "info", "warn", "error", "fatal"}
	for i, level := range levels {
		logEntry := LogEntry{
			Timestamp: time.Now(),
			Level:     level,
			Message:   fmt.Sprintf("This is a %s level log message", level),
			Service:   "example-app",
			Attributes: map[string]interface{}{
				"index": i,
				"test": true,
			},
		}
		
		if err := sendLog(endpoint, logEntry); err != nil {
			log.Printf("Failed to send %s log: %v", level, err)
		} else {
			fmt.Printf("Sent %s log\n", level)
		}
		
		time.Sleep(100 * time.Millisecond)
	}
}

func sendLog(endpoint string, log LogEntry) error {
	data, err := json.Marshal(log)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	return nil
}

func sendLogs(endpoint string, logs []LogEntry) error {
	data, err := json.Marshal(logs)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	return nil
}