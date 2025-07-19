package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Config holds the agent configuration
type Config struct {
	// Endpoint is the URL to send logs to
	Endpoint string
	// BatchSize is the number of logs to batch before sending
	BatchSize int
	// FlushInterval is how often to flush logs
	FlushInterval time.Duration
	// MaxRetries is the maximum number of retries for failed sends
	MaxRetries int
	// Service name to tag logs with
	Service string
	// Additional attributes to add to all logs
	Attributes map[string]interface{}
	// HTTPTimeout for requests
	HTTPTimeout time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Endpoint:      "http://localhost:20002/api/v1/ingest/logs",
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		Service:       "unknown",
		Attributes:    make(map[string]interface{}),
		HTTPTimeout:   10 * time.Second,
	}
}

// Agent collects and ships logs to Click-Lite
type Agent struct {
	config     *Config
	buffer     []LogEntry
	bufferMu   sync.Mutex
	client     *http.Client
	stopChan   chan struct{}
	flushChan  chan struct{}
	wg         sync.WaitGroup
}

// LogEntry represents a log entry
type LogEntry struct {
	ID         string                 `json:"id,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Service    string                 `json:"service"`
	TraceID    string                 `json:"trace_id,omitempty"`
	SpanID     string                 `json:"span_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// New creates a new log agent
func New(config *Config) *Agent {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &Agent{
		config: config,
		buffer: make([]LogEntry, 0, config.BatchSize),
		client: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		stopChan:  make(chan struct{}),
		flushChan: make(chan struct{}, 1),
	}
}

// Start starts the agent
func (a *Agent) Start() {
	a.wg.Add(1)
	go a.run()
}

// Stop gracefully stops the agent
func (a *Agent) Stop() {
	close(a.stopChan)
	a.wg.Wait()
}

// Log sends a log entry
func (a *Agent) Log(level, message string) {
	a.LogWithFields(level, message, nil)
}

// LogWithFields sends a log entry with additional fields
func (a *Agent) LogWithFields(level, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp:  time.Now(),
		Level:      level,
		Message:    message,
		Service:    a.config.Service,
		Attributes: make(map[string]interface{}),
	}
	
	// Add default attributes
	for k, v := range a.config.Attributes {
		entry.Attributes[k] = v
	}
	
	// Add provided fields
	for k, v := range fields {
		entry.Attributes[k] = v
	}
	
	a.addToBuffer(entry)
}

// LogError logs an error
func (a *Agent) LogError(err error, message string) {
	a.LogWithFields("error", message, map[string]interface{}{
		"error": err.Error(),
	})
}

// addToBuffer adds a log entry to the buffer
func (a *Agent) addToBuffer(entry LogEntry) {
	a.bufferMu.Lock()
	a.buffer = append(a.buffer, entry)
	shouldFlush := len(a.buffer) >= a.config.BatchSize
	a.bufferMu.Unlock()
	
	if shouldFlush {
		select {
		case a.flushChan <- struct{}{}:
		default:
		}
	}
}

// run is the main agent loop
func (a *Agent) run() {
	defer a.wg.Done()
	
	ticker := time.NewTicker(a.config.FlushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-a.stopChan:
			a.flush()
			return
		case <-ticker.C:
			a.flush()
		case <-a.flushChan:
			a.flush()
		}
	}
}

// flush sends the buffered logs
func (a *Agent) flush() {
	a.bufferMu.Lock()
	if len(a.buffer) == 0 {
		a.bufferMu.Unlock()
		return
	}
	
	// Copy and reset buffer
	batch := make([]LogEntry, len(a.buffer))
	copy(batch, a.buffer)
	a.buffer = a.buffer[:0]
	a.bufferMu.Unlock()
	
	// Send with retries
	for i := 0; i < a.config.MaxRetries; i++ {
		if err := a.send(batch); err != nil {
			log.Error().Err(err).Int("attempt", i+1).Msg("Failed to send logs")
			if i < a.config.MaxRetries-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
			}
			continue
		}
		return
	}
	
	log.Error().Int("batch_size", len(batch)).Msg("Failed to send logs after all retries")
}

// send sends a batch of logs to the server
func (a *Agent) send(batch []LogEntry) error {
	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}
	
	req, err := http.NewRequest("POST", a.config.Endpoint, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	return nil
}

// Convenience functions for different log levels

func (a *Agent) Debug(message string) {
	a.Log("debug", message)
}

func (a *Agent) Info(message string) {
	a.Log("info", message)
}

func (a *Agent) Warn(message string) {
	a.Log("warn", message)
}

func (a *Agent) Error(message string) {
	a.Log("error", message)
}

func (a *Agent) Fatal(message string) {
	a.Log("fatal", message)
}