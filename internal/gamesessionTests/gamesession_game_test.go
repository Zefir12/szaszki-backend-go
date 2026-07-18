package internal

import (
	"sync"
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

// func TestShouldEndGame(t *testing.T) {
// 	_ = setupTestConfig()

// 	g := createTestGameSession()
// 	b, _ := chess.ParseFEN("rnbqkbnr/pppppppp/5N2/8/8/8/PPPPPPPP/RNBQKB1R b KQkq - 0 2")
// 	g.Board = b.Clone()
// 	g.BlackCards = []int8{4, 4, 3, 3, 2}

// 	gameEnded := g.ShouldEndGame(chess.Black)

// 	if gameEnded {
// 		t.Error("Game shouldnt end")
// 	}
// 	if g.BlackHp != 2 {
// 		t.Error("Black should lost hp")
// 	}
// }
