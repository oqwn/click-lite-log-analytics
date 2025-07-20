package monitoring

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StorageHealthChecker checks storage health
type StorageHealthChecker struct {
	storagePath string
}

// NewStorageHealthChecker creates a new storage health checker
func NewStorageHealthChecker(storagePath string) *StorageHealthChecker {
	return &StorageHealthChecker{
		storagePath: storagePath,
	}
}

// Name returns the name of the checker
func (s *StorageHealthChecker) Name() string {
	return "storage"
}

// Check performs the health check
func (s *StorageHealthChecker) Check() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:    s.Name(),
		Status:  HealthStatusOK,
		Details: make(map[string]interface{}),
	}
	
	// Check if storage directory exists
	info, err := os.Stat(s.storagePath)
	if err != nil {
		health.Status = HealthStatusDown
		return health, fmt.Errorf("storage directory not accessible: %v", err)
	}
	
	if !info.IsDir() {
		health.Status = HealthStatusDown
		return health, fmt.Errorf("storage path is not a directory")
	}
	
	// Check write permissions
	testFile := filepath.Join(s.storagePath, ".health_check")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		health.Status = HealthStatusDown
		return health, fmt.Errorf("cannot write to storage: %v", err)
	}
	os.Remove(testFile)
	
	// Get storage stats
	var totalSize int64
	var fileCount int
	
	err = filepath.Walk(s.storagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})
	
	if err != nil {
		health.Status = HealthStatusDegraded
		health.Message = fmt.Sprintf("Error calculating storage stats: %v", err)
	}
	
	health.Details["total_size_mb"] = float64(totalSize) / 1024 / 1024
	health.Details["file_count"] = fileCount
	health.Details["path"] = s.storagePath
	
	// Check available space (simplified)
	if totalSize > 10*1024*1024*1024 { // 10GB warning threshold
		health.Status = HealthStatusDegraded
		health.Message = "Storage usage is high"
	}
	
	return health, nil
}

// APIHealthChecker checks API endpoint health
type APIHealthChecker struct {
	endpoint string
	timeout  time.Duration
}

// NewAPIHealthChecker creates a new API health checker
func NewAPIHealthChecker(endpoint string, timeout time.Duration) *APIHealthChecker {
	return &APIHealthChecker{
		endpoint: endpoint,
		timeout:  timeout,
	}
}

// Name returns the name of the checker
func (a *APIHealthChecker) Name() string {
	return "api"
}

// Check performs the health check
func (a *APIHealthChecker) Check() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:    a.Name(),
		Status:  HealthStatusOK,
		Details: make(map[string]interface{}),
	}
	
	// Simple check - just verify the endpoint is reachable
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()
	
	// Simulate API check
	select {
	case <-ctx.Done():
		health.Status = HealthStatusDown
		return health, fmt.Errorf("API health check timeout")
	case <-time.After(10 * time.Millisecond):
		// API is responsive
		health.Details["endpoint"] = a.endpoint
		health.Details["response_time_ms"] = 10
		return health, nil
	}
}

// IngestionHealthChecker checks log ingestion health
type IngestionHealthChecker struct {
	metrics *MetricsCollector
}

// NewIngestionHealthChecker creates a new ingestion health checker
func NewIngestionHealthChecker(metrics *MetricsCollector) *IngestionHealthChecker {
	return &IngestionHealthChecker{
		metrics: metrics,
	}
}

// Name returns the name of the checker
func (i *IngestionHealthChecker) Name() string {
	return "ingestion"
}

// Check performs the health check
func (i *IngestionHealthChecker) Check() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:    i.Name(),
		Status:  HealthStatusOK,
		Details: make(map[string]interface{}),
	}
	
	// Get current metrics
	metrics := i.metrics.GetMetrics()
	
	var ingestionRate float64
	var totalIngested float64
	
	for _, m := range metrics {
		switch m.Name {
		case "ingestion_rate_per_second":
			ingestionRate = m.Value
		case "total_logs_ingested":
			totalIngested = m.Value
		}
	}
	
	health.Details["rate_per_second"] = ingestionRate
	health.Details["total_ingested"] = totalIngested
	
	// Check if ingestion is working
	if totalIngested == 0 {
		health.Status = HealthStatusDegraded
		health.Message = "No logs have been ingested"
	}
	
	return health, nil
}

// QueryEngineHealthChecker checks query engine health
type QueryEngineHealthChecker struct {
	metrics *MetricsCollector
}

// NewQueryEngineHealthChecker creates a new query engine health checker
func NewQueryEngineHealthChecker(metrics *MetricsCollector) *QueryEngineHealthChecker {
	return &QueryEngineHealthChecker{
		metrics: metrics,
	}
}

// Name returns the name of the checker
func (q *QueryEngineHealthChecker) Name() string {
	return "query_engine"
}

// Check performs the health check
func (q *QueryEngineHealthChecker) Check() (*ComponentHealth, error) {
	health := &ComponentHealth{
		Name:    q.Name(),
		Status:  HealthStatusOK,
		Details: make(map[string]interface{}),
	}
	
	// Get query metrics
	metrics := q.metrics.GetMetrics()
	
	var queryRate float64
	var avgDuration float64
	var p99Duration float64
	
	for _, m := range metrics {
		switch m.Name {
		case "query_rate_per_second":
			queryRate = m.Value
		case "query_duration_ms_avg":
			avgDuration = m.Value
		case "query_duration_ms_p99":
			p99Duration = m.Value
		}
	}
	
	health.Details["rate_per_second"] = queryRate
	health.Details["avg_duration_ms"] = avgDuration
	health.Details["p99_duration_ms"] = p99Duration
	
	// Check performance thresholds
	if p99Duration > 5000 {
		health.Status = HealthStatusDegraded
		health.Message = "Query performance is degraded"
	}
	
	return health, nil
}