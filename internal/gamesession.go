package internal

import (
	"encoding/json"
	"log"
	"sync"

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
	GameActive   bool
	Mu           sync.RWMutex
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

func (g *GameSession) shouldEndGame() bool {
	g.Mu.RLock()
	defer g.Mu.RUnlock()

	// Check if game is too old
	// if time.Since(g.LastActivity) > 10*time.Minute {
	// 	return true
	// }

	// Check if players are still connected
	connectedPlayers := 0
	for _, player := range g.Players {
		if player.ConnCount() > 0 && !player.IsDisconnected() {
			connectedPlayers++
		}
	}

	// End game if less than 2 players connected
	return connectedPlayers < 2
}

func (g *GameSession) broadcastGameState() {
	if !g.GameActive {
		return
	}

	// Convert bitboard representation to square array
	squareArray := g.Board.ToSquareArray()

	// Pack the 64-square board as bytes
	boardBytes := make([]byte, 64)
	for i := 0; i < 64; i++ {
		boardBytes[i] = byte(squareArray[i])
	}

	for i, player := range g.Players {
		if player.ConnCount() == 0 {
			continue
		}

		sideToMove := g.Board.SideToMove()

		// Pack complete game state
		payload, err := bh.Pack(
			[]bh.FieldType{bh.Uint32, bh.Uint8, bh.Uint8, bh.Uint8, bh.Int8, bh.Uint8, bh.Uint16},
			[]any{
				g.ID,
				uint8(i),
				sideToMove,
				g.Board.Flags & 15, // Only castling bits (mask out WhiteToMove bit)
				g.Board.EnPassantSquare,
				g.Board.HalfmoveClock,
				g.Board.FullmoveNumber,
			},
		)
		if err != nil {
			log.Printf("Game %d: error packing game state header: %v", g.ID, err)
			continue
		}

		// Append board data
		payload = append(payload, boardBytes...)

		err = player.WriteMsg(ServerCmds.GameState, payload)
		if err != nil {
			log.Printf("Game %d: error sending game state to player %d: %v", g.ID, player.UserID, err)
		}
	}
}
