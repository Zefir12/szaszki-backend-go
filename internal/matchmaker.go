package internal

import (
	"log"
	"sync"
)

type Matchmaker struct {
	queue []*ClientConn
	mu    sync.Mutex
}

var instance *Matchmaker

func InitMatchmaker() {
	instance = &Matchmaker{
		queue: make([]*ClientConn, 0),
	}
}

func GetMatchmaker() *Matchmaker {
	return instance
}

func (m *Matchmaker) Enqueue(client *ClientConn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Prevent duplicates
	for _, c := range m.queue {
		if c == client {
			log.Printf("Client %d is already in the matchmaking queue", client.UserID)
			return
		}
	}

	m.queue = append(m.queue, client)
	log.Printf("Client %d entered matchmaking queue", client.UserID)

	if len(m.queue) >= 2 {
		p1 := m.queue[0]
		p2 := m.queue[1]
		m.queue = m.queue[2:]

		go m.startGame([]*ClientConn{p1, p2})
	}
}

func (m *Matchmaker) startGame(players []*ClientConn) {
	log.Printf("Starting game with players: %d and %d", players[0].UserID, players[1].UserID)
	GetGameKeeper().CreateGame([]*ClientConn{players[0], players[1]})
}
