package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
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

type Stats struct {
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	totalLatency    int64
}

func main() {
	endpoint := "http://localhost:20002/api/v1/ingest/logs"
	
	// Test parameters
	concurrentWorkers := 10
	logsPerWorker := 1000
	batchSize := 10
	
	fmt.Printf("Starting stress test:\n")
	fmt.Printf("- Workers: %d\n", concurrentWorkers)
	fmt.Printf("- Logs per worker: %d\n", logsPerWorker)
	fmt.Printf("- Batch size: %d\n", batchSize)
	fmt.Printf("- Total logs: %d\n", concurrentWorkers*logsPerWorker)
	fmt.Println()
	
	// Run stress test
	stats := &Stats{}
	start := time.Now()
	
	var wg sync.WaitGroup
	for i := 0; i < concurrentWorkers; i++ {
		wg.Add(1)
		go worker(i, endpoint, logsPerWorker, batchSize, stats, &wg)
	}
	
	// Monitor progress
	go monitorProgress(stats, start)
	
	wg.Wait()
	duration := time.Since(start)
	
	// Print final results
	fmt.Println("\n\nStress test completed!")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total requests: %d\n", atomic.LoadInt64(&stats.totalRequests))
	fmt.Printf("Successful requests: %d\n", atomic.LoadInt64(&stats.successRequests))
	fmt.Printf("Failed requests: %d\n", atomic.LoadInt64(&stats.failedRequests))
	fmt.Printf("Total logs sent: %d\n", concurrentWorkers*logsPerWorker)
	fmt.Printf("Logs/second: %.2f\n", float64(concurrentWorkers*logsPerWorker)/duration.Seconds())
	fmt.Printf("Average latency: %.2fms\n", float64(atomic.LoadInt64(&stats.totalLatency))/float64(atomic.LoadInt64(&stats.totalRequests)))
	
	// Run burst test
	fmt.Println("\n--- Running burst test ---")
	runBurstTest(endpoint)
}

func worker(id int, endpoint string, totalLogs, batchSize int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	for i := 0; i < totalLogs; i += batchSize {
		// Generate batch
		logs := make([]LogEntry, batchSize)
		for j := 0; j < batchSize; j++ {
			logs[j] = generateStressLog(id, i+j)
		}
		
		// Send batch
		start := time.Now()
		if err := sendBatch(client, endpoint, logs); err != nil {
			atomic.AddInt64(&stats.failedRequests, 1)
		} else {
			atomic.AddInt64(&stats.successRequests, 1)
		}
		latency := time.Since(start).Milliseconds()
		
		atomic.AddInt64(&stats.totalRequests, 1)
		atomic.AddInt64(&stats.totalLatency, latency)
		
		// Small delay to prevent overwhelming the server
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	}
}

func generateStressLog(workerID, index int) LogEntry {
	levels := []string{"debug", "info", "warn", "error"}
	services := []string{"stress-test", "worker", "loader", "generator"}
	
	return LogEntry{
		Timestamp: time.Now(),
		Level:     levels[rand.Intn(len(levels))],
		Message:   fmt.Sprintf("Stress test log from worker %d, index %d", workerID, index),
		Service:   services[rand.Intn(len(services))],
		TraceID:   fmt.Sprintf("stress-%d-%d", workerID, index/10),
		SpanID:    fmt.Sprintf("span-%d-%d", workerID, index),
		Attributes: map[string]interface{}{
			"worker_id":     workerID,
			"index":         index,
			"test_run":      time.Now().Unix(),
			"random_value":  rand.Float64(),
			"data_size":     rand.Intn(1000),
		},
	}
}

func sendBatch(client *http.Client, endpoint string, logs []LogEntry) error {
	data, err := json.Marshal(logs)
	if err != nil {
		return err
	}
	
	resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	return nil
}

func monitorProgress(stats *Stats, start time.Time) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		total := atomic.LoadInt64(&stats.totalRequests)
		success := atomic.LoadInt64(&stats.successRequests)
		failed := atomic.LoadInt64(&stats.failedRequests)
		elapsed := time.Since(start).Seconds()
		rate := float64(total) / elapsed
		
		fmt.Printf("\rProgress: %d requests (%.0f req/s), Success: %d, Failed: %d",
			total, rate, success, failed)
	}
}

func runBurstTest(endpoint string) {
	fmt.Println("Sending 10,000 logs in a single burst...")
	
	// Generate large batch
	logs := make([]LogEntry, 10000)
	for i := 0; i < 10000; i++ {
		logs[i] = generateStressLog(999, i)
	}
	
	// Use bulk endpoint for burst
	bulkEndpoint := "http://localhost:20002/api/v1/ingest/bulk"
	request := map[string]interface{}{
		"logs": logs,
		"options": map[string]bool{
			"skip_broadcast": true,
		},
	}
	
	data, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("Failed to marshal burst data: %v", err)
	}
	
	start := time.Now()
	resp, err := http.Post(bulkEndpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatalf("Failed to send burst: %v", err)
	}
	defer resp.Body.Close()
	
	duration := time.Since(start)
	
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
		fmt.Printf("Burst completed in %v\n", duration)
		fmt.Printf("Rate: %.0f logs/second\n", 10000/duration.Seconds())
	} else {
		fmt.Printf("Burst failed with status %d\n", resp.StatusCode)
	}
}