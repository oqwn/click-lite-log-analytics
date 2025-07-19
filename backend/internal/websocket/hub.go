package websocket

import (
	"encoding/json"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Info().Str("client_id", client.id).Msg("Client connected")

			// Send welcome message
			welcome := models.WebSocketMessage{
				Type: "connection",
				Data: map[string]string{
					"status":  "connected",
					"message": "Connected to log stream",
				},
			}
			if msg, err := json.Marshal(welcome); err == nil {
				client.send <- msg
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.mu.Unlock()
				log.Info().Str("client_id", client.id).Msg("Client disconnected")
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close it
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastLog sends a log entry to all connected clients
func (h *Hub) BroadcastLog(log *models.Log) {
	message := models.WebSocketMessage{
		Type: "log",
		Data: log,
	}

	if msg, err := json.Marshal(message); err == nil {
		h.broadcast <- msg
	}
}

// BroadcastToClients sends a message to specific clients based on their filters
func (h *Hub) BroadcastToClients(logEntry *models.Log) {
	message := models.WebSocketMessage{
		Type: "log",
		Data: logEntry,
	}

	msgBytes, err := json.Marshal(message)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Check if log matches client's filters
		if client.MatchesFilters(logEntry) && !client.isPaused {
			select {
			case client.send <- msgBytes:
			default:
				// Client's send channel is full
				log.Warn().Str("client_id", client.id).Msg("Client send buffer full")
			}
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}