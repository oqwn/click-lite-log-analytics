package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type BulkRequest struct {
	Logs    []LogEntry `json:"logs"`
	Options struct {
		SkipBroadcast bool `json:"skip_broadcast"`
	} `json:"options"`
}

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
	endpoint := "http://localhost:20002/api/v1/ingest/bulk"
	
	// Generate a large batch of logs
	batchSize := 1000
	logs := generateLogs(batchSize)
	
	fmt.Printf("Sending %d logs in bulk...\n", batchSize)
	start := time.Now()
	
	request := BulkRequest{
		Logs: logs,
	}
	request.Options.SkipBroadcast = true // Skip WebSocket broadcast for bulk
	
	if err := sendBulkLogs(endpoint, request); err != nil {
		log.Fatalf("Failed to send bulk logs: %v", err)
	}
	
	duration := time.Since(start)
	fmt.Printf("Successfully sent %d logs in %v\n", batchSize, duration)
	fmt.Printf("Rate: %.2f logs/second\n", float64(batchSize)/duration.Seconds())
	
	// Send smaller batches continuously
	fmt.Println("\nStarting continuous bulk sending (press Ctrl+C to stop)...")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		smallBatch := generateLogs(100)
		request := BulkRequest{
			Logs: smallBatch,
		}
		
		if err := sendBulkLogs(endpoint, request); err != nil {
			log.Printf("Failed to send batch: %v", err)
		} else {
			fmt.Printf("Sent batch of %d logs at %s\n", len(smallBatch), time.Now().Format("15:04:05"))
		}
	}
}

func generateLogs(count int) []LogEntry {
	levels := []string{"debug", "info", "warn", "error"}
	services := []string{"api-gateway", "user-service", "order-service", "payment-service", "inventory-service"}
	messages := []string{
		"Request processed successfully",
		"Cache hit for key",
		"Database query executed",
		"External API call completed",
		"Background job started",
		"Queue message processed",
		"Configuration reloaded",
		"Health check passed",
		"Metric recorded",
		"Event published",
	}
	
	logs := make([]LogEntry, count)
	
	for i := 0; i < count; i++ {
		logs[i] = LogEntry{
			Timestamp: time.Now().Add(time.Duration(-count+i) * time.Second),
			Level:     levels[rand.Intn(len(levels))],
			Message:   messages[rand.Intn(len(messages))],
			Service:   services[rand.Intn(len(services))],
			TraceID:   fmt.Sprintf("trace-%d", rand.Intn(1000)),
			SpanID:    fmt.Sprintf("span-%d", rand.Intn(10000)),
			Attributes: map[string]interface{}{
				"user_id":     fmt.Sprintf("user-%d", rand.Intn(1000)),
				"request_id":  fmt.Sprintf("req-%d", rand.Intn(100000)),
				"latency_ms":  rand.Intn(500),
				"status_code": 200 + rand.Intn(5)*100,
				"method":      []string{"GET", "POST", "PUT", "DELETE"}[rand.Intn(4)],
				"path":        fmt.Sprintf("/api/v1/%s", []string{"users", "orders", "products"}[rand.Intn(3)]),
			},
		}
	}
	
	return logs
}

func sendBulkLogs(endpoint string, request BulkRequest) error {
	data, err := json.Marshal(request)
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