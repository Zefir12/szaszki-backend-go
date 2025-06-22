package internal

import (
	"log"
	"sync"
	"time"
)

type Matchmaker struct {
	queue      chan *ClientConn
	quit       chan struct{}
	mode       uint16
	acceptChan chan playerResponse
	pending    map[int]*pendingMatch
	matchLock  sync.Mutex
}

type pendingMatch struct {
	players  []*ClientConn
	timer    *time.Timer
	accepted map[*ClientConn]bool
	mode     uint16
}

type playerResponse struct {
	matchID int
	client  *ClientConn
	accept  bool
}

var matchmakers map[uint16]*Matchmaker

func InitAllMatchmakers(bufferSize int) {
	matchmakers = make(map[uint16]*Matchmaker)
	modes := GetAllModes()
	for _, mode := range modes {
		m := &Matchmaker{
			queue:      make(chan *ClientConn, bufferSize),
			quit:       make(chan struct{}),
			mode:       mode,
			acceptChan: make(chan playerResponse),
			pending:    make(map[int]*pendingMatch),
		}
		matchmakers[mode] = m
		go m.loop()
	}
}

func EnqueuePlayerForMode(client *ClientConn, mode uint16) {
	m, ok := matchmakers[mode]
	if !ok {
		log.Printf("No matchmaker for mode %d", mode)
		return
	}
	m.Enqueue(client)
}

func (m *Matchmaker) Enqueue(client *ClientConn) {
	select {
	case m.queue <- client:
		log.Printf("Client %d entered matchmaking queue", client.UserID)
	default:
		log.Printf("Matchmaking queue full for client %d", client.UserID)
	}
}

func (m *Matchmaker) loop() {
	waiting := make([]*ClientConn, 0)
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
					players:  []*ClientConn{p1, p2},
					accepted: make(map[*ClientConn]bool),
					mode:     m.mode,
				}
				pm.timer = time.AfterFunc(15*time.Second, func() {
					m.handleTimeout(matchID)
				})

				m.matchLock.Lock()
				m.pending[matchID] = pm
				m.matchLock.Unlock()

				for _, player := range pm.players {
					WriteMsg(player.Conns, ServerCmds.GameFound, nil)
				}
			}

		case resp := <-m.acceptChan:
			m.handlePlayerResponse(resp.matchID, resp.client, resp.accept)

		case <-m.quit:
			return
		}
	}
}

func (m *Matchmaker) handlePlayerResponse(matchID int, client *ClientConn, accept bool) {
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
				WriteMsg(p.Conns, ServerCmds.GameDeclined, nil)
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
		WriteMsg(p.Conns, ServerCmds.GameSearchTimeout, nil)
		// Optionally requeue players to try matching again
		m.Enqueue(p)
	}
}

func (m *Matchmaker) Stop() {
	close(m.quit)
}

func (m *Matchmaker) startGame(players []*ClientConn) {
	log.Printf("Starting game with players: %d and %d", players[0].UserID, players[1].UserID)
	GetGameKeeper().CreateGame(players, m.mode)
}
