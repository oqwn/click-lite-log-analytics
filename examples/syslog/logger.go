package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

// Syslog severity levels
const (
	LOG_EMERG   = 0
	LOG_ALERT   = 1
	LOG_CRIT    = 2
	LOG_ERR     = 3
	LOG_WARNING = 4
	LOG_NOTICE  = 5
	LOG_INFO    = 6
	LOG_DEBUG   = 7
)

// Syslog facility codes
const (
	LOG_KERN     = 0
	LOG_USER     = 1
	LOG_MAIL     = 2
	LOG_DAEMON   = 3
	LOG_AUTH     = 4
	LOG_SYSLOG   = 5
	LOG_LPR      = 6
	LOG_NEWS     = 7
	LOG_UUCP     = 8
	LOG_CRON     = 9
	LOG_AUTHPRIV = 10
	LOG_FTP      = 11
	LOG_LOCAL0   = 16
	LOG_LOCAL1   = 17
	LOG_LOCAL2   = 18
	LOG_LOCAL3   = 19
	LOG_LOCAL4   = 20
	LOG_LOCAL5   = 21
	LOG_LOCAL6   = 22
	LOG_LOCAL7   = 23
)

func main() {
	// Connect to syslog server
	conn, err := net.Dial("udp", "localhost:20004")
	if err != nil {
		log.Fatalf("Failed to connect to syslog server: %v", err)
	}
	defer conn.Close()
	
	fmt.Println("Connected to syslog server")
	
	// Send RFC3164 formatted messages
	hostname := "example-host"
	tag := "myapp"
	pid := 12345
	
	// Example 1: Send various severity levels
	sendSyslog(conn, LOG_LOCAL0, LOG_INFO, hostname, tag, pid, 
		"Application started successfully")
	
	sendSyslog(conn, LOG_LOCAL0, LOG_WARNING, hostname, tag, pid, 
		"Configuration file not found, using defaults")
	
	sendSyslog(conn, LOG_LOCAL0, LOG_ERR, hostname, tag, pid, 
		"Failed to connect to database, retrying...")
	
	sendSyslog(conn, LOG_LOCAL0, LOG_DEBUG, hostname, tag, pid, 
		"Debug mode enabled, verbose logging active")
	
	// Example 2: Send RFC5424 formatted messages
	sendSyslogRFC5424(conn, LOG_LOCAL1, LOG_INFO, hostname, "webapp", "1234", "ID47",
		`[exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"] Application event`)
	
	sendSyslogRFC5424(conn, LOG_LOCAL1, LOG_NOTICE, hostname, "webapp", "1234", "ID48",
		`[exampleSDID@32473 iut="9" eventSource="Security" eventID="2022"] User login successful`)
	
	// Example 3: Simulate application lifecycle events
	fmt.Println("\nSimulating application lifecycle...")
	
	// Startup
	sendSyslog(conn, LOG_DAEMON, LOG_NOTICE, hostname, "lifecycle", pid,
		"Service starting up")
	time.Sleep(1 * time.Second)
	
	sendSyslog(conn, LOG_DAEMON, LOG_INFO, hostname, "lifecycle", pid,
		"Loading configuration from /etc/myapp/config.yml")
	time.Sleep(500 * time.Millisecond)
	
	sendSyslog(conn, LOG_DAEMON, LOG_INFO, hostname, "lifecycle", pid,
		"Connecting to upstream services")
	time.Sleep(500 * time.Millisecond)
	
	sendSyslog(conn, LOG_DAEMON, LOG_NOTICE, hostname, "lifecycle", pid,
		"Service ready to accept connections")
	
	// Normal operation
	for i := 0; i < 5; i++ {
		sendSyslog(conn, LOG_USER, LOG_INFO, hostname, "worker", pid,
			fmt.Sprintf("Processing job #%d", i+1))
		time.Sleep(2 * time.Second)
	}
	
	// Shutdown
	sendSyslog(conn, LOG_DAEMON, LOG_NOTICE, hostname, "lifecycle", pid,
		"Received shutdown signal")
	time.Sleep(500 * time.Millisecond)
	
	sendSyslog(conn, LOG_DAEMON, LOG_INFO, hostname, "lifecycle", pid,
		"Gracefully stopping workers")
	time.Sleep(500 * time.Millisecond)
	
	sendSyslog(conn, LOG_DAEMON, LOG_NOTICE, hostname, "lifecycle", pid,
		"Service stopped")
	
	fmt.Println("\nSyslog examples completed")
}

// sendSyslog sends an RFC3164 formatted syslog message
func sendSyslog(conn net.Conn, facility, severity int, hostname, tag string, pid int, message string) {
	priority := facility*8 + severity
	timestamp := time.Now().Format("Jan _2 15:04:05")
	
	// RFC3164 format: <priority>timestamp hostname tag[pid]: message
	syslogMsg := fmt.Sprintf("<%d>%s %s %s[%d]: %s",
		priority, timestamp, hostname, tag, pid, message)
	
	_, err := conn.Write([]byte(syslogMsg))
	if err != nil {
		log.Printf("Failed to send syslog: %v", err)
		return
	}
	
	fmt.Printf("Sent RFC3164: %s\n", syslogMsg)
}

// sendSyslogRFC5424 sends an RFC5424 formatted syslog message
func sendSyslogRFC5424(conn net.Conn, facility, severity int, hostname, appName, procID, msgID, message string) {
	priority := facility*8 + severity
	version := 1
	timestamp := time.Now().Format(time.RFC3339)
	
	// RFC5424 format: <priority>version timestamp hostname app-name procid msgid [structured-data] msg
	syslogMsg := fmt.Sprintf("<%d>%d %s %s %s %s %s %s",
		priority, version, timestamp, hostname, appName, procID, msgID, message)
	
	_, err := conn.Write([]byte(syslogMsg))
	if err != nil {
		log.Printf("Failed to send syslog: %v", err)
		return
	}
	
	fmt.Printf("Sent RFC5424: %s\n", syslogMsg)
}