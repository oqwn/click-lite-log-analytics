package monitoring

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

// Metric represents a single metric
type Metric struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Value       float64                `json:"value"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description,omitempty"`
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	mu              sync.RWMutex
	counters        map[string]*int64
	gauges          map[string]*float64
	histograms      map[string]*Histogram
	descriptions    map[string]string
	ingestionRate   *RateCounter
	queryRate       *RateCounter
}

// Histogram tracks distribution of values
type Histogram struct {
	mu         sync.Mutex
	count      int64
	sum        float64
	min        float64
	max        float64
	buckets    []float64
	values     []int64
}

// RateCounter tracks rate over time
type RateCounter struct {
	mu            sync.Mutex
	windowSize    time.Duration
	buckets       []int64
	bucketTime    time.Duration
	currentBucket int
	lastUpdate    time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters:      make(map[string]*int64),
		gauges:        make(map[string]*float64),
		histograms:    make(map[string]*Histogram),
		descriptions:  make(map[string]string),
		ingestionRate: NewRateCounter(time.Minute, time.Second),
		queryRate:     NewRateCounter(time.Minute, time.Second),
	}
}

// IncrementCounter increments a counter metric
func (m *MetricsCollector) IncrementCounter(name string, delta int64) {
	m.mu.Lock()
	counter, exists := m.counters[name]
	if !exists {
		var c int64
		m.counters[name] = &c
		counter = &c
	}
	m.mu.Unlock()
	
	atomic.AddInt64(counter, delta)
}

// SetGauge sets a gauge metric value
func (m *MetricsCollector) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.gauges[name]; !exists {
		m.gauges[name] = new(float64)
	}
	*m.gauges[name] = value
}

// RecordHistogram records a value in a histogram
func (m *MetricsCollector) RecordHistogram(name string, value float64) {
	m.mu.Lock()
	hist, exists := m.histograms[name]
	if !exists {
		hist = NewHistogram([]float64{0.1, 0.5, 1, 5, 10, 50, 100, 500, 1000})
		m.histograms[name] = hist
	}
	m.mu.Unlock()
	
	hist.Record(value)
}

// SetDescription sets description for a metric
func (m *MetricsCollector) SetDescription(name string, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.descriptions[name] = description
}

// GetMetrics returns all current metrics
func (m *MetricsCollector) GetMetrics() []Metric {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var metrics []Metric
	timestamp := time.Now()
	
	// Collect counters
	for name, counter := range m.counters {
		value := atomic.LoadInt64(counter)
		metrics = append(metrics, Metric{
			Name:        name,
			Type:        string(MetricTypeCounter),
			Value:       float64(value),
			Timestamp:   timestamp,
			Description: m.descriptions[name],
		})
	}
	
	// Collect gauges
	for name, gauge := range m.gauges {
		metrics = append(metrics, Metric{
			Name:        name,
			Type:        string(MetricTypeGauge),
			Value:       *gauge,
			Timestamp:   timestamp,
			Description: m.descriptions[name],
		})
	}
	
	// Collect histograms
	for name, hist := range m.histograms {
		stats := hist.GetStats()
		for statName, value := range stats {
			metrics = append(metrics, Metric{
				Name: name + "_" + statName,
				Type: string(MetricTypeGauge),
				Value: value,
				Timestamp: timestamp,
				Description: m.descriptions[name],
			})
		}
	}
	
	// Add rate metrics
	metrics = append(metrics, Metric{
		Name:        "ingestion_rate_per_second",
		Type:        string(MetricTypeGauge),
		Value:       m.ingestionRate.GetRate(),
		Timestamp:   timestamp,
		Description: "Log ingestion rate per second",
	})
	
	metrics = append(metrics, Metric{
		Name:        "query_rate_per_second",
		Type:        string(MetricTypeGauge),
		Value:       m.queryRate.GetRate(),
		Timestamp:   timestamp,
		Description: "Query execution rate per second",
	})
	
	return metrics
}

// RecordIngestion records a log ingestion event
func (m *MetricsCollector) RecordIngestion(count int) {
	m.IncrementCounter("total_logs_ingested", int64(count))
	m.ingestionRate.Increment(count)
}

// RecordQuery records a query execution
func (m *MetricsCollector) RecordQuery(duration time.Duration) {
	m.IncrementCounter("total_queries_executed", 1)
	m.RecordHistogram("query_duration_ms", float64(duration.Milliseconds()))
	m.queryRate.Increment(1)
}

// RecordStorageSize records current storage size
func (m *MetricsCollector) RecordStorageSize(sizeBytes int64) {
	m.SetGauge("storage_size_bytes", float64(sizeBytes))
	m.SetGauge("storage_size_mb", float64(sizeBytes)/1024/1024)
}

// NewHistogram creates a new histogram
func NewHistogram(buckets []float64) *Histogram {
	return &Histogram{
		buckets: buckets,
		values:  make([]int64, len(buckets)+1),
		min:     1e9,
		max:     -1e9,
	}
}

// Record records a value in the histogram
func (h *Histogram) Record(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.count++
	h.sum += value
	
	if value < h.min {
		h.min = value
	}
	if value > h.max {
		h.max = value
	}
	
	// Find the right bucket
	bucketIndex := len(h.buckets)
	for i, threshold := range h.buckets {
		if value <= threshold {
			bucketIndex = i
			break
		}
	}
	h.values[bucketIndex]++
}

// GetStats returns histogram statistics
func (h *Histogram) GetStats() map[string]float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.count == 0 {
		return map[string]float64{
			"count": 0,
			"sum":   0,
			"avg":   0,
			"min":   0,
			"max":   0,
			"p50":   0,
			"p90":   0,
			"p99":   0,
		}
	}
	
	return map[string]float64{
		"count": float64(h.count),
		"sum":   h.sum,
		"avg":   h.sum / float64(h.count),
		"min":   h.min,
		"max":   h.max,
		"p50":   h.getPercentile(0.5),
		"p90":   h.getPercentile(0.9),
		"p99":   h.getPercentile(0.99),
	}
}

func (h *Histogram) getPercentile(p float64) float64 {
	// Simple approximation - return the bucket threshold
	target := int64(float64(h.count) * p)
	cumulative := int64(0)
	
	for i, count := range h.values {
		cumulative += count
		if cumulative >= target {
			if i < len(h.buckets) {
				return h.buckets[i]
			}
			return h.max
		}
	}
	
	return h.max
}

// NewRateCounter creates a new rate counter
func NewRateCounter(windowSize, bucketTime time.Duration) *RateCounter {
	numBuckets := int(windowSize / bucketTime)
	return &RateCounter{
		windowSize: windowSize,
		buckets:    make([]int64, numBuckets),
		bucketTime: bucketTime,
		lastUpdate: time.Now(),
	}
}

// Increment increments the counter
func (r *RateCounter) Increment(count int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.rotateBuckets()
	r.buckets[r.currentBucket] += int64(count)
}

// GetRate returns the current rate per second
func (r *RateCounter) GetRate() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.rotateBuckets()
	
	sum := int64(0)
	for _, count := range r.buckets {
		sum += count
	}
	
	return float64(sum) / r.windowSize.Seconds()
}

func (r *RateCounter) rotateBuckets() {
	now := time.Now()
	elapsed := now.Sub(r.lastUpdate)
	
	bucketsToRotate := int(elapsed / r.bucketTime)
	if bucketsToRotate > 0 {
		if bucketsToRotate >= len(r.buckets) {
			// Clear all buckets
			for i := range r.buckets {
				r.buckets[i] = 0
			}
			r.currentBucket = 0
		} else {
			// Rotate buckets
			for i := 0; i < bucketsToRotate; i++ {
				r.currentBucket = (r.currentBucket + 1) % len(r.buckets)
				r.buckets[r.currentBucket] = 0
			}
		}
		r.lastUpdate = now
	}
}