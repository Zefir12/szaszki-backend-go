package internal

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/zefir/szaszki-go-backend/config"
	"github.com/zefir/szaszki-go-backend/grpc"
	bh "github.com/zefir/szaszki-go-backend/internal/binaryHelpers"
	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
	"github.com/zefir/szaszki-go-backend/logger"
)

type GameResult struct {
	Winner uint32
	Loser  uint32
	Reason string // "checkmate", "stalemate", "time", "resignation", "hp_loss"
	PGN    string
}

type GameSession struct {
	ID           uint32
	Players      []*Client
	Mode         uint16
	Board        chess.Board
	MoveHistory  []chess.Move
	TimeHistory  []time.Duration
	CardHistory  [][]chess.CardSwap
	MoveChannel  chan PlayerMove
	GameActive   bool
	WhiteTime    time.Duration
	BlackTime    time.Duration
	WhiteCards   []int8
	BlackCards   []int8
	WhiteHp      int8
	BlackHp      int8
	LastMoveTime time.Time
	GameResult   *GameResult
	clockTicker  *time.Ticker
	clockDone    chan bool
	Mu           sync.RWMutex
}

const (
	DefaultWhiteTime = 10 * time.Minute // or whatever your time control is
	DefaultBlackTime = 10 * time.Minute
)

type PlayerMove struct {
	From          int8
	To            int8
	PromoteTo     int8
	CardsToReroll [5]int8
	Player        *Client
}

type GameStartMsg struct {
	GameMode  uint16 `json:"game_mode"`
	PlayerIDs []int  `json:"player_ids"`
	GameID    uint32 `json:"game_id"`
}

// Improved clock with proper shutdown
func (g *GameSession) runClock() {
	g.clockTicker = time.NewTicker(100 * time.Millisecond) // More responsive
	defer g.clockTicker.Stop()

	lastBroadcast := time.Now()
	broadcastInterval := 500 * time.Millisecond

	for {
		select {
		case <-g.clockDone:
			return
		case now := <-g.clockTicker.C:
			g.Mu.Lock()

			if !g.GameActive {
				g.Mu.Unlock()
				return
			}

			elapsed := now.Sub(g.LastMoveTime)

			if g.Board.SideToMove() == chess.White {
				g.WhiteTime -= elapsed
				if g.WhiteTime <= 0 {
					g.WhiteTime = 0
					g.GameActive = false
					g.Mu.Unlock()
					g.endGame(chess.Black, chess.White, "time")
					return
				}
			} else {
				g.BlackTime -= elapsed
				if g.BlackTime <= 0 {
					g.BlackTime = 0
					g.GameActive = false
					g.Mu.Unlock()
					g.endGame(chess.White, chess.Black, "time")
					return
				}
			}

			g.LastMoveTime = now
			shouldBroadcast := now.Sub(lastBroadcast) >= broadcastInterval
			g.Mu.Unlock()

			// Broadcast time less frequently to reduce network traffic
			if shouldBroadcast {
				g.BroadcastTime()
				lastBroadcast = now
			}
		}
	}
}

func (g *GameSession) stopClock() {
	if g.clockDone != nil {
		close(g.clockDone)
	}
	if g.clockTicker != nil {
		g.clockTicker.Stop()
	}
}

func (g *GameSession) Run() {
	cfg, err := config.Instance.Get()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	logger.Log.Info().Uint32("gameId", g.ID).Msg("Game started!")

	g.Board = chess.NewStartingPosition()
	//g.SideToMove = chess.White

	// Initialize timers
	g.WhiteTime = DefaultWhiteTime
	g.BlackTime = DefaultBlackTime
	g.LastMoveTime = time.Now()

	// Initialize hp
	g.WhiteHp = 3
	g.BlackHp = 3

	g.clockDone = make(chan bool)

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

	go g.runClock()

	// Game loop
	for {
		// wait for move from current player
		move := <-g.MoveChannel
		//logger.Log.Info().Uint32("gameId", g.ID).Int("from", int(move.From)).Int("to", int(move.To)).Int("promoteTo", int(move.PromoteTo)).Uint32("playerId", move.Player.UserID).Msg("Received move")

		g.Mu.Lock()
		if !g.GameActive {
			g.Mu.Unlock()
			break
		}
		g.Mu.Unlock()

		color := g.Board.SideToMove()
		if g.Players[g.Board.SideToMove()] != move.Player {
			logger.Log.Warn().Uint32("playerId", move.Player.UserID).Uint32("gameId", g.ID).Msg("ignoring move from wrong player")
			log.Printf("Move from wrong player")
			continue
		}

		if !chess.IsMoveLegal(&g.Board, move.From, move.To, move.PromoteTo) {
			log.Printf("illigal move: from: %s, to: %s", IndexToSqaureName(move.From), IndexToSqaureName(move.To))
			// reject move, ask player again
			continue
		}

		removedCardIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
		madeMove := chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)

		if color == chess.White {
			g.TimeHistory = append(g.TimeHistory, g.WhiteTime)
		} else {
			g.TimeHistory = append(g.TimeHistory, g.BlackTime)
		}

		g.LastMoveTime = time.Now()
		g.MoveHistory = append(g.MoveHistory, madeMove)
		swapHistory := g.CardLogicAdding(removedCardIndexes, cfg)
		g.CardHistory = append(g.CardHistory, swapHistory)

		g.BroadcastCards()

		g.BroadcastMove(move.From, move.To, move.PromoteTo, cfg)

		g.BroadcastTime()

		g.Board.FlipSideToMove()

		if shouldEnd, winner, loser, reason := g.CheckGameEnd(g.Board.SideToMove()); shouldEnd {
			g.endGame(winner, loser, reason)
			break
		}
	}
}

func (g *GameSession) Surrender(client *Client) {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	if !g.GameActive {
		return
	}

	var winner, loser int8
	for i, p := range g.Players {
		if p.UserID == client.UserID {
			loser = int8(i)
			winner = 1 - int8(i)
			break
		}
	}

	g.GameActive = false
	g.Mu.Unlock()
	g.endGame(winner, loser, "resignation")
	g.Mu.Lock()
}

func IndexToSqaureName(index int8) string {
	if index < 0 || index > 63 {
		return "??"
	}

	file := index % 8
	rank := 8 - (index / 8)

	return fmt.Sprintf("%c%d", 'a'+file, rank)
}

func (g *GameSession) BroadcastMove(from, to, promote int8, cfg *config.ConfigValues) {
	if cfg.SHOW_EXTRA_LOGS {
		log.Printf("Broadcasting move: from=%s, to=%s, promote=%d, g.ID=%d",
			IndexToSqaureName(from), IndexToSqaureName(to), promote, g.ID,
		)
	}

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
	g.Mu.RLock()
	whiteTime := g.WhiteTime
	blackTime := g.BlackTime
	gameID := g.ID
	sideToMove := g.Board.SideToMove()
	g.Mu.RUnlock()

	payload, err := bh.Pack(
		[]bh.FieldType{bh.Int32, bh.Int32, bh.Uint32, bh.Int8},
		[]any{int32(whiteTime.Milliseconds()), int32(blackTime.Milliseconds()), gameID, sideToMove},
	)
	if err != nil {
		logger.Log.Warn().Err(err).Uint32("gameId", gameID).Msg("couldnt pack time status")
		return
	}

	for _, p := range g.Players {
		_ = p.WriteMsg(ServerCmds.TimeStatus, payload)
	}
}

func (g *GameSession) CardLogicRemoval(from int8, cardsToReroll [5]int8, cfg *config.ConfigValues) []int {
	var removedCardIndexes []int
	cards := &g.WhiteCards

	if g.Board.SideToMove() == chess.Black {
		cards = &g.BlackCards
	}

	piece := g.Board.GetPieceType(from, g.Board.SideToMove())

	// First pass: mark cards for reroll
	for i := 0; i < len(*cards) && i < len(cardsToReroll); i++ {
		if cardsToReroll[i] == 1 {
			removedCardIndexes = append(removedCardIndexes, i)
		}
	}

	// Second pass: find matching piece (if not already marked for reroll)
	for i, c := range *cards {
		if cardsToReroll[i] == 1 {
			continue // Already marked
		}
		enginePiece := chess.CardToEnginePiece(c)
		if enginePiece == piece {
			removedCardIndexes = append(removedCardIndexes, i)
			break
		}
	}
	return removedCardIndexes
}

func (g *GameSession) CardLogicAdding(removedCardIndexes []int, cfg *config.ConfigValues) []chess.CardSwap {
	swaps := []chess.CardSwap{}
	if g.Board.SideToMove() == chess.White {
		for _, idx := range removedCardIndexes {
			newCard := chess.GetRandomValidCard(&g.Board, chess.White)
			if idx >= 0 && idx < len(g.WhiteCards) {
				g.WhiteCards[idx] = newCard
				swaps = append(swaps, chess.CardSwap{IndexReplaced: int8(idx), NewCard: newCard})
			}
		}
	} else {
		for _, idx := range removedCardIndexes {
			newCard := chess.GetRandomValidCard(&g.Board, chess.Black)
			if idx >= 0 && idx < len(g.BlackCards) {
				g.BlackCards[idx] = newCard
				swaps = append(swaps, chess.CardSwap{IndexReplaced: int8(idx), NewCard: newCard})
			}
		}
	}
	return swaps
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

func (g *GameSession) BroadcastGameEnded() {
	payload, _ := json.Marshal(struct {
		GameID uint32 `json:"game_id"`
		Winner uint32 `json:"winner"`
		Loser  uint32 `json:"loser"`
		Reason string `json:"reason"`
	}{
		g.ID,
		1,
		2,
		"time",
	})

	for _, p := range g.Players {
		_ = p.WriteMsg(ServerCmds.GameEnded, payload)
	}
}

func (g *GameSession) HasPlayableMove(cards []int8, color int8) bool {
	var cardPieces uint8 = 0
	for _, card := range cards {
		pieceType := chess.CardToEnginePiece(card)
		if pieceType < 0 {
			continue
		}
		cardPieces |= 1 << pieceType
	}

	return g.Board.GeAllPiecesThatCanMoveLegallyThisTurn(color, cardPieces) > 0
}

// CheckGameEnd returns (shouldEnd, winner, loser, reason)
func (g *GameSession) CheckGameEnd(color int8) (bool, int8, int8, string) {
	g.Mu.RLock()
	defer g.Mu.RUnlock()

	inCheck := g.Board.IsInCheck(color)

	var currentCards []int8
	if color == chess.White {
		currentCards = g.WhiteCards
	} else {
		currentCards = g.BlackCards
	}

	hasPlayableMove := g.HasPlayableMove(currentCards, color)

	if !hasPlayableMove {
		if g.Board.HasAnyLegalMove(color) {
			// Player has legal moves but no matching cards - HP loss
			if color == chess.White {
				g.WhiteHp -= 1
				if g.WhiteHp <= 0 {
					return true, chess.Black, chess.White, "hp_loss"
				}
			} else {
				g.BlackHp -= 1
				if g.BlackHp <= 0 {
					return true, chess.White, chess.Black, "hp_loss"
				}
			}
			g.BroadcastHp()
		} else {
			// No legal moves at all
			if inCheck {
				winner := int8(1 - color)
				return true, winner, color, "checkmate"
			} else {
				return true, -1, -1, "stalemate"
			}
		}
	}

	return false, -1, -1, ""
}

// endGame handles all game ending logic
func (g *GameSession) endGame(winner, loser int8, reason string) {
	g.Mu.Lock()
	g.GameActive = false
	g.Mu.Unlock()

	// Stop the clock immediately
	g.stopClock()

	// Prepare result
	var winnerID, loserID uint32
	var pgnResult string

	if reason == "stalemate" {
		pgnResult = "1/2-1/2"
		if len(g.Players) >= 2 {
			winnerID = 0
			loserID = 0
		}
	} else {
		if winner == chess.White {
			pgnResult = "1-0"
		} else {
			pgnResult = "0-1"
		}

		if winner >= 0 && winner < int8(len(g.Players)) {
			winnerID = g.Players[winner].UserID
		}
		if loser >= 0 && loser < int8(len(g.Players)) {
			loserID = g.Players[loser].UserID
		}
	}

	g.GameResult = &GameResult{
		Winner: winnerID,
		Loser:  loserID,
		Reason: reason,
	}

	logger.Log.Info().
		Uint32("gameId", g.ID).
		Uint32("winner", winnerID).
		Uint32("loser", loserID).
		Str("reason", reason).
		Msg("Game ended")

	// Save game
	g.saveGame(pgnResult)

	// Broadcast to players
	g.BroadcastGameEnded()

	// Clean up player states
	for _, p := range g.Players {
		p.CurrentlyPlaying = false
	}
}

func (g *GameSession) saveGame(result string) {
	initialWhiteCards := []int8{6, 6, 6, 5, 5} // Store these at game start
	initialBlackCards := []int8{6, 6, 6, 5, 5}

	pgnGen := chess.NewPGNGenerator(
		fmt.Sprintf("Player_%d", g.Players[0].UserID),
		fmt.Sprintf("Player_%d", g.Players[1].UserID),
	)

	totalMinutes := int(DefaultWhiteTime.Minutes())
	pgnGen.TimeControl = fmt.Sprintf("%d+0", totalMinutes*60)
	pgnGen.Result = result

	pgn := pgnGen.GeneratePGN(
		g.MoveHistory,
		g.TimeHistory,
		g.CardHistory,
		initialWhiteCards,
		initialBlackCards,
	)

	fmt.Println(pgn)

	if g.GameResult != nil {
		g.GameResult.PGN = pgn
	}

	_, err := grpc.SaveGame(g.ID, g.Players[0].UserID, g.Players[1].UserID, pgn)
	if err != nil {
		logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("Failed to save game")
	}
}

func (g *GameSession) BroadcastGameState(targetClient *Client, conn *net.Conn) {
	cfg, err := config.Instance.Get()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if cfg.SHOW_EXTRA_LOGS {
		println("Triggered brodcasting gamestate for: ", targetClient.UserID)
	}
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
		if targetClient.UserID != player.UserID {
			continue
		}
		if cfg.SHOW_EXTRA_LOGS {
			println("Sending gamestate to: ", player.UserID)
		}

		sideToMove := g.Board.SideToMove()

		var enemyID uint32
		if len(g.Players) == 2 {
			if i == 0 {
				enemyID = g.Players[1].UserID
			} else {
				enemyID = g.Players[0].UserID
			}
		}

		// Pack complete game state
		payload, err := bh.Pack(
			[]bh.FieldType{bh.Uint32, bh.Uint32, bh.Uint8, bh.Uint8, bh.Uint8, bh.Int8, bh.Uint8, bh.Uint16},
			[]any{
				g.ID,
				enemyID,
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

		// --- Append HP ---
		hpBytes, err := bh.Pack(
			[]bh.FieldType{bh.Int8, bh.Int8},
			[]any{g.WhiteHp, g.BlackHp},
		)
		if err != nil {
			logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("error packing HP")
			continue
		}
		payload = append(payload, hpBytes...)

		// --- Append Cards ---
		payload = append(payload, byte(len(g.WhiteCards)))
		payload = append(payload, byte(len(g.BlackCards)))
		for _, c := range g.WhiteCards {
			payload = append(payload, byte(c))
		}
		for _, c := range g.BlackCards {
			payload = append(payload, byte(c))
		}

		//append alst move

		err = WriteMsgToSingleConn(*conn, ServerCmds.GameFullStatus, payload)
		if err != nil {
			logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Uint32("playerId", player.UserID).Msg("error sending game state to player")
		}
	}
}
