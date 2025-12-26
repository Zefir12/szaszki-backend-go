package chess

import (
	"fmt"
	"log"
	"math/bits"
	"strconv"
	"strings"
)

//https://en.wikipedia.org/wiki/Bitboard

type Bitboard uint64

const (
	White = 1
	Black = 0
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

var castlingRightsBySquare = [64]uint8{
	^WQ, 0xFF, 0xFF, 0xFF, ^(WK | WQ), 0xFF, 0xFF, ^WK, // rank 1 (White)
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // rank 2
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // rank 3
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // rank 4
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // rank 5
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // rank 6
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, // rank 7
	^BQ, 0xFF, 0xFF, 0xFF, ^(BK | BQ), 0xFF, 0xFF, ^BK, // rank 8 (Black)
}

type Move struct {
	From, To, Promotion int8
}

const (
	WK uint8 = 1 << 0 // White kingside
	WQ uint8 = 1 << 1 // White queenside
	BK uint8 = 1 << 2 // Black kingside
	BQ uint8 = 1 << 3 // Black queenside

	WhiteToMove uint8 = 1 << 4
)

var knightMoves [64]Bitboard
var kingMoves [64]Bitboard

var rookDeltas = [4]int{8, -8, 1, -1}
var bishopDeltas = [4]int{9, -9, 7, -7}

const (
	Pawn = iota
	Knight
	Bishop
	Rook
	Queen
	King
)

const (
	CanPawn   uint8 = 1 << iota // 1
	CanKnight                   // 2
	CanBishop                   // 4
	CanRook                     // 8
	CanQueen                    // 16
	CanKing                     // 32
)

// === Board Representation ===
type Board struct {
	Pawns, Knights, Bishops, Rooks, Queens, Kings [2]Bitboard
	Occupied                                      [2]Bitboard
	Hash                                          uint64
	EnPassantSquare                               int8
	Flags                                         uint8  // bitmask: 1 = WK, 2 = WQ, 4 = BK, 8 = BQ, 16 = WhiteToMove
	HalfmoveClock                                 uint8  // for 50-move rule
	FullmoveNumber                                uint16 // increments after black's move
}

func (b *Board) Clone() Board {
	return *b // shallow copy
}

// Helper method to get side to move
func (b *Board) SideToMove() uint8 {
	if b.Flags&WhiteToMove != 0 {
		return 0 // White
	}
	return 1 // Black
}

func init() {
	for i := range 64 {
		knightMoves[i] = generateKnightMoves(i)
		kingMoves[i] = generateKingMoves(i)
	}
}

func IndexToSqaureName(index int8) string {
	if index < 0 || index > 63 {
		return "??"
	}

	file := index % 8
	rank := (index / 8) + 1 // Fixed: rank 0 -> 1, rank 1 -> 2, etc.

	return fmt.Sprintf("%c%d", 'a'+file, rank)
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
	if enemy == 0 {
		return 0
	}
	if isWhite {
		left := (pawns << 7) & ^fileH
		right := (pawns << 9) & ^fileA
		return (left | right) & enemy
	}
	left := (pawns >> 9) & ^fileH
	right := (pawns >> 7) & ^fileA
	return (left | right) & enemy
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
	b.Occupied[White] = 0x000000000000FFFF // b.Pawns[White] | b.Rooks[White] | b.Knights[White] | b.Bishops[White] | b.Queens[White] | b.Kings[White]
	b.Occupied[Black] = 0xFFFF000000000000 // b.Pawns[Black] | b.Rooks[Black] | b.Knights[Black] | b.Bishops[Black] | b.Queens[Black] | b.Kings[Black]

	b.Flags |= WK | WQ | BK | BQ | WhiteToMove // enable all castling and set side to white

	b.EnPassantSquare = -1
	b.FullmoveNumber = 1

	return b
}

// Helper to convert square name to index
func SquareNameToIndex(name string) int8 {
	if len(name) != 2 {
		return -1
	}
	file := int8(name[0] - 'a')
	rank := int8(name[1] - '1')
	return rank*8 + file
}

func slidingAttacksRook(sq int, occupied Bitboard) Bitboard {
	var attacks Bitboard
	for _, d := range rookDeltas {
		for s := sq + d; s >= 0 && s < 64; s += d {
			if (d == 1 || d == -1) && s/8 != (s-d)/8 {
				break
			}
			attacks |= (1 << s)
			if (occupied & (1 << s)) != 0 {
				break
			}
		}
	}
	return attacks
}

func slidingAttacksBishop(sq int, occupied Bitboard) Bitboard {
	var attacks Bitboard
	for _, d := range bishopDeltas {
		for s := sq + d; s >= 0 && s < 64; s += d {
			if abs(s%8-((s-d)%8)) != 1 {
				break
			}
			attacks |= (1 << s)
			if (occupied & (1 << s)) != 0 {
				break
			}
		}
	}
	return attacks
}

func slidingAttacksQueen(sq int, occupied Bitboard) Bitboard {
	return slidingAttacksRook(sq, occupied) | slidingAttacksBishop(sq, occupied)
}

//go:inline
func abs(x int) int {
	mask := x >> 31
	return (x + mask) ^ mask
}

func (board *Board) GetPieceType(square int8, color int8) int {
	return GetPieceType(board, square, color)
}

func GetPieceType(board *Board, square int8, color int8) int {
	bb := Bitboard(1) << square
	if board.Pawns[color]&bb != 0 {
		return Pawn
	}
	if board.Knights[color]&bb != 0 {
		return Knight
	}
	if board.Bishops[color]&bb != 0 {
		return Bishop
	}
	if board.Rooks[color]&bb != 0 {
		return Rook
	}
	if board.Queens[color]&bb != 0 {
		return Queen
	}
	if board.Kings[color]&bb != 0 {
		return King
	}
	return -1 // No piece
}

func (b *Board) HasAnyLegalMove(color int8) bool {
	return b.GetAllPiecesThatCanMoveLegallyThisTurnCheckAll(color) != 0
}

func IsSquareAttacked(sq int, b *Board, attackerColor int8) bool {
	occ := b.Occupied[White] | b.Occupied[Black]
	target := Bitboard(1) << sq

	// === Pawn attacks (fast, no function call) ===
	if attackerColor == White {
		// White pawns attack upward (from rank-1 perspective)
		if ((target>>7)&^fileA)&b.Pawns[White] != 0 {
			return true
		}
		if ((target>>9)&^fileH)&b.Pawns[White] != 0 {
			return true
		}
	} else {
		// Black pawns attack downward
		if ((target<<7)&^fileH)&b.Pawns[Black] != 0 {
			return true
		}
		if ((target<<9)&^fileA)&b.Pawns[Black] != 0 {
			return true
		}
	}

	// === Knights ===
	if knightMoves[sq]&b.Knights[attackerColor] != 0 {
		return true
	}

	// === Kings ===
	if kingMoves[sq]&b.Kings[attackerColor] != 0 {
		return true
	}

	// === Bishops / Queens ===
	bq := b.Bishops[attackerColor] | b.Queens[attackerColor]
	if bq != 0 {
		if slidingAttacksBishop(sq, occ)&bq != 0 {
			return true
		}
	}

	// === Rooks / Queens ===
	rq := b.Rooks[attackerColor] | b.Queens[attackerColor]
	if rq != 0 {
		if slidingAttacksRook(sq, occ)&rq != 0 {
			return true
		}
	}

	return false
}

func (b *Board) IsInCheck(color int8) bool {
	kingSq := bits.TrailingZeros64(uint64(b.Kings[color]))
	return IsSquareAttacked(kingSq, b, 1-color)
}

func IsPseudoLegal(board *Board, from, to int8, piece int, color int8) bool {
	toBB := Bitboard(1) << to
	occupied := board.Occupied[White] | board.Occupied[Black]

	// Check if target square contains own piece (illegal for all pieces)
	if toBB&board.Occupied[color] != 0 {
		return false
	}

	switch piece {
	case Pawn:
		empty := ^occupied
		single := SinglePawnPush(Bitboard(1)<<from, empty, color == White)
		double := DoublePawnPush(Bitboard(1)<<from, empty, color == White)
		attacks := PawnAttacks(Bitboard(1)<<from, board.Occupied[1-color], color == White)

		// En passant
		var enPassant Bitboard
		if board.EnPassantSquare >= 0 {
			enPassantBB := Bitboard(1) << board.EnPassantSquare
			enPassant = PawnAttacks(Bitboard(1)<<from, enPassantBB, color == White)
		}

		return (single|double|attacks|enPassant)&toBB != 0

	case Knight:
		return knightMoves[from]&toBB != 0

	case Bishop:
		attacks := slidingAttacksBishop(int(from), occupied)
		return attacks&toBB != 0

	case Rook:
		attacks := slidingAttacksRook(int(from), occupied)
		return attacks&toBB != 0

	case Queen:
		attacks := slidingAttacksQueen(int(from), occupied)
		return attacks&toBB != 0

	case King:
		// Check normal king moves
		if kingMoves[from]&toBB != 0 {
			return true
		}
		// Check castling
		if abs(int(to-from)) == 2 {
			return CanCastle(board, color, to > from)
		}
		return false
	}

	return false
}

func IsMoveLegal(board *Board, from, to, promoteTo int8) bool {
	color := Black
	if board.Flags&WhiteToMove != 0 {
		color = White
	}

	piece := GetPieceType(board, from, int8(color))
	if piece == -1 {
		return false
	}

	// Basic pawn direction check
	if piece == Pawn {
		direction := to - from
		if color == White && direction <= 0 {
			return false // White pawns must move forward (positive direction)
		}
		if color == Black && direction >= 0 {
			return false // Black pawns must move backward (negative direction)
		}
	}

	// NEW: Check if the move is pseudo-legal for this piece type
	if !IsPseudoLegal(board, from, to, piece, int8(color)) {
		return false
	}

	temp := *board
	MakeMove(&temp, from, to, promoteTo, true)

	kingSq := bits.TrailingZeros64(uint64(temp.Kings[color]))
	enemyColor := 1 - color
	return !IsSquareAttacked(kingSq, &temp, int8(enemyColor))
}

func (b *Board) CanAnyPawnMove(color int8) bool {
	occupied := b.Occupied[White] | b.Occupied[Black]
	pawns := b.Pawns[color]
	enemy := b.Occupied[1-color]
	isWhite := color == White

	if single := SinglePawnPush(pawns, ^occupied, isWhite); single != 0 {
		return true
	}
	if attacks := PawnAttacks(pawns, enemy, isWhite); attacks != 0 {
		return true
	}
	if double := DoublePawnPush(pawns, ^occupied, isWhite); double != 0 {
		return true
	}
	if b.EnPassantSquare >= 0 {
		enPassantBB := Bitboard(1) << b.EnPassantSquare
		if enPassant := PawnAttacks(pawns, enPassantBB, isWhite); enPassant != 0 {
			return true
		}
	}
	return false
}

func (b *Board) CanAnyRookMove(color int8) bool {
	occupied := b.Occupied[White] | b.Occupied[Black]
	rooks := b.Rooks[color]

	for bb := rooks; bb != 0; {
		from := int(PopLSB(&bb)) // get the rook square
		// Compute rook attacks, excluding friendly pieces
		moves := slidingAttacksRook(from, occupied) & ^b.Occupied[color]

		// If any move is legal, return early
		if moves != 0 {
			return true
		}
	}
	return false
}

func (b *Board) CanAnyBishopMove(color int8) bool {
	occupied := b.Occupied[White] | b.Occupied[Black]
	bishops := b.Bishops[color]

	for bb := bishops; bb != 0; {
		from := int(PopLSB(&bb)) // get the ishop square
		// Compute rook attacks, excluding friendly pieces
		moves := slidingAttacksBishop(from, occupied) & ^b.Occupied[color]

		// If any move is legal, return early
		if moves != 0 {
			return true
		}
	}
	return false
}

func (b *Board) CanAnyQueenMove(color int8) bool {
	occupied := b.Occupied[White] | b.Occupied[Black]
	queens := b.Queens[color]

	for bb := queens; bb != 0; {
		from := int(PopLSB(&bb)) // get the ishop square
		// Compute queen attacks, excluding friendly pieces
		moves := slidingAttacksQueen(from, occupied) & ^b.Occupied[color]

		// If any move is legal, return early
		if moves != 0 {
			return true
		}
	}
	return false
}

func (b *Board) movePieceLight(from, to int8, color int8, piece int) (captured Bitboard) {
	fromBB := Bitboard(1) << from
	toBB := Bitboard(1) << to

	// detect capture
	captured = b.Occupied[1-color] & toBB

	// remove captured piece
	if captured != 0 {
		b.Occupied[1-color] &^= toBB
	}

	// move piece
	b.Occupied[color] &^= fromBB
	b.Occupied[color] |= toBB

	// pawn-specific
	switch piece {
	case Pawn:
		b.Pawns[color] &^= fromBB
		b.Pawns[color] |= toBB
	case Knight:
		b.Knights[color] &^= fromBB
		b.Knights[color] |= toBB
	case Rook:
		b.Rooks[color] &^= fromBB
		b.Rooks[color] |= toBB
	case King:
		b.Kings[color] &^= fromBB
		b.Kings[color] |= toBB
	case Queen:
		b.Queens[color] &^= fromBB
		b.Queens[color] |= toBB
	case Bishop:
		b.Bishops[color] &^= fromBB
		b.Bishops[color] |= toBB
	}

	return captured
}

func (b *Board) undoMovePieceLight(from, to int8, color int8, captured Bitboard, piece int) {
	fromBB := Bitboard(1) << from
	toBB := Bitboard(1) << to

	b.Occupied[color] &^= toBB
	b.Occupied[color] |= fromBB

	// pawn-specific
	switch piece {
	case Pawn:
		b.Pawns[color] &^= toBB
		b.Pawns[color] |= fromBB
	case Knight:
		b.Knights[color] &^= toBB
		b.Knights[color] |= fromBB
	case Rook:
		b.Rooks[color] &^= toBB
		b.Rooks[color] |= fromBB
	case King:
		b.Kings[color] &^= toBB
		b.Kings[color] |= fromBB
	case Queen:
		b.Queens[color] &^= toBB
		b.Queens[color] |= fromBB
	case Bishop:
		b.Bishops[color] &^= toBB
		b.Bishops[color] |= fromBB
	}

	if captured != 0 {
		b.Occupied[1-color] |= captured
	}
}

func (b *Board) CanAnyPawnMoveSafe(color int8) bool {
	occupied := b.Occupied[White] | b.Occupied[Black]
	pawns := b.Pawns[color]
	enemy := b.Occupied[1-color]
	isWhite := color == White
	enemyColor := 1 - color

	kingSq := int(bits.TrailingZeros64(uint64(b.Kings[color])))

	for bb := pawns; bb != 0; {
		from := int8(PopLSB(&bb))
		fromBB := Bitboard(1) << from

		// 1️⃣ Single push
		if moves := SinglePawnPush(fromBB, ^occupied, isWhite); moves != 0 {
			to := int8(bits.TrailingZeros64(uint64(moves)))
			cap := b.movePieceLight(from, to, color, Pawn)
			if !IsSquareAttacked(kingSq, b, int8(enemyColor)) {
				b.undoMovePieceLight(from, to, color, cap, Pawn)
				return true
			}
			b.undoMovePieceLight(from, to, color, cap, Pawn)
		}

		// 2️⃣ Captures
		if moves := PawnAttacks(fromBB, enemy, isWhite); moves != 0 {
			for m := moves; m != 0; {
				to := int8(PopLSB(&m))
				cap := b.movePieceLight(from, to, color, Pawn)
				if !IsSquareAttacked(kingSq, b, int8(enemyColor)) {
					b.undoMovePieceLight(from, to, color, cap, Pawn)
					return true
				}
				b.undoMovePieceLight(from, to, color, cap, Pawn)
			}
		}

		// 3️⃣ Double push
		if moves := DoublePawnPush(fromBB, ^occupied, isWhite); moves != 0 {
			to := int8(bits.TrailingZeros64(uint64(moves)))
			cap := b.movePieceLight(from, to, color, Pawn)
			if !IsSquareAttacked(kingSq, b, int8(enemyColor)) {
				b.undoMovePieceLight(from, to, color, cap, Pawn)
				return true
			}
			b.undoMovePieceLight(from, to, color, cap, Pawn)
		}

		// 4️⃣ En passant (CRITICAL: remove captured pawn!)
		if b.EnPassantSquare >= 0 {
			epBB := Bitboard(1) << b.EnPassantSquare
			if PawnAttacks(fromBB, epBB, isWhite) != 0 {
				capSq := int8(b.EnPassantSquare + map[bool]int8{true: -8, false: 8}[isWhite])
				capturedPawnBB := Bitboard(1) << capSq

				// simulate EP
				b.Pawns[color] &^= fromBB
				b.Pawns[color] |= epBB
				b.Occupied[color] &^= fromBB
				b.Occupied[color] |= epBB
				b.Pawns[enemyColor] &^= capturedPawnBB
				b.Occupied[enemyColor] &^= capturedPawnBB

				if !IsSquareAttacked(kingSq, b, int8(enemyColor)) {
					// undo EP
					b.Pawns[color] &^= epBB
					b.Pawns[color] |= fromBB
					b.Occupied[color] &^= epBB
					b.Occupied[color] |= fromBB
					b.Pawns[enemyColor] |= capturedPawnBB
					b.Occupied[enemyColor] |= capturedPawnBB
					return true
				}

				// undo EP
				b.Pawns[color] &^= epBB
				b.Pawns[color] |= fromBB
				b.Occupied[color] &^= epBB
				b.Occupied[color] |= fromBB
				b.Pawns[enemyColor] |= capturedPawnBB
				b.Occupied[enemyColor] |= capturedPawnBB
			}
		}
	}

	return false
}

func (b *Board) CanAnyRookMoveSafe(color int8) bool {
	occupied := b.Occupied[White] | b.Occupied[Black]
	enemyColor := 1 - color
	kingSq := int(bits.TrailingZeros64(uint64(b.Kings[color])))

	for bb := b.Rooks[color]; bb != 0; {
		from := int8(PopLSB(&bb))
		fromBB := Bitboard(1) << from

		moves := slidingAttacksRook(int(from), occupied) & ^b.Occupied[color]
		for m := moves; m != 0; {
			to := int8(PopLSB(&m))
			toBB := Bitboard(1) << to

			// move rook
			b.Rooks[color] &^= fromBB
			b.Rooks[color] |= toBB
			cap := b.movePieceLight(from, to, color, Rook)

			if !IsSquareAttacked(kingSq, b, enemyColor) {
				b.undoMovePieceLight(from, to, color, cap, Rook)
				b.Rooks[color] &^= toBB
				b.Rooks[color] |= fromBB
				return true
			}

			// undo
			b.undoMovePieceLight(from, to, color, cap, Rook)
			b.Rooks[color] &^= toBB
			b.Rooks[color] |= fromBB
		}
	}
	return false
}

func (b *Board) CanAnyBishopMoveSafe(color int8) bool {
	occupied := b.Occupied[White] | b.Occupied[Black]
	enemyColor := 1 - color
	kingSq := int(bits.TrailingZeros64(uint64(b.Kings[color])))

	for bb := b.Bishops[color]; bb != 0; {
		from := int8(PopLSB(&bb))
		fromBB := Bitboard(1) << from

		moves := slidingAttacksBishop(int(from), occupied) & ^b.Occupied[color]
		for m := moves; m != 0; {
			to := int8(PopLSB(&m))
			toBB := Bitboard(1) << to

			b.Bishops[color] &^= fromBB
			b.Bishops[color] |= toBB
			cap := b.movePieceLight(from, to, color, Bishop)

			if !IsSquareAttacked(kingSq, b, enemyColor) {
				b.undoMovePieceLight(from, to, color, cap, Bishop)
				b.Bishops[color] &^= toBB
				b.Bishops[color] |= fromBB
				return true
			}

			b.undoMovePieceLight(from, to, color, cap, Bishop)
			b.Bishops[color] &^= toBB
			b.Bishops[color] |= fromBB
		}
	}
	return false
}

func (b *Board) CanAnyQueenMoveSafe(color int8) bool {
	occupied := b.Occupied[White] | b.Occupied[Black]
	enemyColor := 1 - color
	kingSq := int(bits.TrailingZeros64(uint64(b.Kings[color])))

	for bb := b.Queens[color]; bb != 0; {
		from := int8(PopLSB(&bb))
		fromBB := Bitboard(1) << from

		moves := slidingAttacksQueen(int(from), occupied) & ^b.Occupied[color]
		for m := moves; m != 0; {
			to := int8(PopLSB(&m))
			toBB := Bitboard(1) << to

			b.Queens[color] &^= fromBB
			b.Queens[color] |= toBB
			cap := b.movePieceLight(from, to, color, Queen)

			if !IsSquareAttacked(kingSq, b, enemyColor) {
				b.undoMovePieceLight(from, to, color, cap, Queen)
				b.Queens[color] &^= toBB
				b.Queens[color] |= fromBB
				return true
			}

			b.undoMovePieceLight(from, to, color, cap, Queen)
			b.Queens[color] &^= toBB
			b.Queens[color] |= fromBB
		}
	}
	return false
}

func (b *Board) CanAnyKnightMoveSafe(color int8) bool {
	knights := b.Knights[color]
	enemyColor := 1 - color

	kingSq := int8(bits.TrailingZeros64(uint64(b.Kings[color])))

	for bb := knights; bb != 0; {
		from := int8(PopLSB(&bb))
		fromBB := Bitboard(1) << from

		moves := knightMoves[from] & ^b.Occupied[color]
		for m := moves; m != 0; {
			to := int8(PopLSB(&m))

			// make light move
			captured := b.Occupied[enemyColor] & (Bitboard(1) << to)

			b.Knights[color] &^= fromBB
			b.Knights[color] |= Bitboard(1) << to
			b.Occupied[color] &^= fromBB
			b.Occupied[color] |= Bitboard(1) << to
			b.Occupied[enemyColor] &^= captured

			if !IsSquareAttacked(int(kingSq), b, enemyColor) {
				// undo
				b.Knights[color] &^= Bitboard(1) << to
				b.Knights[color] |= fromBB
				b.Occupied[color] &^= Bitboard(1) << to
				b.Occupied[color] |= fromBB
				b.Occupied[enemyColor] |= captured
				return true
			}

			// undo
			b.Knights[color] &^= Bitboard(1) << to
			b.Knights[color] |= fromBB
			b.Occupied[color] &^= Bitboard(1) << to
			b.Occupied[color] |= fromBB
			b.Occupied[enemyColor] |= captured
		}
	}
	return false
}

func (b *Board) CanAnyKingMoveSafe(color int8) bool {
	enemyColor := 1 - color
	kingBB := b.Kings[color]

	from := int8(bits.TrailingZeros64(uint64(kingBB)))
	fromBB := Bitboard(1) << from

	moves := kingMoves[from] & ^b.Occupied[color]

	for m := moves; m != 0; {
		to := int8(PopLSB(&m))
		toBB := Bitboard(1) << to

		captured := b.Occupied[enemyColor] & toBB

		// move king
		b.Kings[color] = toBB
		b.Occupied[color] &^= fromBB
		b.Occupied[color] |= toBB
		b.Occupied[enemyColor] &^= captured

		// king must not be attacked on destination square
		if !IsSquareAttacked(int(to), b, enemyColor) {
			b.Kings[color] = fromBB
			b.Occupied[color] &^= toBB
			b.Occupied[color] |= fromBB
			b.Occupied[enemyColor] |= captured
			return true
		}

		// undo
		b.Kings[color] = fromBB
		b.Occupied[color] &^= toBB
		b.Occupied[color] |= fromBB
		b.Occupied[enemyColor] |= captured
	}

	return false
}

func (b *Board) GeAllPiecesThatCanMoveLegallyThisTurn(color int8, piecesToCheck uint8) uint8 {
	var result uint8 = 0

	if piecesToCheck&CanPawn != 0 && b.CanAnyPawnMoveSafe(color) {
		result |= CanPawn
	}
	if piecesToCheck&CanKnight != 0 && b.CanAnyKnightMoveSafe(color) {
		result |= CanKnight
	}
	if piecesToCheck&CanBishop != 0 && b.CanAnyBishopMoveSafe(color) {
		result |= CanBishop
	}
	if piecesToCheck&CanRook != 0 && b.CanAnyRookMoveSafe(color) {
		result |= CanRook
	}
	if piecesToCheck&CanQueen != 0 && b.CanAnyQueenMoveSafe(color) {
		result |= CanQueen
	}
	if piecesToCheck&CanKing != 0 && b.CanAnyKingMoveSafe(color) {
		result |= CanKing
	}

	return result
}

func (b *Board) GetAllPiecesThatCanMoveLegallyThisTurnCheckAll(color int8) uint8 {
	var result uint8

	if b.CanAnyPawnMoveSafe(color) {
		result |= CanPawn
	}
	if b.CanAnyKnightMoveSafe(color) {
		result |= CanKnight
	}
	if b.CanAnyBishopMoveSafe(color) {
		result |= CanBishop
	}
	if b.CanAnyRookMoveSafe(color) {
		result |= CanRook
	}
	if b.CanAnyQueenMoveSafe(color) {
		result |= CanQueen
	}
	if b.CanAnyKingMoveSafe(color) {
		result |= CanKing
	}

	return result
}

func MakeMove(board *Board, from, to int8, promoteTo int8, testingFuture bool) Move {

	fromBB := Bitboard(1) << from
	toBB := Bitboard(1) << to
	color := Black // 0
	if board.Flags&WhiteToMove != 0 {
		color = White // 1
	}
	enemyColor := 1 - color
	movingPiece := GetPieceType(board, from, int8(color))
	capturedPiece := GetPieceType(board, to, int8(enemyColor))

	// Check for en passant BEFORE updating the en passant square
	isEnPassant := false
	var enPassantCaptureSq int8
	if movingPiece == Pawn && to == board.EnPassantSquare && board.EnPassantSquare >= 0 {
		isEnPassant = true
		// Calculate the actual square where the captured pawn is
		if color == White {
			enPassantCaptureSq = to - 8 // White captures downward
		} else {
			enPassantCaptureSq = to + 8 // Black captures upward
		}
	}

	if movingPiece == King && abs(int(to-from)) == 2 {
		var rookFrom, rookTo int8
		if to > from {
			rookFrom = to + 1
			rookTo = to - 1
		} else {
			rookFrom = to - 2
			rookTo = to + 1
		}
		rookFromBB := Bitboard(1) << rookFrom
		rookToBB := Bitboard(1) << rookTo

		if (board.Rooks[color] & rookFromBB) != 0 {
			// move rook bits
			board.Rooks[color] &^= rookFromBB
			board.Rooks[color] |= rookToBB

			// update occupied for rook
			board.Occupied[color] &^= rookFromBB
			board.Occupied[color] |= rookToBB
		} else {
			log.Printf("Expected rook for castling not found at %s", IndexToSqaureName(rookFrom))
		}
	}

	// Clear captured piece (only if there's a capture)
	if capturedPiece >= 0 {
		switch capturedPiece {
		case Pawn:
			board.Pawns[enemyColor] &^= toBB
		case Knight:
			board.Knights[enemyColor] &^= toBB
		case Bishop:
			board.Bishops[enemyColor] &^= toBB
		case Rook:
			board.Rooks[enemyColor] &^= toBB
		case Queen:
			board.Queens[enemyColor] &^= toBB
		case King:
			board.Kings[enemyColor] &^= toBB
		}
	}

	if isEnPassant {
		capSqBB := Bitboard(1) << enPassantCaptureSq
		board.Pawns[enemyColor] &^= capSqBB
		board.Occupied[enemyColor] &^= capSqBB
	}

	// Move the piece and update hash
	switch movingPiece {
	case Pawn:
		board.Pawns[color] &^= fromBB
		// Check for promotion
		if (color == White && (toBB&rank8) != 0) || (color == Black && (toBB&rank1) != 0) {
			switch promoteTo {
			case 1:
				board.Rooks[color] |= toBB
			case 2:
				board.Knights[color] |= toBB
			case 3:
				board.Bishops[color] |= toBB
			case 4:
				board.Queens[color] |= toBB
			default:
				board.Pawns[color] |= toBB
			}
		} else {
			board.Pawns[color] |= toBB
		}
	case Knight:
		board.Knights[color] &^= fromBB
		board.Knights[color] |= toBB
	case Bishop:
		board.Bishops[color] &^= fromBB
		board.Bishops[color] |= toBB
	case Rook:
		board.Rooks[color] &^= fromBB
		board.Rooks[color] |= toBB
	case Queen:
		board.Queens[color] &^= fromBB
		board.Queens[color] |= toBB
	case King:
		board.Kings[color] &^= fromBB
		board.Kings[color] |= toBB
	}

	// Update occupancy bitboards
	board.Occupied[color] &^= fromBB
	board.Occupied[color] |= toBB
	if capturedPiece >= 0 {
		board.Occupied[enemyColor] &^= toBB
	}

	// Update en passant square (AFTER checking for en passant capture)
	newEnPassant := int8(-1)
	if movingPiece == Pawn && abs(int(to-from)) == 16 {
		newEnPassant = (from + to) / 2
	}
	board.EnPassantSquare = newEnPassant

	// Update castling rights using lookup table
	board.Flags &= castlingRightsBySquare[from] & castlingRightsBySquare[to]

	board.UpdateMoveCounters(capturedPiece != -1, movingPiece == Pawn)

	// Toggle side to move
	board.Flags ^= WhiteToMove

	///#####test debug
	//log.Printf("FEN: %s", board.ToFEN())
	///#####test debiug

	return Move{From: from, To: to, Promotion: promoteTo}
}

func (b *Board) MakeMove(from, to int8, promoteTo int8, testingFuture bool) {
	MakeMove(b, from, to, promoteTo, testingFuture)
}

func CanCastle(b *Board, color int8, kingSide bool) bool {
	rights := b.Flags & 0x0F
	if color == White {
		if kingSide && (rights&WK == 0) {
			return false
		}
		if !kingSide && (rights&WQ == 0) {
			return false
		}
	} else {
		if kingSide && (rights&BK == 0) {
			return false
		}
		if !kingSide && (rights&BQ == 0) {
			return false
		}
	}

	// Squares between king and rook must be empty
	var between Bitboard
	var kingPos int
	if color == White {
		kingPos = 4 // e1
		if kingSide {
			between = Bitboard(0x60)
		} else {
			between = Bitboard(0x0E)
		}
	} else {
		kingPos = 60 // e8
		if kingSide {
			between = Bitboard(0x6000000000000000)
		} else {
			between = Bitboard(0x0E00000000000000)
		}
	}
	allOccupied := b.Occupied[White] | b.Occupied[Black]
	if allOccupied&between != 0 {
		return false
	}

	// King must not be in check, and squares it moves over must not be attacked
	kingSquares := []int{kingPos}
	if kingSide {
		kingSquares = append(kingSquares, kingPos+1, kingPos+2)
	} else {
		kingSquares = append(kingSquares, kingPos-1, kingPos-2)
	}

	for _, sq := range kingSquares {
		if IsSquareAttacked(sq, b, 1-color) {
			return false
		}
	}

	return true
}

func (b *Board) ToSquareArray() [64]uint8 {
	var squares [64]uint8

	// Define piece types (adjust these constants to match your engine)
	const (
		Empty  = 0
		Pawn   = 1
		Knight = 2
		Bishop = 3
		Rook   = 4
		Queen  = 5
		King   = 6
	)

	// For each color (0 = white, 1 = black)
	for color := 0; color < 2; color++ {
		colorOffset := uint8(color * 8) // 0 for white, 8 for black pieces

		// Check each piece type
		pieces := []struct {
			bitboard  Bitboard
			pieceType uint8
		}{
			{b.Pawns[color], Pawn},
			{b.Knights[color], Knight},
			{b.Bishops[color], Bishop},
			{b.Rooks[color], Rook},
			{b.Queens[color], Queen},
			{b.Kings[color], King},
		}

		for _, piece := range pieces {
			bb := piece.bitboard
			for bb != 0 {
				square := bits.TrailingZeros64(uint64(bb))
				squares[square] = piece.pieceType + colorOffset
				bb &= bb - 1 // Clear the least significant bit
			}
		}
	}

	return squares
}

func (b *Board) UpdateMoveCounters(capturedPiece bool, pawnMove bool) {
	// Reset halfmove clock on pawn move or capture
	if pawnMove || capturedPiece {
		b.HalfmoveClock = 0
	} else {
		b.HalfmoveClock++
	}

	// Increment fullmove number after black's move
	if b.Flags&16 == 0 { // If it was black's turn (now switching to white)
		b.FullmoveNumber++
	}
}

func (b *Board) ToByteArray() []byte {
	squares := b.ToSquareArray()
	bytes := make([]byte, len(squares))
	for i, s := range squares {
		bytes[i] = byte(s)
	}
	return bytes
}

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

	halfmove, _ := strconv.ParseUint(parts[4], 10, 8)
	fullmove, _ := strconv.ParseUint(parts[5], 10, 16)

	board.HalfmoveClock = uint8(halfmove)
	board.FullmoveNumber = uint16(fullmove)

	//parse
	return board, nil
}

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
