package chess

import "math/rand"

func InitCardsWithDuplicates() ([]int8, []int8) {
	whiteCards := make([]int8, 5)
	blackCards := make([]int8, 5)

	// for i := 0; i < 5; i++ {
	// 	whiteCards[i] = int8(rand.Intn(6)) + 1 // 1..6
	// 	blackCards[i] = int8(rand.Intn(6)) + 1 // 1..6
	// }

	whiteCards = []int8{6, 6, 6, 5, 5}
	blackCards = []int8{6, 6, 6, 5, 5}

	return whiteCards, blackCards
}

func GetRandomCard() int8 {
	return int8(rand.Intn(6)) + 1
}

func GetRandomValidCard(board *Board, color int8) int8 {
	// All pieces flag
	allPieces := CanPawn | CanKnight | CanBishop | CanRook | CanQueen | CanKing

	// Get all pieces that can legally move for this color
	movers := board.GeAllPiecesThatCanMoveLegallyThisTurn(color, allPieces)

	// If no legal moves, fallback to random card
	if movers == 0 {
		return GetRandomCard()
	}

	// Collect all set bits (piece types)
	var available []int8
	for i := range uint8(6) { // 0=Pawn, 1=Knight, ..., 5=King
		if movers&(1<<i) != 0 {
			available = append(available, int8(i))
		}
	}

	// Pick a random piece type
	enginePiece := available[rand.Intn(len(available))]

	// Convert back to card
	return EnginePieceToCard(enginePiece)
}

func EnginePieceToCard(piece int8) int8 {
	switch piece {
	case 0:
		return 6
	case 1:
		return 5
	case 2:
		return 4
	case 3:
		return 3
	case 4:
		return 2
	case 5:
		return 1
	default:
		return -1
	}
}

func CardToEnginePiece(card int8) int {
	switch card {
	case 6:
		return Pawn
	case 5:
		return Knight
	case 4:
		return Bishop
	case 3:
		return Rook
	case 2:
		return Queen
	case 1:
		return King
	default:
		return -1
	}
}

func CardName(card int8) string {
	switch card {
	case 6:
		return "Pawn"
	case 5:
		return "Knight"
	case 4:
		return "Bishop"
	case 3:
		return "Rook"
	case 2:
		return "Queen"
	case 1:
		return "King"
	default:
		return "Unknown"
	}
}

func CardListToString(cards []int8) string {
	if len(cards) == 0 {
		return "[]"
	}
	out := "["
	for i, c := range cards {
		if i > 0 {
			out += ", "
		}
		out += CardName(c)
	}
	out += "]"
	return out
}
