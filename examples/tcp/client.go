package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	// Connect to TCP server
	conn, err := net.Dial("tcp", "localhost:20003")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	fmt.Println("Connected to TCP log server")
	
	// Send plain text logs
	messages := []string{
		"Application starting up",
		"Connected to database successfully",
		"Server listening on port 8080",
		"Processing user request for ID: 12345",
		"Cache initialized with 1000 entries",
		"Background job scheduler started",
		"Health check endpoint registered",
		"Graceful shutdown initiated",
	}
	
	reader := bufio.NewReader(conn)
	
	for i, msg := range messages {
		// Send log message
		logLine := fmt.Sprintf("[%s] INFO: %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
		_, err := conn.Write([]byte(logLine))
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
		
		fmt.Printf("Sent log %d: %s", i+1, logLine)
		fmt.Printf("Server response: %s", response)
		
		time.Sleep(500 * time.Millisecond)
	}
	
	// Send continuous logs
	fmt.Println("\nStarting continuous log stream (press Ctrl+C to stop)...")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	counter := 0
	for range ticker.C {
		counter++
		logLine := fmt.Sprintf("[%s] INFO: Heartbeat message #%d from TCP client\n", 
			time.Now().Format("2006-01-02 15:04:05"), counter)
		
		_, err := conn.Write([]byte(logLine))
		if err != nil {
			log.Printf("Failed to send heartbeat: %v", err)
			break
		}
		
		// Read acknowledgment
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Failed to read response: %v", err)
			break
		}
		
		fmt.Printf("Sent heartbeat #%d, response: %s", counter, response)
	}
}