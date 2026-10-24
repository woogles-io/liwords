package league

import (
	"github.com/domino14/word-golib/tilemapping"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
)

// tilesPlayedByPlayer counts the tiles each player has on the board, from a
// game's events. A phony taken off the board takes its tiles back.
func tilesPlayedByPlayer(events []*macondopb.GameEvent, tm *tilemapping.TileMapping) [2]int32 {
	var counts, lastPlay [2]int32
	for _, evt := range events {
		p := evt.PlayerIndex
		if p > 1 {
			continue
		}
		switch evt.Type {
		case macondopb.GameEvent_TILE_PLACEMENT_MOVE:
			mls, err := tilemapping.ToMachineLetters(evt.PlayedTiles, tm)
			if err != nil {
				continue
			}
			n := int32(0)
			for _, ml := range mls {
				if ml != 0 {
					n++
				}
			}
			counts[p] += n
			lastPlay[p] = n
		case macondopb.GameEvent_PHONY_TILES_RETURNED:
			counts[p] -= lastPlay[p]
			lastPlay[p] = 0
		}
	}
	return counts
}

// tileProgress returns, for a game in progress, the tiles on the board for
// player 0 and player 1, and the tiles not yet played (bag and both racks:
// every tile of the distribution not on the board).
func tileProgress(h *macondopb.GameHistory, dist *tilemapping.LetterDistribution) (played [2]int32, unplayed int32) {
	played = tilesPlayedByPlayer(h.Events, dist.TileMapping())
	unplayed = int32(dist.NumTotalLetters()) - played[0] - played[1]
	return played, unplayed
}

// tileProgressFromBytes is tileProgress for a stored game history. ok is
// false when the history or its letter distribution cannot be read.
func tileProgressFromBytes(cfg *config.Config, historyBytes []byte) (played [2]int32, unplayed int32, ok bool) {
	if cfg == nil || len(historyBytes) == 0 {
		return played, 0, false
	}
	var h macondopb.GameHistory
	if err := proto.Unmarshal(historyBytes, &h); err != nil {
		log.Warn().Err(err).Msg("tile-progress-history")
		return played, 0, false
	}
	dist, err := tilemapping.GetDistribution(cfg.WGLConfig(), h.LetterDistribution)
	if err != nil {
		log.Warn().Err(err).Str("dist", h.LetterDistribution).Msg("tile-progress-distribution")
		return played, 0, false
	}
	played, unplayed = tileProgress(&h, dist)
	return played, unplayed, true
}
