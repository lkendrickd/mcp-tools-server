package server

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Client represents a single SSE client connection.
type Client struct {
	id      string
	send    chan []byte // Channel to send messages to this client.
	logger  *slog.Logger
	isAlive bool
}

// SSEManager handles all active SSE client connections.
type SSEManager struct {
	clients map[string]*Client
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewSSEManager creates a new SSEManager.
func NewSSEManager(logger *slog.Logger) *SSEManager {
	return &SSEManager{
		clients: make(map[string]*Client),
		logger:  logger,
	}
}

// AddClient registers a new client and returns it.
func (m *SSEManager) AddClient() *Client {
	m.mu.Lock()
	defer m.mu.Unlock()

	clientID := uuid.NewString()
	client := &Client{
		id:      clientID,
		send:    make(chan []byte, 256), // Buffered channel
		logger:  m.logger.With("clientID", clientID),
		isAlive: true,
	}

	m.clients[client.id] = client
	m.logger.Info("SSE client added", "clientID", client.id)
	return client
}

// RemoveClient unregisters a client.
func (m *SSEManager) RemoveClient(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, ok := m.clients[id]; ok {
		client.isAlive = false
		close(client.send)
		delete(m.clients, id)
		m.logger.Info("SSE client removed", "clientID", id)
	}
}

// Send sends a message to a specific client.
// It returns an error if the client is not found or the send times out.
func (m *SSEManager) Send(clientID string, message []byte) error {
	m.mu.RLock()
	client, ok := m.clients[clientID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("client not found: %s", clientID)
	}

	if !client.isAlive {
		return fmt.Errorf("client channel closed: %s", clientID)
	}

	select {
	case client.send <- message:
		return nil
	case <-time.After(2 * time.Second): // 2-second timeout
		return fmt.Errorf("timeout sending message to client %s", clientID)
	}
}

// Broadcast sends a message to all connected clients.
func (m *SSEManager) Broadcast(message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, client := range m.clients {
		if client.isAlive {
			select {
			case client.send <- message:
				// Message sent
			default:
				// Channel is full, log it and move on.
				// This prevents a slow client from blocking all broadcasts.
				m.logger.Warn("Failed to broadcast to client, channel full", "clientID", id)
			}
		}
	}
}
