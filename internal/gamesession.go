package internal

import (
	"encoding/json"
	"log"
)

// GameSession holds game-specific state
type GameSession struct {
	ID      int
	Players []*ClientConn
	Mode    uint16
	// You can add board, turn, scores, etc.
}

type GameStartMsg struct {
	GameMode  uint16 `json:"game_mode"`
	PlayerIDs []int  `json:"player_ids"`
	GameID    int    `json:"game_id"`
}

func (g *GameSession) Run() {
	log.Printf("Game %d started!", g.ID)

	// Collect player IDs
	var playerIDs []int
	for _, p := range g.Players {
		p.CurrentlyPlaying = true
		playerIDs = append(playerIDs, int(p.UserID)) // assuming Player has an ID field of type int
	}

	msg := GameStartMsg{
		GameMode:  g.Mode, // assuming you have a Mode uint16 field in GameSession
		PlayerIDs: playerIDs,
		GameID:    g.ID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshaling game start message: %v", err)
		return
	}

	for _, player := range g.Players {
		err := WriteMsg(player.Conns, ServerCmds.GameStarted, data) // adapt WriteMsg to send data bytes
		if err != nil {
			log.Printf("error sending message to player %d: %v", player.UserID, err)
		}
	}
}
