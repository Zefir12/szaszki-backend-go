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
	movers := board.GetAllPiecesThatCanMoveThisTurn(color, false)
	if len(movers) == 0 {
		return GetRandomCard() // fallback
	}

	// Deduplicate piece types
	uniquePieces := make(map[int8]struct{})
	for _, p := range movers {
		uniquePieces[p] = struct{}{}
	}

	// Convert map keys to slice
	var pieceTypes []int8
	for p := range uniquePieces {
		pieceTypes = append(pieceTypes, p)
	}

	// Pick a random piece type
	enginePiece := pieceTypes[rand.Intn(len(pieceTypes))]

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
