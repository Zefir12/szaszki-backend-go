package chess

import (
	"testing"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

func TestGetAllPiecesThatCanLegallyMoveThisTurn_StartingPosition(t *testing.T) {
	board := chess.NewStartingPosition()

	// Both sides should be able to move pawns and knights
	piecesToCheck := chess.CanPawn | chess.CanKnight
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(chess.White, piecesToCheck) != piecesToCheck {
		t.Error("White should be able to move pawns and knights from starting position")
	}

	if board.GeAllPiecesThatCanMoveLegallyThisTurn(chess.Black, piecesToCheck) != piecesToCheck {
		t.Error("Black should be able to move pawns and knights from starting position")
	}
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_CheckSituation(t *testing.T) {
	board, _ := chess.ParseFEN("8/8/k7/1b6/8/3R4/4K3/8 w - - 0 1")

	// Both sides should be able to move pawns and knights
	piecesToCheck := chess.CanKing
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(chess.White, piecesToCheck) != piecesToCheck {
		t.Error("White should be able to move only king")
	}

	board, _ = chess.ParseFEN("8/8/k7/1r6/8/3B4/4K3/8 b - - 0 1")
	piecesToCheck = chess.CanKing
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(chess.Black, piecesToCheck) != piecesToCheck {
		t.Error("Black should be able to move only king")
	}
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_Stalemate(t *testing.T) {
	// Black is stalemated
	board, _ := chess.ParseFEN("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")

	piecesToCheck := chess.CanPawn | chess.CanKnight | chess.CanBishop | chess.CanRook | chess.CanQueen | chess.CanKing
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(chess.Black, piecesToCheck) != 0 {
		t.Error("Black should have no legal moves (stalemate)")
	}
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_Checkmate(t *testing.T) {
	// Black is checkmated
	board, _ := chess.ParseFEN("7k/6Q1/6K1/8/8/8/8/8 b - - 0 1")

	piecesToCheck := chess.CanPawn | chess.CanKnight | chess.CanBishop | chess.CanRook | chess.CanQueen | chess.CanKing
	if board.GeAllPiecesThatCanMoveLegallyThisTurn(chess.Black, piecesToCheck) != 0 {
		t.Error("Black should have no legal moves (checkmate)")
	}
}

func TestGetAllPiecesThatCanLegallyMoveThisTurn_OnlyCheckRooks(t *testing.T) {
	board, _ := chess.ParseFEN("8/8/8/8/8/8/8/R3K2k b - - 1 1")

	piecesToCheck := chess.CanRook
	result := board.GeAllPiecesThatCanMoveLegallyThisTurn(chess.White, piecesToCheck)

	if result != chess.CanRook {
		t.Errorf("Only rook should be movable, got %b", result)
	}
}
