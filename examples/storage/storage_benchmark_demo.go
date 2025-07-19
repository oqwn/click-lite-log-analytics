package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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

type BenchmarkResults struct {
	TotalLogs           int64
	Duration            time.Duration
	LogsPerSecond       float64
	AvgCompressionRatio float64
	PartitionsCreated   int64
	TotalStorageSize    string
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
	fmt.Println("🏁 Storage Benchmark - Testing storage performance under load")
	fmt.Println("======================================================================")
	
	// Benchmark configuration
	benchmarks := []struct {
		name           string
		totalLogs      int
		concurrency    int
		batchSize      int
		description    string
	}{
		{
			name:        "small_load",
			totalLogs:   10000,
			concurrency: 5,
			batchSize:   100,
			description: "Small load test (10K logs)",
		},
		{
			name:        "medium_load",
			totalLogs:   50000,
			concurrency: 10,
			batchSize:   500,
			description: "Medium load test (50K logs)",
		},
		{
			name:        "large_load",
			totalLogs:   100000,
			concurrency: 20,
			batchSize:   1000,
			description: "Large load test (100K logs)",
		},
	}
	
	allResults := make([]BenchmarkResults, 0, len(benchmarks))
	
	for i, benchmark := range benchmarks {
		fmt.Printf("\n🚀 Running %s...\n", benchmark.description)
		fmt.Printf("   📊 Configuration: %d logs, %d workers, batch size %d\n", 
			benchmark.totalLogs, benchmark.concurrency, benchmark.batchSize)
		
		// Get initial stats
		initialStats, _ := getStorageStats()
		
		// Run benchmark
		results, err := runBenchmark(benchmark.name, benchmark.totalLogs, 
			benchmark.concurrency, benchmark.batchSize)
		if err != nil {
			log.Printf("❌ Benchmark %s failed: %v", benchmark.name, err)
			continue
		}
		
		// Get final stats
		finalStats, _ := getStorageStats()
		
		// Calculate storage metrics
		if initialStats != nil && finalStats != nil {
			results.PartitionsCreated = finalStats.PartitionCount - initialStats.PartitionCount
			results.AvgCompressionRatio = finalStats.CompressionRatio
			results.TotalStorageSize = finalStats.CompressedSize
		}
		
		allResults = append(allResults, *results)
		
		// Print results
		printBenchmarkResults(benchmark.description, results)
		
		// Wait between benchmarks
		if i < len(benchmarks)-1 {
			fmt.Println("\n⏳ Waiting for system to stabilize...")
			time.Sleep(5 * time.Second)
		}
	}
	
	// Print summary
	printSummary(allResults)
	
	// Storage efficiency analysis
	if err := analyzeStorageEfficiency(); err != nil {
		log.Printf("❌ Storage analysis failed: %v", err)
	}
}

func runBenchmark(name string, totalLogs, concurrency, batchSize int) (*BenchmarkResults, error) {
	endpoint := "http://localhost:20002/api/v1/ingest/bulk"
	
	var totalSent int64
	var wg sync.WaitGroup
	start := time.Now()
	
	// Create work channel
	workChan := make(chan int, concurrency*2)
	
	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for batchNum := range workChan {
				logs := generateBenchmarkLogs(name, workerID, batchNum, batchSize)
				
				if err := sendBulkLogs(endpoint, logs); err != nil {
					log.Printf("Worker %d failed to send batch %d: %v", workerID, batchNum, err)
					continue
				}
				
				atomic.AddInt64(&totalSent, int64(len(logs)))
				
				// Progress indicator
				if batchNum%10 == 0 {
					sent := atomic.LoadInt64(&totalSent)
					fmt.Printf("\r   📈 Progress: %d/%d logs (%.1f%%)", 
						sent, totalLogs, float64(sent)/float64(totalLogs)*100)
				}
			}
		}(i)
	}
	
	// Send work
	totalBatches := (totalLogs + batchSize - 1) / batchSize
	for i := 0; i < totalBatches; i++ {
		workChan <- i
	}
	close(workChan)
	
	// Wait for completion
	wg.Wait()
	duration := time.Since(start)
	
	// Clear progress line
	fmt.Printf("\r   ✅ Completed: %d logs in %v\n", totalSent, duration)
	
	// Wait for processing
	time.Sleep(3 * time.Second)
	
	results := &BenchmarkResults{
		TotalLogs:     totalSent,
		Duration:      duration,
		LogsPerSecond: float64(totalSent) / duration.Seconds(),
	}
	
	return results, nil
}

func generateBenchmarkLogs(testName string, workerID, batchNum, count int) []LogEntry {
	logs := make([]LogEntry, count)
	
	services := []string{"api-gateway", "user-service", "order-service", "payment-service", "inventory-service"}
	levels := []string{"debug", "info", "warn", "error"}
	
	for i := 0; i < count; i++ {
		logs[i] = LogEntry{
			Timestamp: time.Now().Add(time.Duration(i-count) * time.Millisecond),
			Level:     levels[i%len(levels)],
			Message:   fmt.Sprintf("Benchmark %s: worker %d, batch %d, log %d", testName, workerID, batchNum, i),
			Service:   services[i%len(services)],
			TraceID:   fmt.Sprintf("bench-%s-w%d-b%d-trace%d", testName, workerID, batchNum, i/10),
			SpanID:    fmt.Sprintf("span-%d", i),
			Attributes: map[string]interface{}{
				"benchmark":    testName,
				"worker_id":    workerID,
				"batch_num":    batchNum,
				"sequence":     i,
				"timestamp":    time.Now().Unix(),
				"random_data":  generateRandomAttributes(),
			},
		}
	}
	
	return logs
}

func generateRandomAttributes() map[string]interface{} {
	return map[string]interface{}{
		"request_id":    fmt.Sprintf("req-%d", time.Now().UnixNano()),
		"user_id":       fmt.Sprintf("user-%d", time.Now().UnixNano()%10000),
		"session_id":    fmt.Sprintf("sess-%d", time.Now().UnixNano()%1000),
		"ip_address":    fmt.Sprintf("192.168.%d.%d", time.Now().UnixNano()%255, time.Now().UnixNano()%255),
		"response_time": time.Now().UnixNano() % 1000,
		"status_code":   []int{200, 201, 400, 404, 500}[time.Now().UnixNano()%5],
	}
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

func getStorageStats() (*StorageStats, error) {
	resp, err := http.Get("http://localhost:20002/api/v1/storage/stats")
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

func printBenchmarkResults(description string, results *BenchmarkResults) {
	fmt.Printf("\n📊 %s Results:\n", description)
	fmt.Printf("   📈 Total Logs: %d\n", results.TotalLogs)
	fmt.Printf("   ⏱️  Duration: %v\n", results.Duration)
	fmt.Printf("   🚀 Throughput: %.0f logs/second\n", results.LogsPerSecond)
	
	if results.AvgCompressionRatio > 0 {
		fmt.Printf("   🗜️  Compression: %.1f%% space saved\n", (1-results.AvgCompressionRatio)*100)
	}
	
	if results.PartitionsCreated > 0 {
		fmt.Printf("   🗂️  Partitions Created: %d\n", results.PartitionsCreated)
	}
	
	if results.TotalStorageSize != "" {
		fmt.Printf("   💾 Storage Size: %s\n", results.TotalStorageSize)
	}
	
	// Performance rating
	if results.LogsPerSecond > 50000 {
		fmt.Println("   🏆 Performance: Excellent (>50K logs/sec)")
	} else if results.LogsPerSecond > 20000 {
		fmt.Println("   ✅ Performance: Good (20K-50K logs/sec)")
	} else if results.LogsPerSecond > 10000 {
		fmt.Println("   ⚠️  Performance: Moderate (10K-20K logs/sec)")
	} else {
		fmt.Println("   ❌ Performance: Poor (<10K logs/sec)")
	}
}

func printSummary(results []BenchmarkResults) {
	fmt.Println("\n🎯 Benchmark Summary:")
	fmt.Println("==================================================")
	
	var totalLogs int64
	var totalDuration time.Duration
	var maxThroughput float64
	
	for i, result := range results {
		totalLogs += result.TotalLogs
		totalDuration += result.Duration
		if result.LogsPerSecond > maxThroughput {
			maxThroughput = result.LogsPerSecond
		}
		
		fmt.Printf("  %d. %d logs in %v (%.0f logs/sec)\n", 
			i+1, result.TotalLogs, result.Duration, result.LogsPerSecond)
	}
	
	fmt.Printf("\n📊 Overall Statistics:\n")
	fmt.Printf("   📈 Total Logs Processed: %d\n", totalLogs)
	fmt.Printf("   ⏱️  Total Time: %v\n", totalDuration)
	fmt.Printf("   🚀 Peak Throughput: %.0f logs/second\n", maxThroughput)
	fmt.Printf("   📊 Average Throughput: %.0f logs/second\n", 
		float64(totalLogs)/totalDuration.Seconds())
}

func analyzeStorageEfficiency() error {
	fmt.Println("\n🔍 Storage Efficiency Analysis:")
	
	stats, err := getStorageStats()
	if err != nil {
		return err
	}
	
	fmt.Printf("   📊 Final Statistics:\n")
	fmt.Printf("     📈 Total Rows: %d\n", stats.TotalRows)
	fmt.Printf("     💾 Compressed Size: %s\n", stats.CompressedSize)
	fmt.Printf("     📦 Uncompressed Size: %s\n", stats.UncompressedSize)
	fmt.Printf("     🗜️  Compression Ratio: %.4f (%.1f%% space saved)\n", 
		stats.CompressionRatio, (1-stats.CompressionRatio)*100)
	fmt.Printf("     🗂️  Partitions: %d\n", stats.PartitionCount)
	fmt.Printf("     📅 Date Range: %s to %s\n", stats.OldestDate, stats.NewestDate)
	
	// Efficiency recommendations
	fmt.Println("\n💡 Storage Optimizations Applied:")
	fmt.Println("   ✅ ZSTD compression for excellent space efficiency")
	fmt.Println("   ✅ Daily partitioning for fast TTL cleanup")
	fmt.Println("   ✅ Materialized columns for query optimization")
	fmt.Println("   ✅ Specialized indexes for common query patterns")
	fmt.Println("   ✅ Automated cleanup routines")
	fmt.Println("   ✅ Tiered storage (hot/cold/archive)")
	
	return nil
}