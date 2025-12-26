package chess

import (
	"testing"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

func TestCanAnyRookMoveSafe_ClearPath(t *testing.T) {
	// White rook on a1 with empty file
	startingFen := "R7/7k/8/8/8/7K/8/8 w - - 0 1"

	board, _ := chess.ParseFEN(startingFen)
	if !board.CanAnyRookMoveSafe(chess.White) {
		t.Error("White rook on a1 should be able to move freely")
	}
	if startingFen != board.ToFEN() {
		t.Error("Board changed after check!")
	}

	// Black rook on h8 with empty file
	startingFen = "7r/5k2/8/8/8/4K3/8/8 b - - 0 1"
	board, _ = chess.ParseFEN(startingFen)
	if !board.CanAnyRookMoveSafe(chess.Black) {
		t.Error("Black rook on h8 should be able to move freely")
	}
	if startingFen != board.ToFEN() {
		t.Error("Board changed after check!")
	}
}

func TestCanAnyRookMoveSafe_BlockedRooks(t *testing.T) {
	// Rooks completely blocked by own pieces
	startingFen := "RRRRRRKR/PPPPPPPP/8/8/8/8/pppppppp/rrrrrrkr w - - 0 1"
	board, _ := chess.ParseFEN(startingFen)
	if board.CanAnyRookMoveSafe(chess.White) {
		t.Error("White rooks should all be blocked")
	}
	if startingFen != board.ToFEN() {
		t.Error("Board changed after check!")
	}
	if board.CanAnyRookMoveSafe(chess.Black) {
		t.Error("Black rooks should all be blocked")
	}
	if startingFen != board.ToFEN() {
		t.Error("Board changed after check!")
	}
}

func TestCanAnyPawnMove_StartingPosition(t *testing.T) {
	board := chess.NewStartingPosition()

	// Both sides should be able to move pawns
	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to move pawns from starting position")
	}

	if !board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to move pawns from starting position")
	}
}

func TestCanAnyPawnMove_NoPawns(t *testing.T) {
	board, _ := chess.ParseFEN("rnbqkbnr/8/8/8/8/8/8/RNBQKBNR w KQkq - 0 1")

	if board.CanAnyPawnMove(chess.White) {
		t.Error("White should not be able to move pawns when they have none")
	}

	if board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should not be able to move pawns when they have none")
	}
}

func TestCanAnyPawnMove_BlockedPawns(t *testing.T) {
	// All white and black pawns blocked
	board, _ := chess.ParseFEN("4k3/8/pppppppp/rrrrrrrr/RRRRRRRR/PPPPPPPP/8/5K2 w - - 0 1")

	if board.CanAnyPawnMove(chess.White) {
		t.Error("White pawns should be completely blocked")
	}

	if board.CanAnyPawnMove(chess.Black) {
		t.Error("Black pawns should be completely blocked")
	}
}

func TestCanAnyPawnMove_SinglePush(t *testing.T) {
	// White pawn on e2 can push to e3
	board, _ := chess.ParseFEN("8/8/8/8/8/8/4P3/8 w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to push pawn from e2")
	}

	// Black pawn on e7 can push to e6
	board, _ = chess.ParseFEN("8/4p3/8/8/8/8/8/8 b - - 0 1")

	if !board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to push pawn from e7")
	}
}

func TestCanAnyPawnMove_DoublePush(t *testing.T) {
	// White pawn on e2 can double push
	board, _ := chess.ParseFEN("8/8/8/8/8/8/4P3/8 w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to double push from e2")
	}

	// Black pawn on e7 can double push
	board, _ = chess.ParseFEN("8/4p3/8/8/8/8/8/8 b - - 0 1")

	if !board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to double push from e7")
	}
}

func TestCanAnyPawnMove_DoublePushBlocked(t *testing.T) {
	// White pawn on e2, piece on e4 (can still single push to e3)
	board, _ := chess.ParseFEN("8/8/8/8/4n3/8/4P3/8 w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should still be able to single push")
	}

	// Completely blocked
	board, _ = chess.ParseFEN("8/8/8/8/8/4n3/4P3/8 w - - 0 1")

	if board.CanAnyPawnMove(chess.White) {
		t.Error("White pawn should be completely blocked")
	}
}

func TestCanAnyPawnMove_CaptureAvailable(t *testing.T) {
	// White pawn can capture black knight
	board, _ := chess.ParseFEN("8/8/8/8/3n4/2P5/8/8 w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to capture on d4")
	}

	// Black pawn can capture white knight
	board, _ = chess.ParseFEN("8/8/2p5/3N4/8/8/8/8 b - - 0 1")

	if !board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to capture on d5")
	}
}

func TestCanAnyPawnMove_EnPassantAvailable(t *testing.T) {
	// White pawn can capture en passant on d6
	board, _ := chess.ParseFEN("rnbqkb1r/ppp1pppp/7n/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to capture en passant on e6")
	}

	// Black pawn can capture en passant on e3
	board, _ = chess.ParseFEN("8/8/8/8/3Pp3/8/8/8 b - e3 0 1")

	if !board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to capture en passant on e3")
	}
}

func TestCanAnyPawnMove_PromotionRank(t *testing.T) {
	// White pawn on 7th rank can promote
	board, _ := chess.ParseFEN("8/4P3/8/8/8/8/8/8 w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to push pawn to promotion")
	}

	// Black pawn on 2nd rank can promote
	board, _ = chess.ParseFEN("8/8/8/8/8/8/4p3/8 b - - 0 1")

	if !board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to push pawn to promotion")
	}
}

func TestCanAnyPawnMove_ComplexPosition(t *testing.T) {
	// Multiple pawns, some blocked, some free
	board, _ := chess.ParseFEN("7k/8/8/2ppp3/3P4/8/PP6/6K1 w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to move pawns (a2, b2, or captures with d4)")
	}

	board, _ = chess.ParseFEN("8/8/8/2ppp3/3P4/8/PP6/8 b - - 0 1")

	if !board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to move pawns (c5, e5 forward or d5 captures)")
	}
}

func TestCanAnyPawnMove_OnlyCapturesPossible(t *testing.T) {
	// White pawn blocked in front but can capture
	board, _ := chess.ParseFEN("7k/8/8/8/8/2npn3/3P4/7K w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to capture on c3 or e3")
	}
}

func TestCanAnyPawnMove_AllMovesBlocked(t *testing.T) {
	// Pawn completely surrounded by own pieces
	board, _ := chess.ParseFEN("7k/8/8/8/3R4/2RPR3/8/7K w - - 0 1")

	if board.CanAnyPawnMove(chess.White) {
		t.Error("White pawns should all be blocked")
	}
}

func TestCanAnyPawnMove_MixedBlockedAndFree(t *testing.T) {
	// Some pawns blocked, one free
	board, _ := chess.ParseFEN("8/8/8/pppp4/BBBB4/PPPP4/7P/8 w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to move h2 pawn")
	}
}

func TestCanAnyPawnMove_PawnsBlockedFromPromoting(t *testing.T) {
	// Pawn on 7th rank, blocked from promoting and no other move
	board, _ := chess.ParseFEN("4n2k/4P3/8/8/8/8/8/7K w - - 0 1")

	if board.CanAnyPawnMove(chess.White) {
		t.Error("White pawn should be blocked from promoting")
	}

	//Pawn on 2th rank, blocked from promoting and no other move
	board, _ = chess.ParseFEN("7k/8/8/8/8/8/4p3/4N2K b - - 0 1")

	if board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to capture and promote")
	}
}

func TestCanAnyPawnMove_CaptureOnPromotionRank(t *testing.T) {
	// White pawn can capture and promote
	board, _ := chess.ParseFEN("3n3k/4P3/8/8/8/8/8/7K w - - 0 1")

	if !board.CanAnyPawnMove(chess.White) {
		t.Error("White should be able to capture and promote")
	}

	// Black pawn can capture and promote
	board, _ = chess.ParseFEN("7k/8/8/8/8/8/4p3/3N3K b - - 0 1")

	if !board.CanAnyPawnMove(chess.Black) {
		t.Error("Black should be able to capture and promote")
	}
}

func TestCanAnyRookMove_StartingPosition(t *testing.T) {
	board := chess.NewStartingPosition()

	// At the very start, rooks are blocked by pawns
	if board.CanAnyRookMove(chess.White) {
		t.Error("White rooks should not be able to move from starting position")
	}
	if board.CanAnyRookMove(chess.Black) {
		t.Error("Black rooks should not be able to move from starting position")
	}
}

func TestCanAnyRookMove_ClearPath(t *testing.T) {
	// White rook on a1 with empty file
	board, _ := chess.ParseFEN("R7/7k/8/8/8/7K/8/8 w - - 0 1")
	if !board.CanAnyRookMove(chess.White) {
		t.Error("White rook on a1 should be able to move freely")
	}

	// Black rook on h8 with empty file
	board, _ = chess.ParseFEN("7r/5k2/8/8/8/4K3/8/8 b - - 0 1")
	if !board.CanAnyRookMove(chess.Black) {
		t.Error("Black rook on h8 should be able to move freely")
	}
}

func TestCanAnyRookMove_BlockedRooks(t *testing.T) {
	// Rooks completely blocked by own pieces
	board, _ := chess.ParseFEN("RRRRRRKR/PPPPPPPP/8/8/8/8/pppppppp/rrrrrrkr w - - 0 1")
	if board.CanAnyRookMove(chess.White) {
		t.Error("White rooks should all be blocked")
	}
	if board.CanAnyRookMove(chess.Black) {
		t.Error("Black rooks should all be blocked")
	}
}

func TestCanAnyRookMove_SomeBlockedSomeFree(t *testing.T) {
	// White rook on a1 blocked, rook on h1 free
	board, _ := chess.ParseFEN("RN3K1R/NN6/8/8/8/8/8/5k2 w - - 0 1")
	if !board.CanAnyRookMove(chess.White) {
		t.Error("White should be able to move at least one rook")
	}
}

func TestCanAnyRookMove_CaptureAvailable(t *testing.T) {
	// White rook can capture black piece
	board, _ := chess.ParseFEN("RN3k2/pN6/8/8/8/8/5K2/8 w - - 0 1")
	if !board.CanAnyRookMove(chess.White) {
		t.Error("White rook should be able to capture black pawn")
	}

	// Black rook can capture white piece
	board, _ = chess.ParseFEN("8/8/4k3/8/3K4/8/6nP/6nr b - - 0 1")
	if !board.CanAnyRookMove(chess.Black) {
		t.Error("Black rook should be able to capture white pawn")
	}
}

func TestCanAnyRookMoveSafe_StartingPosition(t *testing.T) {
	board := chess.NewStartingPosition()

	// At the very start, rooks are blocked by pawns
	if board.CanAnyRookMoveSafe(chess.White) {
		t.Error("White rooks should not be able to move from starting position")
	}
	if board.CanAnyRookMoveSafe(chess.Black) {
		t.Error("Black rooks should not be able to move from starting position")
	}
}

func TestCanAnyRookMoveSafe_SomeBlockedSomeFree(t *testing.T) {
	// White rook on a1 blocked, rook on h1 free
	board, _ := chess.ParseFEN("RN3K1R/NN6/8/8/8/8/8/5k2 w - - 0 1")
	if !board.CanAnyRookMoveSafe(chess.White) {
		t.Error("White should be able to move at least one rook")
	}
}

func TestCanAnyRookMoveSafe_CantMoveCouseCheck(t *testing.T) {
	// White rook has clear path to move but king would be left in check
	board, _ := chess.ParseFEN("8/8/k7/1b6/8/3R4/4K3/8 w - - 0 1")
	if board.CanAnyRookMoveSafe(chess.White) {
		t.Error("White rook shouldn't be able to move")
	}

	// Black rook has clear path to move but king would be left in check
	board, _ = chess.ParseFEN("8/8/k7/1r6/8/3B4/4K3/8 b - - 0 1")
	if board.CanAnyRookMoveSafe(chess.Black) {
		t.Error("Black rook shouldn't be able to move")
	}
}
