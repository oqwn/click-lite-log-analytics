package errors

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// ErrorDetector detects and analyzes error patterns in logs
type ErrorDetector struct {
	mu               sync.RWMutex
	patterns         []ErrorPattern
	errorStats       map[string]*ErrorStats
	anomalyDetector  *AnomalyDetector
	windowSize       time.Duration
	alertThresholds  AlertThresholds
}

// ErrorPattern defines patterns for detecting errors
type ErrorPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Severity    string
	Category    string
	Description string
}

// ErrorStats tracks error statistics
type ErrorStats struct {
	Pattern      string                 `json:"pattern"`
	Category     string                 `json:"category"`
	Count        int64                  `json:"count"`
	FirstSeen    time.Time              `json:"first_seen"`
	LastSeen     time.Time              `json:"last_seen"`
	Services     map[string]int64       `json:"services"`
	Samples      []ErrorSample          `json:"samples"`
	Rate         float64                `json:"rate"`
	Trend        string                 `json:"trend"` // increasing, decreasing, stable
}

// ErrorSample represents a sample error log
type ErrorSample struct {
	LogID     string    `json:"log_id"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Message   string    `json:"message"`
	TraceID   string    `json:"trace_id,omitempty"`
}

// AlertThresholds defines thresholds for error alerts
type AlertThresholds struct {
	ErrorRatePerMinute float64
	ErrorBurstSize     int
	AnomalyStdDev      float64
}

// AnomalyDetector detects anomalies in error rates
type AnomalyDetector struct {
	mu             sync.RWMutex
	history        []float64
	mean           float64
	stdDev         float64
	windowSize     int
	lastUpdate     time.Time
}

// NewErrorDetector creates a new error detector
func NewErrorDetector() *ErrorDetector {
	ed := &ErrorDetector{
		errorStats: make(map[string]*ErrorStats),
		windowSize: 5 * time.Minute,
		alertThresholds: AlertThresholds{
			ErrorRatePerMinute: 10.0,
			ErrorBurstSize:     50,
			AnomalyStdDev:      2.0,
		},
		patterns: []ErrorPattern{
			// Application errors
			{
				Name:     "Exception",
				Pattern:  regexp.MustCompile(`(?i)(exception|error):\s*(.+)`),
				Severity: "high",
				Category: "application",
			},
			{
				Name:     "StackTrace",
				Pattern:  regexp.MustCompile(`(?i)^\s*at\s+[\w.$]+\(.*\)|\s+at\s+.+:\d+`),
				Severity: "high",
				Category: "application",
			},
			{
				Name:     "NullPointer",
				Pattern:  regexp.MustCompile(`(?i)(null\s*pointer|null\s*reference|nil\s*pointer)`),
				Severity: "high",
				Category: "application",
			},
			{
				Name:     "OutOfMemory",
				Pattern:  regexp.MustCompile(`(?i)(out\s*of\s*memory|oom|memory\s*exhausted)`),
				Severity: "critical",
				Category: "resource",
			},
			
			// HTTP errors
			{
				Name:     "HTTP4xx",
				Pattern:  regexp.MustCompile(`\b4\d{2}\b|(?i)(bad\s*request|unauthorized|forbidden|not\s*found)`),
				Severity: "medium",
				Category: "http",
			},
			{
				Name:     "HTTP5xx",
				Pattern:  regexp.MustCompile(`\b5\d{2}\b|(?i)(internal\s*server|gateway|service\s*unavailable)`),
				Severity: "high",
				Category: "http",
			},
			
			// Database errors
			{
				Name:     "DatabaseConnection",
				Pattern:  regexp.MustCompile(`(?i)(connection\s*(refused|failed|timeout)|can't\s*connect|lost\s*connection)`),
				Severity: "high",
				Category: "database",
			},
			{
				Name:     "QueryError",
				Pattern:  regexp.MustCompile(`(?i)(sql\s*error|query\s*failed|syntax\s*error|deadlock)`),
				Severity: "medium",
				Category: "database",
			},
			
			// System errors
			{
				Name:     "DiskSpace",
				Pattern:  regexp.MustCompile(`(?i)(disk\s*(full|space)|no\s*space\s*left)`),
				Severity: "critical",
				Category: "system",
			},
			{
				Name:     "Permission",
				Pattern:  regexp.MustCompile(`(?i)(permission\s*denied|access\s*denied|unauthorized\s*access)`),
				Severity: "medium",
				Category: "security",
			},
			{
				Name:     "Timeout",
				Pattern:  regexp.MustCompile(`(?i)(timeout|timed?\s*out|deadline\s*exceeded)`),
				Severity: "medium",
				Category: "network",
			},
			
			// Generic patterns
			{
				Name:     "Failed",
				Pattern:  regexp.MustCompile(`(?i)\bfailed?\b|\bfailure\b`),
				Severity: "medium",
				Category: "generic",
			},
			{
				Name:     "Critical",
				Pattern:  regexp.MustCompile(`(?i)\bcritical\b|\bfatal\b|\bpanic\b`),
				Severity: "critical",
				Category: "generic",
			},
		},
	}

	ed.anomalyDetector = NewAnomalyDetector(100) // 100 data points window
	
	// Start cleanup routine
	go ed.cleanupOldStats()

	return ed
}

// ProcessLog analyzes a log entry for errors
func (ed *ErrorDetector) ProcessLog(log *models.Log) []string {
	detectedErrors := []string{}

	// Check log level first
	if ed.isErrorLevel(log.Level) {
		ed.recordError("LogLevel", log.Level, "level", log)
		detectedErrors = append(detectedErrors, fmt.Sprintf("level:%s", log.Level))
	}

	// Check message against patterns
	for _, pattern := range ed.patterns {
		if pattern.Pattern.MatchString(log.Message) {
			ed.recordError(pattern.Name, pattern.Category, pattern.Category, log)
			detectedErrors = append(detectedErrors, fmt.Sprintf("%s:%s", pattern.Category, pattern.Name))
		}
	}

	// Check attributes for error indicators
	if log.Attributes != nil {
		if errMsg, ok := log.Attributes["error"].(string); ok && errMsg != "" {
			ed.recordError("AttributeError", "attribute", "application", log)
			detectedErrors = append(detectedErrors, "attribute:error")
		}
		
		if statusCode, ok := log.Attributes["status_code"].(float64); ok {
			if statusCode >= 400 {
				category := "http4xx"
				if statusCode >= 500 {
					category = "http5xx"
				}
				ed.recordError(fmt.Sprintf("HTTP%d", int(statusCode)), category, "http", log)
				detectedErrors = append(detectedErrors, fmt.Sprintf("http:%d", int(statusCode)))
			}
		}
	}

	return detectedErrors
}

// isErrorLevel checks if log level indicates an error
func (ed *ErrorDetector) isErrorLevel(level string) bool {
	errorLevels := []string{"error", "err", "fatal", "panic", "critical", "crit", "alert", "emerg"}
	levelLower := strings.ToLower(level)
	for _, errLevel := range errorLevels {
		if levelLower == errLevel {
			return true
		}
	}
	return false
}

// recordError records error statistics
func (ed *ErrorDetector) recordError(pattern, category, errorType string, log *models.Log) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	key := fmt.Sprintf("%s:%s", category, pattern)
	stats, exists := ed.errorStats[key]
	if !exists {
		stats = &ErrorStats{
			Pattern:   pattern,
			Category:  category,
			FirstSeen: log.Timestamp,
			Services:  make(map[string]int64),
			Samples:   make([]ErrorSample, 0, 10),
		}
		ed.errorStats[key] = stats
	}

	// Update stats
	stats.Count++
	stats.LastSeen = log.Timestamp
	stats.Services[log.Service]++

	// Keep up to 10 recent samples
	if len(stats.Samples) < 10 {
		stats.Samples = append(stats.Samples, ErrorSample{
			LogID:     log.ID,
			Timestamp: log.Timestamp,
			Service:   log.Service,
			Message:   log.Message,
			TraceID:   log.TraceID,
		})
	}

	// Update rate (errors per minute)
	duration := time.Since(stats.FirstSeen).Minutes()
	if duration > 0 {
		stats.Rate = float64(stats.Count) / duration
	}

	// Update anomaly detector
	ed.anomalyDetector.AddDataPoint(stats.Rate)
}

// GetErrorStats returns current error statistics
func (ed *ErrorDetector) GetErrorStats() map[string]*ErrorStats {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	// Calculate trends
	for _, stats := range ed.errorStats {
		stats.Trend = ed.calculateTrend(stats)
	}

	return ed.errorStats
}

// calculateTrend calculates error trend
func (ed *ErrorDetector) calculateTrend(stats *ErrorStats) string {
	// Simple trend calculation based on recent rate changes
	recentDuration := time.Since(stats.LastSeen)
	if recentDuration > ed.windowSize {
		return "stable" // No recent errors
	}

	// Compare current rate with historical average
	avgRate := float64(stats.Count) / time.Since(stats.FirstSeen).Minutes()
	if stats.Rate > avgRate*1.2 {
		return "increasing"
	} else if stats.Rate < avgRate*0.8 {
		return "decreasing"
	}

	return "stable"
}

// GetAnomalies detects anomalies in error rates
func (ed *ErrorDetector) GetAnomalies() []ErrorAnomaly {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	anomalies := []ErrorAnomaly{}
	
	for key, stats := range ed.errorStats {
		// Check rate threshold
		if stats.Rate > ed.alertThresholds.ErrorRatePerMinute {
			anomalies = append(anomalies, ErrorAnomaly{
				Type:        "high_error_rate",
				Pattern:     key,
				Category:    stats.Category,
				CurrentRate: stats.Rate,
				Threshold:   ed.alertThresholds.ErrorRatePerMinute,
				Severity:    "warning",
				Message:     fmt.Sprintf("Error rate %.2f/min exceeds threshold %.2f/min", stats.Rate, ed.alertThresholds.ErrorRatePerMinute),
			})
		}

		// Check anomaly detection
		if ed.anomalyDetector.IsAnomaly(stats.Rate, ed.alertThresholds.AnomalyStdDev) {
			anomalies = append(anomalies, ErrorAnomaly{
				Type:        "anomaly",
				Pattern:     key,
				Category:    stats.Category,
				CurrentRate: stats.Rate,
				Threshold:   ed.anomalyDetector.mean + ed.alertThresholds.AnomalyStdDev*ed.anomalyDetector.stdDev,
				Severity:    "critical",
				Message:     fmt.Sprintf("Error rate %.2f/min is anomalous (%.1f std devs from mean)", stats.Rate, (stats.Rate-ed.anomalyDetector.mean)/ed.anomalyDetector.stdDev),
			})
		}
	}

	return anomalies
}

// ErrorAnomaly represents an error anomaly
type ErrorAnomaly struct {
	Type        string  `json:"type"`
	Pattern     string  `json:"pattern"`
	Category    string  `json:"category"`
	CurrentRate float64 `json:"current_rate"`
	Threshold   float64 `json:"threshold"`
	Severity    string  `json:"severity"`
	Message     string  `json:"message"`
}

// cleanupOldStats removes old error statistics
func (ed *ErrorDetector) cleanupOldStats() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ed.mu.Lock()
		now := time.Now()
		for key, stats := range ed.errorStats {
			if now.Sub(stats.LastSeen) > 24*time.Hour {
				delete(ed.errorStats, key)
			}
		}
		ed.mu.Unlock()
	}
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(windowSize int) *AnomalyDetector {
	return &AnomalyDetector{
		history:    make([]float64, 0, windowSize),
		windowSize: windowSize,
	}
}

// AddDataPoint adds a new data point and updates statistics
func (ad *AnomalyDetector) AddDataPoint(value float64) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.history = append(ad.history, value)
	if len(ad.history) > ad.windowSize {
		ad.history = ad.history[1:]
	}

	ad.updateStats()
	ad.lastUpdate = time.Now()
}

// updateStats recalculates mean and standard deviation
func (ad *AnomalyDetector) updateStats() {
	if len(ad.history) == 0 {
		return
	}

	// Calculate mean
	sum := 0.0
	for _, v := range ad.history {
		sum += v
	}
	ad.mean = sum / float64(len(ad.history))

	// Calculate standard deviation
	variance := 0.0
	for _, v := range ad.history {
		variance += (v - ad.mean) * (v - ad.mean)
	}
	ad.stdDev = 0.0
	if len(ad.history) > 1 {
		ad.stdDev = variance / float64(len(ad.history)-1)
		if ad.stdDev > 0 {
			ad.stdDev = ad.stdDev * ad.stdDev // sqrt
		}
	}
}

// IsAnomaly checks if a value is anomalous
func (ad *AnomalyDetector) IsAnomaly(value float64, stdDevThreshold float64) bool {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	if len(ad.history) < 10 || ad.stdDev == 0 {
		return false // Not enough data
	}

	deviation := (value - ad.mean) / ad.stdDev
	return deviation > stdDevThreshold || deviation < -stdDevThreshold
}