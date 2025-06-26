package internal

import (
	"log"
	"net"
	"sync"
)

type Client struct {
	Conns            map[uint64]net.Conn
	UserID           uint32
	CurrentlyPlaying bool
	QueuedInModes    map[uint16]bool
	Mu               sync.Mutex
	disconnected     bool // Track if client is already being disconnected
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
	log.Println("conn added", id)
}

func AddClient(userID uint32, connID uint64, conn net.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if client, ok := clients[userID]; ok {
		// Client exists, add new connection
		client.AddConn(connID, conn)
	} else {
		// Create new client and add connection
		c := &Client{
			UserID:        userID,
			Conns:         make(map[uint64]net.Conn),
			QueuedInModes: make(map[uint16]bool),
		}
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
		// Check if client is disconnected, if so create a new one
		client.Mu.Lock()
		disconnected := client.disconnected
		client.Mu.Unlock()

		if disconnected {
			log.Println("client was disconnected, creating new one", userID)
			// Remove the old disconnected client
			delete(clients, userID)
		} else {
			log.Println("got client", userID)
			return client
		}
	}

	// Create new Client
	client := &Client{
		UserID:        userID,
		Conns:         make(map[uint64]net.Conn),
		QueuedInModes: make(map[uint16]bool),
	}
	log.Println("created client", userID)
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

func RemoveClient(userID uint32) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if client, ok := clients[userID]; ok {
		// Mark as disconnected first
		client.Mu.Lock()
		client.disconnected = true

		// Close all connections
		for connID, conn := range client.Conns {
			log.Println("closing connection", connID, "for user", userID)
			conn.Close()
		}
		client.Conns = make(map[uint64]net.Conn) // Clear the map
		client.Mu.Unlock()

		delete(clients, userID)
		log.Println("client removed", userID)
	}
}

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

	if c.disconnected {
		log.Println("client already disconnected, ignoring conn removal", connID)
		return 0
	}

	if c.Conns != nil {
		if _, exists := c.Conns[connID]; exists {
			log.Println("conn removed", connID)
			delete(c.Conns, connID)

			log.Println("remaining conns for client", c.UserID, ":")
			for id := range c.Conns {
				log.Println(" - connID:", id)
			}

			if len(c.Conns) <= 0 {
				c.disconnected = true // Mark as disconnected before handling
				c.Mu.Unlock()         // Unlock before calling handleDisconnect to avoid deadlock
				c.handleDisconnect()
				RemoveClient(c.UserID)
				c.Mu.Lock() // Re-lock for defer unlock
			}
		} else {
			log.Println("connection", connID, "not found for removal")
		}
	}
	return len(c.Conns)
}

func (c *Client) handleDisconnect() {
	log.Println("handling disconnect for client", c.UserID)

	// Remove from all matchmakers - make this more robust
	var wg sync.WaitGroup
	for i, m := range matchmakers {
		wg.Add(1)
		go func(matchmaker *Matchmaker, index uint16) {
			defer wg.Done()
			select {
			case matchmaker.remove <- c:
				log.Printf("Successfully sent client %d to matchmaker %d remove queue", c.UserID, index)
			default:
				log.Printf("Warning: could not send client %d to matchmaker %d remove queue (channel full)", c.UserID, index)
			}
		}(m, i)
	}

	// Optional: wait for all matchmaker removals to complete
	// wg.Wait()
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
