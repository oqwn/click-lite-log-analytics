package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type SavedQuery struct {
	ID          string           `json:"id,omitempty"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Query       string           `json:"query"`
	Parameters  []QueryParameter `json:"parameters,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	Category    string           `json:"category,omitempty"`
}

type QueryParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description,omitempty"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Required     bool        `json:"required"`
}

func main() {
	baseURL := "http://localhost:20002/api/v1/query"
	
	fmt.Println("💾 Saved Queries Demo - Managing and executing saved queries")
	fmt.Println("===========================================================")
	
	// Test 1: List built-in templates
	fmt.Println("\n📋 Test 1: List built-in query templates")
	listQueries(baseURL + "/saved?templates_only=true")
	
	// Test 2: Create a custom saved query
	fmt.Println("\n💾 Test 2: Create a custom saved query")
	customQuery := SavedQuery{
		Name:        "Top Error Sources",
		Description: "Find services generating the most errors in a time range",
		Query: `SELECT 
			service,
			COUNT() as error_count,
			COUNT(DISTINCT trace_id) as affected_requests,
			MIN(timestamp) as first_error,
			MAX(timestamp) as last_error
		FROM logs
		WHERE level = 'error'
			AND timestamp >= now() - INTERVAL :hours HOUR
		GROUP BY service
		ORDER BY error_count DESC
		LIMIT :top_n`,
		Parameters: []QueryParameter{
			{
				Name:         "hours",
				Type:         "number",
				Description:  "Look back period in hours",
				DefaultValue: 24,
				Required:     true,
			},
			{
				Name:         "top_n",
				Type:         "number",
				Description:  "Number of top services to return",
				DefaultValue: 10,
				Required:     false,
			},
		},
		Tags:     []string{"errors", "monitoring", "custom"},
		Category: "Error Analysis",
	}
	
	savedID := createQuery(baseURL+"/saved", customQuery)
	
	// Test 3: Execute the saved query
	if savedID != "" {
		fmt.Println("\n🚀 Test 3: Execute saved query with parameters")
		params := map[string]interface{}{
			"hours": 48,
			"top_n": 5,
		}
		executeSavedQuery(baseURL+"/saved/"+savedID+"/execute", params)
	}
	
	// Test 4: Update the saved query
	if savedID != "" {
		fmt.Println("\n✏️  Test 4: Update saved query")
		updates := map[string]interface{}{
			"description": "Updated: Find services with most errors (enhanced)",
			"tags":        []string{"errors", "monitoring", "custom", "updated"},
		}
		updateQuery(baseURL+"/saved/"+savedID, updates)
	}
	
	// Test 5: Execute a built-in template
	fmt.Println("\n🚀 Test 5: Execute built-in template (Errors by Service)")
	templateParams := map[string]interface{}{
		"time_range": 6,
		"limit":      10,
	}
	executeSavedQuery(baseURL+"/saved/template-errors-by-service/execute", templateParams)
	
	// Test 6: Search queries by tag
	fmt.Println("\n🔍 Test 6: Search queries by tag")
	listQueries(baseURL + "/saved?tags=errors")
	
	// Test 7: List queries by category
	fmt.Println("\n📁 Test 7: List queries by category")
	listQueries(baseURL + "/saved?category=Error Analysis")
	
	// Test 8: Delete custom query
	if savedID != "" {
		fmt.Println("\n🗑️  Test 8: Delete custom query")
		deleteQuery(baseURL+"/saved/"+savedID)
	}
	
	fmt.Println("\n✅ Saved Queries Demo completed!")
}

func listQueries(url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("❌ Failed to list queries: %v", err)
		return
	}
	defer resp.Body.Close()
	
	var result struct {
		Queries []SavedQuery `json:"queries"`
		Count   int          `json:"count"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("❌ Failed to decode response: %v", err)
		return
	}
	
	fmt.Printf("Found %d queries:\n", result.Count)
	for _, q := range result.Queries {
		fmt.Printf("  📄 %s (%s)\n", q.Name, q.ID)
		fmt.Printf("     %s\n", q.Description)
		if len(q.Tags) > 0 {
			fmt.Printf("     Tags: %v\n", q.Tags)
		}
		if q.Category != "" {
			fmt.Printf("     Category: %s\n", q.Category)
		}
		fmt.Println()
	}
}

func createQuery(url string, query SavedQuery) string {
	data, err := json.Marshal(query)
	if err != nil {
		log.Printf("❌ Failed to marshal query: %v", err)
		return ""
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("❌ Failed to create query: %v", err)
		return ""
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Server returned status %d", resp.StatusCode)
		return ""
	}
	
	var saved SavedQuery
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		log.Printf("❌ Failed to decode response: %v", err)
		return ""
	}
	
	fmt.Printf("✅ Created query: %s (ID: %s)\n", saved.Name, saved.ID)
	return saved.ID
}

func executeSavedQuery(url string, params map[string]interface{}) {
	data, err := json.Marshal(params)
	if err != nil {
		log.Printf("❌ Failed to marshal parameters: %v", err)
		return
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("❌ Failed to execute query: %v", err)
		return
	}
	defer resp.Body.Close()
	
	var result struct {
		Columns       []interface{} `json:"columns"`
		Rows          []interface{} `json:"rows"`
		RowCount      int           `json:"row_count"`
		ExecutionTime int64         `json:"execution_time_ms"`
		Query         string        `json:"query"`
		Error         string        `json:"error,omitempty"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("❌ Failed to decode response: %v", err)
		return
	}
	
	if result.Error != "" {
		fmt.Printf("❌ Query error: %s\n", result.Error)
		return
	}
	
	fmt.Printf("✅ Query executed: %s\n", result.Query)
	fmt.Printf("   Execution time: %dms\n", result.ExecutionTime)
	fmt.Printf("   Rows returned: %d\n", result.RowCount)
}

func updateQuery(url string, updates map[string]interface{}) {
	data, err := json.Marshal(updates)
	if err != nil {
		log.Printf("❌ Failed to marshal updates: %v", err)
		return
	}
	
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Failed to update query: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Server returned status %d", resp.StatusCode)
		return
	}
	
	fmt.Println("✅ Query updated successfully")
}

func deleteQuery(url string) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("❌ Failed to create request: %v", err)
		return
	}
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Failed to delete query: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Query deleted successfully")
	} else {
		fmt.Printf("❌ Server returned status %d\n", resp.StatusCode)
	}
}