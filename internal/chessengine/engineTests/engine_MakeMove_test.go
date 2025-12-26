package chess

import (
	"strings"
	"testing"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

// Test MakeMove function
func TestMakeMove(t *testing.T) {
	tests := []struct {
		name        string
		startFEN    string
		from        string
		to          string
		promoteTo   int8
		expectedFEN string
	}{
		{
			name:        "En passant capture",
			startFEN:    "rnbqkbnr/ppp1p1pp/8/3pPp2/8/8/PPPP1PPP/RNBQKBNR w KQkq f6 0 3",
			from:        "e5",
			to:          "f6",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/ppp1p1pp/5P2/3p4/8/8/PPPP1PPP/RNBQKBNR b KQkq - 0 3",
		},
		{
			name:        "Simple pawn move e2-e4",
			startFEN:    "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			from:        "e2",
			to:          "e4",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		},
		{
			name:        "Knight move g1-f3",
			startFEN:    "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			from:        "g1",
			to:          "f3",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/pppppppp/8/8/8/5N2/PPPPPPPP/RNBQKB1R b KQkq - 1 1",
		},
		{
			name:        "Black pawn move e7-e5",
			startFEN:    "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			from:        "e7",
			to:          "e5",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
		},
		{
			name:        "Bishop move f1-c4",
			startFEN:    "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
			from:        "f1",
			to:          "c4",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/pppp1ppp/8/4p3/2B1P3/8/PPPP1PPP/RNBQK1NR b KQkq - 1 2",
		},
		{
			name:        "Knight move b8-c6",
			startFEN:    "rnbqkbnr/pppp1ppp/8/4p3/2B1P3/8/PPPP1PPP/RNBQK1NR b KQkq - 1 2",
			from:        "b8",
			to:          "c6",
			promoteTo:   0,
			expectedFEN: "r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/8/PPPP1PPP/RNBQK1NR w KQkq - 2 3",
		},
		{
			name:        "Capture - pawn takes pawn",
			startFEN:    "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
			from:        "e4",
			to:          "e5",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/pppp1ppp/8/4P3/8/8/PPPP1PPP/RNBQKBNR b KQkq - 0 2",
		},
		{
			name:        "White kingside castling",
			startFEN:    "rnbqkbnr/ppppp1pp/5p2/8/8/5N2/PPPPPPPP/RNBQK2R w KQkq - 0 1",
			from:        "e1",
			to:          "g1",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/ppppp1pp/5p2/8/8/5N2/PPPPPPPP/RNBQ1RK1 b kq - 1 1",
		},
		{
			name:        "White queenside castling",
			startFEN:    "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/R3KBNR w KQkq - 0 1",
			from:        "e1",
			to:          "c1",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/2KR1BNR b kq - 1 1",
		},
		{
			name:        "Black kingside castling",
			startFEN:    "rnbqk2r/pppppppp/5n2/8/8/5N2/PPPPPPPP/RNBQKB1R b KQkq - 0 1",
			from:        "e8",
			to:          "g8",
			promoteTo:   0,
			expectedFEN: "rnbq1rk1/pppppppp/5n2/8/8/5N2/PPPPPPPP/RNBQKB1R w KQ - 1 2",
		},
		{
			name:        "Black queenside castling",
			startFEN:    "r3kbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 1",
			from:        "e8",
			to:          "c8",
			promoteTo:   0,
			expectedFEN: "2kr1bnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQ - 1 2",
		},
		{
			name:        "Pawn promotion to Queen",
			startFEN:    "4k3/P7/8/8/8/8/8/4K3 w - - 0 1",
			from:        "a7",
			to:          "a8",
			promoteTo:   4, // Queen
			expectedFEN: "Q3k3/8/8/8/8/8/8/4K3 b - - 0 1",
		},
		{
			name:        "Pawn promotion to Knight",
			startFEN:    "4k3/P7/8/8/8/8/8/4K3 w - - 0 1",
			from:        "a7",
			to:          "a8",
			promoteTo:   2, // Knight
			expectedFEN: "N3k3/8/8/8/8/8/8/4K3 b - - 0 1",
		},
		{
			name:        "Rook move loses castling rights",
			startFEN:    "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			from:        "a1",
			to:          "b1",
			promoteTo:   0,
			expectedFEN: "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/1R2K2R b Kkq - 1 1",
		},
		{
			name:        "King move loses all castling rights",
			startFEN:    "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R3K2R w KQkq - 0 1",
			from:        "e1",
			to:          "d1",
			promoteTo:   0,
			expectedFEN: "r3k2r/pppppppp/8/8/8/8/PPPPPPPP/R2K3R b kq - 1 1",
		},
		{
			name:        "Black pawns moves forward after white",
			startFEN:    "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			from:        "e7",
			to:          "e5",
			promoteTo:   0,
			expectedFEN: "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := chess.ParseFEN(tt.startFEN)
			if err != nil {
				t.Fatalf("Failed to parse starting FEN: %v", err)
			}

			from := chess.SquareNameToIndex(tt.from)
			to := chess.SquareNameToIndex(tt.to)

			if from < 0 || to < 0 {
				t.Fatalf("Invalid square names: from=%s, to=%s", tt.from, tt.to)
			}

			// Make the move
			chess.MakeMove(board, from, to, tt.promoteTo, false)

			// Get resulting FEN
			resultFEN := board.ToFEN()

			if tt.expectedFEN != resultFEN {
				t.Errorf("FEN mismatch\nExpected: %s\n     Got: %s", tt.expectedFEN, resultFEN)
			}

		})
	}
}

// Test sequence of moves (Scholar's Mate)
func TestMoveSequence_ScholarsMate(t *testing.T) {
	board := chess.NewStartingPosition()

	moves := []struct {
		from string
		to   string
		fen  string
	}{
		{"e2", "e4", "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"},
		{"e7", "e5", "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2"},
		{"f1", "c4", "rnbqkbnr/pppp1ppp/8/4p3/2B1P3/8/PPPP1PPP/RNBQK1NR b KQkq - 1 2"},
		{"b8", "c6", "r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/8/PPPP1PPP/RNBQK1NR w KQkq - 2 3"},
		{"d1", "h5", "r1bqkbnr/pppp1ppp/2n5/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 3 3"},
		{"g8", "f6", "r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4"},
		{"h5", "f7", "r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4"},
	}

	for i, move := range moves {
		from := chess.SquareNameToIndex(move.from)
		to := chess.SquareNameToIndex(move.to)

		chess.MakeMove(&board, from, to, 0, false)

		resultFEN := board.ToFEN()
		expectedParts := strings.Fields(move.fen)
		resultParts := strings.Fields(resultFEN)

		// Compare first 4 fields
		for j := 0; j < 4; j++ {
			if expectedParts[j] != resultParts[j] {
				t.Errorf("Move %d (%s-%s) field %d mismatch\nExpected: %s\nGot: %s",
					i+1, move.from, move.to, j, expectedParts[j], resultParts[j])
			}
		}
	}

	// Verify checkmate
	if !board.IsInCheck(chess.Black) {
		t.Error("Black should be in check")
	}

	if board.HasAnyLegalMove(chess.Black) {
		t.Error("Black should have no legal moves (checkmate)")
	}
}
