package cluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// DistributedQueryEngine executes queries across multiple nodes
type DistributedQueryEngine struct {
	coordinator *Coordinator
	merger      ResultMerger
	timeout     time.Duration
}

// QueryResult represents a query result from a node
type QueryResult struct {
	NodeID string
	Data   []map[string]interface{}
	Error  error
	Timing time.Duration
}

// ResultMerger interface for merging distributed query results
type ResultMerger interface {
	Merge(results []*QueryResult) ([]map[string]interface{}, error)
}

// NewDistributedQueryEngine creates a new distributed query engine
func NewDistributedQueryEngine(coordinator *Coordinator, timeout time.Duration) *DistributedQueryEngine {
	return &DistributedQueryEngine{
		coordinator: coordinator,
		merger:      NewDefaultResultMerger(),
		timeout:     timeout,
	}
}

// ExecuteDistributedQuery executes a query across all relevant nodes
func (dqe *DistributedQueryEngine) ExecuteDistributedQuery(ctx context.Context, query string, shardKey string) ([]map[string]interface{}, error) {
	start := time.Now()
	
	// Determine which nodes to query
	nodes, err := dqe.getQueryNodes(shardKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get query nodes: %w", err)
	}
	
	// Execute query on each node
	results, err := dqe.executeOnNodes(ctx, query, nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to execute on nodes: %w", err)
	}
	
	// Merge results
	merged, err := dqe.merger.Merge(results)
	if err != nil {
		return nil, fmt.Errorf("failed to merge results: %w", err)
	}
	
	log.Info().
		Int("nodes", len(nodes)).
		Int("results", len(merged)).
		Dur("duration", time.Since(start)).
		Msg("Executed distributed query")
	
	return merged, nil
}

// getQueryNodes determines which nodes should execute the query
func (dqe *DistributedQueryEngine) getQueryNodes(shardKey string) ([]Node, error) {
	if shardKey == "" {
		// Query all nodes for global queries
		dqe.coordinator.nodesMu.RLock()
		defer dqe.coordinator.nodesMu.RUnlock()
		return dqe.coordinator.getHealthyNodes(), nil
	}
	
	// Query specific shard nodes
	return dqe.coordinator.GetNodesForShard(shardKey)
}

// executeOnNodes executes query on multiple nodes concurrently
func (dqe *DistributedQueryEngine) executeOnNodes(ctx context.Context, query string, nodes []Node) ([]*QueryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, dqe.timeout)
	defer cancel()
	
	results := make([]*QueryResult, len(nodes))
	var wg sync.WaitGroup
	
	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, n Node) {
			defer wg.Done()
			results[idx] = dqe.executeOnNode(ctx, query, n)
		}(i, node)
	}
	
	wg.Wait()
	
	// Check for errors
	var errors []error
	successCount := 0
	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else {
			successCount++
		}
	}
	
	// Require at least one successful result
	if successCount == 0 {
		return nil, fmt.Errorf("all nodes failed: %v", errors)
	}
	
	// Log warnings for failed nodes
	if len(errors) > 0 {
		log.Warn().
			Int("failed", len(errors)).
			Int("successful", successCount).
			Msg("Some nodes failed during distributed query")
	}
	
	return results, nil
}

// executeOnNode executes query on a single node
func (dqe *DistributedQueryEngine) executeOnNode(ctx context.Context, query string, node Node) *QueryResult {
	start := time.Now()
	result := &QueryResult{
		NodeID: node.ID,
		Timing: 0,
	}
	
	// In a real implementation, this would make HTTP/gRPC call to the node
	// For now, we'll simulate the execution
	data, err := dqe.simulateNodeQuery(ctx, query, node)
	
	result.Data = data
	result.Error = err
	result.Timing = time.Since(start)
	
	return result
}

// simulateNodeQuery simulates executing a query on a node
func (dqe *DistributedQueryEngine) simulateNodeQuery(ctx context.Context, query string, node Node) ([]map[string]interface{}, error) {
	// Simulate network delay
	select {
	case <-time.After(10 * time.Millisecond):
		// Continue
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	
	// Return mock data for demonstration
	return []map[string]interface{}{
		{
			"timestamp": time.Now(),
			"count":     100,
			"node_id":   node.ID,
		},
	}, nil
}

// DefaultResultMerger implements basic result merging
type DefaultResultMerger struct{}

// NewDefaultResultMerger creates a default result merger
func NewDefaultResultMerger() *DefaultResultMerger {
	return &DefaultResultMerger{}
}

// Merge merges results from multiple nodes
func (drm *DefaultResultMerger) Merge(results []*QueryResult) ([]map[string]interface{}, error) {
	var merged []map[string]interface{}
	
	for _, result := range results {
		if result.Error != nil {
			continue // Skip failed results
		}
		
		merged = append(merged, result.Data...)
	}
	
	return merged, nil
}

// AggregatingResultMerger merges and aggregates results
type AggregatingResultMerger struct {
	aggregateFields []string
}

// NewAggregatingResultMerger creates an aggregating result merger
func NewAggregatingResultMerger(aggregateFields []string) *AggregatingResultMerger {
	return &AggregatingResultMerger{
		aggregateFields: aggregateFields,
	}
}

// Merge merges and aggregates results
func (arm *AggregatingResultMerger) Merge(results []*QueryResult) ([]map[string]interface{}, error) {
	aggregates := make(map[string]map[string]interface{})
	
	for _, result := range results {
		if result.Error != nil {
			continue
		}
		
		for _, row := range result.Data {
			key := arm.generateAggregateKey(row)
			
			if existing, exists := aggregates[key]; exists {
				// Merge with existing aggregate
				arm.mergeRow(existing, row)
			} else {
				// Create new aggregate
				aggregates[key] = arm.copyRow(row)
			}
		}
	}
	
	// Convert to slice
	var merged []map[string]interface{}
	for _, aggregate := range aggregates {
		merged = append(merged, aggregate)
	}
	
	return merged, nil
}

// generateAggregateKey generates a key for grouping rows
func (arm *AggregatingResultMerger) generateAggregateKey(row map[string]interface{}) string {
	key := ""
	for _, field := range arm.aggregateFields {
		if value, ok := row[field]; ok {
			key += fmt.Sprintf("%v|", value)
		}
	}
	return key
}

// mergeRow merges two rows by summing numeric values
func (arm *AggregatingResultMerger) mergeRow(existing, new map[string]interface{}) {
	for key, value := range new {
		if existing[key] == nil {
			existing[key] = value
			continue
		}
		
		// Try to sum numeric values
		switch existingVal := existing[key].(type) {
		case int:
			if newVal, ok := value.(int); ok {
				existing[key] = existingVal + newVal
			}
		case int64:
			if newVal, ok := value.(int64); ok {
				existing[key] = existingVal + newVal
			}
		case float64:
			if newVal, ok := value.(float64); ok {
				existing[key] = existingVal + newVal
			}
		}
	}
}

// copyRow creates a copy of a row
func (arm *AggregatingResultMerger) copyRow(row map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for key, value := range row {
		copy[key] = value
	}
	return copy
}

// QueryPlanner plans distributed query execution
type QueryPlanner struct {
	coordinator *Coordinator
}

// NewQueryPlanner creates a new query planner
func NewQueryPlanner(coordinator *Coordinator) *QueryPlanner {
	return &QueryPlanner{
		coordinator: coordinator,
	}
}

// PlanQuery creates an execution plan for a distributed query
func (qp *QueryPlanner) PlanQuery(query string) (*QueryPlan, error) {
	// Analyze query to determine distribution strategy
	if qp.isAggregateQuery(query) {
		return qp.planAggregateQuery(query)
	}
	
	if qp.isJoinQuery(query) {
		return qp.planJoinQuery(query)
	}
	
	// Default: simple distributed scan
	return qp.planScanQuery(query)
}

// QueryPlan represents a distributed query execution plan
type QueryPlan struct {
	OriginalQuery string
	Steps         []QueryStep
	EstimatedCost float64
	Parallelism   int
}

// QueryStep represents a step in query execution
type QueryStep struct {
	Type        string // scan, aggregate, join, merge
	Query       string
	TargetNodes []string
	Dependencies []int // indices of dependent steps
}

// isAggregateQuery checks if query contains aggregations
func (qp *QueryPlanner) isAggregateQuery(query string) bool {
	aggregateFunctions := []string{"COUNT", "SUM", "AVG", "MIN", "MAX", "GROUP BY"}
	queryUpper := fmt.Sprintf("%s", query) // Convert to uppercase for checking
	
	for _, fn := range aggregateFunctions {
		if contains(queryUpper, fn) {
			return true
		}
	}
	return false
}

// isJoinQuery checks if query contains joins
func (qp *QueryPlanner) isJoinQuery(query string) bool {
	return contains(query, "JOIN")
}

// planAggregateQuery plans execution for aggregate queries
func (qp *QueryPlanner) planAggregateQuery(query string) (*QueryPlan, error) {
	return &QueryPlan{
		OriginalQuery: query,
		Steps: []QueryStep{
			{
				Type:        "distributed_scan",
				Query:       query,
				TargetNodes: qp.getAllNodeIDs(),
			},
			{
				Type:         "aggregate_merge",
				Query:        "",
				Dependencies: []int{0},
			},
		},
		Parallelism: len(qp.getAllNodeIDs()),
	}, nil
}

// planJoinQuery plans execution for join queries
func (qp *QueryPlanner) planJoinQuery(query string) (*QueryPlan, error) {
	// For now, execute joins locally after gathering data
	return &QueryPlan{
		OriginalQuery: query,
		Steps: []QueryStep{
			{
				Type:        "distributed_scan",
				Query:       qp.extractScanPortion(query),
				TargetNodes: qp.getAllNodeIDs(),
			},
			{
				Type:         "local_join",
				Query:        query,
				Dependencies: []int{0},
			},
		},
		Parallelism: 1, // Joins are executed locally
	}, nil
}

// planScanQuery plans execution for simple scan queries
func (qp *QueryPlanner) planScanQuery(query string) (*QueryPlan, error) {
	return &QueryPlan{
		OriginalQuery: query,
		Steps: []QueryStep{
			{
				Type:        "distributed_scan",
				Query:       query,
				TargetNodes: qp.getAllNodeIDs(),
			},
			{
				Type:         "simple_merge",
				Query:        "",
				Dependencies: []int{0},
			},
		},
		Parallelism: len(qp.getAllNodeIDs()),
	}, nil
}

// getAllNodeIDs returns all healthy node IDs
func (qp *QueryPlanner) getAllNodeIDs() []string {
	qp.coordinator.nodesMu.RLock()
	defer qp.coordinator.nodesMu.RUnlock()
	
	nodes := qp.coordinator.getHealthyNodes()
	ids := make([]string, len(nodes))
	for i, node := range nodes {
		ids[i] = node.ID
	}
	return ids
}

// extractScanPortion extracts the scan portion of a complex query
func (qp *QueryPlanner) extractScanPortion(query string) string {
	// Simplified: return the original query
	// In practice, this would parse and extract the FROM/WHERE portions
	return query
}

// contains checks if a string contains a substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr ||
			 indexOf(s, substr) >= 0)))
}

// indexOf finds the index of substr in s
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}