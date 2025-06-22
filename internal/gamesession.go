package internal

import (
	"encoding/json"
	"log"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

type GameSession struct {
	ID         uint32
	Players    []*Client
	Mode       uint16
	Board      chess.Board
	SideToMove int // 0 = White, 1 = Black
}

type GameStartMsg struct {
	GameMode  uint16 `json:"game_mode"`
	PlayerIDs []int  `json:"player_ids"`
	GameID    uint32 `json:"game_id"`
}

func (g *GameSession) Run() {
	log.Printf("Game %d started!", g.ID)

	g.Board = chess.NewStartingPosition()
	g.SideToMove = chess.White
	g.Board.Hash = chess.ComputeHash(&g.Board)

	var playerIDs []int
	for _, p := range g.Players {
		p.CurrentlyPlaying = true
		playerIDs = append(playerIDs, int(p.UserID))
	}

	msg := GameStartMsg{
		GameMode:  g.Mode,
		PlayerIDs: playerIDs,
		GameID:    g.ID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshaling game start message: %v", err)
		return
	}

	for _, player := range g.Players {
		err := player.WriteMsg(ServerCmds.GameStarted, data)
		if err != nil {
			log.Printf("error sending message to player %d: %v", player.UserID, err)
		}
	}

	// Game loop
	for {
		// wait for move from current player
		from, to, err := g.waitForPlayerMove(g.SideToMove)
		if err != nil {
			log.Printf("error receiving move: %v", err)
			return
		}

		// check legality
		if !chess.IsMoveLegal(&g.Board, from, to) {
			// reject move, ask player again
			continue
		}

		chess.MakeMove(&g.Board, from, to)

		// update side to move
		g.SideToMove = 1 - g.SideToMove

		// broadcast updated board or move to players

		// TODO: check for game end (checkmate, stalemate, etc)
	}
}

func (g *GameSession) BroadcastMove(from, to int8) {
	type MoveMsg struct {
		From int8   `json:"from"`
		To   int8   `json:"to"`
		Hash uint64 `json:"hash"`
	}
	msg := MoveMsg{
		From: from,
		To:   to,
		Hash: g.Board.Hash,
	}
	data, _ := json.Marshal(msg)
	for _, p := range g.Players {
		_ = p.WriteMsg(ServerCmds.MoveHappend, data)
	}
}

func (g *GameSession) waitForPlayerMove(color int) (from, to int8, err error) {
	// Example: wait for move JSON from player with color to move
	// Parse message, extract from/to squares (0-63)

	// For now, stub values:
	return 12, 28, nil
}
