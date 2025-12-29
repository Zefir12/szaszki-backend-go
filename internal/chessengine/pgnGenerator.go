package chess

import (
	"fmt"
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

// Example usage in GameSession.saveGame():
/*
func (g *GameSession) saveGame() {
	// Determine result
	result := "1-0" // White wins
	if winner == Black {
		result = "0-1"
	} else if isDraw {
		result = "1/2-1/2"
	}

	// Get initial cards (store these at game start)
	initialWhiteCards := []int8{6, 6, 6, 5, 5} // Store these when game starts
	initialBlackCards := []int8{6, 6, 6, 5, 5}

	// Create PGN generator
	pgnGen := chess.NewPGNGenerator(
		fmt.Sprintf("Player_%d", g.Players[0].UserID),
		fmt.Sprintf("Player_%d", g.Players[1].UserID),
	)

	// Set time control based on game settings
	totalMinutes := int(DefaultWhiteTime.Minutes())
	pgnGen.TimeControl = fmt.Sprintf("%d+0", totalMinutes*60)
	pgnGen.Result = result

	// Generate PGN
	pgn := pgnGen.GeneratePGN(
		g.MoveHistory,
		g.TimeHistory,
		g.CardHistory,
		initialWhiteCards,
		initialBlackCards,
	)

	// Convert to protobuf format
	var moveHistoryProto []*pb.Move
	for _, move := range g.MoveHistory {
		moveHistoryProto = append(moveHistoryProto, &pb.Move{
			From:      int32(move.From),
			To:        int32(move.To),
			Promotion: int32(move.Promotion),
		})
	}

	gameState := &pb.GameState{
		MoveHistory: moveHistoryProto,
	}

	_, err := grpc.SaveGame(g.ID, g.Players[0].UserID, g.Players[1].UserID, gameState, pgn)
	if err != nil {
		logger.Log.Warn().Err(err).Uint32("gameId", g.ID).Msg("Failed to save game")
	}
}
*/
