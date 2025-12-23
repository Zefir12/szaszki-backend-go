package internal

import (
	"net"
	"sync"
	"time"

	"github.com/zefir/szaszki-go-backend/logger"
)

type Client struct {
	Conns            map[uint64]net.Conn
	UserID           uint32
	CurrentlyPlaying bool
	QueuedInModes    map[uint16]bool
	lastSeen         time.Time
	Mu               sync.Mutex
	disconnected     bool // Track if client is already being disconnected

	initialMessages [][]byte // ← message queue for initial messages
}

var (
	clients   = make(map[uint32]*Client)
	clientsMu sync.RWMutex
)

func (c *Client) AddConn(id uint64, conn net.Conn) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.Conns == nil {
		c.Conns = make(map[uint64]net.Conn)
	}
	c.Conns[id] = conn
	logger.Log.Info().Uint32("clientId", c.UserID).Uint64("connId", id).Msg("Added conn to client")
}

func AddClient(userID uint32, connID uint64, conn net.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if client, ok := clients[userID]; ok {
		// Client exists, add new connection
		client.AddConn(connID, conn)
		logger.Log.Info().Uint32("clientId", client.UserID).Uint64("connId", connID).Msg("Added new connection to client")
	} else {
		// Create new client and add connection
		c := &Client{
			UserID:        userID,
			Conns:         make(map[uint64]net.Conn),
			QueuedInModes: make(map[uint16]bool),
		}
		logger.Log.Info().Uint32("clientId", client.UserID).Uint64("connId", connID).Msg("Making new client for connection and adding connection")
		c.Conns[connID] = conn
		clients[userID] = c
	}
}

func (c *Client) AddQueuedMode(mode uint16) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.disconnected {
		return // Don't add modes to disconnected clients
	}
	c.QueuedInModes[mode] = true
}

func (c *Client) RemoveQueuedMode(mode uint16) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	delete(c.QueuedInModes, mode)
}

func (c *Client) IsQueuedInMode(mode uint16) bool {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.QueuedInModes[mode]
}

func GetClientOrCreate(userID uint32) *Client {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if client, ok := clients[userID]; ok {
		client.Mu.Lock()
		client.disconnected = false // ← reclaim old client!
		client.lastSeen = time.Now()
		client.Mu.Unlock()
		return client
	}

	// Create new Client
	client := &Client{
		UserID:        userID,
		Conns:         make(map[uint64]net.Conn),
		QueuedInModes: make(map[uint16]bool),
	}
	logger.Log.Info().Uint32("clientId", client.UserID).Msg("Client created")
	clients[userID] = client
	return client
}

func GetClient(userID uint32) (*Client, bool) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	client, ok := clients[userID]
	if !ok {
		return nil, false
	}

	// Check if client is disconnected
	client.Mu.Lock()
	disconnected := client.disconnected
	client.Mu.Unlock()

	if disconnected {
		return nil, false
	}

	return client, true
}

// func RemoveClient(userID uint32) {
// 	clientsMu.Lock()
// 	defer clientsMu.Unlock()

// 	if client, ok := clients[userID]; ok {
// 		// Mark as disconnected first
// 		client.Mu.Lock()
// 		client.disconnected = true

// 		// Close all connections
// 		for connID, conn := range client.Conns {
// 			logger.Log.Info().Uint32("clientId", client.UserID).Uint64("connId", connID).Msg("Closing connection for client")
// 			conn.Close()
// 		}
// 		client.Conns = make(map[uint64]net.Conn) // Clear the map
// 		client.Mu.Unlock()

// 		delete(clients, userID)
// 		logger.Log.Info().Uint32("clientId", client.UserID).Msg("Client removed")
// 	}
// }

func GetAllClients() map[uint32]*Client {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	// make a copy to avoid race conditions
	copied := make(map[uint32]*Client)
	for k, v := range clients {
		// Only include non-disconnected clients
		v.Mu.Lock()
		if !v.disconnected {
			copied[k] = v
		}
		v.Mu.Unlock()
	}
	return copied
}

func (c *Client) RemoveConn(connID uint64) int {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	if c.Conns != nil {
		delete(c.Conns, connID)
	}

	remaining := len(c.Conns)
	logger.Log.Info().Uint32("clientId", c.UserID).Int("remainingConns", remaining).Msg("Connection removed")

	if remaining == 0 {
		// mark soft-disconnected, but DO NOT remove client
		c.disconnected = false // Allow reclaim
		// optional: store timestamp for idle timeout
	}

	return remaining
}

func (c *Client) ConnCount() int {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.disconnected {
		return 0
	}
	return len(c.Conns)
}

// Helper method to check if client is disconnected
func (c *Client) IsDisconnected() bool {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.disconnected
}
