package ingestion

import (
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/websocket"
)

// SyslogServer handles Syslog protocol log ingestion (RFC3164 and RFC5424)
type SyslogServer struct {
	addr           string
	batchProcessor *BatchProcessor
	wsHub          *websocket.Hub
	conn           net.PacketConn
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// Syslog severity levels
var severityLevels = map[int]string{
	0: "emergency",
	1: "alert",
	2: "critical",
	3: "error",
	4: "warning",
	5: "notice",
	6: "info",
	7: "debug",
}

// Syslog facility names
var facilityNames = map[int]string{
	0:  "kernel",
	1:  "user",
	2:  "mail",
	3:  "system",
	4:  "security",
	5:  "syslogd",
	6:  "line-printer",
	7:  "network-news",
	8:  "uucp",
	9:  "clock",
	10: "security2",
	11: "ftp",
	12: "ntp",
	13: "log-audit",
	14: "log-alert",
	15: "clock2",
	16: "local0",
	17: "local1",
	18: "local2",
	19: "local3",
	20: "local4",
	21: "local5",
	22: "local6",
	23: "local7",
}

// NewSyslogServer creates a new Syslog ingestion server
func NewSyslogServer(addr string, batchProcessor *BatchProcessor, wsHub *websocket.Hub) *SyslogServer {
	return &SyslogServer{
		addr:           addr,
		batchProcessor: batchProcessor,
		wsHub:          wsHub,
		stopChan:       make(chan struct{}),
	}
}

// Start starts the Syslog server
func (s *SyslogServer) Start() error {
	conn, err := net.ListenPacket("udp", s.addr)
	if err != nil {
		return err
	}
	
	s.conn = conn
	log.Info().Str("addr", s.addr).Msg("Syslog ingestion server started")
	
	s.wg.Add(1)
	go s.receiveMessages()
	
	return nil
}

// receiveMessages receives and processes syslog messages
func (s *SyslogServer) receiveMessages() {
	defer s.wg.Done()
	
	buffer := make([]byte, 65536) // 64KB buffer
	
	for {
		select {
		case <-s.stopChan:
			return
		default:
		}
		
		// Set read deadline
		s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		
		n, addr, err := s.conn.ReadFrom(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Error().Err(err).Msg("Error reading syslog message")
			continue
		}
		
		message := string(buffer[:n])
		s.processSyslogMessage(message, addr.String())
	}
}

// processSyslogMessage parses and processes a syslog message
func (s *SyslogServer) processSyslogMessage(message, source string) {
	logEntry := s.parseSyslogMessage(message)
	
	// Set source address as attribute
	if logEntry.Attributes == nil {
		logEntry.Attributes = make(map[string]interface{})
	}
	logEntry.Attributes["source_addr"] = source
	
	// Set defaults
	if logEntry.ID == "" {
		logEntry.ID = uuid.New().String()
	}
	if logEntry.Timestamp.IsZero() {
		logEntry.Timestamp = time.Now()
	}
	
	// Add to batch processor
	s.batchProcessor.Add(*logEntry)
	
	// Broadcast to WebSocket clients
	s.wsHub.BroadcastLog(logEntry)
}

// parseSyslogMessage parses RFC3164 or RFC5424 syslog messages
func (s *SyslogServer) parseSyslogMessage(message string) *models.Log {
	// Try RFC5424 format first
	if logEntry := s.parseRFC5424(message); logEntry != nil {
		return logEntry
	}
	
	// Fall back to RFC3164 format
	return s.parseRFC3164(message)
}

// parseRFC5424 parses RFC5424 formatted syslog messages
func (s *SyslogServer) parseRFC5424(message string) *models.Log {
	// RFC5424: <priority>version timestamp hostname app-name procid msgid [structured-data] msg
	re := regexp.MustCompile(`^<(\d+)>(\d+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\[.*?\]|-)\s*(.*)$`)
	matches := re.FindStringSubmatch(message)
	
	if len(matches) < 10 {
		return nil
	}
	
	priority, _ := strconv.Atoi(matches[1])
	severity := priority & 0x07
	facility := priority >> 3
	
	timestamp, _ := time.Parse(time.RFC3339, matches[3])
	
	logEntry := &models.Log{
		Timestamp: timestamp,
		Level:     severityLevels[severity],
		Service:   matches[5], // app-name
		Message:   matches[9],
		Attributes: map[string]interface{}{
			"facility":        facilityNames[facility],
			"hostname":        matches[4],
			"procid":          matches[6],
			"msgid":           matches[7],
			"structured_data": matches[8],
			"version":         matches[2],
			"format":          "RFC5424",
		},
	}
	
	return logEntry
}

// parseRFC3164 parses RFC3164 formatted syslog messages
func (s *SyslogServer) parseRFC3164(message string) *models.Log {
	// RFC3164: <priority>timestamp hostname tag[pid]: message
	re := regexp.MustCompile(`^<(\d+)>(\w+\s+\d+\s+\d+:\d+:\d+)\s+(\S+)\s+(\S+?)(\[(\d+)\])?:\s*(.*)$`)
	matches := re.FindStringSubmatch(message)
	
	if len(matches) < 8 {
		// Fallback for simple format
		return &models.Log{
			Timestamp: time.Now(),
			Level:     "info",
			Service:   "syslog",
			Message:   message,
			Attributes: map[string]interface{}{
				"format": "unknown",
			},
		}
	}
	
	priority, _ := strconv.Atoi(matches[1])
	severity := priority & 0x07
	facility := priority >> 3
	
	// Parse timestamp (current year assumed)
	timestamp, _ := time.Parse("Jan _2 15:04:05", matches[2])
	timestamp = timestamp.AddDate(time.Now().Year(), 0, 0)
	
	logEntry := &models.Log{
		Timestamp: timestamp,
		Level:     severityLevels[severity],
		Service:   matches[4], // tag
		Message:   matches[7],
		Attributes: map[string]interface{}{
			"facility": facilityNames[facility],
			"hostname": matches[3],
			"format":   "RFC3164",
		},
	}
	
	// Add PID if present
	if matches[6] != "" {
		logEntry.Attributes["pid"] = matches[6]
	}
	
	// Extract additional fields from message if JSON
	if strings.HasPrefix(matches[7], "{") && strings.HasSuffix(matches[7], "}") {
		// Message might contain JSON data
		logEntry.Attributes["raw_message"] = matches[7]
	}
	
	return logEntry
}

// Stop gracefully shuts down the Syslog server
func (s *SyslogServer) Stop() error {
	close(s.stopChan)
	
	if s.conn != nil {
		s.conn.Close()
	}
	
	s.wg.Wait()
	return nil
}