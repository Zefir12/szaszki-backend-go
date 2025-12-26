package chess

import (
	"fmt"
	"math/bits"
	"strconv"
)

func (b *Board) ToPGN(moveHistory []Move) string {
	temp := *b
	var pgn string

	for i, move := range moveHistory {

		if i%2 == 0 {
			pgn += fmt.Sprintf("%d. ", i/2+1)
		}

		san := moveToSAN(&temp, move)
		pgn += san + " "

		MakeMove(&temp, move.From, move.To, move.Promotion, false)
	}

	return pgn
}

func pieceLetter(p int8) string {
	switch p {
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
	}
	return ""
}

func CurrentColor(b *Board) int8 {
	return int8((b.Flags&WhiteToMove)>>4) ^ 1
}

func kingInCheck(b *Board, color int8) bool {
	sq := bits.TrailingZeros64(uint64(b.Kings[color]))
	return IsSquareAttacked(int(sq), b, 1-color)
}

func kingInCheckmate(b *Board, color int8) bool {
	if !kingInCheck(b, color) {
		return false
	}
	// if no legal move exists → mate
	return !b.HasAnyLegalMove(color)
}

func (b *Board) FindPieces(piece, color int8) []int8 {
	var bb Bitboard

	switch piece {
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

	var result []int8
	bb64 := uint64(bb)

	for bb64 != 0 {
		sq := int8(bits.TrailingZeros64(bb64))
		result = append(result, sq)
		bb64 &= bb64 - 1
	}

	return result
}

func disambiguate(board *Board, move Move, piece, color int8) (fileConflict bool, rankConflict bool) {
	from := move.From
	to := move.To

	// find all pieces of same type/color that can also move to "to"
	samePieces := board.FindPieces(piece, color)

	var candidates []int8
	for _, sq := range samePieces {
		if sq == from {
			continue
		}
		if IsMoveLegal(board, sq, to, piece) {
			candidates = append(candidates, sq)
		}
	}

	if len(candidates) == 0 {
		return false, false // no ambiguity
	}

	needFile := false
	needRank := false

	fromFile := from % 8
	fromRank := from / 8

	// check if any candidate shares file or rank
	for _, sq := range candidates {
		if sq%8 == fromFile {
			needRank = true
		}
		if sq/8 == fromRank {
			needFile = true
		}
	}

	// if file OR rank alone is not enough → use both
	if needFile && needRank {
		return true, true
	}

	// if only one condition needed:
	return needFile, needRank
}

func moveToSAN(board *Board, move Move) string {
	from := move.From
	to := move.To

	color := CurrentColor(board)
	enemy := 1 - color

	piece := GetPieceType(board, from, color)
	capturedPiece := GetPieceType(board, to, enemy)

	// ---- handle castling ----
	if piece == King {
		if (from == 4 && to == 6) || (from == 60 && to == 62) {
			return "O-O"
		}
		if (from == 4 && to == 2) || (from == 60 && to == 58) {
			return "O-O-O"
		}
	}

	san := ""

	// ---- piece letter (blank for pawn) ----
	if piece != Pawn {
		san += pieceLetter(int8(piece))
	}

	// ---- disambiguation ----
	if piece != Pawn { // only pieces need this
		needFile, needRank := disambiguate(board, move, int8(piece), color)
		if needFile {
			san += string('a' + rune(from%8))
		}
		if needRank {
			san += strconv.Itoa(int(from/8) + 1)
		}
	}

	// ---- pawn capture: add file ----
	if piece == Pawn && capturedPiece != -1 {
		san += string('a' + rune(from%8))
	}

	// ---- capture symbol ----
	if capturedPiece != -1 {
		san += "x"
	}

	// ---- destination square ----
	san += squareToString(to)

	// ---- promotion ----
	if move.Promotion != 0 {
		san += "=" + pieceLetter(move.Promotion)
	}

	// ---- check or mate ----
	temp := *board
	MakeMove(&temp, move.From, move.To, move.Promotion, false)
	if kingInCheckmate(&temp, enemy) {
		san += "#"
	} else if kingInCheck(&temp, enemy) {
		san += "+"
	}

	return san
}

func squareToString(s int8) string {
	file := s % 8
	rank := s / 8
	// Fixed: properly convert file to letter and rank to number
	return string('a'+rune(file)) + strconv.Itoa(int(rank)+1)
}
