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
		log.Println("got client", userID)
		return client
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
	return client, ok
}

func RemoveClient(userID uint32) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	delete(clients, userID)
	log.Println("client removed", userID)
}

func GetAllClients() map[uint32]*Client {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	// make a copy to avoid race conditions
	copied := make(map[uint32]*Client)
	for k, v := range clients {
		copied[k] = v
	}
	return copied
}

func (c *Client) RemoveConn(connID uint64) int {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.Conns != nil {
		log.Println("conn removed", connID)
		delete(c.Conns, connID)

		log.Println("remaining conns for client", c.UserID, ":")
		for id := range c.Conns {
			log.Println(" - connID:", id)
		}

		if len(c.Conns) <= 0 {
			RemoveClient(c.UserID)
		}
	}
	return len(c.Conns)
}

func (c *Client) ConnCount() int {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return len(c.Conns)
}
