package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// ClickHouseAdapter implements DatabaseInterface for ClickHouse
type ClickHouseAdapter struct {
	baseURL string
	client  *http.Client
}

// NewClickHouseAdapter creates a new ClickHouse adapter
func NewClickHouseAdapter(baseURL string) *ClickHouseAdapter {
	return &ClickHouseAdapter{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Exec executes a query without returning results
func (c *ClickHouseAdapter) Exec(query string) error {
	resp, err := c.client.Post(c.baseURL, "text/plain", strings.NewReader(query))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ClickHouse error: %s", string(body))
	}
	
	return nil
}

// Query executes a query and returns results
func (c *ClickHouseAdapter) Query(query string) ([]map[string]interface{}, error) {
	// Add FORMAT JSONEachRow for easier parsing
	if !strings.Contains(strings.ToUpper(query), "FORMAT") {
		query += " FORMAT JSONEachRow"
	}
	
	resp, err := c.client.Post(c.baseURL, "text/plain", strings.NewReader(query))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ClickHouse error: %s", string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var results []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			log.Warn().Err(err).Str("line", line).Msg("Failed to parse query result row")
			continue
		}
		
		results = append(results, row)
	}
	
	return results, nil
}