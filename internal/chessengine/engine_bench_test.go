package chess

import (
	"strings"
	"testing"
)

// Benchmark SANToMove parsing
func BenchmarkSANToMove(b *testing.B) {
	board := NewStartingPosition()
	MakeMove(&board, 12, 28, 0, false) // e2-e4

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SANToMove(&board, "e5")
	}
}

// Benchmark MakeMove
func BenchmarkMakeMove(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board := NewStartingPosition()
		MakeMove(&board, 12, 28, 0, false) // e2-e4
	}
}

// Benchmark IsMoveLegal
func BenchmarkIsMoveLegal(b *testing.B) {
	board := NewStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsMoveLegal(&board, 12, 28, 0) // e2-e4
	}
}

// Benchmark IsPseudoLegal
func BenchmarkIsPseudoLegal(b *testing.B) {
	board := NewStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsPseudoLegal(&board, 12, 28, Pawn, White) // e2-e4
	}
}

// Benchmark IsSquareAttacked
func BenchmarkIsSquareAttacked(b *testing.B) {
	board := NewStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsSquareAttacked(28, &board, Black) // is e4 attacked by black?
	}
}

// Benchmark full game replay
func BenchmarkFullGameReplay(b *testing.B) {
	pgn := `1. e4 e5 2. d4 exd4 3. Qxd4 Nc6 4. Qd1 Nf6 5. Nc3 Bb4 6. Bd3 O-O 7. Bd2 Re8 8.
f3 d5 9. Nge2 dxe4 10. Nxe4 Bxd2+ 11. Qxd2 Bf5 12. Nxf6+ Qxf6 13. O-O Rad8 14.
Rad1 Bxd3 15. cxd3 Ne5 16. d4 Nc4 17. Qc3 Ne3 18. d5 Nxf1 19. Qxf6 gxf6 20. Kxf1
Re5 21. Nf4 c6 22. d6 Kf8 23. b4 f5 24. a3 f6 25. Nh5 Kf7 26. f4 Rd5 27. Rxd5
cxd5 28. Kf2 Rxd6 29. Kg3 d4 30. Kh4 d3 31. Ng3 d2 32. Nxf5 Rd5 33. Ne3 Rd4 34.
Kg3 d1=Q 35. Nxd1 Rxd1 36. Kg4 Ra1 37. g3 Kg6 38. h4 Rxa3 39. h5+ Kg7 40. Kf5
Ra4 41. h6+ Kg8 42. Kxf6 Rxb4 43. f5 a5 44. g4 a4 45. g5 a3 46. g6 hxg6 47. fxg6
Rb6+ 48. Kf5 a2 49. h7+ Kg7 50. h8=Q+ Kxh8 51. g7+ Kxg7 52. Ke5 a1=Q+ 53. Kf5
Qf6+ 54. Ke4 Rb5 55. Ke3 Qe5+ 56. Kf3 Rb4 57. Kf2 Qf4+ 58. Ke2 Rb3 59. Kd1 Qe3
60. Kc2 Qc3+ 61. Kd1 Rb2 1/2-1/2`

	games, _ := ParsePGNReader(strings.NewReader(pgn))
	game := games[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board := NewStartingPosition()
		for _, san := range game.Moves {
			move, _ := SANToMove(&board, san)
			MakeMove(&board, move.From, move.To, move.Promotion, false)
		}
	}
}

// Benchmark different piece move generations
func BenchmarkPawnMoveGeneration(b *testing.B) {
	board := NewStartingPosition()
	occupied := board.Occupied[White] | board.Occupied[Black]
	empty := ^occupied

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SinglePawnPush(board.Pawns[White], empty, true)
		DoublePawnPush(board.Pawns[White], empty, true)
		PawnAttacks(board.Pawns[White], board.Occupied[Black], true)
	}
}

func BenchmarkIsInCheck(b *testing.B) {
	board := NewStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.IsInCheck(White)
	}
}

func BenchmarkBishopMoveGeneration(b *testing.B) {
	board := NewStartingPosition()
	occupied := board.Occupied[White] | board.Occupied[Black]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slidingAttacksBishop(2, occupied)
	}
}

func BenchmarkRookMoveGeneration(b *testing.B) {
	board := NewStartingPosition()
	occupied := board.Occupied[White] | board.Occupied[Black]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slidingAttacksRook(0, occupied)
	}
}

func BenchmarkQueenMoveGeneration(b *testing.B) {
	board := NewStartingPosition()
	occupied := board.Occupied[White] | board.Occupied[Black]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slidingAttacksQueen(3, occupied)
	}
}

// Benchmark board cloning
func BenchmarkBoardClone(b *testing.B) {
	board := NewStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = board.Clone()
	}
}

// Benchmark FEN generation
func BenchmarkToFEN(b *testing.B) {
	board := NewStartingPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.ToFEN()
	}
}

// Benchmark position after complex middle game
func BenchmarkComplexPositionLegalMoves(b *testing.B) {
	// Position after move 15 from the test game
	board := NewStartingPosition()
	moves := []struct{ from, to int8 }{
		{12, 28}, {52, 36}, {11, 27}, {36, 27}, {3, 27},
		{57, 42}, {27, 3}, {62, 45}, {1, 18}, {58, 25},
		{2, 19}, {60, 62}, {5, 11}, {56, 60}, {14, 21},
	}

	for _, m := range moves {
		MakeMove(&board, m.from, m.to, 0, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.HasAnyLegalMove(Black)
	}
}

func BenchmarkCanAnyPawnMove_NoPawns(b *testing.B) {
	board, _ := ParseFEN("rnbqkbnr/8/8/8/8/8/8/RNBQKBNR w KQkq - 0 1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CanAnyPawnMove(White)
	}
}

func BenchmarkCanAnyPawnMove_BlockedPawns(b *testing.B) {
	board, _ := ParseFEN("7k/8/pppppppp/nnnnnnnn/NNNNNNNN/PPPPPPPP/8/7K w - - 0 1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CanAnyPawnMove(White)
	}
}

func BenchmarkCanAnyPawnMove_ComplexPosition(b *testing.B) {
	board, _ := ParseFEN("r1b3r1/ppppkppP/2n2n2/1B2p3/4P1q1/b4N2/PPPP1P1P/RNB1QR1K w - - 5 11")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CanAnyPawnMove(White)
	}
}

func BenchmarkCanAnyPawnMove_EndgamePosition(b *testing.B) {
	board, _ := ParseFEN("8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CanAnyPawnMove(White)
	}
}

func BenchmarkCanAnyRookMove_NoRooks(b *testing.B) {
	board, _ := ParseFEN("1nbqkbn1/pppppppp/8/8/8/8/PPPPPPPP/1NBQKBN1 w - - 0 1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CanAnyRookMove(White)
	}
}

func BenchmarkCanAnyRookMove_ComplexPosition(b *testing.B) {
	board, _ := ParseFEN("r1b3r1/ppppkppP/2n2n2/1B2p3/4P1q1/b4N2/PPPP1P1P/RNB1QR1K w - - 5 11")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CanAnyRookMove(White)
	}
}

func BenchmarkCanAnyRookMoveSafe_NoRooks(b *testing.B) {
	board, _ := ParseFEN("1nbqkbn1/pppppppp/8/8/8/8/PPPPPPPP/1NBQKBN1 w - - 0 1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CanAnyRookMoveSafe(White)
	}
}

func BenchmarkCanAnyRookMoveSafe_ComplexPosition(b *testing.B) {
	board, _ := ParseFEN("r1b3r1/ppppkppP/2n2n2/1B2p3/4P1q1/b4N2/PPPP1P1P/RNB1QR1K w - - 5 11")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CanAnyRookMoveSafe(White)
	}
}
