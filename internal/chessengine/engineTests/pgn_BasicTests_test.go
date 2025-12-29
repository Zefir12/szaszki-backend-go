package chess

import (
	"strings"
	"testing"
	"time"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

func TestPGNGeneration(t *testing.T) {
	testCases := []struct {
		name          string
		moves         []chess.Move
		times         []time.Duration
		cards         [][]chess.CardSwap
		whiteCards    []int8
		blackCards    []int8
		expectedMoves string // Expected move text portion
	}{
		{
			name: "Simple opening",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e2-e4
				{From: 52, To: 36, Promotion: 0}, // e7-e5
			},
			times: []time.Duration{
				10*time.Minute - 2*time.Second,
				10*time.Minute - 3*time.Second,
			},
			cards:         [][]chess.CardSwap{{}, {}},
			whiteCards:    []int8{6, 6, 6, 5, 5},
			blackCards:    []int8{6, 6, 6, 5, 5},
			expectedMoves: "1. e4 { [%clk 9:58] } e5 { [%clk 9:57] }",
		},
		{
			name: "Scholar's mate",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e2-e4
				{From: 52, To: 36, Promotion: 0}, // e7-e5
				{From: 5, To: 26, Promotion: 0},  // f1-c4
				{From: 57, To: 42, Promotion: 0}, // b8-c6
				{From: 3, To: 39, Promotion: 0},  // d1-h5
				{From: 62, To: 45, Promotion: 0}, // g8-f6
				{From: 39, To: 53, Promotion: 0}, // h5-f7# (checkmate)
			},
			times: []time.Duration{
				10 * time.Minute,
				10 * time.Minute,
				9*time.Minute + 55*time.Second,
				9*time.Minute + 50*time.Second,
				9*time.Minute + 45*time.Second,
				9*time.Minute + 40*time.Second,
				9*time.Minute + 35*time.Second,
			},
			cards:         make([][]chess.CardSwap, 7),
			whiteCards:    []int8{6, 6, 6, 5, 5},
			blackCards:    []int8{6, 6, 6, 5, 5},
			expectedMoves: "1. e4",
		},
		{
			name: "Castling kingside",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e2-e4
				{From: 52, To: 36, Promotion: 0}, // e7-e5
				{From: 6, To: 21, Promotion: 0},  // g1-f3
				{From: 62, To: 45, Promotion: 0}, // g8-f6
				{From: 5, To: 26, Promotion: 0},  // f1-c4
				{From: 61, To: 34, Promotion: 0}, // f8-c5
				{From: 4, To: 6, Promotion: 0},   // e1-g1 (O-O)
			},
			times: []time.Duration{
				10 * time.Minute,
				10 * time.Minute,
				9*time.Minute + 55*time.Second,
				9*time.Minute + 50*time.Second,
				9*time.Minute + 45*time.Second,
				9*time.Minute + 40*time.Second,
				9*time.Minute + 35*time.Second,
			},
			cards:         make([][]chess.CardSwap, 7),
			whiteCards:    []int8{6, 6, 6, 5, 5},
			blackCards:    []int8{6, 6, 6, 5, 5},
			expectedMoves: "4. O-O",
		},
		{
			name: "Pawn promotion",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e2-e4
				{From: 51, To: 35, Promotion: 0}, // d7-d5
				{From: 28, To: 35, Promotion: 0}, // e4xd5
				{From: 59, To: 52, Promotion: 0}, // d8-d7
				{From: 35, To: 43, Promotion: 0}, // d5-d6
				{From: 52, To: 43, Promotion: 0}, // d7xd6
				{From: 11, To: 27, Promotion: 0}, // d2-d4
				{From: 48, To: 32, Promotion: 0}, // a7-a5
				{From: 27, To: 35, Promotion: 0}, // d4-d5
				{From: 32, To: 24, Promotion: 0}, // a5-a4
				{From: 35, To: 43, Promotion: 0}, // d5-d6
				{From: 24, To: 16, Promotion: 0}, // a4-a3
				{From: 43, To: 51, Promotion: 0}, // d6-d7+
				{From: 60, To: 52, Promotion: 0}, // e8-d8
				{From: 51, To: 59, Promotion: 4}, // d7-d8=Q+
			},
			times:         make([]time.Duration, 15),
			cards:         make([][]chess.CardSwap, 15),
			whiteCards:    []int8{6, 6, 6, 5, 5},
			blackCards:    []int8{6, 6, 6, 5, 5},
			expectedMoves: "8. d8=Q+",
		},
		{
			name: "With card swaps",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e2-e4
				{From: 52, To: 36, Promotion: 0}, // e7-e5
			},
			times: []time.Duration{
				9*time.Minute + 58*time.Second,
				9*time.Minute + 55*time.Second,
			},
			cards: [][]chess.CardSwap{
				{
					{IndexReplaced: 0, NewCard: 5},
					{IndexReplaced: 2, NewCard: 4},
				},
				{
					{IndexReplaced: 0, NewCard: 3},
				},
			},
			whiteCards:    []int8{6, 6, 6, 5, 5},
			blackCards:    []int8{6, 6, 6, 5, 5},
			expectedMoves: "1. e4 { [%clk 9:58] Cards: Pawn→Knight, Pawn→Bishop }",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgnGen := chess.NewPGNGenerator("TestWhite", "TestBlack")
			pgnGen.Result = "1-0"
			pgnGen.TimeControl = "600+0"

			pgn := pgnGen.GeneratePGN(
				tc.moves,
				tc.times,
				tc.cards,
				tc.whiteCards,
				tc.blackCards,
			)

			// Check that PGN contains required headers
			requiredHeaders := []string{
				"[Event ",
				"[Site ",
				"[Date ",
				"[White \"TestWhite\"]",
				"[Black \"TestBlack\"]",
				"[Result \"1-0\"]",
			}

			for _, header := range requiredHeaders {
				if !strings.Contains(pgn, header) {
					t.Errorf("PGN missing required header: %s", header)
				}
			}

			// Check that expected move text appears
			if !strings.Contains(pgn, tc.expectedMoves) {
				t.Errorf("PGN does not contain expected moves.\nExpected substring: %s\nGot PGN:\n%s",
					tc.expectedMoves, pgn)
			}

			// Verify PGN ends with result
			if !strings.HasSuffix(strings.TrimSpace(pgn), "1-0") {
				t.Errorf("PGN should end with result")
			}
		})
	}
}

func TestPGNRoundTrip(t *testing.T) {
	// Test that we can generate PGN and it's valid enough to parse back
	testCases := []struct {
		name  string
		moves []chess.Move
	}{
		{
			name: "Italian game opening",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e4
				{From: 52, To: 36, Promotion: 0}, // e5
				{From: 6, To: 21, Promotion: 0},  // Nf3
				{From: 57, To: 42, Promotion: 0}, // Nc6
				{From: 5, To: 26, Promotion: 0},  // Bc4
				{From: 61, To: 34, Promotion: 0}, // Bc5
			},
		},
		{
			name: "Queen's gambit declined",
			moves: []chess.Move{
				{From: 11, To: 27, Promotion: 0}, // d4
				{From: 51, To: 35, Promotion: 0}, // d5
				{From: 10, To: 26, Promotion: 0}, // c4
				{From: 52, To: 36, Promotion: 0}, // e6
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgnGen := chess.NewPGNGenerator("White", "Black")
			pgnGen.Result = "*"

			times := make([]time.Duration, len(tc.moves))
			for i := range times {
				times[i] = 10 * time.Minute
			}
			cards := make([][]chess.CardSwap, len(tc.moves))

			pgn := pgnGen.GeneratePGN(
				tc.moves,
				times,
				cards,
				[]int8{6, 6, 6, 5, 5},
				[]int8{6, 6, 6, 5, 5},
			)

			// Verify we can reconstruct the position
			board := chess.NewStartingPosition()
			for i, move := range tc.moves {
				if !chess.IsMoveLegal(&board, move.From, move.To, move.Promotion) {
					t.Fatalf("Move %d is illegal: from=%d to=%d", i+1, move.From, move.To)
				}
				chess.MakeMove(&board, move.From, move.To, move.Promotion, false)
				board.FlipSideToMove()
			}

			// Basic sanity check - PGN should be multi-line
			lines := strings.Split(pgn, "\n")
			if len(lines) < 5 {
				t.Errorf("PGN seems too short, got %d lines:\n%s", len(lines), pgn)
			}
		})
	}
}

func TestClockTimeFormatting(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		expected string
	}{
		{10 * time.Minute, "10:00"},
		{9*time.Minute + 58*time.Second, "9:58"},
		{1*time.Minute + 5*time.Second, "1:05"},
		{30 * time.Second, "0:30"},
		{1 * time.Second, "0:01"},
		{0 * time.Second, "0:00"},
		{1*time.Hour + 30*time.Minute + 45*time.Second, "1:30:45"},
	}

	pgnGen := chess.NewPGNGenerator("W", "B")

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			// Generate a simple PGN with this time
			moves := []chess.Move{{From: 12, To: 28, Promotion: 0}}
			times := []time.Duration{tc.duration}
			cards := [][]chess.CardSwap{{}}

			pgn := pgnGen.GeneratePGN(
				moves,
				times,
				cards,
				[]int8{6},
				[]int8{6},
			)

			expectedClk := "[%clk " + tc.expected + "]"
			if !strings.Contains(pgn, expectedClk) {
				t.Errorf("Expected clock time %s not found in PGN:\n%s", expectedClk, pgn)
			}
		})
	}
}

func TestCardSwapFormatting(t *testing.T) {
	testCases := []struct {
		name     string
		swaps    []chess.CardSwap
		expected string
	}{
		{
			name:     "No swaps",
			swaps:    []chess.CardSwap{},
			expected: "",
		},
		{
			name: "Single swap",
			swaps: []chess.CardSwap{
				{IndexReplaced: 0, NewCard: chess.CardKing},
			},
			expected: "Cards: Pawn→King",
		},
		{
			name: "Multiple swaps",
			swaps: []chess.CardSwap{
				{IndexReplaced: 0, NewCard: 5},
				{IndexReplaced: 1, NewCard: 4},
				{IndexReplaced: 2, NewCard: 3},
			},
			expected: "Cards: Pawn→Knight, Pawn→Bishop, Bishop→Rook",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgnGen := chess.NewPGNGenerator("W", "B")
			moves := []chess.Move{{From: 12, To: 28, Promotion: 0}}
			times := []time.Duration{10 * time.Minute}
			cards := [][]chess.CardSwap{tc.swaps}

			pgn := pgnGen.GeneratePGN(
				moves,
				times,
				cards,
				[]int8{6, 6, 4, 5, 5},
				[]int8{6, 6, 6, 5, 5},
			)

			if !strings.Contains(pgn, tc.expected) {
				t.Errorf("Expected card format not found.\nExpected: %s\nGot PGN:\n%s",
					tc.expected, pgn)
			}
		})
	}
}

func TestPGNHeaders(t *testing.T) {
	pgnGen := chess.NewPGNGenerator("Magnus Carlsen", "Hikaru Nakamura")
	pgnGen.Event = "World Championship"
	pgnGen.Site = "Dubai"
	pgnGen.Date = "2025.12.29"
	pgnGen.Result = "1/2-1/2"
	pgnGen.TimeControl = "5400+30"

	moves := []chess.Move{{From: 12, To: 28, Promotion: 0}}
	times := []time.Duration{90 * time.Minute}
	cards := [][]chess.CardSwap{{}}

	pgn := pgnGen.GeneratePGN(
		moves,
		times,
		cards,
		[]int8{6, 6, 6, 5, 5},
		[]int8{6, 6, 6, 5, 5},
	)

	expectedHeaders := map[string]string{
		"Event":       "World Championship",
		"Site":        "Dubai",
		"Date":        "2025.12.29",
		"White":       "Magnus Carlsen",
		"Black":       "Hikaru Nakamura",
		"Result":      "1/2-1/2",
		"TimeControl": "5400+30",
	}

	for key, value := range expectedHeaders {
		expected := "[" + key + " \"" + value + "\"]"
		if !strings.Contains(pgn, expected) {
			t.Errorf("Expected header not found: %s\nGot PGN:\n%s", expected, pgn)
		}
	}
}

func TestEmptyGame(t *testing.T) {
	pgnGen := chess.NewPGNGenerator("White", "Black")
	pgnGen.Result = "*"

	pgn := pgnGen.GeneratePGN(
		[]chess.Move{},
		[]time.Duration{},
		[][]chess.CardSwap{},
		[]int8{6, 6, 6, 5, 5},
		[]int8{6, 6, 6, 5, 5},
	)

	// Should still have headers
	if !strings.Contains(pgn, "[White \"White\"]") {
		t.Error("Empty game PGN missing headers")
	}

	// Should end with result
	if !strings.HasSuffix(strings.TrimSpace(pgn), "*") {
		t.Error("Empty game should end with * result")
	}
}

func TestPGNRoundTripWithRealGame(t *testing.T) {
	// Real game from chess.com
	inputPGN := `[Event "Live Chess"]
[Site "Chess.com"]
[Date "2025.09.09"]
[Round "-"]
[White "only_strong_moves"]
[Black "AlexKhripachenko"]
[Result "0-1"]
[CurrentPosition "8/8/3q3P/8/1k6/6Q1/6PK/4q3 w - - 0 67"]
[Timezone "UTC"]
[ECO "C45"]
[ECOUrl "https://www.chess.com/openings/Scotch-Game-Classical-Potter-Variation-5...Bb6-6.Qe2"]
[UTCDate "2025.09.09"]
[UTCTime "18:17:04"]
[WhiteElo "2898"]
[BlackElo "3009"]
[TimeControl "180"]
[Termination "AlexKhripachenko won on time"]
[StartTime "18:17:04"]
[EndDate "2025.09.09"]
[EndTime "18:23:23"]
[Link "https://www.chess.com/analysis/game/live/142926290362/analysis"]

1. e4 e5 2. Nf3 Nc6 3. d4 exd4 4. Nxd4 Bc5 5. Nb3 Bb6 6. Qe2 d6 7. Nc3 a5 8. a3 a4 9. Nd2 Nd4 10. Qd1 Nf6 11. Nc4 Bc5 12. Bg5 h6 13. Bh4 O-O 14. Bd3 Re8 15. O-O c6 16. Re1 b5 17. Ne3 Ne6 18. Qf3 Ng5 19. Qd1 Qb6 20. Bxg5 hxg5 21. Kh1 Be6 22. Qf3 b4 23. axb4 Qxb4 24. Ned1 Bg4 25. Qg3 Bxd1 26. Raxd1 Qxb2 27. e5 Rxe5 28. Rxe5 dxe5 29. Ne4 Nxe4 30. Bxe4 a3 31. Qh3 g6 32. Bxg6 fxg6 33. Qe6+ Kh8 34. Qf6+ Kg8 35. Qxg6+ Kh8 36. Qh6+ Kg8 37. Qxg5+ Kh8 38. Qh6+ Kg8 39. Qxc6 Rf8 40. Qd5+ Rf7 41. Qxc5 a2 42. Qa5 Qxc2 43. Rg1 Rxf2 44. h3 Qb2 45. Qd8+ Kf7 46. Qd7+ Kf6 47. Qd8+ Kf5 48. Qf8+ Ke4 49. Qa8+ Ke3 50. Qa7+ Ke2 51. Qa6+ Kd2 52. Qa5+ Kc2 53. Qc5+ Qc3 54. Qxf2+ Kb3 55. Qb6+ Ka3 56. Qa7+ Kb2 57. Qf2+ Ka3 58. Re1 a1=Q 59. Rxa1+ Qxa1+ 60. Kh2 e4 61. Qg3+ Kb4 62. h4 Qd4 63. h5 e3 64. Qh3 e2 65. h6 Qd6+ 66. Qg3 e1=Q 0-1`

	expectedFinalFEN := "8/8/3q3P/8/1k6/6Q1/6PK/4q3 w - - 0 67"

	// Parse the PGN
	games, err := chess.ParsePGNReader(strings.NewReader(inputPGN))
	if err != nil {
		t.Fatalf("Failed to parse input PGN: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(games))
	}

	game := games[0]

	// Play through the moves and record them
	board := chess.NewStartingPosition()
	var moveHistory []chess.Move

	for i, san := range game.Moves {
		move, err := chess.SANToMove(&board, san)
		if err != nil {
			t.Fatalf("Move %d (%s) failed: %v", i+1, san, err)
		}

		moveHistory = append(moveHistory, move)
		chess.MakeMove(&board, move.From, move.To, move.Promotion, false)
		board.FlipSideToMove()
	}

	// Verify we reached the expected position
	engineCore := strings.Join(strings.Fields(board.ToFEN())[:4], " ")
	expectedCore := strings.Join(strings.Fields(expectedFinalFEN)[:4], " ")
	if engineCore != expectedCore {
		t.Fatalf("Final FEN mismatch after parsing\nExpected: %s\nGot:      %s", expectedCore, engineCore)
	}

	// Generate PGN from our move history
	pgnGen := chess.NewPGNGenerator("only_strong_moves", "AlexKhripachenko")
	pgnGen.Event = "Live Chess"
	pgnGen.Site = "Chess.com"
	pgnGen.Date = "2025.09.09"
	pgnGen.Result = "0-1"
	pgnGen.TimeControl = "180"

	// Create dummy time and card data (since we don't have actual times from the PGN)
	timeHistory := make([]time.Duration, len(moveHistory))
	for i := range timeHistory {
		timeHistory[i] = 3 * time.Minute // Dummy time
	}
	cardHistory := make([][]chess.CardSwap, len(moveHistory))

	generatedPGN := pgnGen.GeneratePGN(
		moveHistory,
		timeHistory,
		cardHistory,
		[]int8{6, 6, 6, 5, 5}, // Dummy cards
		[]int8{6, 6, 6, 5, 5},
	)

	t.Logf("Generated PGN:\n%s\n", generatedPGN)

	// Parse our generated PGN
	generatedGames, err := chess.ParsePGNReader(strings.NewReader(generatedPGN))
	if err != nil {
		t.Fatalf("Failed to parse generated PGN: %v", err)
	}
	if len(generatedGames) != 1 {
		t.Fatalf("Expected 1 game from generated PGN, got %d", len(generatedGames))
	}

	generatedGame := generatedGames[0]

	// Compare move counts
	if len(generatedGame.Moves) != len(game.Moves) {
		t.Fatalf("Move count mismatch. Original: %d, Generated: %d", len(game.Moves), len(generatedGame.Moves))
	}

	// Play through generated PGN moves
	board2 := chess.NewStartingPosition()
	for i, san := range generatedGame.Moves {
		move, err := chess.SANToMove(&board2, san)
		if err != nil {
			t.Fatalf("Generated move %d (%s) failed: %v", i+1, san, err)
		}
		chess.MakeMove(&board2, move.From, move.To, move.Promotion, false)
		board2.FlipSideToMove()
	}

	// Verify both end at the same position
	finalFEN1 := strings.Join(strings.Fields(board.ToFEN())[:4], " ")
	finalFEN2 := strings.Join(strings.Fields(board2.ToFEN())[:4], " ")

	if finalFEN1 != finalFEN2 {
		t.Fatalf("Final positions don't match\nOriginal game: %s\nGenerated game: %s", finalFEN1, finalFEN2)
	}

	// Verify we match expected final position
	if finalFEN2 != expectedCore {
		t.Fatalf("Generated game final position mismatch\nExpected: %s\nGot:      %s", expectedCore, finalFEN2)
	}

	t.Logf("✓ Successfully round-tripped %d moves", len(moveHistory))
	t.Logf("✓ Final position: %s", finalFEN2)
}

func TestPGNRoundTripMultipleGames(t *testing.T) {
	tests, err := chess.LoadPGNTests("../../../testdata/games.pgn")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			// Parse original PGN
			games, err := chess.ParsePGNReader(strings.NewReader(tc.PGN))
			if err != nil {
				t.Fatal(err)
			}
			if len(games) != 1 {
				t.Fatalf("Expected 1 game, got %d", len(games))
			}

			game := games[0]

			// Play through and record moves
			board := chess.NewStartingPosition()
			var moveHistory []chess.Move

			for i, san := range game.Moves {
				move, err := chess.SANToMove(&board, san)
				if err != nil {
					t.Fatalf("Move %d (%s) failed: %v", i+1, san, err)
				}
				moveHistory = append(moveHistory, move)
				chess.MakeMove(&board, move.From, move.To, move.Promotion, false)
				board.FlipSideToMove()
			}

			// Verify original position
			originalCore := strings.Join(strings.Fields(board.ToFEN())[:4], " ")
			expectedCore := strings.Join(strings.Fields(tc.FinalFEN)[:4], " ")
			if originalCore != expectedCore {
				t.Fatalf("Original game FEN mismatch\nExpected: %s\nGot:      %s", expectedCore, originalCore)
			}

			// Generate new PGN
			pgnGen := chess.NewPGNGenerator(
				"White",
				"Black",
			)

			// Create dummy time and card data
			timeHistory := make([]time.Duration, len(moveHistory))
			for i := range timeHistory {
				timeHistory[i] = 3 * time.Minute
			}
			cardHistory := make([][]chess.CardSwap, len(moveHistory))

			generatedPGN := pgnGen.GeneratePGN(
				moveHistory,
				timeHistory,
				cardHistory,
				[]int8{6, 6, 6, 5, 5},
				[]int8{6, 6, 6, 5, 5},
			)

			// Parse generated PGN
			generatedGames, err := chess.ParsePGNReader(strings.NewReader(generatedPGN))
			if err != nil {
				t.Fatalf("Failed to parse generated PGN: %v", err)
			}
			if len(generatedGames) != 1 {
				t.Fatalf("Expected 1 generated game, got %d", len(generatedGames))
			}

			generatedGame := generatedGames[0]

			// Play through generated moves
			board2 := chess.NewStartingPosition()
			for i, san := range generatedGame.Moves {
				move, err := chess.SANToMove(&board2, san)
				if err != nil {
					t.Fatalf("Generated move %d (%s) failed: %v", i+1, san, err)
				}
				chess.MakeMove(&board2, move.From, move.To, move.Promotion, false)
				board2.FlipSideToMove()
			}

			// Verify final positions match
			generatedCore := strings.Join(strings.Fields(board2.ToFEN())[:4], " ")
			if originalCore != generatedCore {
				t.Fatalf("Round-trip FEN mismatch\nOriginal:  %s\nGenerated: %s", originalCore, generatedCore)
			}

			t.Logf("✓ Round-tripped %d moves successfully", len(moveHistory))
		})
	}
}

func TestPGNGenerationAccuracy(t *testing.T) {
	// Test specific move notations
	testCases := []struct {
		name     string
		moves    []chess.Move
		expected []string // Expected SAN notation for each move
	}{
		{
			name: "Basic pawn and knight moves",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e4
				{From: 52, To: 36, Promotion: 0}, // e5
				{From: 6, To: 21, Promotion: 0},  // Nf3
				{From: 57, To: 42, Promotion: 0}, // Nc6
			},
			expected: []string{"e4", "e5", "Nf3", "Nc6"},
		},
		{
			name: "Captures",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e4
				{From: 51, To: 35, Promotion: 0}, // d5
				{From: 28, To: 35, Promotion: 0}, // exd5
			},
			expected: []string{"e4", "d5", "exd5"},
		},
		{
			name: "Castling",
			moves: []chess.Move{
				{From: 12, To: 28, Promotion: 0}, // e4
				{From: 52, To: 36, Promotion: 0}, // e5
				{From: 6, To: 21, Promotion: 0},  // Nf3
				{From: 62, To: 45, Promotion: 0}, // Nf6
				{From: 5, To: 26, Promotion: 0},  // Bc4
				{From: 61, To: 34, Promotion: 0}, // Bc5
				{From: 4, To: 6, Promotion: 0},   // O-O
			},
			expected: []string{"e4", "e5", "Nf3", "Nf6", "Bc4", "Bc5", "O-O"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgnGen := chess.NewPGNGenerator("White", "Black")
			pgnGen.Result = "*"

			timeHistory := make([]time.Duration, len(tc.moves))
			for i := range timeHistory {
				timeHistory[i] = 10 * time.Minute
			}
			cardHistory := make([][]chess.CardSwap, len(tc.moves))

			generatedPGN := pgnGen.GeneratePGN(
				tc.moves,
				timeHistory,
				cardHistory,
				[]int8{6, 6, 6, 5, 5},
				[]int8{6, 6, 6, 5, 5},
			)

			// Check each expected move appears in the PGN
			for i, expectedMove := range tc.expected {
				if !strings.Contains(generatedPGN, expectedMove) {
					t.Errorf("Move %d: Expected notation '%s' not found in PGN:\n%s",
						i+1, expectedMove, generatedPGN)
				}
			}
		})
	}
}
