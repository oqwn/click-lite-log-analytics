package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

func main() {
	// Test real-time log streaming with WebSocket
	fmt.Println("Testing WebSocket real-time log streaming...")

	// Start WebSocket client
	go startWebSocketClient()

	// Wait a moment for connection
	time.Sleep(2 * time.Second)

	// Send some test logs
	sendTestLogs()

	// Keep the program running to observe logs
	time.Sleep(30 * time.Second)
}

func startWebSocketClient() {
	u := url.URL{Scheme: "ws", Host: "localhost:20002", Path: "/api/v1/ws"}
	fmt.Printf("Connecting to WebSocket: %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("Failed to connect to WebSocket: %v\n", err)
		return
	}
	defer c.Close()

	// Send filter for error logs only
	filterMsg := models.WebSocketMessage{
		Type: "filter",
		Filters: []models.LogFilter{
			{
				Field:    "level",
				Operator: "equals",
				Value:    "error",
			},
		},
	}

	if err := c.WriteJSON(filterMsg); err != nil {
		fmt.Printf("Failed to send filter: %v\n", err)
		return
	}

	fmt.Println("WebSocket connected, listening for logs...")

	// Read messages
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			fmt.Printf("WebSocket read error: %v\n", err)
			break
		}

		var msg models.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			fmt.Printf("Failed to parse message: %v\n", err)
			continue
		}

		switch msg.Type {
		case "connection":
			fmt.Printf("✅ Connected: %+v\n", msg.Data)
		case "log":
			if logData, ok := msg.Data.(map[string]interface{}); ok {
				fmt.Printf("📝 Log received: [%s] %s: %s\n", 
					logData["level"], 
					logData["service"], 
					logData["message"])
			}
		case "status":
			fmt.Printf("📊 Status: %+v\n", msg.Data)
		}
	}
}

func sendTestLogs() {
	logs := []models.Log{
		{
			Timestamp: time.Now(),
			Level:     "info",
			Service:   "websocket-test",
			Message:   "Test info log message",
			Attributes: map[string]interface{}{
				"test": "websocket",
			},
		},
		{
			Timestamp: time.Now(),
			Level:     "error",
			Service:   "websocket-test",
			Message:   "Test error log message - should be visible",
			Attributes: map[string]interface{}{
				"test":  "websocket",
				"alert": true,
			},
		},
		{
			Timestamp: time.Now(),
			Level:     "debug",
			Service:   "websocket-test",
			Message:   "Test debug log message",
		},
		{
			Timestamp: time.Now(),
			Level:     "error",
			Service:   "websocket-test",
			Message:   "Another error log - should also be visible",
			TraceID:   "trace-123",
		},
	}

	for i, logEntry := range logs {
		fmt.Printf("Sending log %d: %s\n", i+1, logEntry.Message)

		payload := map[string]interface{}{
			"log": logEntry,
			"options": map[string]bool{
				"enable_parsing":    false,
				"enable_validation": false,
			},
		}

		jsonData, _ := json.Marshal(payload)
		resp, err := http.Post(
			"http://localhost:20002/api/v1/logs",
			"application/json",
			bytes.NewBuffer(jsonData),
		)

		if err != nil {
			fmt.Printf("Failed to send log: %v\n", err)
			continue
		}

		resp.Body.Close()
		time.Sleep(1 * time.Second) // Space out the logs
	}

	fmt.Println("✅ All test logs sent")
}