package internal

import (
	"sync"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
	"github.com/zefir/szaszki-go-backend/logger"
)

type GameKeeper struct {
	games  map[uint32]*GameSession
	nextID uint32
	mu     sync.Mutex
}

var keeper *GameKeeper

func InitGameKeeper() {
	keeper = &GameKeeper{
		games:  make(map[uint32]*GameSession),
		nextID: 1,
	}
}

func GetGameKeeper() *GameKeeper {
	return keeper
}

func (g *GameKeeper) CreateGame(players []*Client, mode uint16) *GameSession {
	g.mu.Lock()
	defer g.mu.Unlock()

	startingBoard := chess.NewStartingPosition()
	gamesession := &GameSession{
		ID:           g.nextID,
		Players:      players,
		Mode:         mode,
		Board:        startingBoard,
		BoardHistory: []chess.Board{startingBoard},
		SideToMove:   chess.White,
		MoveChannel:  make(chan PlayerMove, 4),
	}
	g.games[g.nextID] = gamesession
	g.nextID++

	playerIDs := make([]uint32, len(players))
	for i, p := range players {
		playerIDs[i] = p.UserID
	}
	logger.Log.Info().Uint32("gameId", gamesession.ID).Interface("playerIDs", playerIDs).Msg("Game created")
	// Start game loop in separate goroutine
	go gamesession.Run()

	return gamesession
}

func (g *GameKeeper) GetGame(id uint32) (*GameSession, bool) {
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
