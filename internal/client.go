package internal

import (
	"net"
	"sync"
)

type ClientConn struct {
	Conn             net.Conn
	UserID           int32
	CurrentlyPlaying bool
	Mu               sync.Mutex
}

var (
	clients   = make(map[int32]*ClientConn)
	clientsMu sync.RWMutex
)

func AddClient(userID int32, conn net.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	clients[userID] = &ClientConn{Conn: conn, UserID: userID}
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
