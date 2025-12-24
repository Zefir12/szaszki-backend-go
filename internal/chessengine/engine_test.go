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

// FEN parsing and validation
func ParseFEN(fen string) (*Board, error) {
	parts := strings.Fields(fen)
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid FEN: not enough parts")
	}

	board := &Board{}
	board.EnPassantSquare = -1

	// Parse piece placement
	ranks := strings.Split(parts[0], "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("invalid FEN: expected 8 ranks")
	}

	for rankIdx, rankStr := range ranks {
		file := 0
		rank := 7 - rankIdx // FEN starts from rank 8

		for _, ch := range rankStr {
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
			} else {
				if file > 7 {
					return nil, fmt.Errorf("invalid FEN: too many files")
				}

				square := rank*8 + file
				bb := Bitboard(1) << square

				color := White
				if ch >= 'a' && ch <= 'z' {
					color = Black
					ch = ch - 32 // Convert to uppercase
				}

				switch ch {
				case 'P':
					board.Pawns[color] |= bb
				case 'N':
					board.Knights[color] |= bb
				case 'B':
					board.Bishops[color] |= bb
				case 'R':
					board.Rooks[color] |= bb
				case 'Q':
					board.Queens[color] |= bb
				case 'K':
					board.Kings[color] |= bb
				default:
					return nil, fmt.Errorf("invalid piece: %c", ch)
				}

				board.Occupied[color] |= bb
				file++
			}
		}
	}

	// Parse side to move
	if parts[1] == "w" {
		board.Flags |= WhiteToMove
	}

	// Parse castling rights
	if parts[2] != "-" {
		for _, ch := range parts[2] {
			switch ch {
			case 'K':
				board.Flags |= WK
			case 'Q':
				board.Flags |= WQ
			case 'k':
				board.Flags |= BK
			case 'q':
				board.Flags |= BQ
			}
		}
	}

	// Parse en passant square
	if parts[3] != "-" {
		file := int(parts[3][0] - 'a')
		rank := int(parts[3][1] - '1')
		board.EnPassantSquare = int8(rank*8 + file)
	}

	board.Hash = ComputeHash(board)
	return board, nil
}

// Convert board to FEN
func (b *Board) ToFEN() string {
	var fen strings.Builder

	// Piece placement
	for rank := 7; rank >= 0; rank-- {
		empty := 0
		for file := 0; file < 8; file++ {
			square := rank*8 + file
			bb := Bitboard(1) << square

			piece := ""
			if b.Pawns[White]&bb != 0 {
				piece = "P"
			} else if b.Pawns[Black]&bb != 0 {
				piece = "p"
			} else if b.Knights[White]&bb != 0 {
				piece = "N"
			} else if b.Knights[Black]&bb != 0 {
				piece = "n"
			} else if b.Bishops[White]&bb != 0 {
				piece = "B"
			} else if b.Bishops[Black]&bb != 0 {
				piece = "b"
			} else if b.Rooks[White]&bb != 0 {
				piece = "R"
			} else if b.Rooks[Black]&bb != 0 {
				piece = "r"
			} else if b.Queens[White]&bb != 0 {
				piece = "Q"
			} else if b.Queens[Black]&bb != 0 {
				piece = "q"
			} else if b.Kings[White]&bb != 0 {
				piece = "K"
			} else if b.Kings[Black]&bb != 0 {
				piece = "k"
			}

			if piece != "" {
				if empty > 0 {
					fen.WriteString(fmt.Sprintf("%d", empty))
					empty = 0
				}
				fen.WriteString(piece)
			} else {
				empty++
			}
		}
		if empty > 0 {
			fen.WriteString(fmt.Sprintf("%d", empty))
		}
		if rank > 0 {
			fen.WriteString("/")
		}
	}

	// Side to move
	if b.Flags&WhiteToMove != 0 {
		fen.WriteString(" w ")
	} else {
		fen.WriteString(" b ")
	}

	// Castling rights
	castling := ""
	if b.Flags&WK != 0 {
		castling += "K"
	}
	if b.Flags&WQ != 0 {
		castling += "Q"
	}
	if b.Flags&BK != 0 {
		castling += "k"
	}
	if b.Flags&BQ != 0 {
		castling += "q"
	}
	if castling == "" {
		castling = "-"
	}
	fen.WriteString(castling)

	// En passant
	if b.EnPassantSquare >= 0 {
		fen.WriteString(" " + IndexToSqaureName(b.EnPassantSquare))
	} else {
		fen.WriteString(" -")
	}

	// Halfmove clock (for 50-move rule)
	fen.WriteString(fmt.Sprintf(" %d", b.HalfmoveClock))

	// Fullmove number
	fen.WriteString(fmt.Sprintf(" %d", b.FullmoveNumber))

	return fen.String()
}

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

// ExtractFinalFEN reads the PGN text and returns the FEN from the [CurrentPosition "..."] header
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

// Test a full game from PGN
// func TestPGNGame(t *testing.T) {
// 	pgn := `
// [Event "Live Chess"]
// [Site "Chess.com"]
// [Date "2025.12.23"]
// [Round "-"]
// [White "Cheer_Down"]
// [Black "Chessbard1972"]
// [Result "1-0"]
// [Tournament "https://www.chess.com/tournament/live/titled-tuesday-blitz-december-23-2025-6110589"]
// [CurrentPosition "8/8/6P1/7k/5K1N/b7/8/8 b - - 0 95"]
// [Timezone "UTC"]
// [ECO "D30"]
// [ECOUrl "https://www.chess.com/openings/Queens-Gambit-Declined-Pseudo-Tarrasch-Defense-4.e3"]
// [UTCDate "2025.12.23"]
// [UTCTime "16:11:07"]
// [WhiteElo "2649"]
// [BlackElo "2416"]
// [TimeControl "300"]
// [Termination "Cheer_Down won on time"]
// [StartTime "16:11:07"]
// [EndDate "2025.12.23"]
// [EndTime "16:21:15"]
// [Link "https://www.chess.com/analysis/game/live/160819629725/analysis?move=188"]
// [WhiteUrl "https://images.chesscomfiles.com/uploads/v1/user/49542020.3c660b52.50x50o.4b101c77be84.jpeg"]
// [WhiteCountry "75"]
// [WhiteTitle "CM"]
// [BlackUrl "https://images.chesscomfiles.com/uploads/v1/user/73534598.42e69e19.50x50o.8758ab788452.jpeg"]
// [BlackCountry "54"]
// [BlackTitle "FM"]

// 1. d4 d5 2. c4 e6 3. Nf3 c5 4. e3 Nf6 5. a3 a6 6. dxc5 Bxc5 7. b4 Ba7 8. Bb2 O-O
// 9. Nbd2 Qe7 10. Qb3 Rd8 11. Be2 Nc6 12. O-O dxc4 13. Nxc4 b5 14. Nce5 Bb7 15.
// Rac1 Rac8 16. Nxc6 Bxc6 17. Be5 Bd5 18. Qb2 Ne8 19. h3 f6 20. Bd4 Bb8 21. Rxc8
// Rxc8 22. Rc1 Rd8 23. Bd1 Nd6 24. Bc5 Qc7 25. Bxd6 Qxd6 26. g3 Ba7 27. Qc3 Bc4
// 28. Bb3 Bxb3 29. Qxb3 Kf7 30. Qc2 Qd5 31. Kg2 h5 32. Qh7 Bb6 33. Rc2 a5 34. Rd2
// Qf5 35. Qxf5 exf5 36. Rxd8 Bxd8 37. Nd4 axb4 38. axb4 Be7 39. Nxf5 Bxb4 40. Nd4
// Bc5 41. Nxb5 Ke6 42. Nc3 f5 43. Kf1 g5 44. Ke2 Ke5 45. Nb1 f4 46. exf4+ gxf4 47.
// g4 hxg4 48. hxg4 Ke4 49. Nd2+ Ke5 50. Kf3 Ba7 51. Ne4 Bb8 52. Ng5 Kf6 53. Nh3
// Kg6 54. Nxf4+ Kg5 55. Nh3+ Kh4 56. Ng1 Ba7 57. Ne2 Bxf2 58. Nf4 Ba7 59. Ng2+ Kg5
// 60. Kg3 Bb8+ 61. Kf3 Ba7 62. Ne1 Bb8 63. Nd3 Bd6 64. Nf2 Bc7 65. Ne4+ Kg6 66.
// Kg2 Bb8 67. Kf1 Be5 68. Ke2 Bb8 69. Kd3 Ba7 70. Kc4 Be3 71. Kd5 Bc1 72. Ke6 Be3
// 73. Kd7 Bc1 74. Kc6 Be3 75. Kb5 Bc1 76. Kc4 Be3 77. Kc3 Bc1 78. Kd3 Bg5 79. Ke2
// Bf4 80. Kf1 Bg5 81. Kg2 Bh6 82. Kh3 Bg5 83. Nf2 Bc1 84. Nd3 Bg5 85. Ne5+ Kf6 86.
// Nf3 Kg6 87. Kg2 Bf4 88. Kf2 Bc1 89. Ke2 Ba3 90. Kd3 Kf6 91. Ke4 Kg6 92. Kf4 Kf6
// 93. g5+ Kg6 94. Nh4+ Kh5 95. g6 1-0`

// 	// --- Parse PGN ---
// 	games, err := ParsePGNReader(strings.NewReader(pgn))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if len(games) != 1 {
// 		t.Fatalf("Expected 1 game, got %d", len(games))
// 	}

// 	game := games[0]
// 	board := NewStartingPosition()

// 	// --- Replay moves ---
// 	for i, san := range game.Moves {
// 		move, err := SANToMove(&board, san)
// 		if err != nil {
// 			t.Fatalf("Move %d (%s) failed: %v", i+1, san, err)
// 		}
// 		MakeMove(&board, move.From, move.To, move.Promotion, false)
// 		fen := board.ToFEN() // assuming you have a board.FEN() method
// 		t.Logf("Move %d: %s -> FEN: %s", i+1, san, fen)
// 	}

// 	// --- Compare final FEN (ignore clocks) ---
// 	expectedFEN := "8/1p4k1/8/8/8/2q5/1r6/3K4 w - - 18 62"

// 	engineParts := strings.Fields(board.ToFEN())
// 	expectedParts := strings.Fields(expectedFEN)

// 	engineCore := strings.Join(engineParts[:4], " ")
// 	expectedCore := strings.Join(expectedParts[:4], " ")

// 	if engineCore != expectedCore {
// 		t.Fatalf(
// 			"Final FEN mismatch\nExpected: %s\nGot:      %s",
// 			expectedCore,
// 			engineCore,
// 		)
// 	}

// 	t.Log("PGN replay successful and final FEN matches Chess.com")
// }
