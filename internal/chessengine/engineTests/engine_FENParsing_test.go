package chess

import (
	"testing"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

func TestFENParsing(t *testing.T) {
	testCases := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"rnbqkb1r/pppppppp/5n2/8/8/5N2/PPPPPPPP/RNBQKB1R w KQkq - 2 2",
		"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", // Castling position
		"8/8/8/4k3/8/8/4K3/8 w - - 0 1",        // Kings only
	}

	for _, fen := range testCases {
		board, err := chess.ParseFEN(fen)
		if err != nil {
			t.Errorf("Failed to parse FEN: %s\nError: %v", fen, err)
			continue
		}

		reconstructed := board.ToFEN()

		if fen != reconstructed {
			t.Errorf("FEN reconstruction mismatch\nOriginal: %s\nReconstructed: %s", fen, reconstructed)
		}
	}
}

// Test starting position
func TestStartingPosition(t *testing.T) {
	board := chess.NewStartingPosition()
	expectedFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	actualFEN := board.ToFEN()

	if actualFEN != expectedFEN {
		t.Errorf("Starting position FEN mismatch\nExpected: %s\nGot: %s", expectedFEN, actualFEN)
	}
}
