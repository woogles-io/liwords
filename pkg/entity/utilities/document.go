package utilities

import (
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/omgwords/stores"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
)

// helper functions to convert from the old GameHistory etc structs to
// GameDocuments. We can delete this after some time.

func ToGameDocument(g *entity.Game, cfg *config.Config) (*ipc.GameDocument, error) {
	letterdist, err := tilemapping.GetDistribution(cfg.MacondoConfigMap, g.History().LetterDistribution)
	if err != nil {
		return nil, err
	}

	rackConverter := func(rack string, index int) []byte {
		mls, err := tilemapping.ToMachineLetters(rack, letterdist.TileMapping())
		if err != nil {
			panic(err)
		}
		return tilemapping.MachineWord(mls).ToByteArr()
	}

	eventConverter := func(evt *macondo.GameEvent, index int) *ipc.GameEvent {
		cvt := &ipc.GameEvent{
			Rack:                rackConverter(evt.Rack, 0),
			Type:                ipc.GameEvent_Type(evt.Type),
			Cumulative:          evt.Cumulative,
			Row:                 evt.Row,
			Column:              evt.Column,
			Direction:           ipc.GameEvent_Direction(evt.Direction),
			Position:            evt.Position,
			PlayedTiles:         rackConverter(evt.PlayedTiles, 0),
			Exchanged:           rackConverter(evt.Exchanged, 0),
			Score:               evt.Score,
			Bonus:               evt.Bonus,
			EndRackPoints:       evt.EndRackPoints,
			LostScore:           evt.LostScore,
			IsBingo:             evt.IsBingo,
			WordsFormed:         lo.Map(evt.WordsFormed, rackConverter),
			WordsFormedFriendly: evt.WordsFormed,
			MillisRemaining:     evt.MillisRemaining,
			PlayerIndex:         evt.PlayerIndex,
		}

		return cvt
	}

	gdoc := &ipc.GameDocument{
		Players: lo.Map(g.History().Players, func(p *macondo.PlayerInfo, idx int) *ipc.GameDocument_MinimalPlayerInfo {
			return &ipc.GameDocument_MinimalPlayerInfo{
				Nickname: p.Nickname,
				RealName: p.RealName,
				UserId:   p.UserId,
			}
		}),
		Events:        lo.Map(g.History().Events, eventConverter),
		Version:       stores.CurrentGameDocumentVersion,
		Lexicon:       g.LexiconName(),
		Uid:           g.GameID(),
		Racks:         lo.Map(g.History().LastKnownRacks, rackConverter),
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
	}

	populateBoard(g, gdoc, letterdist)
	populateBag(g, gdoc, letterdist)

	return gdoc, nil
}

func populateBoard(g *entity.Game, gdoc *ipc.GameDocument, ld *tilemapping.LetterDistribution) {
	gdoc.Board.NumCols = int32(g.Board().Dim())
	gdoc.Board.NumRows = int32(g.Board().Dim())
	gdoc.Board.IsEmpty = g.Board().IsEmpty()

	gdoc.Board.Tiles = lo.Map(g.Board().GetSquares(), func(ml tilemapping.MachineLetter, idx int) byte {
		return byte(ml)
	})
}

func populateBag(g *entity.Game, gdoc *ipc.GameDocument, ld *tilemapping.LetterDistribution) {
	gdoc.Bag.Tiles = lo.Map(g.Bag().Peek(), func(ml tilemapping.MachineLetter, idx int) byte {
		return byte(ml)
	})
}

// ToGameHistory is a helper function to convert a GameDocument back to a game history.
// Eventually we will not have GameHistory's anymore.
func ToGameHistory(doc *ipc.GameDocument, cfg *config.Config) (*macondo.GameHistory, error) {
	letterdist, err := tilemapping.GetDistribution(cfg.MacondoConfigMap, doc.LetterDistribution)
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

	eventConverter := func(evt *ipc.GameEvent, index int) *macondo.GameEvent {
		cvt := &macondo.GameEvent{
			Rack:            rackConverter(evt.Rack, 0),
			Type:            macondo.GameEvent_Type(evt.Type),
			Cumulative:      evt.Cumulative,
			Row:             evt.Row,
			Column:          evt.Column,
			Direction:       macondo.GameEvent_Direction(evt.Direction),
			Position:        evt.Position,
			PlayedTiles:     rackConverterForPlay(evt.PlayedTiles, 0),
			Exchanged:       rackConverterForPlay(evt.Exchanged, 0),
			Score:           evt.Score,
			Bonus:           evt.Bonus,
			EndRackPoints:   evt.EndRackPoints,
			LostScore:       evt.LostScore,
			IsBingo:         evt.IsBingo,
			WordsFormed:     lo.Map(evt.WordsFormed, rackConverter),
			MillisRemaining: evt.MillisRemaining,
			PlayerIndex:     evt.PlayerIndex,
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
	}
	return history, nil

}
