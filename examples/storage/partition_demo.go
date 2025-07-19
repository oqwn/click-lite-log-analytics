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
	endpoint := "http://localhost:20002/api/v1/ingest/bulk"
	
	fmt.Println("📅 Partition Test - Testing daily partitioning strategy")
	fmt.Println("============================================================")
	
	// Generate logs spanning multiple days
	days := []int{-7, -6, -5, -4, -3, -2, -1, 0} // Last 7 days plus today
	logsPerDay := 500
	
	for _, dayOffset := range days {
		targetDate := time.Now().AddDate(0, 0, dayOffset)
		logs := generateLogsForDay(targetDate, logsPerDay)
		
		fmt.Printf("📤 Sending %d logs for %s...\n", 
			len(logs), targetDate.Format("2006-01-02"))
		
		if err := sendBulkLogs(endpoint, logs); err != nil {
			log.Printf("❌ Failed to send logs for %s: %v", 
				targetDate.Format("2006-01-02"), err)
			continue
		}
		
		fmt.Printf("✅ Successfully sent logs for %s\n", 
			targetDate.Format("2006-01-02"))
		
		// Small delay to avoid overwhelming the server
		time.Sleep(500 * time.Millisecond)
	}
	
	// Wait for processing
	time.Sleep(3 * time.Second)
	
	// Check partitioning results
	fmt.Println("\n📊 Checking partitioning results...")
	if err := checkPartitions(); err != nil {
		log.Printf("❌ Failed to check partitions: %v", err)
	}
	
	fmt.Println("\n🎯 Partition Test Summary:")
	fmt.Printf("📅 Created logs spanning %d days\n", len(days))
	fmt.Printf("📊 %d logs per day\n", logsPerDay)
	fmt.Printf("📈 Total logs: %d\n", len(days)*logsPerDay)
	fmt.Printf("🗂️  Expected partitions: %d (one per day)\n", len(days))
	
	fmt.Println("\n💡 Expected Benefits:")
	fmt.Println("  ⚡ Faster queries when filtering by date")
	fmt.Println("  🗑️  Efficient TTL cleanup (drop entire partitions)")
	fmt.Println("  💾 Better compression within date ranges")
	fmt.Println("  🔍 Improved query planning and optimization")
}

func generateLogsForDay(targetDate time.Time, count int) []LogEntry {
	logs := make([]LogEntry, count)
	
	// Generate logs throughout the day
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 
		0, 0, 0, 0, targetDate.Location())
	
	services := []string{"web-api", "auth-service", "payment-service", "user-service", "order-service"}
	levels := []string{"debug", "info", "warn", "error"}
	
	for i := 0; i < count; i++ {
		// Distribute logs throughout the day
		secondsInDay := 24 * 60 * 60
		randomSeconds := int64(i * secondsInDay / count)
		timestamp := startOfDay.Add(time.Duration(randomSeconds) * time.Second)
		
		logs[i] = LogEntry{
			Timestamp: timestamp,
			Level:     levels[i%len(levels)],
			Message:   fmt.Sprintf("Log entry #%d for %s", i+1, targetDate.Format("2006-01-02")),
			Service:   services[i%len(services)],
			TraceID:   fmt.Sprintf("trace-%s-%04d", targetDate.Format("20060102"), i),
			SpanID:    fmt.Sprintf("span-%04d", i),
			Attributes: map[string]interface{}{
				"partition_date": targetDate.Format("2006-01-02"),
				"hour":          timestamp.Hour(),
				"sequence":      i,
				"day_of_week":   targetDate.Weekday().String(),
			},
		}
	}
	
	return logs
}

func sendBulkLogs(endpoint string, logs []LogEntry) error {
	request := map[string]interface{}{
		"logs": logs,
		"options": map[string]bool{
			"skip_broadcast": true, // Don't overwhelm WebSocket clients
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

func checkPartitions() error {
	// Check storage statistics
	statsEndpoint := "http://localhost:20002/api/v1/storage/stats"
	
	resp, err := http.Get(statsEndpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	var response struct {
		StorageStats struct {
			TotalRows      int64  `json:"total_rows"`
			PartitionCount int64  `json:"partition_count"`
			OldestDate     string `json:"oldest_date"`
			NewestDate     string `json:"newest_date"`
		} `json:"storage_stats"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}
	
	stats := response.StorageStats
	
	fmt.Printf("📊 Partitioning Results:\n")
	fmt.Printf("  📈 Total Rows: %d\n", stats.TotalRows)
	fmt.Printf("  🗂️  Partition Count: %d\n", stats.PartitionCount)
	fmt.Printf("  📅 Date Range: %s to %s\n", stats.OldestDate, stats.NewestDate)
	
	// Validate partitioning
	if stats.PartitionCount >= 7 { // At least 7 days worth
		fmt.Println("  ✅ Partitioning appears to be working correctly!")
	} else {
		fmt.Printf("  ⚠️  Expected more partitions (got %d, expected ~7-8)\n", stats.PartitionCount)
	}
	
	return nil
}