package chess

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// PGNGenerator handles conversion of game data to PGN format
type PGNGenerator struct {
	WhitePlayer string
	BlackPlayer string
	Event       string
	Site        string
	Date        string
	Result      string
	TimeControl string
}

// GeneratePGN creates a PGN string from move history with times and cards
func (pg *PGNGenerator) GeneratePGN(
	moveHistory []Move,
	timeHistory []time.Duration,
	cardHistory [][]CardSwap,
	whiteCards []int8,
	blackCards []int8,
) string {
	var pgn strings.Builder

	// Write PGN headers
	pgn.WriteString(fmt.Sprintf("[Event \"%s\"]\n", pg.Event))
	pgn.WriteString(fmt.Sprintf("[Site \"%s\"]\n", pg.Site))
	pgn.WriteString(fmt.Sprintf("[Date \"%s\"]\n", pg.Date))
	pgn.WriteString(fmt.Sprintf("[White \"%s\"]\n", pg.WhitePlayer))
	pgn.WriteString(fmt.Sprintf("[Black \"%s\"]\n", pg.BlackPlayer))
	pgn.WriteString(fmt.Sprintf("[Result \"%s\"]\n", pg.Result))

	if pg.TimeControl != "" {
		pgn.WriteString(fmt.Sprintf("[TimeControl \"%s\"]\n", pg.TimeControl))
	}

	// Custom headers for card chess variant
	pgn.WriteString("[Variant \"From Position\"]\n")
	pgn.WriteString("[FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1\"]\n")

	// Add initial card state as a custom header
	pgn.WriteString(fmt.Sprintf("[WhiteInitialCards \"%s\"]\n", formatCardsCompact(whiteCards)))
	pgn.WriteString(fmt.Sprintf("[BlackInitialCards \"%s\"]\n", formatCardsCompact(blackCards)))

	pgn.WriteString("\n")

	// Generate the move list
	board := NewStartingPosition()
	moveNumber := 1

	// Track current card state for both players
	currentWhiteCards := make([]int8, len(whiteCards))
	currentBlackCards := make([]int8, len(blackCards))
	copy(currentWhiteCards, whiteCards)
	copy(currentBlackCards, blackCards)

	for i, move := range moveHistory {
		isWhiteMove := (i % 2) == 0

		// Add move number for white's moves
		if isWhiteMove {
			pgn.WriteString(fmt.Sprintf("%d. ", moveNumber))
		}

		// Generate algebraic notation for the move
		algebraic := generateAlgebraicNotation(&board, move)
		pgn.WriteString(algebraic)

		// Add clock time if available
		if i < len(timeHistory) {
			clockStr := formatClockTime(timeHistory[i])
			pgn.WriteString(fmt.Sprintf(" { [%%clk %s]", clockStr))

			// Add card swaps if available
			if i < len(cardHistory) && len(cardHistory[i]) > 0 {
				// Get the current player's cards to show old values
				currentCards := currentBlackCards
				if isWhiteMove {
					currentCards = currentWhiteCards
				}

				cardInfo := formatCardSwapsWithOldValues(cardHistory[i], currentCards)
				pgn.WriteString(fmt.Sprintf(" Cards: %s", cardInfo))

				// Update the card state
				for _, swap := range cardHistory[i] {
					if int(swap.IndexReplaced) < len(currentCards) {
						currentCards[swap.IndexReplaced] = swap.NewCard
					}
				}
			}

			pgn.WriteString(" }")
		}

		pgn.WriteString(" ")

		// Make the move on our tracking board
		MakeMove(&board, move.From, move.To, move.Promotion, false)
		board.FlipSideToMove()

		// Increment move number after black's move
		if !isWhiteMove {
			moveNumber++
		}
	}

	// Add result
	pgn.WriteString(pg.Result)
	pgn.WriteString("\n")

	return pgn.String()
}

// generateAlgebraicNotation converts a move to standard algebraic notation (SAN)
func generateAlgebraicNotation(board *Board, move Move) string {
	piece := board.GetPieceType(move.From, board.SideToMove())

	// Get destination square name
	destSquare := IndexToSqaureName(move.To)

	// Check if it's a capture
	enemyColor := 1 - board.SideToMove()
	isCapture := (board.Occupied[enemyColor] & (Bitboard(1) << move.To)) != 0

	// Check for en passant
	isEnPassant := piece == Pawn && move.To == board.EnPassantSquare && board.EnPassantSquare >= 0
	if isEnPassant {
		isCapture = true
	}

	// Handle castling
	if piece == King && abs(int(move.To-move.From)) == 2 {
		if move.To > move.From {
			return "O-O" // Kingside
		}
		return "O-O-O" // Queenside
	}

	var notation strings.Builder

	// Add piece letter (except for pawns)
	if piece != Pawn {
		notation.WriteString(getPieceLetter(piece))
	}

	// Add disambiguation if needed
	disambiguation := getDisambiguation(board, move, piece)
	notation.WriteString(disambiguation)

	// Add capture indicator
	if isCapture {
		if piece == Pawn {
			// For pawn captures, include the file
			notation.WriteString(string(IndexToSqaureName(move.From)[0]))
		}
		notation.WriteString("x")
	}

	// Add destination square
	notation.WriteString(destSquare)

	// Add promotion
	if move.Promotion > 0 {
		notation.WriteString("=")
		notation.WriteString(getPieceLetter(int(move.Promotion)))
	}

	// Check for check/checkmate (would need to make the move and test)
	tempBoard := board.Clone()
	MakeMove(&tempBoard, move.From, move.To, move.Promotion, true)
	tempBoard.FlipSideToMove()

	if tempBoard.IsInCheck(tempBoard.SideToMove()) {
		if !tempBoard.HasAnyLegalMove(tempBoard.SideToMove()) {
			notation.WriteString("#") // Checkmate
		} else {
			notation.WriteString("+") // Check
		}
	}

	return notation.String()
}

// getDisambiguation determines if we need to add file/rank to disambiguate moves
func getDisambiguation(board *Board, move Move, piece int) string {
	if piece == Pawn || piece == King {
		return "" // Pawns and kings don't need disambiguation
	}

	color := board.SideToMove()
	//toBB := Bitboard(1) << move.To

	// Find all pieces of the same type that could move to the same square
	var candidates []int8
	var pieceBB Bitboard

	switch piece {
	case Knight:
		pieceBB = board.Knights[color]
	case Bishop:
		pieceBB = board.Bishops[color]
	case Rook:
		pieceBB = board.Rooks[color]
	case Queen:
		pieceBB = board.Queens[color]
	}

	for bb := pieceBB; bb != 0; {
		sq := int8(PopLSB(&bb))
		if sq != move.From && IsMoveLegal(board, sq, move.To, 0) {
			candidates = append(candidates, sq)
		}
	}

	if len(candidates) == 0 {
		return "" // No ambiguity
	}

	// Check if file is sufficient
	fromFile := move.From % 8
	needFile := false
	needRank := false

	for _, sq := range candidates {
		if sq%8 == fromFile {
			needRank = true
		} else {
			needFile = true
		}
	}

	if needRank && needFile {
		// Need both file and rank
		return IndexToSqaureName(move.From)
	} else if needRank {
		// Just rank
		return fmt.Sprintf("%d", (move.From/8)+1)
	} else {
		// Just file
		return string('a' + (move.From % 8))
	}
}

// getPieceLetter returns the standard letter for a piece type
func getPieceLetter(piece int) string {
	switch piece {
	case Knight:
		return "N"
	case Bishop:
		return "B"
	case Rook:
		return "R"
	case Queen:
		return "Q"
	case King:
		return "K"
	default:
		return ""
	}
}

// formatClockTime converts a duration to PGN clock format (H:MM:SS)
func formatClockTime(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// formatCardSwaps creates a human-readable string of card changes
// Note: This now just shows which position was replaced and what the new card is
// The old card value is tracked separately in the game state
func formatCardSwaps(swaps []CardSwap) string {
	if len(swaps) == 0 {
		return "none"
	}

	var parts []string
	for _, swap := range swaps {
		parts = append(parts, fmt.Sprintf("slot%d→%s",
			swap.IndexReplaced,
			CardName(swap.NewCard)))
	}
	return strings.Join(parts, ", ")
}

// formatCardSwapsWithOldValues creates a human-readable string showing old→new card changes
func formatCardSwapsWithOldValues(swaps []CardSwap, currentCards []int8) string {
	if len(swaps) == 0 {
		return "none"
	}

	var parts []string
	for _, swap := range swaps {
		oldCard := "Unknown"
		if int(swap.IndexReplaced) < len(currentCards) {
			oldCard = CardName(currentCards[swap.IndexReplaced])
		}
		parts = append(parts, fmt.Sprintf("%s→%s", oldCard, CardName(swap.NewCard)))
	}
	return strings.Join(parts, ", ")
}

// formatCardsCompact creates a compact representation of cards
func formatCardsCompact(cards []int8) string {
	var parts []string
	for _, card := range cards {
		switch card {
		case 1:
			parts = append(parts, "K")
		case 2:
			parts = append(parts, "Q")
		case 3:
			parts = append(parts, "R")
		case 4:
			parts = append(parts, "B")
		case 5:
			parts = append(parts, "N")
		case 6:
			parts = append(parts, "P")
		}
	}
	return strings.Join(parts, ",")
}

// NewPGNGenerator creates a new PGN generator with default values
func NewPGNGenerator(whitePlayer, blackPlayer string) *PGNGenerator {
	now := time.Now()
	return &PGNGenerator{
		WhitePlayer: whitePlayer,
		BlackPlayer: blackPlayer,
		Event:       "Casual Game",
		Site:        "Card Chess",
		Date:        now.Format("2006.01.02"),
		Result:      "*",     // Game in progress
		TimeControl: "600+0", // 10 minutes
	}
}

type PGNGame struct {
	Event  string
	White  string
	Black  string
	Result string
	Moves  []string
}

type PGNTest struct {
	Name     string
	PGN      string
	FinalFEN string
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
	// First, strip all comments { ... }
	text = StripPGNComments(text)

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
	text = strings.ReplaceAll(text, "*", "")

	tokens := strings.Fields(text)
	moves := []string{}

	for _, tok := range tokens {
		if strings.HasSuffix(tok, ".") {
			continue
		}
		if tok != "" {
			moves = append(moves, tok)
		}
	}

	return moves
}

// StripPGNComments removes comments {...} from PGN move text
func StripPGNComments(text string) string {
	var result strings.Builder
	inComment := false
	depth := 0

	for _, ch := range text {
		if ch == '{' {
			inComment = true
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				inComment = false
			}
		} else if !inComment {
			result.WriteRune(ch)
		}
	}

	return result.String()
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
