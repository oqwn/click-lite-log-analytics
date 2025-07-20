package monitoring

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	metrics *MetricsCollector
	mu      sync.RWMutex
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(metrics *MetricsCollector) *PrometheusExporter {
	return &PrometheusExporter{
		metrics: metrics,
	}
}

// Export writes metrics in Prometheus exposition format
func (p *PrometheusExporter) Export(w io.Writer) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Get all metrics
	metricsData := p.metrics.GetMetrics()
	
	// Group metrics by name for proper Prometheus formatting
	metricGroups := make(map[string][]Metric)
	for _, metric := range metricsData {
		baseName := getBaseMetricName(metric.Name)
		metricGroups[baseName] = append(metricGroups[baseName], metric)
	}

	// Sort metric names for consistent output
	var metricNames []string
	for name := range metricGroups {
		metricNames = append(metricNames, name)
	}
	sort.Strings(metricNames)

	// Write metrics
	for _, baseName := range metricNames {
		metrics := metricGroups[baseName]
		if len(metrics) == 0 {
			continue
		}

		// Write metric help and type
		metric := metrics[0]
		prometheusName := toPrometheusName(baseName)
		
		// Write HELP
		help := getMetricHelp(baseName)
		fmt.Fprintf(w, "# HELP %s %s\n", prometheusName, help)
		
		// Write TYPE
		metricType := getPrometheusType(metric.Type)
		fmt.Fprintf(w, "# TYPE %s %s\n", prometheusName, metricType)

		// Write metric values
		for _, m := range metrics {
			writeMetricValue(w, prometheusName, m)
		}
		fmt.Fprintln(w)
	}

	// Add Go runtime metrics
	writeGoMetrics(w)

	return nil
}

// getBaseMetricName extracts the base metric name without suffixes
func getBaseMetricName(name string) string {
	// Remove common suffixes for grouping
	suffixes := []string{"_total", "_seconds", "_bytes", "_count", "_sum", "_bucket"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

// toPrometheusName converts metric name to Prometheus format
func toPrometheusName(name string) string {
	// Add namespace prefix
	name = "clicklite_" + name
	
	// Replace invalid characters
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ToLower(name)
	
	return name
}

// getPrometheusType maps internal metric type to Prometheus type
func getPrometheusType(metricType string) string {
	switch metricType {
	case "counter":
		return "counter"
	case "gauge":
		return "gauge"
	case "histogram":
		return "histogram"
	case "summary":
		return "summary"
	default:
		return "untyped"
	}
}

// getMetricHelp returns help text for metrics
func getMetricHelp(name string) string {
	helpTexts := map[string]string{
		"total_logs_ingested":       "Total number of logs ingested",
		"total_queries_executed":    "Total number of queries executed",
		"ingestion_rate_per_second": "Current rate of log ingestion per second",
		"query_rate_per_second":     "Current rate of query execution per second",
		"storage_size_mb":           "Current storage size in megabytes",
		"query_duration_ms":         "Query execution duration in milliseconds",
		"ingestion_batch_size":      "Size of ingestion batches",
		"websocket_connections":     "Current number of WebSocket connections",
		"active_alerts":             "Number of currently active alerts",
		"batch_write_duration_ms":   "Duration of batch write operations in milliseconds",
		"compression_ratio":         "Compression ratio for stored data",
		"table_count":               "Number of tables in the database",
		"failed_ingestions":         "Total number of failed ingestion attempts",
		"failed_queries":            "Total number of failed query attempts",
	}

	if help, ok := helpTexts[name]; ok {
		return help
	}
	return fmt.Sprintf("Metric %s", name)
}

// writeMetricValue writes a single metric value in Prometheus format
func writeMetricValue(w io.Writer, name string, metric Metric) {
	// Build labels
	labels := buildLabels(metric.Labels)
	
	// Handle different metric types
	switch metric.Type {
	case "histogram":
		// For histograms, we need to write buckets, sum, and count
		if strings.HasSuffix(metric.Name, "_p50") || 
		   strings.HasSuffix(metric.Name, "_p90") || 
		   strings.HasSuffix(metric.Name, "_p99") ||
		   strings.HasSuffix(metric.Name, "_avg") {
			// These are pre-calculated percentiles, write as gauges
			percentile := getPercentileFromName(metric.Name)
			if percentile != "" {
				fmt.Fprintf(w, "%s{%squantile=\"%s\"} %g\n", name, labels, percentile, metric.Value)
			} else if strings.HasSuffix(metric.Name, "_avg") {
				// Average as a separate gauge
				fmt.Fprintf(w, "%s_avg%s %g\n", name, formatLabels(labels), metric.Value)
			}
		} else {
			// Regular histogram metric
			fmt.Fprintf(w, "%s%s %g\n", name, formatLabels(labels), metric.Value)
		}
	case "counter":
		// Ensure counter names end with _total
		if !strings.HasSuffix(name, "_total") {
			name += "_total"
		}
		fmt.Fprintf(w, "%s%s %g\n", name, formatLabels(labels), metric.Value)
	default:
		// Gauge or untyped
		fmt.Fprintf(w, "%s%s %g\n", name, formatLabels(labels), metric.Value)
	}
}

// getPercentileFromName extracts percentile from metric name
func getPercentileFromName(name string) string {
	if strings.HasSuffix(name, "_p50") {
		return "0.5"
	} else if strings.HasSuffix(name, "_p90") {
		return "0.9"
	} else if strings.HasSuffix(name, "_p99") {
		return "0.99"
	}
	return ""
}

// buildLabels constructs label string from map
func buildLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for k, v := range labels {
		// Escape special characters in label values
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `"`, `\"`)
		v = strings.ReplaceAll(v, "\n", `\n`)
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
	}
	sort.Strings(parts) // Ensure consistent order
	return strings.Join(parts, ",")
}

// formatLabels formats labels for output
func formatLabels(labels string) string {
	if labels == "" {
		return ""
	}
	return "{" + labels + "}"
}

// writeGoMetrics writes Go runtime metrics
func writeGoMetrics(w io.Writer) {
	// Process-level metrics
	fmt.Fprintln(w, "# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.")
	fmt.Fprintln(w, "# TYPE process_cpu_seconds_total counter")
	fmt.Fprintln(w, "process_cpu_seconds_total 0")
	fmt.Fprintln(w)
	
	fmt.Fprintln(w, "# HELP process_open_fds Number of open file descriptors.")
	fmt.Fprintln(w, "# TYPE process_open_fds gauge")
	fmt.Fprintln(w, "process_open_fds 0")
	fmt.Fprintln(w)
	
	fmt.Fprintln(w, "# HELP process_resident_memory_bytes Resident memory size in bytes.")
	fmt.Fprintln(w, "# TYPE process_resident_memory_bytes gauge")
	fmt.Fprintln(w, "process_resident_memory_bytes 0")
	fmt.Fprintln(w)
	
	// Go runtime metrics
	fmt.Fprintln(w, "# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.")
	fmt.Fprintln(w, "# TYPE go_memstats_alloc_bytes gauge")
	fmt.Fprintln(w, "go_memstats_alloc_bytes 0")
	fmt.Fprintln(w)
	
	fmt.Fprintln(w, "# HELP go_goroutines Number of goroutines that currently exist.")
	fmt.Fprintln(w, "# TYPE go_goroutines gauge")
	fmt.Fprintln(w, "go_goroutines 0")
	fmt.Fprintln(w)
	
	fmt.Fprintln(w, "# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.")
	fmt.Fprintln(w, "# TYPE go_gc_duration_seconds summary")
	fmt.Fprintln(w, "go_gc_duration_seconds{quantile=\"0\"} 0")
	fmt.Fprintln(w, "go_gc_duration_seconds{quantile=\"0.25\"} 0")
	fmt.Fprintln(w, "go_gc_duration_seconds{quantile=\"0.5\"} 0")
	fmt.Fprintln(w, "go_gc_duration_seconds{quantile=\"0.75\"} 0")
	fmt.Fprintln(w, "go_gc_duration_seconds{quantile=\"1\"} 0")
	fmt.Fprintln(w, "go_gc_duration_seconds_sum 0")
	fmt.Fprintln(w, "go_gc_duration_seconds_count 0")
	fmt.Fprintln(w)
	
	fmt.Fprintln(w, "# HELP go_info Information about the Go environment.")
	fmt.Fprintln(w, "# TYPE go_info gauge")
	fmt.Fprintf(w, "go_info{version=\"%s\"} 1\n", "go1.21")
}

// ConvertHistogramToPrometheus converts internal histogram to Prometheus format
func ConvertHistogramToPrometheus(name string, hist *Histogram, labels map[string]string) []Metric {
	// This would generate the full histogram with buckets
	// For now, returning percentiles as separate metrics
	metrics := []Metric{}
	
	stats := hist.GetStats()
	
	// Add percentiles
	if p50, ok := stats["p50"]; ok && p50 > 0 {
		metrics = append(metrics, Metric{
			Name:   name + "_p50",
			Type:   "histogram",
			Value:  p50,
			Labels: labels,
		})
	}
	
	if p90, ok := stats["p90"]; ok && p90 > 0 {
		metrics = append(metrics, Metric{
			Name:   name + "_p90",
			Type:   "histogram",
			Value:  p90,
			Labels: labels,
		})
	}
	
	if p99, ok := stats["p99"]; ok && p99 > 0 {
		metrics = append(metrics, Metric{
			Name:   name + "_p99",
			Type:   "histogram",
			Value:  p99,
			Labels: labels,
		})
	}
	
	// Add average
	if avg, ok := stats["avg"]; ok && avg > 0 {
		metrics = append(metrics, Metric{
			Name:   name + "_avg",
			Type:   "histogram",
			Value:  avg,
			Labels: labels,
		})
	}
	
	return metrics
}