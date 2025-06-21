package internal

import (
	"log"
	"time"
)

// GameSession holds game-specific state
type GameSession struct {
	ID      int
	Players []*ClientConn
	// You can add board, turn, scores, etc.
}

func (g *GameSession) Run() {
	log.Printf("Game %d started!", g.ID)

	// TODO: Add your actual game logic here.
	// For example: turn management, game ticks, etc.

	// Example placeholder:
	for i := 0; i < 10; i++ {
		log.Printf("Game %d tick %d", g.ID, i)
		time.Sleep(1 * time.Second)
	}

	log.Printf("Game %d ended", g.ID)
}
