package chess

import (
	"fmt"
	"log"
	"math/bits"
	"math/rand"
)

//https://en.wikipedia.org/wiki/Bitboard

type Bitboard uint64

const (
	White = 1
	Black = 0
)

const zobristSeed = 0xCAFEBABE

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

const (
	WK uint8 = 1 << 0 // White kingside
	WQ uint8 = 1 << 1 // White queenside
	BK uint8 = 1 << 2 // Black kingside
	BQ uint8 = 1 << 3 // Black queenside

	WhiteToMove uint8 = 1 << 4
)

var knightMoves [64]Bitboard
var kingMoves [64]Bitboard

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
var zobristCastlingRights [2][2]uint64
var zobristPieces [2][6][64]uint64 // [color][piece][square]
var zobristCastling [16]uint64     // for all castling flag combinations

var rng *rand.Rand

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
	for i := 0; i < 64; i++ {
		knightMoves[i] = generateKnightMoves(i)
		kingMoves[i] = generateKingMoves(i)
	}
	initZobrist()
	initZobristExtended()
}

func initZobrist() {
	rng = rand.New(rand.NewSource(zobristSeed))
	for pieceType := 0; pieceType < 6; pieceType++ {
		for sq := 0; sq < 64; sq++ {
			zobristTable[pieceType][sq] = rng.Uint64()
		}
	}
	for sq := 0; sq < 64; sq++ {
		zobristEnPassant[sq] = rng.Uint64()
	}
	zobristSideToMove = rng.Uint64()
	for color := 0; color < 2; color++ {
		for side := 0; side < 2; side++ {
			zobristCastlingRights[color][side] = rng.Uint64()
		}
	}
}

func initZobristExtended() {
	// Initialize piece tables
	for color := 0; color < 2; color++ {
		for piece := 0; piece < 6; piece++ {
			for sq := 0; sq < 64; sq++ {
				zobristPieces[color][piece][sq] = rng.Uint64()
			}
		}
	}

	// Initialize castling combinations
	for i := 0; i < 16; i++ {
		zobristCastling[i] = rng.Uint64()
	}
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
	b.Occupied[White] = 0x000000000000FFFF // b.Pawns[White] | b.Rooks[White] | b.Knights[White] | b.Bishops[White] | b.Queens[White] | b.Kings[White]
	b.Occupied[Black] = 0xFFFF000000000000 // b.Pawns[Black] | b.Rooks[Black] | b.Knights[Black] | b.Bishops[Black] | b.Queens[Black] | b.Kings[Black]

	b.Flags |= WK | WQ | BK | BQ | WhiteToMove // enable all castling and set side to white

	b.EnPassantSquare = -1
	b.Hash = ComputeHash(&b)

	return b
}

func slidingAttacks(sq int, occupied Bitboard, deltas []int) Bitboard {
	var attacks Bitboard
	for _, d := range deltas {
		for s := sq + d; s >= 0 && s < 64; s += d {
			// Check for horizontal wrap-around
			if (d == 1 || d == -1) && s/8 != sq/8 {
				break
			}
			// Check for diagonal wrap-around
			if (d == 9 || d == -7 || d == 7 || d == -9) && abs(s%8-((s-d)%8)) != 1 {
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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

func IsMoveLegal(board *Board, from, to, promoteTo int8) bool {
	temp := *board // copy board
	MakeMove(&temp, from, to, promoteTo)

	attackerColor := int8((board.Flags&WhiteToMove)>>4) ^ 1

	kingSq := bits.TrailingZeros64(uint64(temp.Kings[attackerColor]))
	return !IsSquareAttacked(kingSq, &temp, 1-attackerColor)
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
	allOccupied := board.Occupied[White] | board.Occupied[Black]
	if slidingAttacks(sq, allOccupied, directions["bishop"])&
		(board.Bishops[attackerColor]|board.Queens[attackerColor]) != 0 {
		return true
	}
	if slidingAttacks(sq, allOccupied, directions["rook"])&
		(board.Rooks[attackerColor]|board.Queens[attackerColor]) != 0 {
		return true
	}
	return false
}

func MakeMove(board *Board, from, to int8, promoteTo int8) Move {

	log.Println(from, to, promoteTo)
	fromBB := Bitboard(1) << from
	toBB := Bitboard(1) << to
	color := int8((board.Flags&WhiteToMove)>>4) ^ 1
	enemyColor := 1 - color

	movingPiece := GetPieceType(board, from, color)
	log.Println("mving piece", movingPiece, from, color)
	capturedPiece := GetPieceType(board, to, enemyColor)

	newHash := board.Hash // Start incremental hash updates

	// Remove moving piece from source
	if movingPiece >= 0 {
		newHash ^= zobristPieces[color][movingPiece][from]
	}

	// Remove captured piece if any
	if capturedPiece >= 0 {
		newHash ^= zobristPieces[enemyColor][capturedPiece][to]
	}

	// Remove old en passant
	if board.EnPassantSquare >= 0 {
		newHash ^= zobristEnPassant[board.EnPassantSquare]
	}

	// Remove old castling rights
	newHash ^= zobristCastling[board.Flags&0x0F]

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

	// Handle en passant capture
	if board.EnPassantSquare == to && movingPiece == Pawn {
		capSq := to - 8
		if color == White {
			capSq = to + 8
		}
		board.Pawns[enemyColor] &^= Bitboard(1) << capSq
		// Remove captured pawn from hash
		newHash ^= zobristPieces[enemyColor][Pawn][capSq]
	}

	// Move the piece and update hash
	finalPiece := movingPiece
	switch movingPiece {
	case Pawn:
		board.Pawns[color] &^= fromBB
		// Check for promotion
		if (color == White && (toBB&rank8) != 0) || (color == Black && (toBB&rank1) != 0) {
			switch promoteTo {
			case 1:
				board.Rooks[color] |= toBB
				finalPiece = Rook
			case 2:
				board.Knights[color] |= toBB
				finalPiece = Knight
			case 3:
				board.Bishops[color] |= toBB
				finalPiece = Bishop
			case 4:
				board.Queens[color] |= toBB
				finalPiece = Queen
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

	// Add piece to destination in hash
	log.Println(color, finalPiece, to)
	newHash ^= zobristPieces[color][finalPiece][to]

	// Update en passant square
	newEnPassant := int8(-1)
	if movingPiece == Pawn && abs(int(to-from)) == 16 {
		newEnPassant = (from + to) / 2
	}
	board.EnPassantSquare = newEnPassant

	// Add new en passant to hash
	if newEnPassant >= 0 {
		newHash ^= zobristEnPassant[newEnPassant]
	}

	// Update occupancy bitboards
	board.Occupied[color] &^= fromBB
	board.Occupied[color] |= toBB
	board.Occupied[enemyColor] &^= toBB

	// Update castling rights using lookup table
	board.Flags &= castlingRightsBySquare[from] & castlingRightsBySquare[to]

	// Add new castling rights to hash
	newHash ^= zobristCastling[board.Flags&0x0F]

	// Toggle side to move
	board.Flags ^= WhiteToMove
	newHash ^= zobristSideToMove

	board.UpdateMoveCounters(capturedPiece != -1, movingPiece == Pawn)

	// Update hash
	board.Hash = newHash
	return Move{From: from, To: to, Promotion: promoteTo}
}

func ComputeHash(b *Board) uint64 {
	var hash uint64 = 0

	// Add pieces
	for color := 0; color < 2; color++ {
		for pieceType := 0; pieceType < 6; pieceType++ {
			var bb Bitboard
			switch pieceType {
			case Pawn:
				bb = b.Pawns[color]
			case Knight:
				bb = b.Knights[color]
			case Bishop:
				bb = b.Bishops[color]
			case Rook:
				bb = b.Rooks[color]
			case Queen:
				bb = b.Queens[color]
			case King:
				bb = b.Kings[color]
			}
			for bb != 0 {
				sq := PopLSB(&bb)
				hash ^= zobristPieces[color][pieceType][sq]
			}
		}
	}

	// Add en passant if present
	if b.EnPassantSquare >= 0 {
		hash ^= zobristEnPassant[b.EnPassantSquare]
	}

	// Add side to move
	if b.Flags&WhiteToMove == 0 { // If it's Black's turn
		hash ^= zobristSideToMove
	}

	// Add castling rights
	hash ^= zobristCastling[b.Flags&0x0F]

	return hash
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

type Move struct {
	From, To, Promotion int8
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

func (b *Board) ToPGN(moveHistory []Move) string {
	var pgn string
	for i, move := range moveHistory {
		if i%2 == 0 {
			pgn += fmt.Sprintf("%d. %s ", i/2+1, moveToString(move))
		} else {
			pgn += fmt.Sprintf("%s ", moveToString(move))
		}
	}
	return pgn
}

func moveToString(move Move) string {
	from := squareToString(move.From)
	to := squareToString(move.To)
	promo := ""
	switch move.Promotion {
	case Rook:
		promo = "R"
	case Knight:
		promo = "N"
	case Bishop:
		promo = "B"
	case Queen:
		promo = "Q"
	}
	return from + to + promo
}

func squareToString(s int8) string {
	file := s % 8
	rank := s / 8
	return string('a'+file) + string('1'+rank)
}
