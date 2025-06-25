package internal

import (
	"encoding/json"
	"log"

	bh "github.com/zefir/szaszki-go-backend/internal/binaryHelpers"
	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

type GameSession struct {
	ID           uint32
	Players      []*Client
	Mode         uint16
	Board        chess.Board
	BoardHistory []chess.Board
	SideToMove   int // 0 = White, 1 = Black
	MoveChannel  chan PlayerMove
}

type PlayerMove struct {
	From   int8
	To     int8
	Player *Client
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
		move := <-g.MoveChannel
		log.Println(move)

		// Confirm move came from the correct player
		// if g.Players[g.SideToMove] != move.Player {
		// 	log.Println("ignoring move from wrong player:", move.Player.UserID)
		// 	continue
		// }

		//is move by correct palyer

		// check legality
		if !chess.IsMoveLegal(&g.Board, move.From, move.To, 1) {
			// reject move, ask player again
			continue
		}

		chess.MakeMove(&g.Board, move.From, move.To, 1)

		// update side to move
		g.SideToMove = 1 - g.SideToMove

		// broadcast updated board or move to players
		g.BroadcastMove(move.From, move.To)

		// TODO: check for game end (checkmate, stalemate, etc)
	}
}

func (g *GameSession) BroadcastMove(from, to int8) {

	payload, err := bh.Pack([]bh.FieldType{bh.Int8, bh.Int8, bh.Uint32}, []any{from, to, g.ID})
	if err != nil {
		log.Println("couldnt pack move")
		return
	}

	for _, p := range g.Players {
		_ = p.WriteMsg(ServerCmds.MoveHappend, payload)
	}
}
