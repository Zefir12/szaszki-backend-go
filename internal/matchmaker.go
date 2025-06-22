package internal

import (
	"encoding/binary"
	"log"
	"sync"
	"time"
)

type Matchmaker struct {
	queue            chan *Client
	quit             chan struct{}
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
			quit:       make(chan struct{}),
			mode:       mode,
			acceptChan: make(chan playerResponse),
			pending:    make(map[uint32]*pendingMatch),
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
		log.Printf("Client %d entered matchmaking queue: %d", client.UserID, m.mode)
	default:
		log.Printf("Matchmaking queue full for client %d", client.UserID)
	}
}

func (m *Matchmaker) loop() {
	waiting := make([]*Client, 0)
	matchIDCounter := uint32(0)

	for {
		select {
		case client := <-m.queue:
			if client.IsQueuedInMode(m.mode) {
				log.Println("user already searching cant eneter another queue", client.UserID)
				continue
			}
			waiting = append(waiting, client)
			client.AddQueuedMode(m.mode)
			if len(waiting) >= 2 {
				p1 := waiting[0]
				p2 := waiting[1]
				waiting = waiting[2:]
				p1.RemoveQueuedMode(m.mode)
				p2.RemoveQueuedMode(m.mode)

				matchIDCounter++
				matchID := matchIDCounter

				pm := &pendingMatch{
					players:  []*Client{p1, p2},
					accepted: make(map[*Client]bool),
					mode:     m.mode,
				}
				pm.timer = time.AfterFunc(10*time.Second, func() {
					m.handleTimeout(matchID)
				})

				m.matchLock.Lock()
				m.pending[matchID] = pm
				m.matchLock.Unlock()

				matchIdBytes := make([]byte, 6) // 4 bytes for uint32 + 2 bytes for uint16

				binary.BigEndian.PutUint32(matchIdBytes[0:4], uint32(matchID))
				binary.BigEndian.PutUint16(matchIdBytes[4:6], uint16(pm.mode))
				log.Printf("Sending matchID bytes: %v", matchIdBytes)

				for _, player := range pm.players {
					player.WriteMsg(ServerCmds.GameFound, matchIdBytes)
				}
			}

		case resp := <-m.acceptChan:
			m.handlePlayerResponse(resp.matchID, resp.client, resp.accept)

		case clientToRemove := <-m.removeClientChan:
			log.Println("Removing client from waiting:", clientToRemove)
			waiting = removeClientFromWaiting(waiting, clientToRemove)
			clientToRemove.RemoveQueuedMode(m.mode)
			// for _, c := range waiting {
			// 	if c == clientToRemove {
			// 		waiting = removeClientFromWaiting(waiting, clientToRemove)
			// 		clientToRemove.CurrentlyInQueue = false
			// 		break // stop looping after we find it
			// 	}
			//}

		case <-m.quit:
			return
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
				if pm.accepted[p] {
					m.Enqueue(p)
				}
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
		if pm.accepted[p] {
			m.Enqueue(p)
		}
	}
}

func (m *Matchmaker) Stop() {
	close(m.quit)
}

func (m *Matchmaker) startGame(players []*Client) {
	log.Printf("Starting game with players: %d and %d", players[0].UserID, players[1].UserID)
	GetGameKeeper().CreateGame(players, m.mode)
}
