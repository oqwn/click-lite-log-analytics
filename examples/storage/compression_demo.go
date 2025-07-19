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

type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Service    string                 `json:"service"`
	TraceID    string                 `json:"trace_id,omitempty"`
	SpanID     string                 `json:"span_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type StorageStats struct {
	TotalRows        int64   `json:"total_rows"`
	CompressedSize   string  `json:"compressed_size"`
	UncompressedSize string  `json:"uncompressed_size"`
	CompressionRatio float64 `json:"compression_ratio"`
	PartitionCount   int64   `json:"partition_count"`
	OldestDate       string  `json:"oldest_date"`
	NewestDate       string  `json:"newest_date"`
}

func main() {
	endpoint := "http://localhost:20002/api/v1/ingest/bulk"
	statsEndpoint := "http://localhost:20002/api/v1/storage/stats"
	
	fmt.Println("🗜️  Compression Test - Testing storage efficiency")
	fmt.Println("============================================================")
	
	// Get initial stats
	initialStats, err := getStorageStats(statsEndpoint)
	if err != nil {
		log.Printf("Warning: Could not get initial stats: %v", err)
	} else {
		fmt.Printf("📊 Initial Stats:\n")
		printStats(initialStats)
		fmt.Println()
	}
	
	// Test 1: Small messages (high compression expected)
	fmt.Println("🧪 Test 1: Small repetitive messages (high compression expected)")
	smallMessages := generateSmallMessages(5000)
	if err := sendBulkLogs(endpoint, smallMessages); err != nil {
		log.Fatalf("Failed to send small messages: %v", err)
	}
	fmt.Printf("✅ Sent %d small messages\n", len(smallMessages))
	
	time.Sleep(2 * time.Second) // Allow processing
	
	// Test 2: Large diverse messages (lower compression expected)
	fmt.Println("\n🧪 Test 2: Large diverse messages (lower compression expected)")
	largeMessages := generateLargeMessages(1000)
	if err := sendBulkLogs(endpoint, largeMessages); err != nil {
		log.Fatalf("Failed to send large messages: %v", err)
	}
	fmt.Printf("✅ Sent %d large messages\n", len(largeMessages))
	
	time.Sleep(2 * time.Second) // Allow processing
	
	// Test 3: JSON-heavy messages (medium compression expected)
	fmt.Println("\n🧪 Test 3: JSON-heavy structured messages (medium compression expected)")
	jsonMessages := generateJSONMessages(2000)
	if err := sendBulkLogs(endpoint, jsonMessages); err != nil {
		log.Fatalf("Failed to send JSON messages: %v", err)
	}
	fmt.Printf("✅ Sent %d JSON-heavy messages\n", len(jsonMessages))
	
	time.Sleep(5 * time.Second) // Allow processing and compression
	
	// Get final stats
	fmt.Println("\n📊 Final Storage Statistics:")
	finalStats, err := getStorageStats(statsEndpoint)
	if err != nil {
		log.Fatalf("Failed to get final stats: %v", err)
	}
	
	printStats(finalStats)
	
	// Calculate efficiency
	fmt.Println("\n🎯 Compression Efficiency Analysis:")
	if finalStats.CompressionRatio > 0 {
		compressionPercent := (1 - finalStats.CompressionRatio) * 100
		fmt.Printf("💾 Space saved: %.1f%%\n", compressionPercent)
		
		if finalStats.CompressionRatio < 0.3 {
			fmt.Println("🏆 Excellent compression! (>70% space saved)")
		} else if finalStats.CompressionRatio < 0.5 {
			fmt.Println("✅ Good compression! (50-70% space saved)")
		} else if finalStats.CompressionRatio < 0.7 {
			fmt.Println("⚠️  Moderate compression (30-50% space saved)")
		} else {
			fmt.Println("❌ Poor compression (<30% space saved)")
		}
	}
	
	fmt.Printf("📈 Total rows: %d\n", finalStats.TotalRows)
	fmt.Printf("📦 Partitions: %d\n", finalStats.PartitionCount)
	fmt.Printf("📅 Date range: %s to %s\n", finalStats.OldestDate, finalStats.NewestDate)
}

func generateSmallMessages(count int) []LogEntry {
	messages := []string{
		"User login successful",
		"User logout",
		"Page view",
		"Button clicked",
		"Form submitted",
		"Error occurred",
		"Warning logged",
		"Info message",
	}
	
	logs := make([]LogEntry, count)
	for i := 0; i < count; i++ {
		logs[i] = LogEntry{
			Timestamp: time.Now().Add(time.Duration(-count+i) * time.Second),
			Level:     "info",
			Message:   messages[rand.Intn(len(messages))],
			Service:   "web-app",
			TraceID:   fmt.Sprintf("trace-%d", rand.Intn(100)), // Limited trace IDs for repetition
			Attributes: map[string]interface{}{
				"user_id": fmt.Sprintf("user-%d", rand.Intn(50)), // Limited users for repetition
				"session": fmt.Sprintf("session-%d", rand.Intn(20)),
			},
		}
	}
	return logs
}

func generateLargeMessages(count int) []LogEntry {
	logs := make([]LogEntry, count)
	for i := 0; i < count; i++ {
		// Generate large unique messages
		message := fmt.Sprintf("Complex operation #%d with unique data: %s. Processing details: %s. Execution context: %s. Additional metadata: %s",
			i,
			generateRandomString(100),
			generateRandomString(150),
			generateRandomString(200),
			generateRandomString(120))
		
		logs[i] = LogEntry{
			Timestamp: time.Now().Add(time.Duration(-count+i) * time.Second),
			Level:     []string{"debug", "info", "warn", "error"}[rand.Intn(4)],
			Message:   message,
			Service:   fmt.Sprintf("service-%d", rand.Intn(1000)),
			TraceID:   fmt.Sprintf("trace-%s", generateRandomString(20)),
			SpanID:    fmt.Sprintf("span-%s", generateRandomString(15)),
			Attributes: map[string]interface{}{
				"unique_id":     generateRandomString(50),
				"large_data":    generateRandomString(300),
				"timestamp":     time.Now().UnixNano(),
				"random_float":  rand.Float64() * 1000000,
				"complex_path":  fmt.Sprintf("/api/v1/complex/%s/%d/details", generateRandomString(30), rand.Intn(10000)),
			},
		}
	}
	return logs
}

func generateJSONMessages(count int) []LogEntry {
	logs := make([]LogEntry, count)
	for i := 0; i < count; i++ {
		logs[i] = LogEntry{
			Timestamp: time.Now().Add(time.Duration(-count+i) * time.Second),
			Level:     "info",
			Message:   fmt.Sprintf("Structured log entry #%d", i),
			Service:   "json-service",
			TraceID:   fmt.Sprintf("trace-%d", rand.Intn(500)),
			Attributes: map[string]interface{}{
				"request_id":    fmt.Sprintf("req-%d", i),
				"user_agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				"ip_address":    fmt.Sprintf("192.168.%d.%d", rand.Intn(255), rand.Intn(255)),
				"method":        []string{"GET", "POST", "PUT", "DELETE"}[rand.Intn(4)],
				"status_code":   []int{200, 201, 400, 404, 500}[rand.Intn(5)],
				"response_time": rand.Intn(1000),
				"headers": map[string]string{
					"Content-Type":   "application/json",
					"Authorization":  "Bearer " + generateRandomString(50),
					"Accept":         "application/json",
					"Cache-Control":  "no-cache",
				},
				"query_params": map[string]interface{}{
					"page":   rand.Intn(100),
					"limit":  rand.Intn(50) + 10,
					"filter": generateRandomString(20),
				},
			},
		}
	}
	return logs
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func sendBulkLogs(endpoint string, logs []LogEntry) error {
	request := map[string]interface{}{
		"logs": logs,
		"options": map[string]bool{
			"skip_broadcast": true,
		},
	}
	
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

func getStorageStats(endpoint string) (*StorageStats, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	var response struct {
		StorageStats *StorageStats `json:"storage_stats"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	
	return response.StorageStats, nil
}

func printStats(stats *StorageStats) {
	if stats == nil {
		fmt.Println("No statistics available")
		return
	}
	
	fmt.Printf("  📊 Total Rows: %d\n", stats.TotalRows)
	fmt.Printf("  💾 Compressed Size: %s\n", stats.CompressedSize)
	fmt.Printf("  📦 Uncompressed Size: %s\n", stats.UncompressedSize)
	fmt.Printf("  🗜️  Compression Ratio: %.4f\n", stats.CompressionRatio)
	fmt.Printf("  🗂️  Partitions: %d\n", stats.PartitionCount)
	
	if stats.OldestDate != "" && stats.NewestDate != "" {
		fmt.Printf("  📅 Date Range: %s to %s\n", stats.OldestDate, stats.NewestDate)
	}
}