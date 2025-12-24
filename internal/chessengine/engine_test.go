package chess

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// Test starting position
func TestStartingPosition(t *testing.T) {
	board := NewStartingPosition()
	expectedFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	actualFEN := board.ToFEN()

	if actualFEN != expectedFEN {
		t.Errorf("Starting position FEN mismatch\nExpected: %s\nGot: %s", expectedFEN, actualFEN)
	}
}

// Test FEN parsing and reconstruction
func TestFENParsing(t *testing.T) {
	testCases := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
		"rnbqkb1r/pppppppp/5n2/8/8/5N2/PPPPPPPP/RNBQKB1R w KQkq - 2 2",
		"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", // Castling position
		"8/8/8/4k3/8/8/4K3/8 w - - 0 1",        // Kings only
	}

	for _, fen := range testCases {
		board, err := ParseFEN(fen)
		if err != nil {
			t.Errorf("Failed to parse FEN: %s\nError: %v", fen, err)
			continue
		}

		reconstructed := board.ToFEN()
		// Compare without move counters (last two fields)
		originalParts := strings.Fields(fen)[:4]
		reconstructedParts := strings.Fields(reconstructed)[:4]

		if strings.Join(originalParts, " ") != strings.Join(reconstructedParts, " ") {
			t.Errorf("FEN reconstruction mismatch\nOriginal: %s\nReconstructed: %s",
				strings.Join(originalParts, " "),
				strings.Join(reconstructedParts, " "))
		}
	}
}

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
			fen:      "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq -",
			from:     "g1",
			to:       "f3",
			expected: "rnbqkbnr/pppppppp/8/8/8/5N2/PPPPPPPP/RNBQKB1R b KQkq -",
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
			board, err := ParseFEN(tt.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}

			from := squareNameToIndex(tt.from)
			to := squareNameToIndex(tt.to)

			if tt.legal {
				isLegal := IsMoveLegal(board, from, to, 0)
				if !isLegal {
					t.Errorf("Move should be legal but was rejected")
					return
				}

				MakeMove(board, from, to, 0, false)
				result := board.ToFEN()

				if !strings.HasPrefix(result, tt.expected) {
					t.Errorf("Position mismatch\nExpected: %s\nGot: %s", tt.expected, result)
				}
			} else {
				isLegal := IsMoveLegal(board, from, to, 0)
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
			fen:   "r3k2r/8/8/8/8/8/8/R3K2R w KQkq -",
			color: White,
			side:  true,
			can:   true,
		},
		{
			name:  "White queenside allowed",
			fen:   "r3k2r/8/8/8/8/8/8/R3K2R w KQkq -",
			color: White,
			side:  false,
			can:   true,
		},
		{
			name:  "Black kingside blocked",
			fen:   "r3k1nr/8/8/8/8/8/8/R3K2R b KQkq -",
			color: Black,
			side:  true,
			can:   false,
		},
		{
			name:  "White castling through check",
			fen:   "r3k2r/8/8/8/8/8/8/R3K2r w KQq -",
			color: White,
			side:  true,
			can:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := ParseFEN(tt.fen)
			if err != nil {
				t.Fatalf("Failed to parse FEN: %v", err)
			}

			result := CanCastle(board, tt.color, tt.side)
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
			fen:     "4k3/8/8/8/8/8/8/4K2r w - -",
			color:   White,
			inCheck: true,
		},
		{
			name:    "Black in check from bishop",
			fen:     "4k3/7p/8/1B6/8/7P/8/4K3 b - - 0 1",
			color:   Black,
			inCheck: true,
		},
		{
			name:    "No check",
			fen:     "4k3/8/8/8/8/8/8/4K3 w - -",
			color:   White,
			inCheck: false,
		},
		{
			name:    "Knight check",
			fen:     "4k3/7p/3N4/8/8/8/7P/4K3 b - - 0 1",
			color:   Black,
			inCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			board, err := ParseFEN(tt.fen)
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

// Helper to convert square name to index
func squareNameToIndex(name string) int8 {
	if len(name) != 2 {
		return -1
	}
	file := int8(name[0] - 'a')
	rank := int8(name[1] - '1')
	return rank*8 + file
}

// PGN Game parser for testing full games
type PGNGame struct {
	Event  string
	White  string
	Black  string
	Result string
	Moves  []string
}

func ParsePGN(filename string) ([]PGNGame, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var games []PGNGame
	var current PGNGame
	var movesSection strings.Builder

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[Event ") {
			current.Event = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[White ") {
			current.White = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[Black ") {
			current.Black = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[Result ") {
			current.Result = extractPGNValue(line)
		} else if line != "" && !strings.HasPrefix(line, "[") {
			movesSection.WriteString(line + " ")
		} else if line == "" && movesSection.Len() > 0 {
			// Parse moves
			current.Moves = parseMoveText(movesSection.String())
			games = append(games, current)
			current = PGNGame{}
			movesSection.Reset()
		}
	}

	return games, scanner.Err()
}

func ParsePGNReader(r io.Reader) ([]PGNGame, error) {
	var games []PGNGame
	var current PGNGame
	var movesSection strings.Builder

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[Event ") {
			current.Event = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[White ") {
			current.White = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[Black ") {
			current.Black = extractPGNValue(line)
		} else if strings.HasPrefix(line, "[Result ") {
			current.Result = extractPGNValue(line)
		} else if line != "" && !strings.HasPrefix(line, "[") {
			movesSection.WriteString(line + " ")
		} else if line == "" && movesSection.Len() > 0 {
			current.Moves = parseMoveText(movesSection.String())
			games = append(games, current)
			current = PGNGame{}
			movesSection.Reset()
		}
	}

	if movesSection.Len() > 0 {
		current.Moves = parseMoveText(movesSection.String())
		games = append(games, current)
	}

	return games, scanner.Err()
}

func extractPGNValue(line string) string {
	start := strings.Index(line, "\"")
	end := strings.LastIndex(line, "\"")
	if start >= 0 && end > start {
		return line[start+1 : end]
	}
	return ""
}

func parseMoveText(text string) []string {
	text = strings.ReplaceAll(text, "\n", " ")

	replacements := []string{
		"+", "", "#", "",
		"!", "", "?", "",
	}

	for i := 0; i < len(replacements); i += 2 {
		text = strings.ReplaceAll(text, replacements[i], replacements[i+1])
	}

	text = strings.ReplaceAll(text, "1-0", "")
	text = strings.ReplaceAll(text, "0-1", "")
	text = strings.ReplaceAll(text, "1/2-1/2", "")

	tokens := strings.Fields(text)
	moves := []string{}

	for _, tok := range tokens {
		if strings.HasSuffix(tok, ".") {
			continue
		}
		moves = append(moves, tok)
	}

	return moves
}

func getBitboardByPiece(b *Board, color int8, piece int) Bitboard {
	switch piece {
	case Pawn:
		return b.Pawns[color]
	case Knight:
		return b.Knights[color]
	case Bishop:
		return b.Bishops[color]
	case Rook:
		return b.Rooks[color]
	case Queen:
		return b.Queens[color]
	case King:
		return b.Kings[color]
	}
	return 0
}

func SANToMove(board *Board, san string) (Move, error) {
	// --- 1. Cleanup ---
	san = strings.TrimSpace(san)
	san = strings.TrimRight(san, "+#")

	if san == "O-O" {
		if board.Flags&WhiteToMove != 0 {
			return Move{From: 4, To: 6}, nil
		}
		return Move{From: 60, To: 62}, nil
	}

	if san == "O-O-O" {
		if board.Flags&WhiteToMove != 0 {
			return Move{From: 4, To: 2}, nil
		}
		return Move{From: 60, To: 58}, nil
	}

	// --- 2. Promotion ---
	promotion := int8(0)
	if i := strings.Index(san, "="); i != -1 {
		switch san[i+1] {
		case 'Q':
			promotion = Queen
		case 'R':
			promotion = Rook
		case 'B':
			promotion = Bishop
		case 'N':
			promotion = Knight
		}
		san = san[:i]
	}

	// --- 3. Piece type ---
	piece := Pawn
	start := 0

	switch san[0] {
	case 'N':
		piece = Knight
		start = 1
	case 'B':
		piece = Bishop
		start = 1
	case 'R':
		piece = Rook
		start = 1
	case 'Q':
		piece = Queen
		start = 1
	case 'K':
		piece = King
		start = 1
	}

	// Remove capture marker
	san = strings.ReplaceAll(san, "x", "")

	// --- 4. Target square ---
	if len(san) < start+2 {
		return Move{}, fmt.Errorf("invalid SAN: %s", san)
	}

	file := san[len(san)-2] - 'a'
	rank := san[len(san)-1] - '1'
	to := int8(rank*8 + file)

	// --- 5. Disambiguation ---
	disFile := int8(-1)
	disRank := int8(-1)
	middle := san[start : len(san)-2]

	for _, ch := range middle {
		if ch >= 'a' && ch <= 'h' {
			disFile = int8(ch - 'a')
		}
		if ch >= '1' && ch <= '8' {
			disRank = int8(ch - '1')
		}
	}

	// --- 6. Find legal origin ---
	color := Black
	if board.Flags&WhiteToMove != 0 {
		color = White
	}

	var candidates []int8
	bb := getBitboardByPiece(board, int8(color), piece)

	// fmt.Printf("SAN: %s\n", san)
	// fmt.Printf("Piece: %d, Color: %d, Target square: %d\n", piece, color, to)
	// fmt.Printf("Disambiguation: file=%d, rank=%d\n", disFile, disRank)
	// fmt.Printf("Bitboard: %064b\n", bb)

	// First pass: strict SAN (with disambiguation)
	bbCopy := bb
	for bbCopy != 0 {
		from := int8(PopLSB(&bbCopy))

		if piece == Pawn {
			// Add file check for pawn moves without explicit capture notation
			fromFile := from % 8
			toFile := to % 8

			// For non-capture pawn moves, file must match
			if disFile == -1 && strings.Index(san, "x") == -1 {
				if fromFile != toFile {
					continue
				}
			}

			// Only consider pawns that can legally move forward
			if disFile != -1 && from%8 != disFile {
				continue
			}
		} else {
			if disFile != -1 && from%8 != disFile {
				continue
			}
			if disRank != -1 && from/8 != disRank {
				continue
			}
		}

		if IsMoveLegal(board, from, to, promotion) {
			//fmt.Printf("Move allowed: %d -> %d\n", from, to)
			return Move{From: from, To: to, Promotion: promotion}, nil
		} else {
			//fmt.Printf("Move blocked: %d -> %d\n", from, to)
		}
	}

	// Second pass: ignore disambiguation
	bbCopy = bb
	for bbCopy != 0 {
		from := int8(PopLSB(&bbCopy))

		if piece == Pawn {
			targetRank := int8(rank)
			if color == White && from/8 >= targetRank {
				continue
			}
			if color == Black && from/8 <= targetRank {
				continue
			}
		}

		if IsMoveLegal(board, from, to, promotion) {
			candidates = append(candidates, from)
		}
	}

	if len(candidates) == 1 {
		return Move{From: candidates[0], To: to, Promotion: promotion}, nil
	}

	return Move{}, fmt.Errorf("cannot resolve SAN move: %s", san)
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_StartingPosition(t *testing.T) {
	board := NewStartingPosition()

	// Both sides should be able to move pawns and knights
	piecesToCheck := CanPawn | CanKnight
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(White, piecesToCheck) != piecesToCheck {
		t.Error("White should be able to move pawns and knights from starting position")
	}

	if board.GeAllPiecesThatCanMoveLegallyThisTurn(Black, piecesToCheck) != piecesToCheck {
		t.Error("Black should be able to move pawns and knights from starting position")
	}
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_CheckSituation(t *testing.T) {
	board, _ := ParseFEN("8/8/k7/1b6/8/3R4/4K3/8 w - - 0 1")

	// Both sides should be able to move pawns and knights
	piecesToCheck := CanKing
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(White, piecesToCheck) != piecesToCheck {
		t.Error("White should be able to move only king")
	}

	board, _ = ParseFEN("8/8/k7/1r6/8/3B4/4K3/8 b - - 0 1")
	piecesToCheck = CanKing
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(Black, piecesToCheck) != piecesToCheck {
		t.Error("Black should be able to move only king")
	}
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_Stalemate(t *testing.T) {
	// Black is stalemated
	board, _ := ParseFEN("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")

	piecesToCheck := CanPawn | CanKnight | CanBishop | CanRook | CanQueen | CanKing
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(Black, piecesToCheck) != 0 {
		t.Error("Black should have no legal moves (stalemate)")
	}
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_Checkmate(t *testing.T) {
	// Black is checkmated
	board, _ := ParseFEN("7k/6Q1/6K1/8/8/8/8/8 b - - 0 1")

	piecesToCheck := CanPawn | CanKnight | CanBishop | CanRook | CanQueen | CanKing
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(Black, piecesToCheck) != 0 {
		t.Error("Black should have no legal moves (checkmate)")
	}
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_OnlyCheckRooks(t *testing.T) {
	board, _ := ParseFEN("8/8/8/8/8/8/8/R3K2k b - - 1 1")

	piecesToCheck := CanRook
	result := board.GeAllPiecesThatCanMoveLegallyThisTurn(White, piecesToCheck)

	if result != CanRook {
		t.Errorf("Only rook should be movable, got %b", result)
	}
}

type PGNTest struct {
	Name     string
	PGN      string
	FinalFEN string
}

func LoadPGNTests(path string) ([]PGNTest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var games []PGNTest

	var currentName string
	var currentPGN []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") {
			// Start of a new game
			if len(currentPGN) > 0 {
				// Save the previous game
				pgnText := strings.Join(currentPGN, "\n")
				finalFEN, err := ExtractFinalFEN(pgnText)
				if err != nil {
					return nil, fmt.Errorf("could not extract final FEN for game %s: %v", currentName, err)
				}
				games = append(games, PGNTest{
					Name:     currentName,
					PGN:      pgnText,
					FinalFEN: finalFEN,
				})
			}
			// Start new game
			currentName = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			currentPGN = []string{}
			continue
		}

		currentPGN = append(currentPGN, line)
	}

	// Add the last game
	if len(currentPGN) > 0 {
		pgnText := strings.Join(currentPGN, "\n")
		finalFEN, err := ExtractFinalFEN(pgnText)
		if err != nil {
			return nil, fmt.Errorf("could not extract final FEN for game %s: %v", currentName, err)
		}
		games = append(games, PGNTest{
			Name:     currentName,
			PGN:      pgnText,
			FinalFEN: finalFEN,
		})
	}

	return games, nil
}

func TestPGNGames(t *testing.T) {
	tests, err := LoadPGNTests("../../testdata/games.pgn")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			games, err := ParsePGNReader(strings.NewReader(tc.PGN))
			if err != nil {
				t.Fatal(err)
			}
			if len(games) != 1 {
				t.Fatalf("Expected 1 game, got %d", len(games))
			}

			game := games[0]
			board := NewStartingPosition()
			for i, san := range game.Moves {
				move, err := SANToMove(&board, san)
				if err != nil {
					t.Fatalf("Move %d (%s) failed: %v", i+1, san, err)
				}
				MakeMove(&board, move.From, move.To, move.Promotion, false)
			}

			engineCore := strings.Join(strings.Fields(board.ToFEN())[:4], " ")
			expectedCore := strings.Join(strings.Fields(tc.FinalFEN)[:4], " ")
			if engineCore != expectedCore {
				t.Fatalf("Final FEN mismatch\nExpected: %s\nGot:      %s", expectedCore, engineCore)
			}
		})
	}
}

func ExtractFinalFEN(pgn string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(pgn))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[CurrentPosition") {
			// line format: [CurrentPosition "FEN"]
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start == -1 || end == -1 || end <= start {
				return "", errors.New("invalid CurrentPosition header format")
			}
			return line[start+1 : end], nil
		}
	}
	return "", errors.New("CurrentPosition header not found")
}

func TestCanAnyPawnMove_StartingPosition(t *testing.T) {
	board := NewStartingPosition()

	// Both sides should be able to move pawns
	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to move pawns from starting position")
	}

	if !board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to move pawns from starting position")
	}
}

func TestCanAnyPawnMove_NoPawns(t *testing.T) {
	board, _ := ParseFEN("rnbqkbnr/8/8/8/8/8/8/RNBQKBNR w KQkq - 0 1")

	if board.CanAnyPawnMove(White) {
		t.Error("White should not be able to move pawns when they have none")
	}

	if board.CanAnyPawnMove(Black) {
		t.Error("Black should not be able to move pawns when they have none")
	}
}

func TestCanAnyPawnMove_BlockedPawns(t *testing.T) {
	// All white and black pawns blocked
	board, _ := ParseFEN("4k3/8/pppppppp/rrrrrrrr/RRRRRRRR/PPPPPPPP/8/5K2 w - - 0 1")

	if board.CanAnyPawnMove(White) {
		t.Error("White pawns should be completely blocked")
	}

	if board.CanAnyPawnMove(Black) {
		t.Error("Black pawns should be completely blocked")
	}
}

func TestCanAnyPawnMove_SinglePush(t *testing.T) {
	// White pawn on e2 can push to e3
	board, _ := ParseFEN("8/8/8/8/8/8/4P3/8 w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to push pawn from e2")
	}

	// Black pawn on e7 can push to e6
	board, _ = ParseFEN("8/4p3/8/8/8/8/8/8 b - - 0 1")

	if !board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to push pawn from e7")
	}
}

func TestCanAnyPawnMove_DoublePush(t *testing.T) {
	// White pawn on e2 can double push
	board, _ := ParseFEN("8/8/8/8/8/8/4P3/8 w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to double push from e2")
	}

	// Black pawn on e7 can double push
	board, _ = ParseFEN("8/4p3/8/8/8/8/8/8 b - - 0 1")

	if !board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to double push from e7")
	}
}

func TestCanAnyPawnMove_DoublePushBlocked(t *testing.T) {
	// White pawn on e2, piece on e4 (can still single push to e3)
	board, _ := ParseFEN("8/8/8/8/4n3/8/4P3/8 w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should still be able to single push")
	}

	// Completely blocked
	board, _ = ParseFEN("8/8/8/8/8/4n3/4P3/8 w - - 0 1")

	if board.CanAnyPawnMove(White) {
		t.Error("White pawn should be completely blocked")
	}
}

func TestCanAnyPawnMove_CaptureAvailable(t *testing.T) {
	// White pawn can capture black knight
	board, _ := ParseFEN("8/8/8/8/3n4/2P5/8/8 w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to capture on d4")
	}

	// Black pawn can capture white knight
	board, _ = ParseFEN("8/8/2p5/3N4/8/8/8/8 b - - 0 1")

	if !board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to capture on d5")
	}
}

func TestCanAnyPawnMove_EnPassantAvailable(t *testing.T) {
	// White pawn can capture en passant on d6
	board, _ := ParseFEN("rnbqkb1r/ppp1pppp/7n/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to capture en passant on e6")
	}

	// Black pawn can capture en passant on e3
	board, _ = ParseFEN("8/8/8/8/3Pp3/8/8/8 b - e3 0 1")

	if !board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to capture en passant on e3")
	}
}

func TestCanAnyPawnMove_PromotionRank(t *testing.T) {
	// White pawn on 7th rank can promote
	board, _ := ParseFEN("8/4P3/8/8/8/8/8/8 w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to push pawn to promotion")
	}

	// Black pawn on 2nd rank can promote
	board, _ = ParseFEN("8/8/8/8/8/8/4p3/8 b - - 0 1")

	if !board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to push pawn to promotion")
	}
}

func TestCanAnyPawnMove_ComplexPosition(t *testing.T) {
	// Multiple pawns, some blocked, some free
	board, _ := ParseFEN("7k/8/8/2ppp3/3P4/8/PP6/6K1 w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to move pawns (a2, b2, or captures with d4)")
	}

	board, _ = ParseFEN("8/8/8/2ppp3/3P4/8/PP6/8 b - - 0 1")

	if !board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to move pawns (c5, e5 forward or d5 captures)")
	}
}

func TestCanAnyPawnMove_OnlyCapturesPossible(t *testing.T) {
	// White pawn blocked in front but can capture
	board, _ := ParseFEN("7k/8/8/8/8/2npn3/3P4/7K w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to capture on c3 or e3")
	}
}

func TestCanAnyPawnMove_AllMovesBlocked(t *testing.T) {
	// Pawn completely surrounded by own pieces
	board, _ := ParseFEN("7k/8/8/8/3R4/2RPR3/8/7K w - - 0 1")

	if board.CanAnyPawnMove(White) {
		t.Error("White pawns should all be blocked")
	}
}

func TestCanAnyPawnMove_MixedBlockedAndFree(t *testing.T) {
	// Some pawns blocked, one free
	board, _ := ParseFEN("8/8/8/pppp4/BBBB4/PPPP4/7P/8 w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to move h2 pawn")
	}
}

func TestCanAnyPawnMove_PawnsBlockedFromPromoting(t *testing.T) {
	// Pawn on 7th rank, blocked from promoting and no other move
	board, _ := ParseFEN("4n2k/4P3/8/8/8/8/8/7K w - - 0 1")

	if board.CanAnyPawnMove(White) {
		t.Error("White pawn should be blocked from promoting")
	}

	//Pawn on 2th rank, blocked from promoting and no other move
	board, _ = ParseFEN("7k/8/8/8/8/8/4p3/4N2K b - - 0 1")

	if board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to capture and promote")
	}
}

func TestCanAnyPawnMove_CaptureOnPromotionRank(t *testing.T) {
	// White pawn can capture and promote
	board, _ := ParseFEN("3n3k/4P3/8/8/8/8/8/7K w - - 0 1")

	if !board.CanAnyPawnMove(White) {
		t.Error("White should be able to capture and promote")
	}

	// Black pawn can capture and promote
	board, _ = ParseFEN("7k/8/8/8/8/8/4p3/3N3K b - - 0 1")

	if !board.CanAnyPawnMove(Black) {
		t.Error("Black should be able to capture and promote")
	}
}

func TestCanAnyRookMove_StartingPosition(t *testing.T) {
	board := NewStartingPosition()

	// At the very start, rooks are blocked by pawns
	if board.CanAnyRookMove(White) {
		t.Error("White rooks should not be able to move from starting position")
	}
	if board.CanAnyRookMove(Black) {
		t.Error("Black rooks should not be able to move from starting position")
	}
}

func TestCanAnyRookMove_ClearPath(t *testing.T) {
	// White rook on a1 with empty file
	board, _ := ParseFEN("R7/7k/8/8/8/7K/8/8 w - - 0 1")
	if !board.CanAnyRookMove(White) {
		t.Error("White rook on a1 should be able to move freely")
	}

	// Black rook on h8 with empty file
	board, _ = ParseFEN("7r/5k2/8/8/8/4K3/8/8 b - - 0 1")
	if !board.CanAnyRookMove(Black) {
		t.Error("Black rook on h8 should be able to move freely")
	}
}

func TestCanAnyRookMove_BlockedRooks(t *testing.T) {
	// Rooks completely blocked by own pieces
	board, _ := ParseFEN("RRRRRRKR/PPPPPPPP/8/8/8/8/pppppppp/rrrrrrkr w - - 0 1")
	if board.CanAnyRookMove(White) {
		t.Error("White rooks should all be blocked")
	}
	if board.CanAnyRookMove(Black) {
		t.Error("Black rooks should all be blocked")
	}
}

func TestCanAnyRookMove_SomeBlockedSomeFree(t *testing.T) {
	// White rook on a1 blocked, rook on h1 free
	board, _ := ParseFEN("RN3K1R/NN6/8/8/8/8/8/5k2 w - - 0 1")
	if !board.CanAnyRookMove(White) {
		t.Error("White should be able to move at least one rook")
	}
}

func TestCanAnyRookMove_CaptureAvailable(t *testing.T) {
	// White rook can capture black piece
	board, _ := ParseFEN("RN3k2/pN6/8/8/8/8/5K2/8 w - - 0 1")
	if !board.CanAnyRookMove(White) {
		t.Error("White rook should be able to capture black pawn")
	}

	// Black rook can capture white piece
	board, _ = ParseFEN("8/8/4k3/8/3K4/8/6nP/6nr b - - 0 1")
	if !board.CanAnyRookMove(Black) {
		t.Error("Black rook should be able to capture white pawn")
	}
}

func TestCanAnyRookMoveSafe_StartingPosition(t *testing.T) {
	board := NewStartingPosition()

	// At the very start, rooks are blocked by pawns
	if board.CanAnyRookMoveSafe(White) {
		t.Error("White rooks should not be able to move from starting position")
	}
	if board.CanAnyRookMoveSafe(Black) {
		t.Error("Black rooks should not be able to move from starting position")
	}
}

func TestCanAnyRookMoveSafe_ClearPath(t *testing.T) {
	// White rook on a1 with empty file
	board, _ := ParseFEN("R7/7k/8/8/8/7K/8/8 w - - 0 1")
	if !board.CanAnyRookMoveSafe(White) {
		t.Error("White rook on a1 should be able to move freely")
	}

	// Black rook on h8 with empty file
	board, _ = ParseFEN("7r/5k2/8/8/8/4K3/8/8 b - - 0 1")
	if !board.CanAnyRookMoveSafe(Black) {
		t.Error("Black rook on h8 should be able to move freely")
	}
}

func TestCanAnyRookMoveSafe_BlockedRooks(t *testing.T) {
	// Rooks completely blocked by own pieces
	board, _ := ParseFEN("RRRRRRKR/PPPPPPPP/8/8/8/8/pppppppp/rrrrrrkr w - - 0 1")
	if board.CanAnyRookMoveSafe(White) {
		t.Error("White rooks should all be blocked")
	}
	if board.CanAnyRookMoveSafe(Black) {
		t.Error("Black rooks should all be blocked")
	}
}

func TestCanAnyRookMoveSafe_SomeBlockedSomeFree(t *testing.T) {
	// White rook on a1 blocked, rook on h1 free
	board, _ := ParseFEN("RN3K1R/NN6/8/8/8/8/8/5k2 w - - 0 1")
	if !board.CanAnyRookMoveSafe(White) {
		t.Error("White should be able to move at least one rook")
	}
}

func TestCanAnyRookMoveSafe_CantMoveCouseCheck(t *testing.T) {
	// White rook has clear path to move but king would be left in check
	board, _ := ParseFEN("8/8/k7/1b6/8/3R4/4K3/8 w - - 0 1")
	if board.CanAnyRookMoveSafe(White) {
		t.Error("White rook shouldn't be able to move")
	}

	// Black rook has clear path to move but king would be left in check
	board, _ = ParseFEN("8/8/k7/1r6/8/3B4/4K3/8 b - - 0 1")
	if board.CanAnyRookMoveSafe(Black) {
		t.Error("Black rook shouldn't be able to move")
	}
}
