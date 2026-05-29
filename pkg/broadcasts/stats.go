package broadcasts

import (
	"encoding/json"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameStatJSON is the structure stored in broadcast_games.stats (JSONB).
// Keep field names stable — they are persisted to the database.
type GameStatJSON struct {
	Player1Score  int    `json:"p1_score"`
	Player2Score  int    `json:"p2_score"`
	Winner        int    `json:"winner"`  // 0=p1, 1=p2, -1=tie
	Player1Bingos int    `json:"p1_bingos"`
	Player2Bingos int    `json:"p2_bingos"`
	MaxPlayScore  int    `json:"max_play_score"`
	MaxPlayWord   string `json:"max_play_word"`
	MoveCount     int    `json:"move_count"`
	WalkOffBingo  bool   `json:"walk_off_bingo"`
	Player1Rating int    `json:"p1_rating"`
	Player2Rating int    `json:"p2_rating"`
}

// MarshalGameStat serializes a GameStatJSON to JSONB bytes.
func MarshalGameStat(s *GameStatJSON) ([]byte, error) {
	return json.Marshal(s)
}

// UnmarshalGameStat deserializes JSONB bytes into a GameStatJSON.
func UnmarshalGameStat(data []byte) (*GameStatJSON, error) {
	var s GameStatJSON
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ComputeGameStat derives per-game summary statistics from a completed GameDocument.
// p1Name/p2Name are the canonical player names from the broadcast_games DB row.
// p1Rating and p2Rating are from the feed cache; pass 0 if unknown.
// The doc player order may differ from the DB row order (e.g. when the annotator
// entered the game with the first-mover at index 0, but the DB has a different
// player as player1). Name matching corrects for this.
func ComputeGameStat(doc *ipc.GameDocument, p1Name, p2Name string, p1Rating, p2Rating int) *GameStatJSON {
	// Find which doc player index corresponds to the DB's player1.
	// Default to 0/1; flip to 1/0 when doc.Players[1] matches p1Name.
	p1Idx, p2Idx := 0, 1
	if len(doc.Players) >= 2 && doc.Players[1].Nickname == p1Name {
		p1Idx, p2Idx = 1, 0
	}

	s := &GameStatJSON{
		Player1Rating: p1Rating,
		Player2Rating: p2Rating,
	}

	if len(doc.CurrentScores) >= 2 {
		s.Player1Score = int(doc.CurrentScores[p1Idx])
		s.Player2Score = int(doc.CurrentScores[p2Idx])
	}

	// Map the doc winner index to 0=player1, 1=player2, -1=tie.
	switch int(doc.GetWinner()) {
	case p1Idx:
		s.Winner = 0
	case p2Idx:
		s.Winner = 1
	default:
		s.Winner = -1
	}

	lastTileMoveIdx := -1
	lastTileMoveIsBingo := false

	for i, evt := range doc.Events {
		switch evt.GetType() {
		case ipc.GameEvent_TILE_PLACEMENT_MOVE:
			s.MoveCount++
			pi := int(evt.GetPlayerIndex())
			if pi == p1Idx {
				if evt.IsBingo {
					s.Player1Bingos++
				}
			} else {
				if evt.IsBingo {
					s.Player2Bingos++
				}
			}
			sc := int(evt.GetScore())
			if sc > s.MaxPlayScore {
				s.MaxPlayScore = sc
				if len(evt.WordsFormedFriendly) > 0 {
					s.MaxPlayWord = evt.WordsFormedFriendly[0]
				}
			}
			lastTileMoveIdx = i
			lastTileMoveIsBingo = evt.IsBingo
		case ipc.GameEvent_EXCHANGE:
			s.MoveCount++
		}
	}

	// Walk-off bingo: the final tile-placement move was a bingo that ended the game.
	// We detect this by checking if any END_RACK_PTS event follows the last tile placement.
	if lastTileMoveIdx >= 0 && lastTileMoveIsBingo {
		for _, evt := range doc.Events[lastTileMoveIdx+1:] {
			if evt.GetType() == ipc.GameEvent_END_RACK_PTS {
				s.WalkOffBingo = true
				break
			}
		}
	}

	return s
}
