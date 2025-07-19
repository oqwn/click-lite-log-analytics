package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type QueryRequest struct {
	Query      string                 `json:"query"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Timeout    int                    `json:"timeout,omitempty"`
	MaxRows    int                    `json:"max_rows,omitempty"`
}

type QueryResponse struct {
	Columns       []ColumnInfo           `json:"columns"`
	Rows          []map[string]interface{} `json:"rows"`
	RowCount      int                    `json:"row_count"`
	ExecutionTime int64                  `json:"execution_time_ms"`
	Query         string                 `json:"query"`
	Error         string                 `json:"error,omitempty"`
}

type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

func main() {
	endpoint := "http://localhost:20002/api/v1/query/execute"
	
	fmt.Println("🔍 SQL Query Demo - Testing direct SQL access")
	fmt.Println("=============================================")
	
	// First, let's add some test data
	addTestData()
	time.Sleep(2 * time.Second)
	
	// Test 1: Simple SELECT query
	fmt.Println("\n📊 Test 1: Simple SELECT query")
	simpleQuery := QueryRequest{
		Query:   "SELECT level, COUNT() as count FROM logs GROUP BY level ORDER BY count DESC",
		Timeout: 10,
	}
	executeAndDisplay(endpoint, simpleQuery)
	
	// Test 2: Time-based aggregation
	fmt.Println("\n📊 Test 2: Time-based aggregation")
	timeQuery := QueryRequest{
		Query: `SELECT 
			toStartOfMinute(timestamp) as minute,
			COUNT() as log_count,
			COUNT(DISTINCT service) as service_count
		FROM logs 
		WHERE timestamp >= now() - INTERVAL 1 HOUR
		GROUP BY minute
		ORDER BY minute DESC
		LIMIT 10`,
		Timeout: 10,
	}
	executeAndDisplay(endpoint, timeQuery)
	
	// Test 3: Query with parameters
	fmt.Println("\n📊 Test 3: Parameterized query")
	paramQuery := QueryRequest{
		Query: `SELECT 
			service,
			level,
			COUNT() as count
		FROM logs 
		WHERE timestamp >= now() - INTERVAL :hours HOUR
			AND level = :level
		GROUP BY service, level
		ORDER BY count DESC
		LIMIT :limit`,
		Parameters: map[string]interface{}{
			"hours": 24,
			"level": "error",
			"limit": 5,
		},
		Timeout: 10,
	}
	executeAndDisplay(endpoint, paramQuery)
	
	// Test 4: Complex analytical query
	fmt.Println("\n📊 Test 4: Complex analytical query")
	analyticsQuery := QueryRequest{
		Query: `SELECT 
			service,
			level,
			COUNT() as total_logs,
			COUNT(DISTINCT trace_id) as unique_traces,
			avg(length(message)) as avg_message_length,
			max(length(message)) as max_message_length,
			quantile(0.95)(length(message)) as p95_message_length
		FROM logs 
		WHERE timestamp >= today()
		GROUP BY service, level
		HAVING total_logs > 10
		ORDER BY total_logs DESC`,
		MaxRows: 20,
	}
	executeAndDisplay(endpoint, analyticsQuery)
	
	// Test 5: Search query
	fmt.Println("\n📊 Test 5: Full-text search simulation")
	searchQuery := QueryRequest{
		Query: `SELECT 
			timestamp,
			level,
			service,
			message
		FROM logs 
		WHERE position(lower(message), lower(:search_term)) > 0
			AND timestamp >= now() - INTERVAL 1 DAY
		ORDER BY timestamp DESC
		LIMIT 10`,
		Parameters: map[string]interface{}{
			"search_term": "error",
		},
	}
	executeAndDisplay(endpoint, searchQuery)
	
	// Test 6: Query validation (should fail)
	fmt.Println("\n📊 Test 6: Query validation (intentional failure)")
	badQuery := QueryRequest{
		Query: "DROP TABLE logs", // This should be rejected
	}
	executeAndDisplay(endpoint, badQuery)
	
	fmt.Println("\n✅ SQL Query Demo completed!")
}

func executeAndDisplay(endpoint string, req QueryRequest) {
	fmt.Printf("Executing: %.100s...\n", req.Query)
	
	data, err := json.Marshal(req)
	if err != nil {
		log.Printf("❌ Failed to marshal request: %v", err)
		return
	}
	
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("❌ Failed to execute query: %v", err)
		return
	}
	defer resp.Body.Close()
	
	var result QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("❌ Failed to decode response: %v", err)
		return
	}
	
	if result.Error != "" {
		fmt.Printf("❌ Query error: %s\n", result.Error)
		return
	}
	
	// Display results
	fmt.Printf("✅ Execution time: %dms\n", result.ExecutionTime)
	fmt.Printf("📊 Rows returned: %d\n", result.RowCount)
	
	if len(result.Columns) > 0 {
		// Print column headers
		fmt.Print("\n")
		for _, col := range result.Columns {
			fmt.Printf("%-20s ", col.Name)
		}
		fmt.Print("\n")
		for range result.Columns {
			fmt.Print("--------------------")
		}
		fmt.Print("\n")
		
		// Print rows (limit to first 5 for display)
		displayCount := 5
		if result.RowCount < displayCount {
			displayCount = result.RowCount
		}
		
		for i := 0; i < displayCount; i++ {
			row := result.Rows[i]
			for _, col := range result.Columns {
				value := fmt.Sprintf("%v", row[col.Name])
				if len(value) > 18 {
					value = value[:18] + ".."
				}
				fmt.Printf("%-20s ", value)
			}
			fmt.Print("\n")
		}
		
		if result.RowCount > displayCount {
			fmt.Printf("... and %d more rows\n", result.RowCount-displayCount)
		}
	}
	
	fmt.Println()
}

func addTestData() {
	fmt.Println("📝 Adding test data...")
	
	// Add various log entries for testing
	endpoint := "http://localhost:20002/api/v1/ingest/bulk"
	
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-30 * time.Minute),
			"level":     "error",
			"message":   "Database connection error: timeout after 5s",
			"service":   "api-gateway",
			"trace_id":  "trace-001",
		},
		{
			"timestamp": time.Now().Add(-25 * time.Minute),
			"level":     "warn",
			"message":   "High memory usage detected: 85%",
			"service":   "worker-service",
			"trace_id":  "trace-002",
		},
		{
			"timestamp": time.Now().Add(-20 * time.Minute),
			"level":     "info",
			"message":   "User login successful",
			"service":   "auth-service",
			"trace_id":  "trace-003",
		},
		{
			"timestamp": time.Now().Add(-15 * time.Minute),
			"level":     "error",
			"message":   "Payment processing failed: invalid card",
			"service":   "payment-service",
			"trace_id":  "trace-004",
		},
		{
			"timestamp": time.Now().Add(-10 * time.Minute),
			"level":     "debug",
			"message":   "Cache miss for key: user-123",
			"service":   "cache-service",
			"trace_id":  "trace-005",
		},
		{
			"timestamp": time.Now().Add(-5 * time.Minute),
			"level":     "info",
			"message":   "Order completed successfully",
			"service":   "order-service",
			"trace_id":  "trace-006",
		},
	}
	
	// Add more logs to make aggregations interesting
	for i := 0; i < 50; i++ {
		logs = append(logs, map[string]interface{}{
			"timestamp": time.Now().Add(-time.Duration(i) * time.Minute),
			"level":     []string{"info", "warn", "error", "debug"}[i%4],
			"message":   fmt.Sprintf("Test log message #%d", i),
			"service":   []string{"api-gateway", "auth-service", "worker-service"}[i%3],
			"trace_id":  fmt.Sprintf("trace-%03d", i),
		})
	}
	
	request := map[string]interface{}{
		"logs": logs,
		"options": map[string]bool{
			"skip_broadcast": true,
		},
	}
	
	data, _ := json.Marshal(request)
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to add test data: %v", err)
		return
	}
	resp.Body.Close()
	
	fmt.Printf("✅ Added %d test log entries\n", len(logs))
}