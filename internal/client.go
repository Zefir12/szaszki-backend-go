package internal

import (
	"net"
	"sync"
)

type ClientConn struct {
	Conns            map[int64]net.Conn
	UserID           int32
	CurrentlyPlaying bool
	CurrentlyInQueue bool
	Mu               sync.Mutex
}

var (
	clients   = make(map[int32]*ClientConn)
	clientsMu sync.RWMutex
)

func (c *ClientConn) AddConn(id int64, conn net.Conn) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if c.Conns == nil {
		c.Conns = make(map[int64]net.Conn)
	}
	c.Conns[id] = conn
}

func AddClient(userID int32, connID int64, conn net.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if client, ok := clients[userID]; ok {
		// Client exists, add new connection
		client.AddConn(connID, conn)
	} else {
		// Create new client and add connection
		c := &ClientConn{
			UserID: userID,
			Conns:  make(map[int64]net.Conn),
		}
		c.Conns[connID] = conn
		clients[userID] = c
	}
}

func GetClient(userID int32) (*ClientConn, bool) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	client, ok := clients[userID]
	return client, ok
}

func RemoveClient(userID int32) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	delete(clients, userID)
}

func GetAllClients() map[int32]*ClientConn {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	// make a copy to avoid race conditions
	copied := make(map[int32]*ClientConn)
	for k, v := range clients {
		copied[k] = v
	}
	return copied
}
