package internal

import (
	"strings"
	"sync"
	"testing"
	"time"

	chess "github.com/zefir/szaszki-go-backend/internal/chessengine"
)

// Mock Client for testing
type MockClient struct {
	UserID           uint32
	CurrentlyPlaying bool
	Messages         []MessageLog
	mu               sync.Mutex
}

type MessageLog struct {
	Command uint8
	Data    []byte
}

func (m *MockClient) WriteMsg(cmd uint8, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, MessageLog{Command: cmd, Data: data})
	return nil
}

func (m *MockClient) GetMessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Messages)
}

func (m *MockClient) GetLastMessage() *MessageLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Messages) == 0 {
		return nil
	}
	return &m.Messages[len(m.Messages)-1]
}

// Create a test game session with mock clients
func createFullTestGameSession() (*GameSession, *MockClient, *MockClient) {
	whitePlayer := &MockClient{UserID: 1, CurrentlyPlaying: false}
	blackPlayer := &MockClient{UserID: 2, CurrentlyPlaying: false}

	return &GameSession{
		ID:           1,
		Players:      []*Client{{UserID: 1}, {UserID: 2}},
		Mode:         1,
		Board:        chess.NewStartingPosition(),
		BoardHistory: []chess.Board{},
		MoveHistory:  []chess.Move{},
		SideToMove:   chess.White,
		MoveChannel:  make(chan PlayerMove, 100),
		GameActive:   true,
		WhiteTime:    10 * time.Minute,
		BlackTime:    10 * time.Minute,
		WhiteCards:   []int8{6, 6, 6, 5, 5},
		BlackCards:   []int8{6, 6, 6, 5, 5},
		WhiteHp:      3,
		BlackHp:      3,
		LastMoveTime: time.Now(),
	}, whitePlayer, blackPlayer
}

// Play a move in the game
func playMove(g *GameSession, player *MockClient, from, to, promote int8, cardsToReroll [5]int8) {
	move := PlayerMove{
		From:          from,
		To:            to,
		PromoteTo:     promote,
		CardsToReroll: cardsToReroll,
		Player:        &Client{UserID: player.UserID},
	}
	g.MoveChannel <- move
}

// Helper function to convert chess.Move to board indices
func sanToMoveIndices(board *chess.Board, san string) (chess.Move, error) {
	return chess.SANToMove(board, san)
}

// Test complete games from PGN files
func TestFullGames_FromPGN(t *testing.T) {
	// Load PGN test cases
	tests, err := chess.LoadPGNTests("../testdata/games.pgn")
	if err != nil {
		t.Fatalf("Failed to load PGN tests: %v", err)
	}

	cfg := setupTestConfig()

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			// Parse the PGN to get moves
			games, err := chess.ParsePGNReader(strings.NewReader(tc.PGN))
			if err != nil {
				t.Fatalf("Failed to parse PGN: %v", err)
			}
			if len(games) != 1 {
				t.Fatalf("Expected 1 game, got %d", len(games))
			}

			game := games[0]
			g, white, black := createFullTestGameSession()

			// Track game completion
			gameDone := make(chan bool, 1)
			moveCount := 0
			var moveError error

			// Run game in goroutine
			go func() {
				for {
					select {
					case move := <-g.MoveChannel:
						moveCount++

						removedIndexes := g.CardLogicRemoval(move.From, move.CardsToReroll, cfg)
						madeMove := chess.MakeMove(&g.Board, move.From, move.To, move.PromoteTo, false)
						g.MoveHistory = append(g.MoveHistory, madeMove)
						g.BoardHistory = append(g.BoardHistory, g.Board)
						g.CardLogicAdding(move.From, removedIndexes, cfg)

						g.SideToMove = 1 - g.SideToMove

						// Check if all moves are played
						if moveCount >= len(game.Moves) {
							gameDone <- true
							return
						}

						// Optional: check if game ended early (checkmate/stalemate)
						if g.shouldEndGame() {
							t.Logf("Game ended early at move %d", moveCount)
							gameDone <- true
							return
						}

					case <-time.After(5 * time.Second):
						moveError = err
						t.Errorf("Game timeout after %d moves", moveCount)
						gameDone <- false
						return
					}
				}
			}()

			// Convert PGN moves and play them
			tempBoard := chess.NewStartingPosition()
			currentPlayer := white

			for i, san := range game.Moves {
				// Convert SAN to move
				move, err := sanToMoveIndices(&tempBoard, san)
				if err != nil {
					t.Fatalf("Move %d (%s) conversion failed: %v", i+1, san, err)
				}

				// Play the move in the game session
				playMove(g, currentPlayer, move.From, move.To, move.Promotion, [5]int8{0, 0, 0, 0, 0})

				// Update temp board to track position
				chess.MakeMove(&tempBoard, move.From, move.To, move.Promotion, false)

				// Switch players
				if currentPlayer == white {
					currentPlayer = black
				} else {
					currentPlayer = white
				}

				// Small delay to allow processing
				//time.Sleep(1 * time.Microsecond)
			}

			// Wait for game to complete
			select {
			case done := <-gameDone:
				if !done {
					t.Fatal("Game did not complete properly")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("Game did not end within timeout")
			}

			if moveError != nil {
				t.Fatalf("Error during game: %v", moveError)
			}

			// Verify final position matches expected FEN
			engineCore := g.Board.ToFEN()
			expectedCore := tc.FinalFEN

			if engineCore != expectedCore {
				t.Errorf("Final FEN mismatch\nExpected: %s\nGot:      %s", expectedCore, engineCore)
			}

			// Verify move count
			if len(g.MoveHistory) != len(game.Moves) {
				t.Errorf("Expected %d moves, got %d", len(game.Moves), len(g.MoveHistory))
			}

			t.Logf("Game '%s' completed successfully with %d moves", tc.Name, len(g.MoveHistory))
		})
	}
}
