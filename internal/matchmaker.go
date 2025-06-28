package internal

import (
	"github.com/zefir/szaszki-go-backend/logger"
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
			logger.Log.Info().Uint32("clientId", newClient.UserID).Uint16("mode", m.mode).Msg("New client joined queue for mode")
			waitingList = append(waitingList, newClient)

		case leaving := <-m.remove:
			logger.Log.Info().Uint32("clientId", leaving.UserID).Uint16("mode", m.mode).Msg("Removing client from mode")
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
			logger.Log.Info().Uint32("clientId", newClient.UserID).Uint16("mode", m.mode).Msg("Draining: New client joined queue for mode")
			waitingList = append(waitingList, newClient)
		case leaving := <-m.remove:
			logger.Log.Info().Uint32("clientId", leaving.UserID).Uint16("mode", m.mode).Msg("Draining: Removing client from mode")
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
			logger.Log.Info().Uint32("clientId", p1.UserID).Uint16("mode", m.mode).Msg("Dropping disconnected client from mode")
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
			logger.Log.Info().Uint32("clientId", p2.UserID).Uint16("mode", m.mode).Msg("Dropping disconnected client from mode")
			filtered = append(filtered, p1)
			continue
		}

		if p1.UserID == p2.UserID {
			logger.Log.Warn().Uint32("clientId", p1.UserID).Uint16("mode", m.mode).Msg("Skipping match: same player queued twice")
			continue
		}

		// ✅ Both connected: create match
		logger.Log.Info().Uint32("p1_clientId", p1.UserID).Uint32("p2_clientId", p2.UserID).Uint16("mode", m.mode).Msg("Matched clients in mode")
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
		logger.Log.Warn().Uint32("clientId", client.UserID).Uint16("mode", m.mode).Msg("Matchmaker doesn't exist, client cant enqueue")
		return
	}
	m.Enqueue(client)
}

func (m *Matchmaker) Enqueue(client *Client) {
	select {
	case m.queue <- client:
		logger.Log.Info().Uint32("clientId", client.UserID).Uint16("mode", m.mode).Msg("Client joined matchmaking queue")
	default:
		logger.Log.Warn().Uint32("clientId", client.UserID).Uint16("mode", m.mode).Msg("Client cant join matchmaking queue")
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
			logger.Log.Warn().Uint32("clientId", target.UserID).Msg("Removed client form list")
			removed = true
		}
	}

	if !removed {
		logger.Log.Warn().Uint32("clientId", target.UserID).Msg("Client not found in list to remove")
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

	logger.Log.Info().
		Uint32("whitePlayerId", players[0].UserID).
		Uint32("blackPlayerId", players[1].UserID).
		Uint16("mode", m.mode).
		Msg("Starting game")
	GetGameKeeper().CreateGame(players, m.mode)
}
