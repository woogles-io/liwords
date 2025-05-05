package utilities

import (
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func rackConverter(rack string, index int, letterdist *tilemapping.LetterDistribution) []byte {
	mls, err := tilemapping.ToMachineLetters(rack, letterdist.TileMapping())
	if err != nil {
		panic(err)
	}
	return tilemapping.MachineWord(mls).ToByteArr()
}

func MacondoEvtToOMGEvt(evt *macondo.GameEvent, index int, letterdist *tilemapping.LetterDistribution) *ipc.GameEvent {
	cvt := &ipc.GameEvent{
		Rack:          rackConverter(evt.Rack, 0, letterdist),
		Type:          ipc.GameEvent_Type(evt.Type),
		Cumulative:    evt.Cumulative,
		Row:           evt.Row,
		Column:        evt.Column,
		Direction:     ipc.GameEvent_Direction(evt.Direction),
		Position:      evt.Position,
		PlayedTiles:   rackConverter(evt.PlayedTiles, 0, letterdist),
		Exchanged:     rackConverter(evt.Exchanged, 0, letterdist),
		Score:         evt.Score,
		Bonus:         evt.Bonus,
		EndRackPoints: evt.EndRackPoints,
		LostScore:     evt.LostScore,
		IsBingo:       evt.IsBingo,
		WordsFormed: lo.Map(evt.WordsFormed, func(rack string, index int) []byte {
			return rackConverter(rack, index, letterdist)
		}),
		WordsFormedFriendly: evt.WordsFormed,
		MillisRemaining:     evt.MillisRemaining,
		PlayerIndex:         evt.PlayerIndex,
	}

	return cvt
}

// helper functions to convert from the old GameHistory etc structs to
// GameDocuments. We can delete this after some time.

func ToGameDocument(g *entity.Game, cfg *config.Config) (*ipc.GameDocument, error) {
	letterdist, err := tilemapping.GetDistribution(cfg.WGLConfig(), g.History().LetterDistribution)
	if err != nil {
		return nil, err
	}

	gdoc := &ipc.GameDocument{
		Players: lo.Map(g.History().Players, func(p *macondo.PlayerInfo, idx int) *ipc.GameDocument_MinimalPlayerInfo {
			return &ipc.GameDocument_MinimalPlayerInfo{
				Nickname: p.Nickname,
				RealName: p.RealName,
				UserId:   p.UserId,
			}
		}),
		Events: lo.Map(g.History().Events, func(evt *macondo.GameEvent, index int) *ipc.GameEvent {
			return MacondoEvtToOMGEvt(evt, index, letterdist)
		}),
		Version: stores.CurrentGameDocumentVersion,
		Lexicon: g.LexiconName(),
		Uid:     g.GameID(),
		Racks: lo.Map(g.History().LastKnownRacks, func(rack string, index int) []byte {
			return rackConverter(rack, index, letterdist)
		}),
		ChallengeRule: ipc.ChallengeRule(g.GameReq.ChallengeRule),
		PlayState:     ipc.PlayState(g.History().PlayState),
		CurrentScores: lo.Map(lo.Range(len(g.PlayerDBIDs)), func(pidx, i int) int32 {
			return int32(g.PointsFor(pidx))
		}),
		Variant:            g.History().Variant,
		Winner:             g.History().Winner,
		BoardLayout:        g.History().BoardLayout,
		LetterDistribution: g.History().LetterDistribution,
		Type:               g.Type,
		TimersStarted:      g.Started,
		EndReason:          g.GameEndReason,
		MetaEventData:      &ipc.MetaEventData{Events: g.MetaEvents.Events},
		CreatedAt:          timestamppb.New(g.CreatedAt),
		Board:              &ipc.GameBoard{},
		Bag:                &ipc.Bag{},
		ScorelessTurns:     uint32(g.ScorelessTurns()),
		PlayerOnTurn:       uint32(g.PlayerOnTurn()),
		Timers: &ipc.Timers{
			TimeOfLastUpdate: g.Timers.TimeOfLastUpdate,
			TimeStarted:      g.TimeStarted(),
			TimeRemaining: lo.Map(g.Timers.TimeRemaining, func(x int, index int) int64 {
				return int64(x)
			}),
			MaxOvertime:      int32(g.Timers.MaxOvertime),
			IncrementSeconds: g.GameReq.IncrementSeconds,
		},
		Description: g.History().Description,
	}

	populateBoard(g, gdoc)
	populateBag(g, gdoc)

	return gdoc, nil
}

func populateBoard(g *entity.Game, gdoc *ipc.GameDocument) {
	gdoc.Board.NumCols = int32(g.Board().Dim())
	gdoc.Board.NumRows = int32(g.Board().Dim())
	gdoc.Board.IsEmpty = g.Board().IsEmpty()

	gdoc.Board.Tiles = lo.Map(g.Board().GetSquares(), func(ml tilemapping.MachineLetter, idx int) byte {
		return byte(ml)
	})
}

func populateBag(g *entity.Game, gdoc *ipc.GameDocument) {
	gdoc.Bag.Tiles = lo.Map(g.Bag().Peek(), func(ml tilemapping.MachineLetter, idx int) byte {
		return byte(ml)
	})
}

// ToGameHistory is a helper function to convert a GameDocument back to a game history.
// Eventually we will not have GameHistory's anymore.
func ToGameHistory(doc *ipc.GameDocument, cfg *config.Config) (*macondo.GameHistory, error) {
	letterdist, err := tilemapping.GetDistribution(cfg.WGLConfig(), doc.LetterDistribution)
	if err != nil {
		return nil, err
	}

	rackConverter := func(bts []byte, idx int) string {
		mw := tilemapping.FromByteArr(bts)
		return mw.UserVisible(letterdist.TileMapping())
	}

	rackConverterForPlay := func(bts []byte, idx int) string {
		mw := tilemapping.FromByteArr(bts)
		return mw.UserVisiblePlayedTiles(letterdist.TileMapping())
	}

	rackConverterForExchange := func(bts []byte, idx int) string {
		mw := tilemapping.FromByteArr(bts)
		return mw.UserVisible(letterdist.TileMapping())
	}
	var finalScores []int32
	if doc.EndReason != ipc.GameEndReason_ABORTED && doc.EndReason != ipc.GameEndReason_CANCELLED {
		// compute final scores from the doc's events:
		finalScores = make([]int32, len(doc.Players))
		for _, event := range doc.Events {
			finalScores[event.PlayerIndex] = event.Cumulative
		}
	}

	eventConverter := func(evt *ipc.GameEvent, index int) *macondo.GameEvent {
		// macondo GameHistory expects the rack for challenge bonus events to be
		// the next rack for the player who was challenged. So we need to pull this
		// from the history.

		rack := evt.Rack
		if evt.Type == ipc.GameEvent_CHALLENGE_BONUS {
			// find the next rack; loop through doc.Racks from the current
			// index until the next event with the same player index, or until
			// we get to the end of the list.
			for index+1 < len(doc.Events) && doc.Events[index+1].PlayerIndex != evt.PlayerIndex {
				index++
			}

			if index+1 < len(doc.Events) &&
				doc.Events[index+1].PlayerIndex == evt.PlayerIndex &&
				doc.Events[index+1].Type != ipc.GameEvent_END_RACK_PTS {
				// Check to make sure the rack we found actually belongs to this
				// player, and is not the "rack" from the END_RACK_PTS event.
				rack = doc.Events[index+1].Rack
			}
			// Otherwise keep whatever rack was in the event.
		}
		var tilesFromRack int
		if evt.Type == ipc.GameEvent_EXCHANGE {
			tilesFromRack = len(evt.Exchanged)
		} else if evt.Type == ipc.GameEvent_TILE_PLACEMENT_MOVE {
			for _, tile := range evt.PlayedTiles {
				if tile != 0 {
					tilesFromRack++
				}
			}
		}

		cvt := &macondo.GameEvent{
			Rack:             rackConverter(rack, 0),
			Type:             macondo.GameEvent_Type(evt.Type),
			Cumulative:       evt.Cumulative,
			Row:              evt.Row,
			Column:           evt.Column,
			Direction:        macondo.GameEvent_Direction(evt.Direction),
			Position:         evt.Position,
			PlayedTiles:      rackConverterForPlay(evt.PlayedTiles, 0),
			Exchanged:        rackConverterForExchange(evt.Exchanged, 0),
			Score:            evt.Score,
			Bonus:            evt.Bonus,
			EndRackPoints:    evt.EndRackPoints,
			LostScore:        evt.LostScore,
			IsBingo:          evt.IsBingo,
			WordsFormed:      lo.Map(evt.WordsFormed, rackConverter),
			MillisRemaining:  evt.MillisRemaining,
			PlayerIndex:      evt.PlayerIndex,
			NumTilesFromRack: uint32(tilesFromRack),
		}

		return cvt
	}

	history := &macondo.GameHistory{
		Players: lo.Map(doc.Players, func(p *ipc.GameDocument_MinimalPlayerInfo, idx int) *macondo.PlayerInfo {
			return &macondo.PlayerInfo{
				Nickname: p.Nickname,
				RealName: p.RealName,
				UserId:   p.UserId,
			}
		}),
		Events:         lo.Map(doc.Events, eventConverter),
		Version:        game.CurrentGameHistoryVersion,
		Lexicon:        doc.Lexicon,
		Uid:            doc.Uid,
		LastKnownRacks: lo.Map(doc.Racks, rackConverter),
		ChallengeRule:  macondo.ChallengeRule(doc.ChallengeRule),
		PlayState:      macondo.PlayState(doc.PlayState),
		// current scores not in history!
		Variant:            doc.Variant,
		Winner:             doc.Winner,
		BoardLayout:        doc.BoardLayout,
		LetterDistribution: doc.LetterDistribution,
		Description:        doc.Description,
		IdAuth:             "io.woogles", // hardcoded for now
		FinalScores:        finalScores,
	}
	return history, nil

}
