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

func playerIDs(players []*ClientConn) []int32 {
	ids := make([]int32, len(players))
	for i, p := range players {
		ids[i] = p.UserID
	}
	return ids
}

func (g *GameKeeper) CreateGame(players []*ClientConn) *GameSession {
	g.mu.Lock()
	defer g.mu.Unlock()

	game := &GameSession{
		ID:      g.nextID,
		Players: players,
	}

	g.games[g.nextID] = game
	g.nextID++

	log.Printf("Game %d created with players: %v", game.ID, playerIDs(players))

	// Mark players as in game and notify them
	for _, player := range players {
		player.CurrentlyPlaying = true
		WriteMsg(player.Conn, ServerCmds.GameFound, nil)

	}

	// Start game loop in separate goroutine
	go game.Run()

	return game
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
