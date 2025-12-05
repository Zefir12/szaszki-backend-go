package adminEndpoints

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/zefir/szaszki-go-backend/internal"
)

// GameInfo is the JSON shape returned to the admin frontend.
type GameInfo struct {
	ID        uint32   `json:"id"`
	Mode      uint16   `json:"mode"`
	PlayerIDs []uint32 `json:"playerIds"`

	GameActive bool   `json:"gameActive"`
	SideToMove string `json:"sideToMove"` // "w" or "b"

	WhiteTimeMs int64 `json:"whiteTimeMs"`
	BlackTimeMs int64 `json:"blackTimeMs"`

	WhiteHp int8 `json:"whiteHp"`
	BlackHp int8 `json:"blackHp"`

	WhiteCards []int8 `json:"whiteCards"`
	BlackCards []int8 `json:"blackCards"`

	HalfmoveClock  uint8  `json:"halfmoveClock"`
	FullmoveNumber uint16 `json:"fullmoveNumber"`

	MoveHistoryLen int `json:"moveHistoryLen"`

	LastMoveTime string `json:"lastMoveTime"` // RFC3339 or empty
}

func GetGamesEndpoint(w http.ResponseWriter, r *http.Request) {
	// get snapshot of games
	games := internal.GetGameKeeper().ListGames()

	resp := make([]GameInfo, 0, len(games))

	for _, g := range games {
		// Acquire read lock to safely read game fields
		g.Mu.RLock()

		// players info
		playerIDs := make([]uint32, 0, len(g.Players))
		for _, p := range g.Players {
			playerIDs = append(playerIDs, p.UserID)
		}

		side := "w"
		if g.SideToMove == 1 {
			side = "w"
		}

		var lastMoveISO string
		if !g.LastMoveTime.IsZero() {
			lastMoveISO = g.LastMoveTime.UTC().Format(time.RFC3339)
		}

		info := GameInfo{
			ID:        g.ID,
			Mode:      g.Mode,
			PlayerIDs: playerIDs,

			GameActive: g.GameActive,
			SideToMove: side,

			WhiteTimeMs: g.WhiteTime.Milliseconds(),
			BlackTimeMs: g.BlackTime.Milliseconds(),

			WhiteHp: g.WhiteHp,
			BlackHp: g.BlackHp,

			WhiteCards: g.WhiteCards,
			BlackCards: g.BlackCards,

			HalfmoveClock:  g.Board.HalfmoveClock,
			FullmoveNumber: g.Board.FullmoveNumber,

			MoveHistoryLen: len(g.MoveHistory),
			LastMoveTime:   lastMoveISO,
		}

		g.Mu.RUnlock()

		resp = append(resp, info)
	}

	// write JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
