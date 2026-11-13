package push

import (
	"encoding/json"
)

// turnMessage is what the service worker (liwords-ui/public/sw.js) shows.
// Deadline is the last moment (unix ms) the player can move; the service
// worker says it in the device's own time, so it never goes stale.
type turnMessage struct {
	GameID   string `json:"gameId"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Deadline int64  `json:"deadline"`
}

// TurnMessage is the push for a correspondence turn, worded as the
// notification an open game tab shows (showTurnNotification in
// liwords-ui/src/utils/notifications.ts).
func TurnMessage(gameID string, opponent string, deadline int64) []byte {
	if opponent == "" {
		opponent = "Opponent"
	}
	b, _ := json.Marshal(turnMessage{
		GameID:   gameID,
		Title:    "Your turn vs " + opponent,
		Body:     "It's your turn to play",
		Deadline: deadline,
	})
	return b
}
