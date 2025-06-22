package internal

import (
	"log"
	"sync"
)

type GameKeeper struct {
	games  map[int]*GameSession
	nextID int
	mu     sync.Mutex
}

var keeper *GameKeeper

func InitGameKeeper() {
	keeper = &GameKeeper{
		games:  make(map[int]*GameSession),
		nextID: 1,
	}
}

func GetGameKeeper() *GameKeeper {
	return keeper
}

func (g *GameKeeper) CreateGame(players []*Client, mode uint16) *GameSession {
	g.mu.Lock()
	defer g.mu.Unlock()

	gamesession := &GameSession{
		ID:      g.nextID,
		Players: players,
		Mode:    mode,
	}

	g.games[g.nextID] = gamesession
	g.nextID++

	log.Printf("Game %d created with players: %v", gamesession.ID, players)
	// Start game loop in separate goroutine
	go gamesession.Run()

	return gamesession
}

func (g *GameKeeper) GetGame(id int) (*GameSession, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	game, exists := g.games[id]
	return game, exists
}

func (g *GameKeeper) ListGames() []*GameSession {
	g.mu.Lock()
	defer g.mu.Unlock()

	sessions := make([]*GameSession, 0, len(g.games))
	for _, game := range g.games {
		sessions = append(sessions, game)
	}
	return sessions
}
