package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// QueryAdapter implements the QueryExecutor interface for ClickHouse
type QueryAdapter struct {
	baseURL  string
	client   *http.Client
	database string
}

// NewQueryAdapter creates a new query adapter
func NewQueryAdapter(baseURL, database string) *QueryAdapter {
	return &QueryAdapter{
		baseURL:  baseURL,
		client:   &http.Client{},
		database: database,
	}
}

// ExecuteQuery executes a SQL query and returns results as map
func (qa *QueryAdapter) ExecuteQuery(ctx context.Context, query string) ([]map[string]interface{}, error) {
	// The logs table is already in the default database, so we don't need to prefix it
	
	// Ensure JSON format for consistent parsing
	if !strings.Contains(strings.ToUpper(query), "FORMAT") {
		query += " FORMAT JSONEachRow"
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", qa.baseURL, strings.NewReader(query))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Execute request
	resp, err := qa.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ClickHouse error: %s", string(body))
	}
	
	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse JSON lines
	var results []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			// Log error but continue processing other rows
			continue
		}
		
		results = append(results, row)
	}
	
	return results, nil
}