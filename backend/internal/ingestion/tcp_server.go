package ingestion

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/websocket"
)

// TCPServer handles TCP log ingestion
type TCPServer struct {
	addr           string
	batchProcessor *BatchProcessor
	wsHub          *websocket.Hub
	listener       net.Listener
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewTCPServer creates a new TCP ingestion server
func NewTCPServer(addr string, batchProcessor *BatchProcessor, wsHub *websocket.Hub) *TCPServer {
	return &TCPServer{
		addr:           addr,
		batchProcessor: batchProcessor,
		wsHub:          wsHub,
		stopChan:       make(chan struct{}),
	}
}

// Start starts the TCP server
func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	
	s.listener = listener
	log.Info().Str("addr", s.addr).Msg("TCP log ingestion server started")
	
	s.wg.Add(1)
	go s.acceptConnections()
	
	return nil
}

// acceptConnections accepts incoming TCP connections
func (s *TCPServer) acceptConnections() {
	defer s.wg.Done()
	
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return
			default:
				log.Error().Err(err).Msg("Failed to accept TCP connection")
				continue
			}
		}
		
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single TCP connection
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()
	
	clientAddr := conn.RemoteAddr().String()
	log.Info().Str("client", clientAddr).Msg("New TCP client connected")
	
	// Set read deadline for idle connections
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024*64), 1024*1024) // 64KB buffer, 1MB max
	
	for scanner.Scan() {
		// Reset read deadline on each message
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		
		select {
		case <-s.stopChan:
			return
		default:
		}
		
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		
		// Try to parse as JSON log
		var logEntry models.Log
		if err := json.Unmarshal(line, &logEntry); err != nil {
			// If not JSON, treat as plain text log
			logEntry = models.Log{
				Message: string(line),
				Level:   "info",
				Service: "tcp-client",
			}
		}
		
		// Process the log
		s.processLog(&logEntry)
		
		// Send acknowledgment
		conn.Write([]byte("OK\n"))
	}
	
	if err := scanner.Err(); err != nil {
		log.Error().Err(err).Str("client", clientAddr).Msg("Error reading from TCP client")
	}
	
	log.Info().Str("client", clientAddr).Msg("TCP client disconnected")
}

// processLog processes a single log entry
func (s *TCPServer) processLog(logEntry *models.Log) {
	// Set defaults
	if logEntry.ID == "" {
		logEntry.ID = uuid.New().String()
	}
	if logEntry.Timestamp.IsZero() {
		logEntry.Timestamp = time.Now()
	}
	if logEntry.Level == "" {
		logEntry.Level = "info"
	}
	if logEntry.Service == "" {
		logEntry.Service = "tcp-client"
	}
	
	// Add to batch processor
	s.batchProcessor.Add(*logEntry)
	
	// Broadcast to WebSocket clients
	s.wsHub.BroadcastLog(logEntry)
}

// Stop gracefully shuts down the TCP server
func (s *TCPServer) Stop() error {
	close(s.stopChan)
	
	if s.listener != nil {
		s.listener.Close()
	}
	
	s.wg.Wait()
	return nil
}