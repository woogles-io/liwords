package entity

import realtime "github.com/domino14/liwords/rpc/api/proto/realtime"

// Clubs and tournaments have a lot in common. A clubhouse, which is backed
// by a Club struct, creates club sessions, which are identical to tournaments.
type Club struct {
	// A unique ID for this club, not user-visible
	UUID string `json:"uuid"`
	Name string `json:"name"`
	// A case-insensitive slug for this club; URL will look like /club/{slug}
	Slug              string                `json:"slug"`
	Description       string                `json:"desc"`
	ExecutiveDirector string                `json:"execDirector"`
	Directors         *TournamentPersons    `json:"directors"`
	DefaultSettings   *realtime.GameRequest `json:"req"`
}
