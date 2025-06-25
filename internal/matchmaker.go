package internal

import (
	"encoding/binary"
	"log"
	"sync"
	"time"
)

type Matchmaker struct {
	queue            chan *Client
	mode             uint16
	acceptChan       chan playerResponse
	removeClientChan chan *Client
	pending          map[uint32]*pendingMatch
	matchLock        sync.Mutex
}

type pendingMatch struct {
	players  []*Client
	timer    *time.Timer
	accepted map[*Client]bool
	mode     uint16
}

type playerResponse struct {
	matchID uint32
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
			mode:       mode,
			acceptChan: make(chan playerResponse),
			pending:    make(map[uint32]*pendingMatch),
		}
		matchmakers[mode] = m
		go m.loop()
		go m.matchmakingLoop()
	}
}

func (m *Matchmaker) matchmakingLoop() {
	matchIDCounter := uint32(0)
	waitingList := make([]*Client, 0)

	for {
		next := <-m.queue // blocks until someone is queued
		waitingList = append(waitingList, next)

	Drain: // Drain any other waiting players (non-blocking)
		for i := 1; i < 99; i++ {
			select {
			case p := <-m.queue:
				waitingList = append(waitingList, p)
			default:
				break Drain
			}
		}

		filtered := waitingList[:0] // reuse underlying array to reduce allocations

		for i := 0; i < len(waitingList); {
			p1 := waitingList[i]
			i++

			// Drop disconnected player
			if len(p1.Conns) == 0 {
				continue
			}

			// Not enough players left for a match
			if i >= len(waitingList) {
				// Keep the leftover connected player
				filtered = append(filtered, p1)
				break
			}

			p2 := waitingList[i]
			i++

			// Drop disconnected second player and push first back into filtered list
			if len(p2.Conns) == 0 {
				filtered = append(filtered, p1)
				continue
			}

			// ✅ Both connected: create match
			log.Println("Matched", p1, "vs", p2)

			// Remove from their queues
			p1.Mu.Lock()
			p1.RemoveQueuedMode(m.mode)
			p1.Mu.Unlock()

			p2.Mu.Lock()
			p2.RemoveQueuedMode(m.mode)
			p2.Mu.Unlock()

			matchIDCounter++
			matchID := matchIDCounter

			pm := &pendingMatch{
				players:  []*Client{p1, p2},
				accepted: make(map[*Client]bool),
				mode:     m.mode,
			}

			m.matchLock.Lock()
			m.pending[matchID] = pm
			m.matchLock.Unlock()

			pm.timer = time.AfterFunc(10*time.Second, func() {
				m.handleTimeout(matchID)
			})

			matchIdBytes := make([]byte, 6)
			binary.BigEndian.PutUint32(matchIdBytes[0:4], uint32(matchID))
			binary.BigEndian.PutUint16(matchIdBytes[4:6], uint16(pm.mode))

			for _, player := range pm.players {
				player.WriteMsg(ServerCmds.GameFound, matchIdBytes)
			}
		}

		waitingList = filtered
		// Sleep before the next round
		time.Sleep(50 * time.Millisecond)
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
		log.Printf("Client %d entered matchmaking queue: %d", client.UserID, m.mode)
	default:
		log.Printf("Matchmaking queue full for client %d", client.UserID)
	}
}

func (m *Matchmaker) loop() {
	for {
		select {
		case resp := <-m.acceptChan:
			m.handlePlayerResponse(resp.matchID, resp.client, resp.accept)

		case clientToRemove := <-m.removeClientChan:
			log.Println("Removing client from waiting:", clientToRemove)
		}
	}
}

func removeClientFromWaiting(waiting []*Client, client *Client) []*Client {
	for i, c := range waiting {
		if c == client {
			return append(waiting[:i], waiting[i+1:]...)
		}
	}
	return waiting
}

func AcceptMatch(client *Client, matchID uint32, accepted bool, mode uint16) {
	m, ok := matchmakers[mode]
	if !ok {
		log.Printf("No matchmaker found for mode %d", mode)
		return
	}

	m.acceptChan <- playerResponse{
		matchID: matchID,
		client:  client,
		accept:  accepted,
	}
}

func (m *Matchmaker) handlePlayerResponse(matchID uint32, client *Client, accept bool) {
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

func (m *Matchmaker) handleTimeout(matchID uint32) {
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
	}
}

func (m *Matchmaker) startGame(players []*Client) {
	log.Printf("Starting game with players: %d and %d", players[0].UserID, players[1].UserID)
	GetGameKeeper().CreateGame(players, m.mode)
}
