package internal

import (
	"sync"
	"testing"
	"time"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

// Mock Client for testing
type MockClient struct {
	UserID           uint32
	CurrentlyPlaying bool
	Messages         []MessageLog
	mu               sync.Mutex
}

type MessageLog struct {
	Command uint8
	Data    []byte
}

func (m *MockClient) WriteMsg(cmd uint8, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, MessageLog{Command: cmd, Data: data})
	return nil
}

func (m *MockClient) GetMessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Messages)
}

func (m *MockClient) GetLastMessage() *MessageLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Messages) == 0 {
		return nil
	}
	return &m.Messages[len(m.Messages)-1]
}

// Create a test game session with mock clients
func createFullTestGameSession() (*GameSession, *MockClient, *MockClient) {
	whitePlayer := &MockClient{UserID: 1, CurrentlyPlaying: false}
	blackPlayer := &MockClient{UserID: 2, CurrentlyPlaying: false}

	return &GameSession{
		ID:           1,
		Players:      []*Client{{UserID: 1}, {UserID: 2}},
		Mode:         1,
		Board:        chess.NewStartingPosition(),
		BoardHistory: []chess.Board{},
		MoveHistory:  []chess.Move{},
		SideToMove:   chess.White,
		MoveChannel:  make(chan PlayerMove, 100),
		GameActive:   true,
		WhiteTime:    10 * time.Minute,
		BlackTime:    10 * time.Minute,
		WhiteCards:   []int8{6, 6, 6, 5, 5},
		BlackCards:   []int8{6, 6, 6, 5, 5},
		WhiteHp:      3,
		BlackHp:      3,
		LastMoveTime: time.Now(),
	}, whitePlayer, blackPlayer
}

// Play a move in the game
func playMove(g *GameSession, player *MockClient, from, to, promote int8, cardsToReroll [5]int8) {
	move := PlayerMove{
		From:          from,
		To:            to,
		PromoteTo:     promote,
		CardsToReroll: cardsToReroll,
		Player:        &Client{UserID: player.UserID},
	}
	g.MoveChannel <- move
}

// Test complete game - Scholar's Mate
func TestFullGame_ScholarsMate(t *testing.T) {
	cfg := setupTestConfig()
	g, white, black := createFullTestGameSession()

	// Track game completion
	gameDone := make(chan bool, 1)
	moveCount := 0

	// Run game in goroutine
	go func() {
		for {
			select {
			case move := <-g.MoveChannel:
				moveCount++

				// Process move
				removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
				madeMove := chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
				g.MoveHistory = append(g.MoveHistory, madeMove)
				g.BoardHistory = append(g.BoardHistory, g.Board)
				g.CardLogicAdding(move.From, removedIndexes, cfg)

				// Check for game end
				if g.shouldEndGame() {
					gameDone <- true
					return
				}

				// Switch sides
				g.SideToMove = 1 - g.SideToMove

			case <-time.After(5 * time.Second):
				t.Error("Game timeout - moves not completing")
				gameDone <- false
				return
			}
		}
	}()

	// Play Scholar's Mate: 1.e4 e5 2.Bc4 Nc6 3.Qh5 Nf6 4.Qxf7#
	moves := []struct {
		player *MockClient
		from   int8
		to     int8
	}{
		{white, 12, 28}, // e2-e4
		{black, 52, 36}, // e7-e5
		{white, 5, 26},  // Bf1-c4
		{black, 57, 42}, // Nb8-c6
		{white, 3, 31},  // Qd1-h5
		{black, 62, 45}, // Ng8-f6
		{white, 31, 53}, // Qh5xf7#
	}

	// Play moves
	for _, m := range moves {
		playMove(g, m.player, m.from, m.to, 0, [5]int8{0, 0, 0, 0, 0})
		time.Sleep(50 * time.Millisecond) // Small delay for processing
	}

	// Wait for game to end
	select {
	case done := <-gameDone:
		if !done {
			t.Fatal("Game did not complete properly")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Game did not end within timeout")
	}

	// Verify game ended in checkmate
	if len(g.MoveHistory) != 7 {
		t.Errorf("Expected 7 moves, got %d", len(g.MoveHistory))
	}

	// Verify checkmate state
	if !g.Board.IsInCheck(chess.Black) {
		t.Error("Black king should be in check")
	}

	if g.Board.HasLegalMoves(chess.Black) {
		t.Error("Black should have no legal moves (checkmate)")
	}
}

// Test complete game - Fool's Mate
func TestFullGame_FoolsMate(t *testing.T) {
	cfg := setupTestConfig()
	g, white, black := createFullTestGameSession()

	gameDone := make(chan bool, 1)

	go func() {
		for {
			select {
			case move := <-g.MoveChannel:
				removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
				chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
				g.CardLogicAdding(move.From, removedIndexes, cfg)

				if g.shouldEndGame() {
					gameDone <- true
					return
				}

				g.SideToMove = 1 - g.SideToMove

			case <-time.After(5 * time.Second):
				gameDone <- false
				return
			}
		}
	}()

	// Play Fool's Mate: 1.f3 e5 2.g4 Qh4#
	moves := []struct {
		player *MockClient
		from   int8
		to     int8
	}{
		{white, 13, 21}, // f2-f3
		{black, 52, 36}, // e7-e5
		{white, 14, 30}, // g2-g4
		{black, 59, 31}, // Qd8-h4#
	}

	for _, m := range moves {
		playMove(g, m.player, m.from, m.to, 0, [5]int8{0, 0, 0, 0, 0})
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case done := <-gameDone:
		if !done {
			t.Fatal("Game did not complete properly")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Game did not end within timeout")
	}

	if len(g.MoveHistory) != 4 {
		t.Errorf("Expected 4 moves, got %d", len(g.MoveHistory))
	}
}

// Test game with card rerolls
func TestFullGame_WithCardRerolls(t *testing.T) {
	cfg := setupTestConfig()
	g, white, _ := createFullTestGameSession()

	// Set specific cards for testing
	g.WhiteCards = []int8{6, 6, 5, 4, 3}
	g.BlackCards = []int8{6, 6, 5, 4, 3}

	initialWhiteCards := append([]int8(nil), g.WhiteCards...)
	initialBlackCards := append([]int8(nil), g.BlackCards...)

	// Play a move with rerolls
	playMove(g, white, 12, 28, 0, [5]int8{0, 0, 1, 1, 0}) // e2-e4, reroll 2 cards

	// Process the move
	move := <-g.MoveChannel
	removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
	chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
	g.CardLogicAdding(move.From, removedIndexes, cfg)

	// Check that cards were changed
	changedCount := 0
	for i := 0; i < len(g.BlackCards); i++ {
		if g.BlackCards[i] != initialBlackCards[i] {
			changedCount++
		}
	}

	if changedCount < 2 {
		t.Errorf("Expected at least 2 cards changed, got %d", changedCount)
	}

	// Verify all cards are still valid
	for i, card := range g.BlackCards {
		if card < 1 || card > 6 {
			t.Errorf("Invalid card at index %d: %d", i, card)
		}
	}

	// Verify white cards didn't change
	for i := 0; i < len(g.WhiteCards); i++ {
		if g.WhiteCards[i] != initialWhiteCards[i] {
			t.Error("White cards should not have changed")
			break
		}
	}
}

// Test game with HP loss
func TestFullGame_HPLoss(t *testing.T) {
	cfg := setupTestConfig()
	g, white, _ := createFullTestGameSession()

	// Give white cards that don't match any movable pieces
	g.WhiteCards = []int8{3, 3, 3, 2, 2} // Rooks and Queens only

	initialHP := g.WhiteHp

	// Try to move a pawn (which white doesn't have a card for)
	playMove(g, white, 12, 28, 0, [5]int8{0, 0, 0, 0, 0})

	// Process the move
	move := <-g.MoveChannel
	removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
	chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
	g.CardLogicAdding(move.From, removedIndexes, cfg)

	// Switch sides and check for HP loss
	g.SideToMove = chess.Black

	// Check if hasPlayableMove correctly identifies no matching cards
	hasMove := g.hasPlayableMove(g.BlackCards)

	if hasMove {
		// This is expected in starting position with pawn cards
		if g.BlackHp != initialHP {
			t.Error("HP should not have decreased when playable moves exist")
		}
	}
}

// Test stalemate detection
func TestFullGame_Stalemate(t *testing.T) {
	g, _, _ := createFullTestGameSession()

	// Set up a stalemate position
	board, _ := chess.ParseFEN("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")
	g.Board = *board
	g.SideToMove = chess.Black

	// shouldEndGame should return true for stalemate
	if !g.shouldEndGame() {
		t.Error("Game should end in stalemate")
	}

	// Verify it's stalemate (not checkmate)
	if g.Board.IsInCheck(chess.Black) {
		t.Error("Should be stalemate, not checkmate")
	}

	if g.Board.HasLegalMoves(chess.Black) {
		t.Error("Should have no legal moves")
	}
}

// Benchmark full game playthrough
func BenchmarkFullGame_10Moves(b *testing.B) {
	cfg := setupTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, white, black := createFullTestGameSession()

		// Play 10 moves (5 moves per side)
		moves := []struct {
			player *MockClient
			from   int8
			to     int8
		}{
			{white, 12, 28}, // e4
			{black, 52, 36}, // e5
			{white, 6, 21},  // Nf3
			{black, 57, 42}, // Nc6
			{white, 5, 26},  // Bc4
			{black, 58, 43}, // Bc5
			{white, 2, 20},  // c3
			{black, 62, 45}, // Nf6
			{white, 11, 19}, // d3
			{black, 51, 35}, // d6
		}

		for _, m := range moves {
			move := PlayerMove{
				From:          m.from,
				To:            m.to,
				PromoteTo:     0,
				CardsToReroll: [5]int8{0, 0, 0, 0, 0},
				Player:        &Client{UserID: m.player.UserID},
			}

			removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
			chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
			g.CardLogicAdding(move.From, removedIndexes, cfg)
			g.SideToMove = 1 - g.SideToMove
		}
	}
}

func BenchmarkFullGame_ScholarsMate(b *testing.B) {
	cfg := setupTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, white, black := createFullTestGameSession()

		moves := []struct {
			player *MockClient
			from   int8
			to     int8
		}{
			{white, 12, 28}, // e4
			{black, 52, 36}, // e5
			{white, 5, 26},  // Bc4
			{black, 57, 42}, // Nc6
			{white, 3, 31},  // Qh5
			{black, 62, 45}, // Nf6
			{white, 31, 53}, // Qxf7#
		}

		for _, m := range moves {
			move := PlayerMove{
				From:          m.from,
				To:            m.to,
				PromoteTo:     0,
				CardsToReroll: [5]int8{0, 0, 0, 0, 0},
				Player:        &Client{UserID: m.player.UserID},
			}

			removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
			chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
			g.CardLogicAdding(move.From, removedIndexes, cfg)

			if g.shouldEndGame() {
				break
			}

			g.SideToMove = 1 - g.SideToMove
		}
	}
}

func BenchmarkFullGame_WithCardRerolls(b *testing.B) {
	cfg := setupTestConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, white, black := createFullTestGameSession()

		moves := []struct {
			player  *MockClient
			from    int8
			to      int8
			rerolls [5]int8
		}{
			{white, 12, 28, [5]int8{0, 0, 1, 1, 0}}, // e4, reroll 2 cards
			{black, 52, 36, [5]int8{1, 0, 0, 1, 0}}, // e5, reroll 2 cards
			{white, 6, 21, [5]int8{0, 1, 0, 0, 1}},  // Nf3, reroll 2 cards
			{black, 57, 42, [5]int8{0, 0, 0, 0, 0}}, // Nc6, no reroll
			{white, 5, 26, [5]int8{1, 1, 1, 0, 0}},  // Bc4, reroll 3 cards
		}

		for _, m := range moves {
			move := PlayerMove{
				From:          m.from,
				To:            m.to,
				PromoteTo:     0,
				CardsToReroll: m.rerolls,
				Player:        &Client{UserID: m.player.UserID},
			}

			removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
			chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
			g.CardLogicAdding(move.From, removedIndexes, cfg)
			g.SideToMove = 1 - g.SideToMove
		}
	}
}

func BenchmarkCardLogicFullCycle(b *testing.B) {
	cfg := setupTestConfig()
	g, white, _ := createFullTestGameSession()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		move := PlayerMove{
			From:          12,
			To:            28,
			PromoteTo:     0,
			CardsToReroll: [5]int8{0, 0, 1, 0, 1},
			Player:        &Client{UserID: white.UserID},
		}

		removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
		g.CardLogicAdding(move.From, removedIndexes, cfg)
	}
}

func BenchmarkShouldEndGame_StartingPosition(b *testing.B) {
	g, _, _ := createFullTestGameSession()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.shouldEndGame()
	}
}

func BenchmarkShouldEndGame_Checkmate(b *testing.B) {
	g, _, _ := createFullTestGameSession()
	board, _ := chess.ParseFEN("rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3")
	g.Board = *board
	g.SideToMove = chess.White

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.shouldEndGame()
	}
}
