package chess

import (
	"bufio"
	"fmt"
	"io"
	"log"
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
					log.Printf("Stored Pawn for color %d at square %d (%s)", color, square, IndexToSqaureName(int8(square)))
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

	fmt.Printf("SAN: %s\n", san)
	fmt.Printf("Piece: %d, Color: %d, Target square: %d\n", piece, color, to)
	fmt.Printf("Disambiguation: file=%d, rank=%d\n", disFile, disRank)
	fmt.Printf("Bitboard: %064b\n", bb)

	// First pass: strict SAN (with disambiguation)
	bbCopy := bb
	for bbCopy != 0 {
		from := int8(PopLSB(&bbCopy))

		if piece == Pawn {
			// Only consider pawns that can legally move forward
			if disFile != -1 && from%8 != disFile {
				continue
			}
			targetRank := int8(rank)
			if color == White && from/8 >= targetRank {
				continue
			}
			if color == Black && from/8 <= targetRank {
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

		fmt.Printf("Checking candidate from %d -> %d\n", from, to)
		if IsMoveLegal(board, from, to, promotion) {
			fmt.Printf("Move allowed: %d -> %d\n", from, to)
			return Move{From: from, To: to, Promotion: promotion}, nil
		} else {
			fmt.Printf("Move blocked: %d -> %d\n", from, to)
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

// Test a full game from PGN
func TestPGNGame(t *testing.T) {
	pgn := `
[Event "Live Chess"]
[Site "Chess.com"]
[Date "2025.12.23"]
[Round "-"]
[White "Chris_Chess198"]
[Black "zefirqa"]
[Result "1/2-1/2"]
[CurrentPosition "8/1p4k1/8/8/8/2q5/1r6/3K4 w - - 18 62"]
[Timezone "UTC"]
[ECO "C22"]
[ECOUrl "https://www.chess.com/openings/Center-Game-Accepted-Normal-Variation"]
[UTCDate "2025.12.23"]
[UTCTime "14:23:22"]
[WhiteElo "748"]
[BlackElo "732"]
[TimeControl "180"]
[Termination "Game drawn by stalemate"]
[StartTime "14:23:22"]
[EndDate "2025.12.23"]
[EndTime "14:29:50"]
[Link "https://www.chess.com/analysis/game/live/147079102508/analysis"]
[WhiteUrl "https://images.chesscomfiles.com/uploads/v1/user/76940020.0643e98e.50x50o.da031ac84993.jpeg"]
[WhiteCountry "2"]
[WhiteTitle ""]
[BlackUrl "https://images.chesscomfiles.com/uploads/v1/user/291321115.6693bc46.50x50o.2d714e92cdad.jpg"]
[BlackCountry "116"]
[BlackTitle ""]

1. e4 e5 2. d4 exd4 3. Qxd4 Nc6 4. Qd1 Nf6 5. Nc3 Bb4 6. Bd3 O-O 7. Bd2 Re8 8.
f3 d5 9. Nge2 dxe4 10. Nxe4 Bxd2+ 11. Qxd2 Bf5 12. Nxf6+ Qxf6 13. O-O Rad8 14.
Rad1 Bxd3 15. cxd3 Ne5 16. d4 Nc4 17. Qc3 Ne3 18. d5 Nxf1 19. Qxf6 gxf6 20. Kxf1
Re5 21. Nf4 c6 22. d6 Kf8 23. b4 f5 24. a3 f6 25. Nh5 Kf7 26. f4 Rd5 27. Rxd5
cxd5 28. Kf2 Rxd6 29. Kg3 d4 30. Kh4 d3 31. Ng3 d2 32. Nxf5 Rd5 33. Ne3 Rd4 34.
Kg3 d1=Q 35. Nxd1 Rxd1 36. Kg4 Ra1 37. g3 Kg6 38. h4 Rxa3 39. h5+ Kg7 40. Kf5
Ra4 41. h6+ Kg8 42. Kxf6 Rxb4 43. f5 a5 44. g4 a4 45. g5 a3 46. g6 hxg6 47. fxg6
Rb6+ 48. Kf5 a2 49. h7+ Kg7 50. h8=Q+ Kxh8 51. g7+ Kxg7 52. Ke5 a1=Q+ 53. Kf5
Qf6+ 54. Ke4 Rb5 55. Ke3 Qe5+ 56. Kf3 Rb4 57. Kf2 Qf4+ 58. Ke2 Rb3 59. Kd1 Qe3
60. Kc2 Qc3+ 61. Kd1 Rb2 1/2-1/2`

	// --- Parse PGN ---
	games, err := ParsePGNReader(strings.NewReader(pgn))
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(games))
	}

	game := games[0]
	board := NewStartingPosition()

	// --- Replay moves ---
	for i, san := range game.Moves {
		move, err := SANToMove(&board, san)
		if err != nil {
			t.Fatalf("Move %d (%s) failed: %v", i+1, san, err)
		}
		MakeMove(&board, move.From, move.To, move.Promotion, false)
		fen := board.ToFEN() // assuming you have a board.FEN() method
		t.Logf("Move %d: %s -> FEN: %s", i+1, san, fen)
	}

	// --- Compare final FEN (ignore clocks) ---
	expectedFEN := "8/1p4k1/8/8/8/2q5/1r6/3K4 w - - 18 62"

	engineParts := strings.Fields(board.ToFEN())
	expectedParts := strings.Fields(expectedFEN)

	engineCore := strings.Join(engineParts[:4], " ")
	expectedCore := strings.Join(expectedParts[:4], " ")

	if engineCore != expectedCore {
		t.Fatalf(
			"Final FEN mismatch\nExpected: %s\nGot:      %s",
			expectedCore,
			engineCore,
		)
	}

	t.Log("PGN replay successful and final FEN matches Chess.com")
}
