package internal

import (
	"log"
	"sync"
	"time"
)

type Matchmaker struct {
	queue      chan *Client
	quit       chan struct{}
	mode       uint16
	acceptChan chan playerResponse
	pending    map[int]*pendingMatch
	matchLock  sync.Mutex
}

type pendingMatch struct {
	players  []*Client
	timer    *time.Timer
	accepted map[*Client]bool
	mode     uint16
}

type playerResponse struct {
	matchID int
	client  *Client
	accept  bool
}

var matchmakers map[uint16]*Matchmaker

func InitAllMatchmakers(bufferSize int) {
	matchmakers = make(map[uint16]*Matchmaker)
	modes := GetAllModes()
	for _, mode := range modes {
		m := &Matchmaker{
			queue:      make(chan *Client, bufferSize),
			quit:       make(chan struct{}),
			mode:       mode,
			acceptChan: make(chan playerResponse),
			pending:    make(map[int]*pendingMatch),
		}
		matchmakers[mode] = m
		go m.loop()
	}
}

func EnqueuePlayerForMode(client *Client, mode uint16) {
	m, ok := matchmakers[mode]
	if !ok {
		log.Printf("No matchmaker for mode %d", mode)
		return
	}
	m.Enqueue(client)
}

func (m *Matchmaker) Enqueue(client *Client) {
	select {
	case m.queue <- client:
		log.Printf("Client %d entered matchmaking queue", client.UserID)
	default:
		log.Printf("Matchmaking queue full for client %d", client.UserID)
	}
}

func (m *Matchmaker) loop() {
	waiting := make([]*Client, 0)
	matchIDCounter := 0

	for {
		select {
		case client := <-m.queue:
			waiting = append(waiting, client)

			if len(waiting) >= 2 {
				p1 := waiting[0]
				p2 := waiting[1]
				waiting = waiting[2:]

				matchIDCounter++
				matchID := matchIDCounter

				pm := &pendingMatch{
					players:  []*Client{p1, p2},
					accepted: make(map[*Client]bool),
					mode:     m.mode,
				}
				pm.timer = time.AfterFunc(15*time.Second, func() {
					m.handleTimeout(matchID)
				})

				m.matchLock.Lock()
				m.pending[matchID] = pm
				m.matchLock.Unlock()

				for _, player := range pm.players {
					player.WriteMsg(ServerCmds.GameFound, nil)
				}
			}

		case resp := <-m.acceptChan:
			m.handlePlayerResponse(resp.matchID, resp.client, resp.accept)

		case <-m.quit:
			return
		}
	}
}

func (m *Matchmaker) handlePlayerResponse(matchID int, client *Client, accept bool) {
	m.matchLock.Lock()
	defer m.matchLock.Unlock()

	pm, exists := m.pending[matchID]
	if !exists {
		log.Printf("No pending match found with ID %d", matchID)
		return
	}

	if !accept {
		// Declined: cancel match, notify players, requeue accepted players if desired
		pm.timer.Stop()
		delete(m.pending, matchID)
		for _, p := range pm.players {
			if p != client {
				p.WriteMsg(ServerCmds.GameDeclined, nil)
				m.Enqueue(p)
			}
		}
		return
	}

	// Mark accepted
	pm.accepted[client] = true

	// Check if all players accepted
	if len(pm.accepted) == len(pm.players) {
		pm.timer.Stop()
		delete(m.pending, matchID)

		go m.startGame(pm.players)
	}
}

func (m *Matchmaker) handleTimeout(matchID int) {
	m.matchLock.Lock()
	defer m.matchLock.Unlock()

	pm, exists := m.pending[matchID]
	if !exists {
		return
	}

	delete(m.pending, matchID)

	for _, p := range pm.players {
		p.WriteMsg(ServerCmds.GameSearchTimeout, nil)
		// Optionally requeue players to try matching again
		m.Enqueue(p)
	}
}

func (m *Matchmaker) Stop() {
	close(m.quit)
}

func (m *Matchmaker) startGame(players []*Client) {
	log.Printf("Starting game with players: %d and %d", players[0].UserID, players[1].UserID)
	GetGameKeeper().CreateGame(players, m.mode)
}
