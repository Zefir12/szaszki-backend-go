package internal

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/zefir/szaszki-go-backend/grpc"
	bh "github.com/zefir/szaszki-go-backend/internal/binaryHelpers"
	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
	"github.com/zefir/szaszki-go-backend/logger"

	pb "github.com/zefir/szaszki-go-backend/grpc/stuff"
)

type GameSession struct {
	ID           uint32
	Players      []*Client
	Mode         uint16
	Board        chess.Board
	BoardHistory []chess.Board
	MoveHistory  []chess.Move
	SideToMove   int // 0 = White, 1 = Black
	MoveChannel  chan PlayerMove
	GameActive   bool
	Mu           sync.RWMutex
}

type PlayerMove struct {
	From      int8
	To        int8
	PromoteTo int8
	Player    *Client
}

type GameStartMsg struct {
	GameMode  uint16 `json:"game_mode"`
	PlayerIDs []int  `json:"player_ids"`
	GameID    uint32 `json:"game_id"`
}

func (g *GameSession) Run() {
	logger.Log.Info().Uint32("gameId", g.ID).Msg("Game started!")

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
		logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("error marshaling game start message")
		return
	}

	for _, player := range g.Players {
		err := player.WriteMsg(ServerCmds.GameStarted, data)
		if err != nil {
			logger.Log.Warn().Err(err).Uint32("playerId", player.UserID).Uint32("gameId", g.ID).Msg("error sending message to player")
		}
	}

	// Game loop
	for {
		// wait for move from current player
		move := <-g.MoveChannel
		logger.Log.Info().Uint32("gameId", g.ID).Int("from", int(move.From)).Int("to", int(move.To)).Int("promoteTo", int(move.PromoteTo)).Uint32("playerId", move.Player.UserID).Msg("Received move")

		// Confirm move came from the correct player
		// if g.Players[g.SideToMove] != move.Player {
		// 	logger.Log.Warn().Uint32("playerId", move.Player.UserID).Uint32("gameId", g.ID).Msg("ignoring move from wrong player")
		// 	continue
		// }

		//is move by correct palyer

		// check legality
		if !chess.IsMoveLegal(&g.Board, move.From, move.To, move.PromoteTo) {
			// reject move, ask player again
			continue
		}

		madeMove := chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo)
		g.MoveHistory = append(g.MoveHistory, madeMove)
		g.BoardHistory = append(g.BoardHistory, g.Board)

		// update side to move
		g.SideToMove = 1 - g.SideToMove

		g.BroadcastMove(move.From, move.To, move.PromoteTo)

		// TODO: check for game end (checkmate, stalemate, etc)
		if g.shouldEndGame() {
			g.saveGame()
			break
		}
	}
}

func (g *GameSession) BroadcastMove(from, to, promote int8) {

	log.Printf("Broadcasting move: from=%d (%T), to=%d (%T), promote=%d (%T), g.ID=%d",
		from, from, to, to, promote, promote, g.ID,
	)
	payload, err := bh.Pack([]bh.FieldType{bh.Int8, bh.Int8, bh.Int8, bh.Uint32}, []any{from, to, promote, g.ID})
	if err != nil {
		logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("couldnt pack move")
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

func (g *GameSession) saveGame() {
	// Convert board history to byte slices
	var boardHistoryBytes [][]byte
	for _, board := range g.BoardHistory {
		boardHistoryBytes = append(boardHistoryBytes, board.ToByteArray())
	}

	// Convert move history to protobuf format
	var moveHistoryProto []*pb.Move
	for _, move := range g.MoveHistory {
		moveHistoryProto = append(moveHistoryProto, &pb.Move{From: int32(move.From), To: int32(move.To), Promotion: int32(move.Promotion)})
	}

	gameState := &pb.GameState{
		BoardHistory: boardHistoryBytes,
		MoveHistory:  moveHistoryProto,
	}

	pgn := g.Board.ToPGN(g.MoveHistory)

	_, err := grpc.SaveGame(g.ID, g.Players[0].UserID, g.Players[1].UserID, gameState, pgn)
	if err != nil {
		logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("Failed to save game")
	}
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
			logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("error packing game state header")
			continue
		}

		// Append board data
		payload = append(payload, boardBytes...)

		err = player.WriteMsg(ServerCmds.GameState, payload)
		if err != nil {
			logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Uint32("playerId", player.UserID).Msg("error sending game state to player")
		}
	}
}
