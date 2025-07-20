package monitoring

import (
	"fmt"
	"sync"
	"time"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
)

// Alert represents a system alert
type Alert struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Severity    AlertSeverity `json:"severity"`
	Status      AlertStatus   `json:"status"`
	Message     string        `json:"message"`
	Source      string        `json:"source"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     *time.Time    `json:"end_time,omitempty"`
	LastUpdated time.Time     `json:"last_updated"`
	Count       int           `json:"count"`
	Details     interface{}   `json:"details,omitempty"`
}

// AlertRule defines a rule for generating alerts
type AlertRule struct {
	Name        string
	Description string
	Severity    AlertSeverity
	Condition   func(metrics []Metric) (bool, string)
	Cooldown    time.Duration
}

// AlertManager manages system alerts
type AlertManager struct {
	mu          sync.RWMutex
	alerts      map[string]*Alert
	rules       []AlertRule
	lastChecked map[string]time.Time
	listeners   []AlertListener
	metrics     *MetricsCollector
}

// AlertListener interface for alert notifications
type AlertListener interface {
	OnAlert(alert *Alert)
}

// NewAlertManager creates a new alert manager
func NewAlertManager(metrics *MetricsCollector) *AlertManager {
	am := &AlertManager{
		alerts:      make(map[string]*Alert),
		rules:       make([]AlertRule, 0),
		lastChecked: make(map[string]time.Time),
		metrics:     metrics,
	}
	
	// Register default alert rules
	am.registerDefaultRules()
	
	return am
}

// AddListener adds an alert listener
func (am *AlertManager) AddListener(listener AlertListener) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.listeners = append(am.listeners, listener)
}

// AddRule adds a custom alert rule
func (am *AlertManager) AddRule(rule AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules = append(am.rules, rule)
}

// CheckAlerts evaluates all alert rules
func (am *AlertManager) CheckAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	metrics := am.metrics.GetMetrics()
	now := time.Now()
	
	for _, rule := range am.rules {
		// Check cooldown
		if lastCheck, exists := am.lastChecked[rule.Name]; exists {
			if now.Sub(lastCheck) < rule.Cooldown {
				continue
			}
		}
		
		// Evaluate condition
		triggered, message := rule.Condition(metrics)
		alertID := fmt.Sprintf("%s_%d", rule.Name, now.Unix())
		
		if triggered {
			// Check if alert already exists
			existingAlert := am.findActiveAlert(rule.Name)
			if existingAlert != nil {
				// Update existing alert
				existingAlert.Count++
				existingAlert.LastUpdated = now
				existingAlert.Message = message
			} else {
				// Create new alert
				alert := &Alert{
					ID:          alertID,
					Name:        rule.Name,
					Severity:    rule.Severity,
					Status:      AlertStatusActive,
					Message:     message,
					Source:      "system",
					StartTime:   now,
					LastUpdated: now,
					Count:       1,
				}
				am.alerts[alertID] = alert
				am.notifyListeners(alert)
			}
			am.lastChecked[rule.Name] = now
		} else {
			// Resolve existing alert if condition is no longer met
			if existingAlert := am.findActiveAlert(rule.Name); existingAlert != nil {
				existingAlert.Status = AlertStatusResolved
				existingAlert.EndTime = &now
				existingAlert.LastUpdated = now
				am.notifyListeners(existingAlert)
			}
		}
	}
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var activeAlerts []*Alert
	for _, alert := range am.alerts {
		if alert.Status == AlertStatusActive {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	
	return activeAlerts
}

// GetAllAlerts returns all alerts (active and resolved)
func (am *AlertManager) GetAllAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var allAlerts []*Alert
	for _, alert := range am.alerts {
		allAlerts = append(allAlerts, alert)
	}
	
	return allAlerts
}

// findActiveAlert finds an active alert by name
func (am *AlertManager) findActiveAlert(name string) *Alert {
	for _, alert := range am.alerts {
		if alert.Name == name && alert.Status == AlertStatusActive {
			return alert
		}
	}
	return nil
}

// notifyListeners notifies all listeners of an alert
func (am *AlertManager) notifyListeners(alert *Alert) {
	for _, listener := range am.listeners {
		go listener.OnAlert(alert)
	}
}

// registerDefaultRules registers default alert rules
func (am *AlertManager) registerDefaultRules() {
	// High ingestion rate alert
	am.AddRule(AlertRule{
		Name:        "high_ingestion_rate",
		Description: "Log ingestion rate is abnormally high",
		Severity:    SeverityWarning,
		Cooldown:    5 * time.Minute,
		Condition: func(metrics []Metric) (bool, string) {
			for _, m := range metrics {
				if m.Name == "ingestion_rate_per_second" && m.Value > 10000 {
					return true, fmt.Sprintf("Ingestion rate is %.0f logs/sec (threshold: 10000)", m.Value)
				}
			}
			return false, ""
		},
	})
	
	// Slow query alert
	am.AddRule(AlertRule{
		Name:        "slow_queries",
		Description: "Queries are taking too long to execute",
		Severity:    SeverityWarning,
		Cooldown:    5 * time.Minute,
		Condition: func(metrics []Metric) (bool, string) {
			for _, m := range metrics {
				if m.Name == "query_duration_ms_p99" && m.Value > 5000 {
					return true, fmt.Sprintf("99th percentile query duration is %.0fms (threshold: 5000ms)", m.Value)
				}
			}
			return false, ""
		},
	})
	
	// High memory usage alert
	am.AddRule(AlertRule{
		Name:        "high_memory_usage",
		Description: "Memory usage is too high",
		Severity:    SeverityCritical,
		Cooldown:    5 * time.Minute,
		Condition: func(metrics []Metric) (bool, string) {
			var allocMB float64
			for _, m := range metrics {
				if m.Name == "memory_alloc_mb" {
					allocMB = m.Value
					break
				}
			}
			
			if allocMB > 1024 { // 1GB threshold
				return true, fmt.Sprintf("Memory usage is %.0fMB (threshold: 1024MB)", allocMB)
			}
			return false, ""
		},
	})
	
	// Storage space alert
	am.AddRule(AlertRule{
		Name:        "low_storage_space",
		Description: "Storage space is running low",
		Severity:    SeverityCritical,
		Cooldown:    30 * time.Minute,
		Condition: func(metrics []Metric) (bool, string) {
			for _, m := range metrics {
				if m.Name == "storage_free_percent" && m.Value < 10 {
					return true, fmt.Sprintf("Only %.1f%% storage space remaining", m.Value)
				}
			}
			return false, ""
		},
	})
	
	// No recent logs alert
	am.AddRule(AlertRule{
		Name:        "no_recent_logs",
		Description: "No logs received recently",
		Severity:    SeverityInfo,
		Cooldown:    10 * time.Minute,
		Condition: func(metrics []Metric) (bool, string) {
			for _, m := range metrics {
				if m.Name == "ingestion_rate_per_second" && m.Value == 0 {
					return true, "No logs received in the last minute"
				}
			}
			return false, ""
		},
	})
}

// LogAlertListener logs alerts to the console
type LogAlertListener struct {
}

// NewLogAlertListener creates a new log alert listener
func NewLogAlertListener(logger interface{}) *LogAlertListener {
	return &LogAlertListener{}
}

// OnAlert handles alert notifications
func (l *LogAlertListener) OnAlert(alert *Alert) {
	msg := fmt.Sprintf("Alert [%s]: %s - %s", alert.Severity, alert.Name, alert.Message)
	fmt.Println(msg)
}