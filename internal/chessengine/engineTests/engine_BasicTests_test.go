package chess

import (
	"strings"
	"testing"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

// PGN Game parser for testing full games

// Test piece movement
func TestBasicMoves(t *testing.T) {
	tests := []struct {
		name     string
		fen      string
		from     string
		to       string
		expected string
		legal    bool
	}{
		{
			name:     "Pawn push e2-e4",
			fen:      "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			from:     "e2",
			to:       "e4",
			expected: "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			legal:    true,
		},
		{
			name:     "Knight move g1-f3",
			fen:      "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			from:     "g1",
			to:       "f3",
			expected: "rnbqkbnr/pppppppp/8/8/8/5N2/PPPPPPPP/RNBQKB1R b KQkq - 1 1",
			legal:    true,
		},
		{
			name:  "Illegal move - pawn backwards",
			fen:   "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
			from:  "e7",
			to:    "e8",
			legal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := chess.ParseFEN(tt.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}

			from := chess.SquareNameToIndex(tt.from)
			to := chess.SquareNameToIndex(tt.to)

			if tt.legal {
				isLegal := chess.IsMoveLegal(board, from, to, 0)
				if !isLegal {
					t.Errorf("Move should be legal but was rejected")
					return
				}

				chess.MakeMove(board, from, to, 0, false)
				board.FlipSideToMove()
				result := board.ToFEN()

				if !strings.HasPrefix(result, tt.expected) {
					t.Errorf("Position mismatch\nExpected: %s\nGot: %s", tt.expected, result)
				}
			} else {
				isLegal := chess.IsMoveLegal(board, from, to, 0)
				if isLegal {
					t.Errorf("Move should be illegal but was accepted")
				}
			}
		})
	}
}

// Test castling
func TestCastling(t *testing.T) {
	tests := []struct {
		name  string
		fen   string
		color int8
		side  bool // true = kingside, false = queenside
		can   bool
	}{
		{
			name:  "White kingside allowed",
			fen:   "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
			color: chess.White,
			side:  true,
			can:   true,
		},
		{
			name:  "White queenside allowed",
			fen:   "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
			color: chess.White,
			side:  false,
			can:   true,
		},
		{
			name:  "Black kingside blocked",
			fen:   "r3k1nr/8/8/8/8/8/8/R3K2R b KQkq - 0 1",
			color: chess.Black,
			side:  true,
			can:   false,
		},
		{
			name:  "White castling through check",
			fen:   "r3k2r/8/8/8/8/8/8/R3K2r w KQq - 0 1",
			color: chess.White,
			side:  true,
			can:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := chess.ParseFEN(tt.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}

			result := chess.CanCastle(board, tt.color, tt.side)
			if result != tt.can {
				t.Errorf("Expected CanCastle=%v, got %v", tt.can, result)
			}
		})
	}
}

// Test check detection
func TestCheckDetection(t *testing.T) {
	tests := []struct {
		name    string
		fen     string
		color   int8
		inCheck bool
	}{
		{
			name:    "White in check from rook",
			fen:     "4k3/8/8/8/8/8/8/4K2r w - - 0 1",
			color:   chess.White,
			inCheck: true,
		},
		{
			name:    "Black in check from bishop",
			fen:     "4k3/7p/8/1B6/8/7P/8/4K3 b - - 0 1",
			color:   chess.Black,
			inCheck: true,
		},
		{
			name:    "No check",
			fen:     "4k3/8/8/8/8/8/8/4K3 w - - 0 1",
			color:   chess.White,
			inCheck: false,
		},
		{
			name:    "Knight check",
			fen:     "4k3/7p/3N4/8/8/8/7P/4K3 b - - 0 1",
			color:   chess.Black,
			inCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := chess.ParseFEN(tt.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}

			result := board.IsInCheck(tt.color)
			if result != tt.inCheck {
				t.Errorf("Expected IsInCheck=%v, got %v", tt.inCheck, result)
			}
		})
	}
}

func TestPGNGames(t *testing.T) {
	tests, err := chess.LoadPGNTests("../../../testdata/games.pgn")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			games, err := chess.ParsePGNReader(strings.NewReader(tc.PGN))
			if err != nil {
				t.Fatal(err)
			}
			if len(games) != 1 {
				t.Fatalf("Expected 1 game, got %d", len(games))
			}

			game := games[0]
			board := chess.NewStartingPosition()
			for i, san := range game.Moves {
				move, err := chess.SANToMove(&board, san)
				if err != nil {
					t.Fatalf("Move %d (%s) failed: %v", i+1, san, err)
				}
				chess.MakeMove(&board, move.From, move.To, move.Promotion, false)
				board.FlipSideToMove()
			}

			engineCore := strings.Join(strings.Fields(board.ToFEN())[:4], " ")
			expectedCore := strings.Join(strings.Fields(tc.FinalFEN)[:4], " ")
			if engineCore != expectedCore {
				t.Fatalf("Final FEN mismatch\nExpected: %s\nGot:      %s", expectedCore, engineCore)
			}
		})
	}
}

func TestIsSquareAttacked(t *testing.T) {
	board := chess.NewStartingPosition()

	if !chess.IsSquareAttacked(43, &board, chess.Black) {
		t.Errorf("Square isnt attacked when it should be")
	}

	if !chess.IsSquareAttacked(16, &board, chess.White) {
		t.Errorf("Square isnt attacked when it should be")
	}

	if chess.IsSquareAttacked(24, &board, chess.Black) {
		t.Errorf("Square is attacked when it shouldn't be")
	}
	if chess.IsSquareAttacked(24, &board, chess.White) {
		t.Errorf("Square is attacked when it shouldn't be")
	}
}

func TestGetPiece(t *testing.T) {
	board := chess.NewStartingPosition()

	result := board.GetPieceType(0, chess.White)
	if result != chess.Rook {
		t.Errorf("Got =%v, expected rook", result)
	}

	result = board.GetPieceType(3, chess.White)
	if result != chess.Queen {
		t.Errorf("Got =%v, expected queen", result)
	}
}
