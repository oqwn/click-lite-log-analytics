package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/cache"
	"github.com/your-username/click-lite-log-analytics/backend/internal/cluster"
	"github.com/your-username/click-lite-log-analytics/backend/internal/optimization"
	"github.com/your-username/click-lite-log-analytics/backend/internal/storage"
)

// PerformanceHandlerChi handles performance optimization endpoints for chi router
type PerformanceHandlerChi struct {
	queryOptimizer   *optimization.QueryOptimizer
	storageOptimizer *storage.StorageOptimizer
	coordinator      *cluster.Coordinator
	cacheStats       *cache.StatsCache
}

// NewPerformanceHandlerChi creates a new performance handler for chi router
func NewPerformanceHandlerChi(
	optimizer *optimization.QueryOptimizer,
	storageOptimizer *storage.StorageOptimizer,
	coordinator *cluster.Coordinator,
	cacheStats *cache.StatsCache,
) *PerformanceHandlerChi {
	return &PerformanceHandlerChi{
		queryOptimizer:   optimizer,
		storageOptimizer: storageOptimizer,
		coordinator:      coordinator,
		cacheStats:       cacheStats,
	}
}

// OptimizeQueryRequest represents query optimization request
type OptimizeQueryRequest struct {
	Query string `json:"query"`
}

// OptimizeQueryResponse represents query optimization response
type OptimizeQueryResponse struct {
	OriginalQuery   string   `json:"original_query"`
	OptimizedQuery  string   `json:"optimized_query"`
	Optimizations   []string `json:"optimizations"`
	EstimatedCost   float64  `json:"estimated_cost"`
	IndexesUsed     []string `json:"indexes_used"`
	PartitionPruning bool    `json:"partition_pruning"`
	Parallelism     int      `json:"parallelism"`
}

// OptimizeQuery optimizes a SQL query
func (h *PerformanceHandlerChi) OptimizeQuery(w http.ResponseWriter, r *http.Request) {
	var req OptimizeQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	plan := h.queryOptimizer.Optimize(req.Query)

	response := OptimizeQueryResponse{
		OriginalQuery:    plan.OriginalQuery,
		OptimizedQuery:   plan.OptimizedQuery,
		Optimizations:    plan.Optimizations,
		EstimatedCost:    plan.EstimatedCost,
		IndexesUsed:      plan.IndexesUsed,
		PartitionPruning: plan.PartitionPruning,
		Parallelism:      plan.Parallelism,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SuggestIndexesRequest represents index suggestion request
type SuggestIndexesRequest struct {
	Queries []string `json:"queries"`
}

// SuggestIndexes suggests database indexes based on query patterns
func (h *PerformanceHandlerChi) SuggestIndexes(w http.ResponseWriter, r *http.Request) {
	var req SuggestIndexesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	suggestions := h.queryOptimizer.SuggestIndexes(req.Queries)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": suggestions,
	})
}

// GetCacheStats returns cache statistics
func (h *PerformanceHandlerChi) GetCacheStats(w http.ResponseWriter, r *http.Request) {
	stats := h.cacheStats.GetStats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ClearCache clears the query cache
func (h *PerformanceHandlerChi) ClearCache(w http.ResponseWriter, r *http.Request) {
	h.cacheStats.Clear()
	log.Info().Msg("Cache cleared")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Cache cleared successfully",
	})
}

// AnalyzeStorageRequest represents storage analysis request
type AnalyzeStorageRequest struct {
	TableName string `json:"table_name"`
}

// AnalyzeStorage analyzes storage usage and provides recommendations
func (h *PerformanceHandlerChi) AnalyzeStorage(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeStorageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	analysis, err := h.storageOptimizer.AnalyzeStorageUsage(r.Context(), req.TableName)
	if err != nil {
		log.Error().Err(err).Str("table", req.TableName).Msg("Failed to analyze storage")
		http.Error(w, "Failed to analyze storage", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// OptimizeStorage optimizes storage partitions
func (h *PerformanceHandlerChi) OptimizeStorage(w http.ResponseWriter, r *http.Request) {
	tableName := r.URL.Query().Get("table")
	if tableName == "" {
		tableName = "logs"
	}

	if err := h.storageOptimizer.OptimizePartitions(r.Context(), tableName); err != nil {
		log.Error().Err(err).Str("table", tableName).Msg("Failed to optimize storage")
		http.Error(w, "Failed to optimize storage", http.StatusInternalServerError)
		return
	}

	log.Info().Str("table", tableName).Msg("Storage optimization completed")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Storage optimization completed",
	})
}

// CreateOptimizedSchema creates optimized table schema
func (h *PerformanceHandlerChi) CreateOptimizedSchema(w http.ResponseWriter, r *http.Request) {
	if err := h.storageOptimizer.OptimizeSchema(r.Context()); err != nil {
		log.Error().Err(err).Msg("Failed to create optimized schema")
		http.Error(w, "Failed to create optimized schema", http.StatusInternalServerError)
		return
	}

	log.Info().Msg("Optimized schema created")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Optimized schema created successfully",
	})
}

// CreateMaterializedViews creates materialized views for performance
func (h *PerformanceHandlerChi) CreateMaterializedViews(w http.ResponseWriter, r *http.Request) {
	if err := h.storageOptimizer.CreateMaterializedViews(r.Context()); err != nil {
		log.Error().Err(err).Msg("Failed to create materialized views")
		http.Error(w, "Failed to create materialized views", http.StatusInternalServerError)
		return
	}

	log.Info().Msg("Materialized views created")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Materialized views created successfully",
	})
}

// GetClusterStatus returns cluster status and node information
func (h *PerformanceHandlerChi) GetClusterStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "healthy",
		"nodes":  0, // Would get from coordinator
		"message": "Cluster coordination not fully implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// RegisterNodeRequest represents node registration request
type RegisterNodeRequest struct {
	ID       string            `json:"id"`
	Address  string            `json:"address"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RegisterNode registers a new cluster node
func (h *PerformanceHandlerChi) RegisterNode(w http.ResponseWriter, r *http.Request) {
	var req RegisterNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	node := cluster.Node{
		ID:       req.ID,
		Address:  req.Address,
		Metadata: req.Metadata,
	}

	if err := h.coordinator.RegisterNode(node); err != nil {
		log.Error().Err(err).Str("node_id", req.ID).Msg("Failed to register node")
		http.Error(w, "Failed to register node", http.StatusInternalServerError)
		return
	}

	log.Info().Str("node_id", req.ID).Msg("Node registered successfully")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Node registered successfully",
	})
}

// RemoveNode removes a cluster node
func (h *PerformanceHandlerChi) RemoveNode(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")
	if nodeID == "" {
		http.Error(w, "Node ID is required", http.StatusBadRequest)
		return
	}

	if err := h.coordinator.RemoveNode(nodeID); err != nil {
		log.Error().Err(err).Str("node_id", nodeID).Msg("Failed to remove node")
		http.Error(w, "Failed to remove node", http.StatusInternalServerError)
		return
	}

	log.Info().Str("node_id", nodeID).Msg("Node removed successfully")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Node removed successfully",
	})
}

// GetPerformanceMetrics returns overall performance metrics
func (h *PerformanceHandlerChi) GetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	// Get cache stats
	cacheStats := h.cacheStats.GetStats()

	// Build comprehensive metrics
	metrics := map[string]interface{}{
		"cache": cacheStats,
		"query_optimization": map[string]interface{}{
			"enabled": true,
			"patterns_count": len(h.queryOptimizer.SuggestIndexes([]string{})), // Simplified
		},
		"storage": map[string]interface{}{
			"optimization_enabled": true,
			"last_optimization": time.Now().Add(-2 * time.Hour), // Mock data
		},
		"cluster": map[string]interface{}{
			"coordination_enabled": true,
			"nodes_count": 1, // Mock data
		},
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// BenchmarkQueryRequest represents query benchmarking request
type BenchmarkQueryRequest struct {
	Query      string `json:"query"`
	Iterations int    `json:"iterations"`
	UseCache   bool   `json:"use_cache"`
}

// BenchmarkQueryResponse represents query benchmarking response
type BenchmarkQueryResponse struct {
	Query               string  `json:"query"`
	Iterations          int     `json:"iterations"`
	AverageExecutionTime float64 `json:"average_execution_time_ms"`
	MinExecutionTime    float64 `json:"min_execution_time_ms"`
	MaxExecutionTime    float64 `json:"max_execution_time_ms"`
	TotalExecutionTime  float64 `json:"total_execution_time_ms"`
	CacheHits           int     `json:"cache_hits"`
	Errors              int     `json:"errors"`
}

// BenchmarkQuery benchmarks query performance
func (h *PerformanceHandlerChi) BenchmarkQuery(w http.ResponseWriter, r *http.Request) {
	var req BenchmarkQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set default iterations
	if req.Iterations <= 0 {
		req.Iterations = 10
	}
	if req.Iterations > 100 {
		req.Iterations = 100 // Limit to prevent abuse
	}

	// Mock benchmarking implementation
	// In a real implementation, this would execute the query multiple times
	response := BenchmarkQueryResponse{
		Query:               req.Query,
		Iterations:          req.Iterations,
		AverageExecutionTime: 250.5, // Mock data
		MinExecutionTime:    180.2,
		MaxExecutionTime:    420.8,
		TotalExecutionTime:  float64(req.Iterations) * 250.5,
		CacheHits:           0,
		Errors:              0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}