package internal

import (
	"log"
)

type Matchmaker struct {
	queue  chan *Client
	remove chan *Client
	mode   uint16
}

var matchmakers map[uint16]*Matchmaker

func InitAllMatchmakers(bufferSize int) {
	matchmakers = make(map[uint16]*Matchmaker)
	modes := GetAllModes()
	for _, mode := range modes {
		m := &Matchmaker{
			queue:  make(chan *Client, bufferSize),
			remove: make(chan *Client, bufferSize),
			mode:   mode,
		}
		matchmakers[mode] = m
		go m.matchmakingLoop()
	}
}

func (m *Matchmaker) matchmakingLoop() {
	waitingList := make([]*Client, 0)

	for {
		select {
		case newClient := <-m.queue:
			log.Printf("New client %d joined queue for mode %d", newClient.UserID, m.mode)
			waitingList = append(waitingList, newClient)

		case leaving := <-m.remove:
			log.Printf("Removing client %d from mode %d", leaving.UserID, m.mode)
			waitingList = removeClientFromList(waitingList, leaving)
		}

		// Always process the waiting list after any event (join or leave)
		if len(waitingList) > 0 {
			waitingList = m.processWaitingList(waitingList)
		}
	}
}

func (m *Matchmaker) processWaitingList(waitingList []*Client) []*Client {
	// First, drain any additional queued/remove operations (non-blocking)
	for {
		select {
		case newClient := <-m.queue:
			log.Printf("Draining: New client %d joined queue for mode %d", newClient.UserID, m.mode)
			waitingList = append(waitingList, newClient)
		case leaving := <-m.remove:
			log.Printf("Draining: Removing client %d from mode %d", leaving.UserID, m.mode)
			waitingList = removeClientFromList(waitingList, leaving)
		default:
			// No more pending operations
			goto ProcessMatches
		}
	}

ProcessMatches:
	if len(waitingList) == 0 {
		return waitingList
	}

	filtered := waitingList[:0] // reuse underlying array to reduce allocations

	for i := 0; i < len(waitingList); {
		p1 := waitingList[i]
		i++

		// Drop disconnected player
		if m.isClientDisconnected(p1) {
			log.Printf("Dropping disconnected client %d from mode %d", p1.UserID, m.mode)
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
		if m.isClientDisconnected(p2) {
			log.Printf("Dropping disconnected client %d from mode %d", p2.UserID, m.mode)
			filtered = append(filtered, p1)
			continue
		}

		if p1.UserID == p2.UserID {
			log.Printf("⚠️  Skipping match: same player queued twice %d in mode %d", p1.UserID, m.mode)
			continue
		}

		// ✅ Both connected: create match
		log.Printf("Matched client %d vs %d in mode %d", p1.UserID, p2.UserID, m.mode)
		m.startGame([]*Client{p1, p2})
	}

	return filtered
}

// Helper method to check if client is disconnected
func (m *Matchmaker) isClientDisconnected(client *Client) bool {
	if client == nil {
		return true
	}

	// Check if client is marked as disconnected
	if client.IsDisconnected() {
		return true
	}

	// Check connection count
	if client.ConnCount() == 0 {
		return true
	}

	// Check if client still exists in global client list
	if _, exists := GetClient(client.UserID); !exists {
		return true
	}

	return false
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

func removeClientFromList(list []*Client, target *Client) []*Client {
	if target == nil {
		return list
	}

	newList := list[:0]
	removed := false

	for _, p := range list {
		if p != nil && p != target && p.UserID != target.UserID {
			newList = append(newList, p)
		} else if p != nil {
			log.Printf("Removed player with id: %d", p.UserID)
			removed = true
		}
	}

	if !removed {
		log.Printf("Warning: Client %d was not found in waiting list for removal", target.UserID)
	}

	return newList
}

// Add method to get queue status for debugging
func (m *Matchmaker) GetQueueStatus() (int, int) {
	return len(m.queue), len(m.remove)
}

// Add method to force process waiting clients (useful for testing)
func (m *Matchmaker) ForceProcess() {
	select {
	case m.queue <- nil: // Send nil to trigger processing
	default:
		// Queue is full, ignore
	}
}

func (m *Matchmaker) startGame(players []*Client) {
	log.Printf("Starting game with players: %d and %d", players[0].UserID, players[1].UserID)
	GetGameKeeper().CreateGame(players, m.mode)
}
