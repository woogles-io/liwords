package gameplay

import (
	"errors"
)

var errAlreadyOpenReq = errors.New("You already have an open abort request")

// SoughtGameStore is an interface for getting a sought game.
type AbortRequestStore interface {
	SaveRequest(gameID, userID, username string) string // returns a request ID
	AlreadyRequestedForGame(gameID, userID string) bool // has userID already requested an abort for this game?
	AcceptRequest(requestID, userID string)             // would check to make sure userID is actually the receiver of requestID
}
