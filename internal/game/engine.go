package game

import (
	"log"

	"github.com/zefir/szaszki-go-backend/internal/ws"
)

type Game struct {
	GameId  int
	Players []*ws.ClientConn
	// Add game state fields
}

func StartGame() {
	log.Printf("Starting game in lobby %d", lobby.ID)

	game := &Game{
		LobbyID: lobby.ID,
		Players: lobby.Players,
	}

	for _, p := range game.Players {
		p.InGame = true
		ws.WriteMsg(p.Conn, ws.ServerCmds.GameStarted, nil)
	}

	go game.Run()
}

func (g *Game) Run() {
	// Handle turns, sync state, etc.
}
