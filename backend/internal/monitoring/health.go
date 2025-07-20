package monitoring

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusOK       HealthStatus = "ok"
	HealthStatusDegraded HealthStatus = "degraded"
	HealthStatusDown     HealthStatus = "down"
)

// ComponentHealth represents health information for a single component
type ComponentHealth struct {
	Name         string            `json:"name"`
	Status       HealthStatus      `json:"status"`
	Message      string            `json:"message,omitempty"`
	LastChecked  time.Time         `json:"last_checked"`
	ResponseTime time.Duration     `json:"response_time_ms"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status      HealthStatus               `json:"status"`
	Timestamp   time.Time                  `json:"timestamp"`
	Version     string                     `json:"version"`
	Uptime      time.Duration              `json:"uptime_seconds"`
	Components  map[string]*ComponentHealth `json:"components"`
	SystemInfo  SystemInfo                 `json:"system_info"`
}

// SystemInfo contains system-level information
type SystemInfo struct {
	GoVersion       string  `json:"go_version"`
	NumGoroutines   int     `json:"num_goroutines"`
	MemoryAllocMB   float64 `json:"memory_alloc_mb"`
	MemoryTotalMB   float64 `json:"memory_total_mb"`
	NumCPU          int     `json:"num_cpu"`
	StorageUsedGB   float64 `json:"storage_used_gb"`
	StorageTotalGB  float64 `json:"storage_total_gb"`
}

// HealthChecker defines the interface for health checks
type HealthChecker interface {
	Name() string
	Check() (*ComponentHealth, error)
}

// HealthMonitor manages health checks for the system
type HealthMonitor struct {
	mu         sync.RWMutex
	checkers   map[string]HealthChecker
	startTime  time.Time
	version    string
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(version string) *HealthMonitor {
	return &HealthMonitor{
		checkers:  make(map[string]HealthChecker),
		startTime: time.Now(),
		version:   version,
	}
}

// RegisterChecker registers a health checker
func (h *HealthMonitor) RegisterChecker(checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[checker.Name()] = checker
}

// GetHealth performs all health checks and returns system health
func (h *HealthMonitor) GetHealth() *SystemHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	health := &SystemHealth{
		Status:     HealthStatusOK,
		Timestamp:  time.Now(),
		Version:    h.version,
		Uptime:     time.Since(h.startTime),
		Components: make(map[string]*ComponentHealth),
		SystemInfo: h.getSystemInfo(),
	}

	// Run all health checks in parallel
	var wg sync.WaitGroup
	results := make(chan struct {
		name   string
		health *ComponentHealth
	}, len(h.checkers))

	for name, checker := range h.checkers {
		wg.Add(1)
		go func(n string, c HealthChecker) {
			defer wg.Done()
			
			start := time.Now()
			componentHealth, err := c.Check()
			if err != nil {
				componentHealth = &ComponentHealth{
					Name:    n,
					Status:  HealthStatusDown,
					Message: err.Error(),
				}
			}
			componentHealth.ResponseTime = time.Since(start)
			componentHealth.LastChecked = time.Now()
			
			results <- struct {
				name   string
				health *ComponentHealth
			}{n, componentHealth}
		}(name, checker)
	}

	wg.Wait()
	close(results)

	// Collect results and determine overall status
	for result := range results {
		health.Components[result.name] = result.health
		
		// Update overall status based on component status
		switch result.health.Status {
		case HealthStatusDown:
			health.Status = HealthStatusDown
		case HealthStatusDegraded:
			if health.Status != HealthStatusDown {
				health.Status = HealthStatusDegraded
			}
		}
	}

	return health
}

// HTTPHandler returns an HTTP handler for health checks
func (h *HealthMonitor) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := h.GetHealth()
		
		// Set appropriate status code
		statusCode := http.StatusOK
		switch health.Status {
		case HealthStatusDegraded:
			statusCode = http.StatusServiceUnavailable
		case HealthStatusDown:
			statusCode = http.StatusServiceUnavailable
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(health)
	}
}

// LivenessHandler returns a simple liveness check handler
func (h *HealthMonitor) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

// ReadinessHandler returns a readiness check handler
func (h *HealthMonitor) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := h.GetHealth()
		
		if health.Status == HealthStatusDown {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "not_ready",
				"components": health.Components,
			})
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	}
}

func (h *HealthMonitor) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return SystemInfo{
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		MemoryAllocMB: float64(m.Alloc) / 1024 / 1024,
		MemoryTotalMB: float64(m.TotalAlloc) / 1024 / 1024,
		NumCPU:        runtime.NumCPU(),
		// Storage metrics will be populated by storage health checker
	}
}