package entity

import "time"

// RecentGamesPage picks a page of a player's game history by position rather
// than offset: the games just older than BeforeGameID, or than Before (a jump
// to a time), or just newer than AfterGameID, or than After (a time).
type RecentGamesPage struct {
	BeforeGameID string
	Before       time.Time
	AfterGameID  string
	After        time.Time
}
