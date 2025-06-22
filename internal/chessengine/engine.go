package chess

import (
	"math/bits"
	"math/rand"
	"time"
)

//https://en.wikipedia.org/wiki/Bitboard

type Bitboard uint64

const (
	White = 0
	Black = 1
)

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

// for zobrist
const (
	Pawn = iota
	Knight
	Bishop
	Rook
	Queen
	King
)

var zobristTable [6][64]uint64
var zobristEnPassant [64]uint64
var zobristSideToMove uint64
var rng *rand.Rand

// === Board Representation ===
type Board struct {
	Pawns, Knights, Bishops, Rooks, Queens, Kings [2]Bitboard
	Occupied                                      [2]Bitboard
	AllOccupied                                   Bitboard
	EnPassantSquare                               int8 // -1 if none
	SideToMove                                    int8
	Hash                                          uint64
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
		initZobrist() // ensure zobrist is ready
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

var directions = map[string][]int{
	"rook":   {8, -8, 1, -1},
	"bishop": {9, -9, 7, -7},
}

func NewStartingPosition() Board {
	var b Board

	// White pieces
	b.Pawns[White] = 0x000000000000FF00
	b.Rooks[White] = 0x0000000000000081
	b.Knights[White] = 0x0000000000000042
	b.Bishops[White] = 0x0000000000000024
	b.Queens[White] = 0x0000000000000008
	b.Kings[White] = 0x0000000000000010

	// Black pieces
	b.Pawns[Black] = 0x00FF000000000000
	b.Rooks[Black] = 0x8100000000000000
	b.Knights[Black] = 0x4200000000000000
	b.Bishops[Black] = 0x2400000000000000
	b.Queens[Black] = 0x0800000000000000
	b.Kings[Black] = 0x1000000000000000

	// Occupied bitboards
	b.Occupied[White] = b.Pawns[White] | b.Rooks[White] | b.Knights[White] | b.Bishops[White] | b.Queens[White] | b.Kings[White]
	b.Occupied[Black] = b.Pawns[Black] | b.Rooks[Black] | b.Knights[Black] | b.Bishops[Black] | b.Queens[Black] | b.Kings[Black]

	b.AllOccupied = b.Occupied[White] | b.Occupied[Black]

	b.EnPassantSquare = -1
	b.Hash = ComputeHash(&b)

	return b
}

func slidingAttacks(sq int, occupied Bitboard, deltas []int) Bitboard {
	var attacks Bitboard
	for _, d := range deltas {
		curr := sq
		for {
			curr += d
			if curr < 0 || curr >= 64 || isEdgeCrossed(sq, curr, d) {
				break
			}
			attacks |= Bitboard(1) << curr
			if (occupied & (Bitboard(1) << curr)) != 0 {
				break
			}
		}
	}
	return attacks
}

func isEdgeCrossed(from, to, delta int) bool {
	fx, fy := from%8, from/8
	tx, ty := to%8, to/8
	dx, dy := abs(tx-fx), abs(ty-fy)
	return dx > 1 && dy > 1
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// === Legal Move Filtering ===
func IsMoveLegal(board *Board, from, to int8) bool {
	temp := *board // copy board
	MakeMove(&temp, from, to)
	kingSq := bits.TrailingZeros64(uint64(temp.Kings[board.SideToMove]))
	return !IsSquareAttacked(kingSq, &temp, 1-board.SideToMove)
}

func IsSquareAttacked(sq int, board *Board, attackerColor int8) bool {
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
	if slidingAttacks(sq, board.AllOccupied, directions["bishop"])&
		(board.Bishops[attackerColor]|board.Queens[attackerColor]) != 0 {
		return true
	}
	if slidingAttacks(sq, board.AllOccupied, directions["rook"])&
		(board.Rooks[attackerColor]|board.Queens[attackerColor]) != 0 {
		return true
	}
	return false
}

// === Make Move (With En Passant + Zobrist) ===
func MakeMove(board *Board, from, to int8) {
	fromBB := Bitboard(1) << from
	toBB := Bitboard(1) << to
	color := board.SideToMove

	// Clear destination from enemy
	board.Pawns[1-color] &^= toBB
	board.Knights[1-color] &^= toBB
	board.Bishops[1-color] &^= toBB
	board.Rooks[1-color] &^= toBB
	board.Queens[1-color] &^= toBB
	board.Kings[1-color] &^= toBB

	// En passant capture
	if board.EnPassantSquare == to && (board.Pawns[color]&fromBB) != 0 {
		capSq := to + 8
		if color == White {
			capSq = to - 8
		}
		board.Pawns[1-color] &^= Bitboard(1) << capSq
	}

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

	// Set en passant target if double push
	board.EnPassantSquare = -1
	if (board.Pawns[color]&toBB) != 0 && abs(int(to-from)) == 16 {
		board.EnPassantSquare = (from + to) / 2
	}

	// Update occupancy
	board.Occupied[color] &^= fromBB
	board.Occupied[color] |= toBB
	board.Occupied[1-color] &^= toBB
	board.AllOccupied = board.Occupied[0] | board.Occupied[1]

	// Update Zobrist
	board.Hash = ComputeHash(board)
}

func initZobrist() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	for pieceType := 0; pieceType < 6; pieceType++ {
		for sq := 0; sq < 64; sq++ {
			zobristTable[pieceType][sq] = rng.Uint64()
		}
	}
	for sq := 0; sq < 64; sq++ {
		zobristEnPassant[sq] = rng.Uint64()
	}
	zobristSideToMove = rng.Uint64()
}

// ComputeHash calculates the zobrist hash of the board.
// colorToMove is 0 for White, 1 for Black.
func ComputeHash(b *Board) uint64 {
	var hash uint64 = 0

	// Add pieces for both colors
	for color := 0; color < 2; color++ {
		pieces := [6]Bitboard{
			b.Pawns[color],
			b.Knights[color],
			b.Bishops[color],
			b.Rooks[color],
			b.Queens[color],
			b.Kings[color],
		}
		for pieceType, bb := range pieces {
			for bb != 0 {
				sq := PopLSB(&bb)
				hash ^= zobristTable[pieceType][sq]
			}
		}
	}

	// Add en passant if present
	if b.EnPassantSquare >= 0 && b.EnPassantSquare < 64 {
		hash ^= zobristEnPassant[b.EnPassantSquare]
	}

	// Add side to move
	if b.SideToMove == Black {
		hash ^= zobristSideToMove
	}

	return hash
}
