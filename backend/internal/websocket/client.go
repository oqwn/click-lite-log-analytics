package websocket

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// TODO: Implement proper origin checking for production
		return true
	},
}

type Client struct {
	id       string
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	filters  []models.LogFilter
	isPaused bool
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upgrade connection")
			return
		}

		client := &Client{
			id:       uuid.New().String(),
			hub:      hub,
			conn:     conn,
			send:     make(chan []byte, 256),
			filters:  []models.LogFilter{},
			isPaused: false,
		}

		client.hub.register <- client

		// Start goroutines for reading and writing
		go client.writePump()
		go client.readPump()
	}
}

// readPump handles incoming messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("client_id", c.id).Msg("WebSocket error")
			}
			break
		}

		// Parse incoming message
		var msg models.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Error().Err(err).Msg("Failed to parse WebSocket message")
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "filter":
			c.handleFilterMessage(msg)
		case "pause":
			c.isPaused = true
			c.sendStatus("paused", "Stream paused")
		case "resume":
			c.isPaused = false
			c.sendStatus("resumed", "Stream resumed")
		case "ping":
			c.sendStatus("pong", "")
		default:
			log.Warn().Str("type", msg.Type).Msg("Unknown message type")
		}
	}
}

// writePump handles outgoing messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleFilterMessage updates the client's filters
func (c *Client) handleFilterMessage(msg models.WebSocketMessage) {
	if msg.Filters != nil {
		c.filters = msg.Filters
		c.sendStatus("filters_updated", "Filters updated successfully")
		log.Debug().Str("client_id", c.id).Interface("filters", c.filters).Msg("Client filters updated")
	}
}

// MatchesFilters checks if a log entry matches the client's filters
func (c *Client) MatchesFilters(log *models.Log) bool {
	// If no filters, all logs match
	if len(c.filters) == 0 {
		return true
	}

	// Check each filter
	for _, filter := range c.filters {
		if !c.matchFilter(log, filter) {
			return false
		}
	}

	return true
}

// matchFilter checks if a log matches a single filter
func (c *Client) matchFilter(log *models.Log, filter models.LogFilter) bool {
	var fieldValue string

	// Get field value
	switch filter.Field {
	case "level":
		fieldValue = log.Level
	case "service":
		fieldValue = log.Service
	case "message":
		fieldValue = log.Message
	case "trace_id":
		fieldValue = log.TraceID
	default:
		// Check attributes
		if val, ok := log.Attributes[filter.Field]; ok {
			fieldValue = strings.ToLower(val.(string))
		}
	}

	fieldValue = strings.ToLower(fieldValue)
	filterValue := strings.ToLower(filter.Value)

	// Apply operator
	switch filter.Operator {
	case "equals", "=":
		return fieldValue == filterValue
	case "contains":
		return strings.Contains(fieldValue, filterValue)
	case "starts_with":
		return strings.HasPrefix(fieldValue, filterValue)
	case "ends_with":
		return strings.HasSuffix(fieldValue, filterValue)
	case "not_equals", "!=":
		return fieldValue != filterValue
	case "not_contains":
		return !strings.Contains(fieldValue, filterValue)
	default:
		return false
	}
}

// sendStatus sends a status message to the client
func (c *Client) sendStatus(status, message string) {
	msg := models.WebSocketMessage{
		Type: "status",
		Data: map[string]string{
			"status":  status,
			"message": message,
		},
	}

	if msgBytes, err := json.Marshal(msg); err == nil {
		select {
		case c.send <- msgBytes:
		default:
			// Send buffer full
		}
	}
}