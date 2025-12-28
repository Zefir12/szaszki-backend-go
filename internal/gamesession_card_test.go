package internal

import (
	"os"
	"testing"
	"time"

	"github.com/zefir/szaszki-go-backend/config"
	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

func setupTestConfig() *config.ConfigValues {
	os.Setenv("SHOW_EXTRA_LOGS", "false")
	os.Setenv("WS_PORT", "8080")
	os.Setenv("GRPC_PORT", "50051")
	os.Setenv("NODE_GRPC_ADDR", "localhost:50052")
	os.Setenv("ADMIN_ENDPOINT_SECRET", "test-secret")

	// Reset singleton using constructor
	config.Instance = config.NewConfig()

	// Disable logs for tests
	config.Instance.SetLogger(func(string, ...any) {})

	cfg, err := config.Instance.Get()
	if err != nil {
		return &config.ConfigValues{
			SHOW_EXTRA_LOGS: false,
		}
	}
	return cfg
}

// Helper to create a test game session
func createTestGameSession() *GameSession {
	return &GameSession{
		ID:           1,
		Players:      []*Client{{UserID: 1}, {UserID: 2}},
		Mode:         1,
		Board:        chess.NewStartingPosition(),
		MoveHistory:  []chess.Move{},
		MoveChannel:  make(chan PlayerMove, 10),
		GameActive:   true,
		WhiteTime:    10 * time.Minute,
		BlackTime:    10 * time.Minute,
		WhiteCards:   []int8{6, 6, 6, 5, 5},
		BlackCards:   []int8{6, 6, 6, 5, 5},
		WhiteHp:      3,
		BlackHp:      3,
		LastMoveTime: time.Now(),
	}
}

func TestCardLogicRemoval_PawnMove(t *testing.T) {
	cfg := setupTestConfig()

	g := createTestGameSession()

	// White moves pawn from e2 (index 12)
	// Black has a pawn card (6) in their hand
	g.WhiteCards = []int8{6, 5, 4, 3, 2}

	// No rerolls requested
	cardsToReroll := [5]int8{0, 0, 0, 0, 0}

	removedIndexes := g.CardLogicRemoval(12, cardsToReroll, cfg)

	// Should remove the pawn card (index 0)
	if len(removedIndexes) != 1 {
		t.Errorf("Expected 1 removed card, got %d", len(removedIndexes))
	}
	if len(removedIndexes) > 0 && removedIndexes[0] != 0 {
		t.Errorf("Expected removed index 0, got %d", removedIndexes[0])
	}
}

func TestCardLogicRemoval_KnightMove(t *testing.T) {
	cfg := setupTestConfig()

	g := createTestGameSession()

	// White moves knight from b1 (index 1)
	// Black has a knight card (5)
	g.WhiteCards = []int8{6, 5, 4, 3, 2}

	cardsToReroll := [5]int8{0, 0, 0, 0, 0}

	removedIndexes := g.CardLogicRemoval(1, cardsToReroll, cfg)

	// Should remove the knight card (index 1)
	if len(removedIndexes) != 1 {
		t.Errorf("Expected 1 removed card, got %d", len(removedIndexes))
	}
	if len(removedIndexes) > 0 && removedIndexes[0] != 1 {
		t.Errorf("Expected removed index 1, got %d", removedIndexes[0])
	}
}

func TestCardLogicRemoval_NoMatch(t *testing.T) {
	cfg := setupTestConfig()

	g := createTestGameSession()

	g.WhiteCards = []int8{5, 5, 4, 3, 2}

	cardsToReroll := [5]int8{0, 0, 0, 0, 0}

	removedIndexes := g.CardLogicRemoval(12, cardsToReroll, cfg)

	// Should remove nothing (no matching card)
	if len(removedIndexes) != 0 {
		t.Errorf("Expected 0 removed cards, got %d", len(removedIndexes))
	}
}

func TestCardLogicRemoval_WithRerolls(t *testing.T) {
	cfg := setupTestConfig()

	g := createTestGameSession()

	// White moves pawn
	g.WhiteCards = []int8{6, 5, 4, 3, 2}

	// Request reroll of cards at indices 1 and 3
	cardsToReroll := [5]int8{0, 1, 0, 1, 0}

	removedIndexes := g.CardLogicRemoval(12, cardsToReroll, cfg)

	// Should remove: rerolled cards (1, 3) + matching pawn card (0) = 3 total
	if len(removedIndexes) != 3 {
		t.Errorf("Expected 3 removed cards, got %d", len(removedIndexes))
	}

	// Check that indices are correct (order doesn't matter)
	indexMap := make(map[int]bool)
	for _, idx := range removedIndexes {
		indexMap[idx] = true
	}

	if !indexMap[0] || !indexMap[1] || !indexMap[3] {
		t.Errorf("Expected indices [0, 1, 3], got %v", removedIndexes)
	}
}

func TestCardLogicRemoval_RerollBlocksMatch(t *testing.T) {
	cfg := setupTestConfig()

	g := createTestGameSession()

	// White moves pawn
	g.WhiteCards = []int8{6, 5, 4, 3, 2}

	// Request reroll of the pawn card itself
	cardsToReroll := [5]int8{1, 0, 0, 0, 0}

	removedIndexes := g.CardLogicRemoval(12, cardsToReroll, cfg)

	// Should only remove the rerolled card, not find it as a match
	if len(removedIndexes) != 1 {
		t.Errorf("Expected 1 removed card (reroll), got %d", len(removedIndexes))
	}
	if len(removedIndexes) > 0 && removedIndexes[0] != 0 {
		t.Errorf("Expected index 0, got %d", removedIndexes[0])
	}
}

func TestHasPlayableMove_WithMatchingCards(t *testing.T) {
	_ = setupTestConfig()

	g := createTestGameSession()
	g.Board = chess.NewStartingPosition()
	g.Board.SetSideToMove(chess.White)

	// White can move pawns and knights at start
	// Give white some pawn cards
	cards := []int8{6, 6, 5, 4, 3}

	result := g.hasPlayableMove(cards, g.Board.SideToMove())

	if !result {
		t.Error("Expected hasPlayableMove to return true with pawn cards")
	}
}

func TestHasPlayableMove_NoMatchingCards(t *testing.T) {
	_ = setupTestConfig()

	g := createTestGameSession()
	g.Board = chess.NewStartingPosition()
	g.Board.SetSideToMove(chess.White)

	// White can only move pawns and knights, but give only other cards
	cards := []int8{4, 4, 3, 3, 2}

	result := g.hasPlayableMove(cards, g.Board.SideToMove())

	if result {
		t.Error("Expected hasPlayableMove to return false without matching cards")
	}
}

func TestHasPlayableMove_Stalemate(t *testing.T) {
	_ = setupTestConfig()

	g := createTestGameSession()
	// Position where king has no legal moves
	board, _ := chess.ParseFEN("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")
	g.Board = *board
	g.Board.SetSideToMove(chess.Black)

	cards := []int8{1, 1, 1, 1, 1} // All king cards

	result := g.hasPlayableMove(cards, g.Board.SideToMove())

	if result {
		t.Error("Expected hasPlayableMove to return false in stalemate")
	}
}

func TestFullCardCycle(t *testing.T) {
	cfg := setupTestConfig()

	g := createTestGameSession()
	g.Board.SetSideToMove(chess.White)
	g.BlackCards = []int8{6, 6, 5, 4, 3}

	initialCardCount := len(g.BlackCards)

	// White moves pawn from e2
	cardsToReroll := [5]int8{0, 0, 1, 0, 0} // Reroll one card

	removedIndexes := g.CardLogicRemoval(12, cardsToReroll, cfg)
	g.CardLogicAdding(removedIndexes, cfg)

	// Should still have 5 cards
	if len(g.BlackCards) != initialCardCount {
		t.Errorf("Card count changed: expected %d, got %d",
			initialCardCount, len(g.BlackCards))
	}

	// All cards should be valid
	for i, card := range g.BlackCards {
		if card < 1 || card > 6 {
			t.Errorf("Invalid card at index %d: %d", i, card)
		}
	}
}

func BenchmarkHasPlayableMove(b *testing.B) {
	_ = setupTestConfig()

	g := createTestGameSession()
	cards := []int8{6, 6, 5, 4, 3}

	for b.Loop() {
		g.hasPlayableMove(cards, g.Board.SideToMove())
	}
}

func BenchmarkFullCardCycle(b *testing.B) {
	cfg := setupTestConfig()
	g := createTestGameSession()
	cardsToReroll := [5]int8{0, 0, 1, 0, 0}

	for b.Loop() {
		removedIndexes := g.CardLogicRemoval(12, cardsToReroll, cfg)
		g.CardLogicAdding(removedIndexes, cfg)
	}
}
