package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Service    string                 `json:"service"`
	TraceID    string                 `json:"trace_id,omitempty"`
	SpanID     string                 `json:"span_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

func main() {
	// Connect to TCP server
	conn, err := net.Dial("tcp", "localhost:20003")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	fmt.Println("Connected to TCP log server (JSON mode)")
	
	reader := bufio.NewReader(conn)
	
	// Send structured JSON logs
	for i := 0; i < 10; i++ {
		logEntry := generateLogEntry(i)
		
		// Convert to JSON
		data, err := json.Marshal(logEntry)
		if err != nil {
			log.Printf("Failed to marshal log: %v", err)
			continue
		}
		
		// Send with newline delimiter
		_, err = conn.Write(append(data, '\n'))
		if err != nil {
			log.Printf("Failed to send log: %v", err)
			continue
		}
		
		// Read acknowledgment
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Failed to read response: %v", err)
			continue
		}
		
		fmt.Printf("Sent JSON log %d: %s\n", i+1, string(data))
		fmt.Printf("Server response: %s", response)
		
		time.Sleep(1 * time.Second)
	}
	
	// Simulate a real application sending various log types
	fmt.Println("\nSimulating application logs...")
	
	// Startup sequence
	sendAppLog(conn, reader, "info", "Application starting", map[string]interface{}{
		"version": "2.1.0",
		"config":  "production",
	})
	
	sendAppLog(conn, reader, "info", "Database connection pool initialized", map[string]interface{}{
		"pool_size": 10,
		"timeout":   30,
	})
	
	sendAppLog(conn, reader, "info", "HTTP server started", map[string]interface{}{
		"port":    8080,
		"workers": 4,
	})
	
	// Simulate request processing
	for i := 0; i < 5; i++ {
		traceID := fmt.Sprintf("req-%d", rand.Intn(10000))
		
		sendAppLog(conn, reader, "info", "Incoming HTTP request", map[string]interface{}{
			"method":   "GET",
			"path":     "/api/users",
			"trace_id": traceID,
		})
		
		sendAppLog(conn, reader, "debug", "Query executed", map[string]interface{}{
			"query":     "SELECT * FROM users LIMIT 10",
			"duration":  rand.Intn(100),
			"trace_id":  traceID,
		})
		
		if rand.Float32() < 0.3 {
			sendAppLog(conn, reader, "error", "Request failed", map[string]interface{}{
				"error":     "Database connection timeout",
				"trace_id":  traceID,
			})
		} else {
			sendAppLog(conn, reader, "info", "Request completed", map[string]interface{}{
				"status":    200,
				"duration":  rand.Intn(500),
				"trace_id":  traceID,
			})
		}
		
		time.Sleep(2 * time.Second)
	}
}

func generateLogEntry(index int) LogEntry {
	levels := []string{"debug", "info", "warn", "error"}
	services := []string{"tcp-app", "worker", "scheduler", "api"}
	
	return LogEntry{
		Timestamp: time.Now(),
		Level:     levels[rand.Intn(len(levels))],
		Message:   fmt.Sprintf("Log message #%d from JSON TCP client", index+1),
		Service:   services[rand.Intn(len(services))],
		TraceID:   fmt.Sprintf("trace-%d", rand.Intn(1000)),
		SpanID:    fmt.Sprintf("span-%d", rand.Intn(10000)),
		Attributes: map[string]interface{}{
			"index":      index,
			"client":     "tcp-json",
			"random":     rand.Float64(),
			"timestamp":  time.Now().Unix(),
		},
	}
}

func sendAppLog(conn net.Conn, reader *bufio.Reader, level, message string, attrs map[string]interface{}) {
	logEntry := LogEntry{
		Timestamp:  time.Now(),
		Level:      level,
		Message:    message,
		Service:    "demo-app",
		Attributes: attrs,
	}
	
	data, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("Failed to marshal log: %v", err)
		return
	}
	
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		log.Printf("Failed to send log: %v", err)
		return
	}
	
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read response: %v", err)
		return
	}
	
	fmt.Printf("[%s] %s: %s %v\n", level, message, response[:len(response)-1], attrs)
}