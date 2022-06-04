package common

import (
	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

const (
	CurrentGameHistoryVersion = 2
)

// This contains functions for migrating a game in-place.
// The Macondo GameHistory will occasionally, hopefully very rarely,
// change schemas, and thus we need to be able to migrate when these
// change.

func MigrateGameHistory(gh *macondo.GameHistory) *macondo.GameHistory {
	if gh.Version < 2 {
		// Either 0 (unspecified) or 1
		// Migrate to v2.
		return migrateToV2(gh)
	}
	// Otherwise, return the history as is.
	return gh
}

func migrateToV2(gh *macondo.GameHistory) *macondo.GameHistory {
	// Version 2 of a macondo GameHistory works as such:
	// - id_auth should be set to `io.woogles`
	// - second_went_first is deprecated. If it is true, we need to flip
	// the order of the players instead and set it to false.
	//   -- this will also affect the `winner`
	// - the `nickname` field in each GameEvent of the history is deprecated.
	//    we should instead use player_index

	gh2 := proto.Clone(gh).(*macondo.GameHistory)

	if len(gh.Players) != 2 {
		log.Error().Interface("players", gh.Players).Str("gid", gh.Uid).
			Msg("bad-gh-players")
		return gh
	}

	gh2.Version = 2
	gh2.IdAuth = "io.woogles"

	if gh2.SecondWentFirst {
		gh2.SecondWentFirst = false
		gh2.Players[0], gh2.Players[1] = gh2.Players[1], gh2.Players[0]
		if gh2.Winner == 0 {
			gh2.Winner = 1
		} else if gh2.Winner == 1 {
			gh2.Winner = 0
		} // otherwise it's a tie
		if len(gh2.FinalScores) == 2 {
			gh2.FinalScores[0], gh2.FinalScores[1] = gh2.FinalScores[1], gh2.FinalScores[0]
		}
		if len(gh2.LastKnownRacks) == 2 {
			gh2.LastKnownRacks[0], gh2.LastKnownRacks[1] = gh2.LastKnownRacks[1], gh2.LastKnownRacks[0]
		}
	}

	nickname0, nickname1 := gh2.Players[0].Nickname, gh2.Players[1].Nickname
	for _, evt := range gh2.Events {
		if evt.Nickname == nickname1 {
			evt.PlayerIndex = 1
		} else if evt.Nickname == nickname0 {
			evt.PlayerIndex = 0 // technically not necessary because of protobuf but let's be explicit
		}
		evt.Nickname = ""
	}

	return gh2
}
