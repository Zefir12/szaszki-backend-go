// Highly Optimized Bitboard-Based Chess Engine Core in Go
package chess

import (
	"math/bits"
)

// === Bitboard Types ===
type Bitboard uint64

const (
	White = 0
	Black = 1
)

// === Constants ===
const (
	fileA Bitboard = 0x0101010101010101
	fileB Bitboard = 0x0202020202020202
	fileC Bitboard = 0x0404040404040404
	fileD Bitboard = 0x0808080808080808
	fileE Bitboard = 0x1010101010101010
	fileF Bitboard = 0x2020202020202020
	fileG Bitboard = 0x4040404040404040
	fileH Bitboard = 0x8080808080808080

	rank1 Bitboard = 0x00000000000000FF
	rank2 Bitboard = 0x000000000000FF00
	rank3 Bitboard = 0x0000000000FF0000
	rank4 Bitboard = 0x00000000FF000000
	rank5 Bitboard = 0x000000FF00000000
	rank6 Bitboard = 0x0000FF0000000000
	rank7 Bitboard = 0x00FF000000000000
	rank8 Bitboard = 0xFF00000000000000
)

// === Board Representation ===
type Board struct {
	Pawns, Knights, Bishops, Rooks, Queens, Kings [2]Bitboard
	Occupied                                      [2]Bitboard
	AllOccupied                                   Bitboard
}

// === Bitboard Helpers ===
func PopLSB(bb *Bitboard) int {
	lsb := *bb & -*bb
	index := bits.TrailingZeros64(uint64(lsb))
	*bb &= *bb - 1
	return index
}

func CountBits(bb Bitboard) int {
	return bits.OnesCount64(uint64(bb))
}

// === Pawn Moves ===
func SinglePawnPush(pawns, empty Bitboard, isWhite bool) Bitboard {
	if isWhite {
		return (pawns << 8) & empty
	}
	return (pawns >> 8) & empty
}

func DoublePawnPush(pawns, empty Bitboard, isWhite bool) Bitboard {
	if isWhite {
		single := (pawns << 8) & empty
		return (single << 8) & empty & rank4
	}
	single := (pawns >> 8) & empty
	return (single >> 8) & empty & rank5
}

func PawnAttacks(pawns Bitboard, enemy Bitboard, isWhite bool) Bitboard {
	if isWhite {
		left := (pawns << 7) & ^fileH & enemy
		right := (pawns << 9) & ^fileA & enemy
		return left | right
	}
	left := (pawns >> 9) & ^fileH & enemy
	right := (pawns >> 7) & ^fileA & enemy
	return left | right
}

// === King & Knight Moves ===
var knightMoves [64]Bitboard
var kingMoves [64]Bitboard

func init() {
	for i := 0; i < 64; i++ {
		knightMoves[i] = generateKnightMoves(i)
		kingMoves[i] = generateKingMoves(i)
	}
}

func generateKnightMoves(sq int) Bitboard {
	pos := Bitboard(1) << sq
	var moves Bitboard
	moves |= (pos << 17) & ^fileA
	moves |= (pos << 15) & ^fileH
	moves |= (pos << 10) & ^(fileA | fileB)
	moves |= (pos << 6) & ^(fileH | fileG)
	moves |= (pos >> 17) & ^fileH
	moves |= (pos >> 15) & ^fileA
	moves |= (pos >> 10) & ^(fileH | fileG)
	moves |= (pos >> 6) & ^(fileA | fileB)
	return moves
}

func generateKingMoves(sq int) Bitboard {
	pos := Bitboard(1) << sq
	var moves Bitboard
	moves |= (pos << 8)
	moves |= (pos >> 8)
	moves |= (pos << 1) & ^fileA
	moves |= (pos >> 1) & ^fileH
	moves |= (pos << 9) & ^fileA
	moves |= (pos << 7) & ^fileH
	moves |= (pos >> 7) & ^fileA
	moves |= (pos >> 9) & ^fileH
	return moves
}

// === Legal Move Filtering ===
func IsMoveLegal(board *Board, from, to int, color int) bool {
	temp := *board // copy board
	MakeMove(&temp, from, to, color)
	kingSq := bits.TrailingZeros64(uint64(temp.Kings[color]))
	return !IsSquareAttacked(kingSq, &temp, 1-color)
}

func IsSquareAttacked(sq int, board *Board, attackerColor int) bool {
	mask := Bitboard(1) << sq
	if knightMoves[sq]&board.Knights[attackerColor] != 0 {
		return true
	}
	if kingMoves[sq]&board.Kings[attackerColor] != 0 {
		return true
	}
	if PawnAttacks(mask, board.Pawns[attackerColor], attackerColor == Black) != 0 {
		return true
	}
	// TODO: Add bishop/rook/queen sliding checks here
	return false
}

// === Make Move (Full Implementation) ===
func MakeMove(board *Board, from, to int, color int) {
	fromBB := Bitboard(1) << from
	toBB := Bitboard(1) << to

	// Clear destination from enemy
	board.Pawns[1-color] &^= toBB
	board.Knights[1-color] &^= toBB
	board.Bishops[1-color] &^= toBB
	board.Rooks[1-color] &^= toBB
	board.Queens[1-color] &^= toBB
	board.Kings[1-color] &^= toBB

	// Move piece from -> to
	pieceLists := []*Bitboard{
		&board.Pawns[color], &board.Knights[color], &board.Bishops[color],
		&board.Rooks[color], &board.Queens[color], &board.Kings[color],
	}

	for _, piece := range pieceLists {
		if *piece&fromBB != 0 {
			*piece &^= fromBB
			*piece |= toBB
			break
		}
	}

	// Update occupancy
	board.Occupied[color] &^= fromBB
	board.Occupied[color] |= toBB
	board.Occupied[1-color] &^= toBB
	board.AllOccupied = board.Occupied[0] | board.Occupied[1]
}
