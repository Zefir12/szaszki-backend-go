package internal

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

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
	WhiteTime    time.Duration
	BlackTime    time.Duration
	WhiteCards   []int8
	BlackCards   []int8
	WhiteHp      int8
	BlackHp      int8
	LastMoveTime time.Time
	Mu           sync.RWMutex
}

const (
	DefaultWhiteTime = 10 * time.Minute // or whatever your time control is
	DefaultBlackTime = 10 * time.Minute
)

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

	// Initialize timers
	g.WhiteTime = DefaultWhiteTime
	g.BlackTime = DefaultBlackTime
	g.LastMoveTime = time.Now()

	// Initialize hp
	g.WhiteHp = 3
	g.BlackHp = 3

	g.WhiteCards, g.BlackCards = chess.InitCardsWithDuplicates()

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

	g.BroadcastCards()
	g.BroadcastTime()
	g.BroadcastHp()

	// Game loop
	for {
		// wait for move from current player
		move := <-g.MoveChannel
		logger.Log.Info().Uint32("gameId", g.ID).Int("from", int(move.From)).Int("to", int(move.To)).Int("promoteTo", int(move.PromoteTo)).Uint32("playerId", move.Player.UserID).Msg("Received move")

		//Confirm move came from the correct player
		// if g.Players[g.SideToMove] == move.Player {
		// 	logger.Log.Warn().Uint32("playerId", move.Player.UserID).Uint32("gameId", g.ID).Msg("ignoring move from wrong player")
		// 	continue
		// }

		//is move by correct palyer

		// // check legality
		// if !chess.IsMoveLegal(&g.Board, move.From, move.To, move.PromoteTo) {
		// 	log.Printf("illigal move: from: %s, to: %s", IndexToSqaureName(move.From), IndexToSqaureName(move.To))
		// 	// reject move, ask player again
		// 	continue
		// }

		madeMove := chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
		g.MoveHistory = append(g.MoveHistory, madeMove)
		g.BoardHistory = append(g.BoardHistory, g.Board)

		g.BroadcastMove(move.From, move.To, move.PromoteTo)

		elapsed := time.Since(g.LastMoveTime)

		if g.SideToMove == chess.White {
			g.WhiteTime -= elapsed
			if g.WhiteTime < 0 {
				g.WhiteTime = 0
			}
		} else {
			g.BlackTime -= elapsed
			if g.BlackTime < 0 {
				g.BlackTime = 0
			}
		}

		g.BroadcastTime()

		g.LastMoveTime = time.Now()

		// TODO: check for game end (checkmate, stalemate, etc)
		if g.shouldEndGame() {
			g.saveGame()
			break
		}

		// update side to move
		g.SideToMove = 1 - g.SideToMove
	}
}

func IndexToSqaureName(index int8) string {
	if index < 0 || index > 63 {
		return "??"
	}

	file := index % 8
	rank := 8 - (index / 8)

	return fmt.Sprintf("%c%d", 'a'+file, rank)
}

func (g *GameSession) BroadcastMove(from, to, promote int8) {

	log.Printf("Broadcasting move: from=%s, to=%s, promote=%d, g.ID=%d",
		IndexToSqaureName(from), IndexToSqaureName(to), promote, g.ID,
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

func (g *GameSession) BroadcastTime() {
	payload, err := bh.Pack(
		[]bh.FieldType{bh.Int32, bh.Int32, bh.Uint32, bh.Int8},
		[]any{int32(g.WhiteTime.Milliseconds()), int32(g.BlackTime.Milliseconds()), g.ID, int8(g.SideToMove)},
	)
	if err != nil {
		logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("couldnt pack time status")
		return
	}

	for _, p := range g.Players {
		_ = p.WriteMsg(ServerCmds.TimeStatus, payload)
	}
}

func (g *GameSession) BroadcastHp() {
	payload, err := bh.Pack(
		[]bh.FieldType{bh.Int8, bh.Int8, bh.Uint32},
		[]any{g.WhiteHp, g.BlackHp, g.ID},
	)
	if err != nil {
		logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("couldnt pack hp data")
		return
	}

	for _, p := range g.Players {
		_ = p.WriteMsg(ServerCmds.HpStatus, payload)
	}
}

func (g *GameSession) BroadcastCards() {
	payload := make([]byte, 0, 2+len(g.WhiteCards)+len(g.BlackCards)+4)

	payload = append(payload, byte(len(g.WhiteCards)))
	payload = append(payload, byte(len(g.BlackCards)))

	// Append cards
	for _, c := range g.WhiteCards {
		payload = append(payload, byte(c))
	}
	for _, c := range g.BlackCards {
		payload = append(payload, byte(c))
	}

	// Append Game ID
	gameIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(gameIDBytes, g.ID)
	payload = append(payload, gameIDBytes...)

	for _, p := range g.Players {
		_ = p.WriteMsg(ServerCmds.CardStatus, payload)
	}
}

func (g *GameSession) shouldEndGame() bool {
	g.Mu.RLock()
	defer g.Mu.RUnlock()

	// Check if game is too old
	// if time.Since(g.LastActivity) > 10*time.Minute {
	// 	return true
	// }

	inCheck := g.Board.IsInCheck(int8(g.SideToMove))
	hasMoves := g.Board.HasLegalMoves(int8(g.SideToMove))

	if inCheck && !hasMoves {
		// checkmate
		log.Println("checkmate")
		return true
	} else if !inCheck && !hasMoves {
		// stalemate
		log.Println("stalemate")
		return true
	}

	if inCheck {
		if g.SideToMove == 0 {
			g.WhiteHp -= 1
		} else {
			g.BlackHp -= 1
		}
		g.BroadcastHp()
	}

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
