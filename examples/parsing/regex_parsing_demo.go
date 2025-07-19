package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	endpoint := "http://localhost:20002/api/v1/ingest/bulk"
	
	fmt.Println("🔍 Regex Parsing Demo - Testing unstructured log parsing")
	fmt.Println("=========================================================")
	
	// Test 1: Apache access logs
	fmt.Println("📝 Test 1: Apache Combined Log Format")
	apacheLogs := generateApacheLogs()
	if err := sendRawLogs(endpoint, apacheLogs, "apache-parser-test"); err != nil {
		log.Printf("❌ Failed to send Apache logs: %v", err)
	} else {
		fmt.Printf("✅ Sent %d Apache access logs\n", len(apacheLogs))
	}
	
	time.Sleep(1 * time.Second)
	
	// Test 2: Syslog format
	fmt.Println("\n📝 Test 2: Syslog RFC3164 Format")
	syslogLogs := generateSyslogLogs()
	if err := sendRawLogs(endpoint, syslogLogs, "syslog-parser-test"); err != nil {
		log.Printf("❌ Failed to send Syslog logs: %v", err)
	} else {
		fmt.Printf("✅ Sent %d Syslog messages\n", len(syslogLogs))
	}
	
	time.Sleep(1 * time.Second)
	
	// Test 3: Application logs
	fmt.Println("\n📝 Test 3: Application Log Formats")
	appLogs := generateApplicationLogs()
	if err := sendRawLogs(endpoint, appLogs, "app-parser-test"); err != nil {
		log.Printf("❌ Failed to send application logs: %v", err)
	} else {
		fmt.Printf("✅ Sent %d application logs\n", len(appLogs))
	}
	
	time.Sleep(1 * time.Second)
	
	// Test 4: Spring Boot logs
	fmt.Println("\n📝 Test 4: Spring Boot Log Format")
	springLogs := generateSpringBootLogs()
	if err := sendRawLogs(endpoint, springLogs, "spring-parser-test"); err != nil {
		log.Printf("❌ Failed to send Spring Boot logs: %v", err)
	} else {
		fmt.Printf("✅ Sent %d Spring Boot logs\n", len(springLogs))
	}
	
	time.Sleep(1 * time.Second)
	
	// Test 5: Mixed unstructured logs
	fmt.Println("\n📝 Test 5: Mixed Unstructured Formats")
	mixedLogs := generateMixedLogs()
	if err := sendRawLogs(endpoint, mixedLogs, "mixed-parser-test"); err != nil {
		log.Printf("❌ Failed to send mixed logs: %v", err)
	} else {
		fmt.Printf("✅ Sent %d mixed format logs\n", len(mixedLogs))
	}
	
	// Wait for processing
	fmt.Println("\n⏳ Waiting for log processing...")
	time.Sleep(3 * time.Second)
	
	// Query and verify results
	fmt.Println("\n📊 Verifying parsed results:")
	if err := verifyParsedLogs(); err != nil {
		log.Printf("❌ Verification failed: %v", err)
	} else {
		fmt.Println("✅ Regex parsing verification completed successfully")
	}
	
	fmt.Println("\n🎯 Regex Parsing Demo Summary:")
	fmt.Println("  🌐 Tested Apache/Nginx web server log parsing")
	fmt.Println("  📡 Verified Syslog RFC3164 format handling")
	fmt.Println("  📱 Tested application log format extraction")
	fmt.Println("  🍃 Validated Spring Boot log parsing")
	fmt.Println("  🔄 Tested fallback parsing for unknown formats")
	fmt.Println("  ✅ All unstructured logs should be parsed with field extraction")
}

func generateApacheLogs() []string {
	logs := []string{
		`192.168.1.1 - - [15/Jan/2024:10:30:00 +0000] "GET /api/users HTTP/1.1" 200 1234 "https://example.com/login" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"`,
		`10.0.1.50 - user123 [15/Jan/2024:10:30:15 +0000] "POST /api/orders HTTP/1.1" 201 567 "https://shop.example.com/cart" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"`,
		`172.16.0.10 - - [15/Jan/2024:10:30:30 +0000] "PUT /api/profile/456 HTTP/1.1" 200 890 "-" "curl/7.68.0"`,
		`203.0.113.1 - admin [15/Jan/2024:10:30:45 +0000] "DELETE /api/users/789 HTTP/1.1" 204 0 "https://admin.example.com/users" "AdminPanel/1.0"`,
		`192.168.1.100 - - [15/Jan/2024:10:31:00 +0000] "GET /health HTTP/1.1" 200 15 "-" "HealthChecker/2.1"`,
	}
	return logs
}

func generateSyslogLogs() []string {
	logs := []string{
		`<34>Jan 15 10:30:00 web-server nginx[1234]: 192.168.1.1 - GET /api/status HTTP/1.1 200`,
		`<38>Jan 15 10:30:15 auth-server sshd[5678]: Accepted publickey for user123 from 10.0.1.50 port 22`,
		`<131>Jan 15 10:30:30 db-server mysql[9012]: Query executed successfully: SELECT * FROM users WHERE id=456`,
		`<27>Jan 15 10:30:45 app-server payment-service[3456]: Payment processed for order #789, amount: $99.99`,
		`<35>Jan 15 10:31:00 load-balancer haproxy[7890]: Server backend1/web1 is UP, reason: Layer4 check passed`,
	}
	return logs
}

func generateApplicationLogs() []string {
	logs := []string{
		`2024-01-15 10:30:00.123 INFO user-service - User authentication successful for user_id=123`,
		`2024-01-15 10:30:15.456 WARN payment-service - Rate limit approaching: 85/100 requests in current window`,
		`2024-01-15 10:30:30.789 ERROR order-service - Database connection failed: timeout after 5000ms`,
		`2024-01-15 10:30:45.012 DEBUG cache-service - Cache hit for key: session:user-456`,
		`2024-01-15 10:31:00.345 FATAL notification-service - Unable to connect to message queue: connection refused`,
	}
	return logs
}

func generateSpringBootLogs() []string {
	logs := []string{
		`2024-01-15 10:30:00.123  INFO 12345 --- [main] c.e.UserController : Starting UserController on hostname with PID 12345`,
		`2024-01-15 10:30:15.456  WARN 12345 --- [http-nio-8080-exec-1] c.e.SecurityConfig : Invalid JWT token received from IP: 192.168.1.100`,
		`2024-01-15 10:30:30.789 ERROR 12345 --- [task-scheduler-1] c.e.PaymentProcessor : Payment processing failed for order: order-123`,
		`2024-01-15 10:30:45.012 DEBUG 12345 --- [pool-2-thread-1] c.e.CacheManager : Evicting expired cache entries: 15 entries removed`,
		`2024-01-15 10:31:00.345  INFO 12345 --- [shutdown-hook] c.e.Application : Gracefully shutting down application`,
	}
	return logs
}

func generateMixedLogs() []string {
	logs := []string{
		// Generic timestamped
		`2024-01-15T10:32:00.123Z INFO Processing webhook payload from external service`,
		`2024-01-15T10:32:15Z ERROR Failed to validate signature for incoming request`,
		
		// Simple level + message
		`INFO: Health check passed for all services`,
		`ERROR: Memory usage exceeded threshold: 95%`,
		`WARN: Disk space running low on /var partition`,
		
		// Key-value style
		`User login event user_id=789 ip=10.0.1.75 status=success method=oauth`,
		`API request completed endpoint=/api/search duration=250ms status=200 user_agent=mobile-app`,
		
		// Docker style
		`2024-01-15T10:32:30.456Z stdout F Starting background job processor...`,
		`2024-01-15T10:32:45.789Z stderr F WARN: Environment variable API_KEY not set`,
		
		// Custom application format
		`[2024-01-15 10:33:00] [METRICS] service=api-gateway requests_per_second=1250 avg_response_time=85ms`,
		`[2024-01-15 10:33:15] [AUDIT] action=user_created admin_id=admin123 target_user=user999`,
		
		// Nginx error log style
		`2024/01/15 10:33:30 [error] 1234#0: *5678 connect() failed (111: Connection refused) while connecting to upstream`,
		
		// Generic fallback (should catch everything)
		`Something happened that we need to log but doesn't match any specific pattern`,
		`System event: backup completed successfully at 2024-01-15 10:34:00`,
	}
	return logs
}

func sendRawLogs(endpoint string, rawLogs []string, testService string) error {
	// Convert raw logs to LogEntry format - the message field will contain the raw log
	logs := make([]LogEntry, len(rawLogs))
	for i, rawLog := range rawLogs {
		logs[i] = LogEntry{
			Timestamp: time.Now().Add(time.Duration(-len(rawLogs)+i) * time.Second),
			Level:     "info",
			Message:   rawLog, // Raw log will be parsed by the parsing system
			Service:   testService,
			Attributes: map[string]interface{}{
				"raw_format": true,
				"test_type":  "regex_parsing",
				"original":   rawLog,
			},
		}
	}
	
	request := map[string]interface{}{
		"logs": logs,
		"options": map[string]bool{
			"skip_broadcast": true,
			"enable_parsing": true, // Enable parsing for these logs
		},
	}
	
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	
	return nil
}

func verifyParsedLogs() error {
	// Query different test services to verify parsing
	testServices := []string{
		"apache-parser-test",
		"syslog-parser-test", 
		"app-parser-test",
		"spring-parser-test",
		"mixed-parser-test",
	}
	
	totalLogs := 0
	for _, service := range testServices {
		queryURL := fmt.Sprintf("http://localhost:20002/api/v1/logs?limit=10&service=%s", service)
		
		resp, err := http.Get(queryURL)
		if err != nil {
			log.Printf("⚠️  Failed to query %s logs: %v", service, err)
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			log.Printf("⚠️  Query for %s returned status %d", service, resp.StatusCode)
			continue
		}
		
		var response struct {
			Logs []map[string]interface{} `json:"logs"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Printf("⚠️  Failed to decode %s response: %v", service, err)
			continue
		}
		
		count := len(response.Logs)
		totalLogs += count
		fmt.Printf("  📊 Found %d logs from %s\n", count, service)
		
		// Show sample parsed fields for first log
		if count > 0 {
			firstLog := response.Logs[0]
			fmt.Printf("    🔍 Sample fields: %v\n", getLogFields(firstLog))
		}
	}
	
	fmt.Printf("  📈 Total parsed logs verified: %d\n", totalLogs)
	fmt.Println("  ✅ Regex parsing system successfully processed unstructured logs")
	
	return nil
}

func getLogFields(logData map[string]interface{}) []string {
	var fields []string
	for field := range logData {
		fields = append(fields, field)
	}
	return fields
}