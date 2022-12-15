// Package cwgame implements the rules for playing a crossword board game.
// It is heavily dependent on the GameDocument object in protobuf.
package cwgame

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame/board"
	"github.com/domino14/liwords/pkg/cwgame/dawg"
	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/domino14/liwords/pkg/cwgame/tiles"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/rs/zerolog/log"
)

const (
	GameDocumentVersion = 1

	RackTileLimit                = 7
	ExchangePermittedTilesInBag  = 7
	MaxConsecutiveScorelessTurns = 6
)

var globalNower Nower = GameTimer{}

type InvalidWordsError struct {
	rm    *runemapping.RuneMapping
	words []runemapping.MachineWord
}

func (e *InvalidWordsError) Error() string {
	var errString strings.Builder

	errString.WriteString("invalid words: ")
	for idx, w := range e.words {
		errString.WriteString(w.UserVisible(e.rm))
		if idx != len(e.words)-1 {
			errString.WriteString(", ")
		}
	}
	return errString.String()
}

func playMove(ctx context.Context, gdoc *ipc.GameDocument, m move, tr int64) error {
	cfg, ok := ctx.Value(config.CtxKeyword).(*config.Config)
	if !ok {
		return errors.New("config does not exist in context")
	}

	if m.mtype == ipc.GameEvent_CHALLENGE {
		return challengeEvent(ctx, cfg, gdoc, tr)
	}

	err := validateMove(cfg, m, gdoc)
	if err != nil {
		return err
	}

	// register time before playing the move
	recordTimeOfMove(gdoc, globalNower, gdoc.PlayerOnTurn, true)

	// Note: in case of error, anything that modifies gdoc should not save
	// gdoc back to the store; this must be enforced.
	switch m.mtype {
	case ipc.GameEvent_TILE_PLACEMENT_MOVE:
		err := playTilePlacementMove(cfg, m, gdoc, tr)
		if err != nil {
			return err
		}

	case ipc.GameEvent_PASS, ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
		evt := eventFromMove(m, gdoc)
		evt.MillisRemaining = int32(tr)
		gdoc.Events = append(gdoc.Events, evt)

		if gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS {
			gdoc.PlayState = ipc.PlayState_GAME_OVER
			gdoc.EndReason = ipc.GameEndReason_STANDARD
			dist, err := tiles.GetDistribution(cfg, gdoc.LetterDistribution)
			if err != nil {
				return err
			}
			// search for the person who went out (they have no rack)
			wentout := -1
			for idx, p := range gdoc.Racks {
				if len(p) == 0 {
					wentout = idx
				}
			}
			if wentout == -1 {
				return errors.New("no empty rack but player went out")
			}
			endRackCalcs(gdoc, dist, wentout)
			addWinnerToHistory(gdoc)
		} else {
			gdoc.ScorelessTurns += 1
		}

	case ipc.GameEvent_EXCHANGE:

		placeholder := make([]runemapping.MachineLetter, RackTileLimit)
		err := tiles.Exchange(gdoc.Bag, m.tilesUsed, placeholder)
		if err != nil {
			return err
		}
		copy(placeholder[len(m.tilesUsed):], m.leave)

		gdoc.Racks[gdoc.PlayerOnTurn] = runemapping.MachineWord(placeholder).ToByteArr()
		gdoc.ScorelessTurns += 1

		evt := eventFromMove(m, gdoc)
		evt.MillisRemaining = int32(tr)
		evt.Exchanged = runemapping.MachineWord(m.tilesUsed).ToByteArr()
		evt.Leave = runemapping.MachineWord(m.leave).ToByteArr()
		gdoc.Events = append(gdoc.Events, evt)

	}
	if gdoc.ScorelessTurns == MaxConsecutiveScorelessTurns {
		dist, err := tiles.GetDistribution(cfg, gdoc.LetterDistribution)
		if err != nil {
			return err
		}
		err = handleConsecutiveScorelessTurns(gdoc, dist)
		if err != nil {
			return err
		}
	} else {
		assignTurnToNextNonquitter(gdoc, gdoc.PlayerOnTurn)
	}
	return nil
}

func playTilePlacementMove(cfg *config.Config, m move, gdoc *ipc.GameDocument, tr int64) error {
	dist, err := tiles.GetDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return err
	}

	// validate the tile play move
	dawg, err := dawg.GetDawg(cfg, gdoc.Lexicon)
	if err != nil {
		return err
	}

	wordsFormed, err := validateTilePlayMove(dawg, dist.RuneMapping(), m, gdoc)
	if err != nil {
		return err
	}
	score, err := board.PlayMove(gdoc.Board, gdoc.BoardLayout, dist,
		m.tilesUsed, m.row, m.col, m.direction == ipc.GameEvent_VERTICAL)
	if err != nil {
		return err
	}
	tilesPlayed := 0
	for _, t := range m.tilesUsed {
		if t != 0 {
			tilesPlayed++
		}
	}

	// calculating and caching cross-scores in board might not be necessary
	// unless we're really hurting for optimizations here.

	// no international rule counts a score of 0 as a scoreless turn
	// if it's from tiles being played on the board (like a blank next
	// to another blank) so always reset this.
	gdoc.ScorelessTurns = 0
	gdoc.CurrentScores[gdoc.PlayerOnTurn] += score

	placeholder := make([]runemapping.MachineLetter, RackTileLimit)
	drew, err := tiles.DrawAtMost(gdoc.Bag, tilesPlayed, placeholder)
	if err != nil {
		return err
	}
	copy(placeholder[drew:], m.leave)
	newRack := placeholder[:drew+len(m.leave)]

	evt := eventFromMove(m, gdoc)

	evt.Score = score
	evt.IsBingo = tilesPlayed == RackTileLimit
	evt.MillisRemaining = int32(tr)
	evt.Leave = runemapping.MachineWord(m.leave).ToByteArr()

	gdoc.Racks[gdoc.PlayerOnTurn] = runemapping.MachineWord(newRack).ToByteArr()
	evt.WordsFormed = make([][]byte, len(wordsFormed))
	for i, w := range wordsFormed {
		evt.WordsFormed[i] = w.ToByteArr()
	}
	gdoc.Events = append(gdoc.Events, evt)

	if len(newRack) == 0 {
		if gdoc.ChallengeRule != ipc.ChallengeRule_ChallengeRule_VOID {
			gdoc.PlayState = ipc.PlayState_WAITING_FOR_FINAL_PASS
		} else {
			gdoc.PlayState = ipc.PlayState_GAME_OVER
			gdoc.EndReason = ipc.GameEndReason_STANDARD
			err = endRackCalcs(gdoc, dist, int(gdoc.PlayerOnTurn))
			if err != nil {
				return err
			}
			addWinnerToHistory(gdoc)
		}
	}
	return nil
}

func validateMove(cfg *config.Config, m move, gdoc *ipc.GameDocument) error {
	if gdoc.PlayState == ipc.PlayState_GAME_OVER {
		return errGameNotActive
	}
	if m.mtype == ipc.GameEvent_EXCHANGE {
		if gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS {
			return errOnlyPassOrChallenge
		}
		if len(gdoc.Bag.Tiles) < ExchangePermittedTilesInBag {
			return errExchangeNotPermitted
		}
		return nil
	} else if m.mtype == ipc.GameEvent_PASS || m.mtype == ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS {
		return nil
	} else if m.mtype == ipc.GameEvent_CHALLENGE {
		return nil
	} else if m.mtype == ipc.GameEvent_TILE_PLACEMENT_MOVE {
		if gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS {
			return errOnlyPassOrChallenge
		}
		return nil
	}
	return errMoveTypeNotUserInputtable

}

func validateTilePlayMove(dawg *dawg.SimpleDawg, rm *runemapping.RuneMapping, m move, gdoc *ipc.GameDocument) (
	[]runemapping.MachineWord, error) {

	// convert play to machine letters

	err := board.ErrorIfIllegalPlay(gdoc.Board, m.row, m.col,
		m.direction == ipc.GameEvent_VERTICAL, m.tilesUsed)
	if err != nil {
		return nil, err
	}

	// The play is legal. What words does it form?
	formedWords, err := board.FormedWords(gdoc.Board, m.row, m.col,
		m.direction == ipc.GameEvent_VERTICAL, m.tilesUsed)
	if err != nil {
		return nil, err
	}
	if gdoc.ChallengeRule == ipc.ChallengeRule_ChallengeRule_VOID {
		// Actually check the validity of the words.
		illegalWords := validateWords(dawg, formedWords, gdoc.Variant)

		if len(illegalWords) > 0 {
			return nil, &InvalidWordsError{rm: rm, words: illegalWords}
		}
	}
	return formedWords, nil
}

func validateWords(dawg *dawg.SimpleDawg, words []runemapping.MachineWord,
	variant string) []runemapping.MachineWord {

	var illegalWords []runemapping.MachineWord
	for _, word := range words {
		var valid bool
		if variant == VarWordSmog || variant == VarWordSmogSuper {
			valid = dawg.HasAnagram(word)
		} else {
			valid = dawg.HasWord(word)
		}
		if !valid {
			illegalWords = append(illegalWords, word)
		}
	}
	return illegalWords
}

func eventFromMove(m move, gdoc *ipc.GameDocument) *ipc.GameEvent {
	evt := &ipc.GameEvent{}

	evt.Type = m.mtype
	evt.PlayerIndex = uint32(gdoc.PlayerOnTurn)
	evt.Cumulative = gdoc.CurrentScores[gdoc.PlayerOnTurn]
	evt.Rack = gdoc.Racks[gdoc.PlayerOnTurn]

	switch m.mtype {
	case ipc.GameEvent_TILE_PLACEMENT_MOVE:
		evt.Position = m.clientEvt.PositionCoords
		evt.PlayedTiles = runemapping.MachineWord(m.tilesUsed).ToByteArr()
		evt.Row = int32(m.row)
		evt.Column = int32(m.col)
		evt.Direction = m.direction

	case ipc.GameEvent_EXCHANGE:
		evt.Exchanged = runemapping.MachineWord(m.tilesUsed).ToByteArr()
	}
	return evt
}

func handleConsecutiveScorelessTurns(gdoc *ipc.GameDocument, dist *tiles.LetterDistribution) error {

	gdoc.PlayState = ipc.PlayState_GAME_OVER
	gdoc.EndReason = ipc.GameEndReason_CONSECUTIVE_ZEROES

	toIterate := make([]int, len(gdoc.Players))
	for idx := range toIterate {
		toIterate[idx] = idx
	}
	if gdoc.PlayerOnTurn != 0 {
		// process player on turn's end rack penalty first.
		toIterate[0], toIterate[gdoc.PlayerOnTurn] = toIterate[gdoc.PlayerOnTurn], toIterate[0]
	}
	for _, p := range toIterate {
		ptsOnRack := dist.WordScore(runemapping.FromByteArr(gdoc.Racks[p]))
		gdoc.CurrentScores[p] -= int32(ptsOnRack)
		penaltyEvt := endRackPenaltyEvt(gdoc, uint32(p), ptsOnRack)
		gdoc.Events = append(gdoc.Events, penaltyEvt)
	}
	return nil
}

func endRackPenaltyEvt(gdoc *ipc.GameDocument, pidx uint32, penalty int) *ipc.GameEvent {
	return &ipc.GameEvent{
		PlayerIndex: pidx,
		Cumulative:  gdoc.CurrentScores[pidx],
		Rack:        gdoc.Racks[pidx],
		LostScore:   int32(penalty),
		Type:        ipc.GameEvent_END_RACK_PENALTY,
	}
}

// Leave returns the leave after playing or using `tiles` in `rack`.
// It returns an error if the tile is in the play but not in the rack
// XXX: This function needs to allocate less.
func Leave(rack, tilesUsed []runemapping.MachineLetter) ([]runemapping.MachineLetter, error) {
	rackletters := map[runemapping.MachineLetter]int{}
	for _, l := range rack {
		rackletters[l]++
	}
	leave := make([]runemapping.MachineLetter, 0)

	for _, t := range tilesUsed {
		if t == 0 {
			// play-through
			continue
		}
		if t.IsBlanked() {
			// it's a blank
			t = 0
		}
		if rackletters[t] != 0 {
			rackletters[t]--
		} else {
			return nil, fmt.Errorf("tile in play but not in rack: %v", t)
		}
	}

	for k, v := range rackletters {
		if v > 0 {
			for i := 0; i < v; i++ {
				leave = append(leave, k)
			}
		}
	}
	sort.Slice(leave, func(i, j int) bool {
		return leave[i] < leave[j]
	})
	return leave, nil
}

func endRackCalcs(gdoc *ipc.GameDocument, dist *tiles.LetterDistribution, wentout int) error {
	unplayedPts := 0
	var otherRack bytes.Buffer

	for _, r := range gdoc.Racks {
		_, err := otherRack.Write(r)
		if err != nil {
			return err
		}
		unplayedPts += dist.WordScore(runemapping.FromByteArr(r))
	}
	unplayedPts *= 2

	gdoc.CurrentScores[wentout] += int32(unplayedPts)
	gdoc.Events = append(gdoc.Events, &ipc.GameEvent{
		PlayerIndex:   uint32(wentout),
		Cumulative:    gdoc.CurrentScores[wentout],
		Rack:          otherRack.Bytes(),
		EndRackPoints: int32(unplayedPts),
		Type:          ipc.GameEvent_END_RACK_PTS,
	})
	return nil
}

func addWinnerToHistory(gdoc *ipc.GameDocument) {
	gdoc.Winner = -1
	maxScoreIndex, maxScore := 0, int32(-1000000)
	allEqual := true
	for i, s := range gdoc.CurrentScores {
		if s > maxScore {
			maxScoreIndex, maxScore = i, s
		}
		if s != gdoc.CurrentScores[0] {
			allEqual = false
		}
	}
	if !allEqual {
		gdoc.Winner = int32(maxScoreIndex)
	}
}

// ChallengeEvent should only be called if there is a history of events.
// It has the logic for appending challenge events and calculating scores
// properly.
// Note that this event can change the history of the game, including
// things like resetting the game ended state (for example if someone plays
// out with a phony).
func challengeEvent(ctx context.Context, cfg *config.Config, gdoc *ipc.GameDocument, tr int64) error {
	if len(gdoc.Events) == 0 {
		return errors.New("this game has no history")
	}
	if gdoc.ChallengeRule == ipc.ChallengeRule_ChallengeRule_VOID {
		return errors.New("challenges are not valid in void")
	}
	lastWordsFormed := gdoc.Events[len(gdoc.Events)-1].WordsFormed
	if len(lastWordsFormed) == 0 {
		return errors.New("there are no words to challenge")
	}
	// record time of the challenge, but do not account for increments;
	// a challenge event shouldn't modify the clock per se.
	recordTimeOfMove(gdoc, globalNower, gdoc.PlayerOnTurn, false)

	dist, err := tiles.GetDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return err
	}
	dawg, err := dawg.GetDawg(cfg, gdoc.Lexicon)
	if err != nil {
		return err
	}

	// Note that the player on turn right now needs to be the player
	// who is making the challenge.
	lastMWs := make([]runemapping.MachineWord, len(lastWordsFormed))
	for i, w := range lastWordsFormed {
		lastMWs[i] = runemapping.FromByteArr(w)
	}

	illegalWords := validateWords(dawg, lastMWs, gdoc.Variant)
	playLegal := len(illegalWords) == 0

	lastEvent := gdoc.Events[len(gdoc.Events)-1]
	cumeScoreBeforeChallenge := lastEvent.Cumulative

	challengee := lastEvent.PlayerIndex

	offBoardEvent := &ipc.GameEvent{
		PlayerIndex: challengee,
		Type:        ipc.GameEvent_PHONY_TILES_RETURNED,
		LostScore:   lastEvent.Score,
		Cumulative:  cumeScoreBeforeChallenge - lastEvent.Score,
		Rack:        lastEvent.Rack,
		PlayedTiles: lastEvent.PlayedTiles,
		// Note: these millis remaining would be the challenger's
		MillisRemaining: int32(tr),
	}

	// This ideal system makes it so someone always loses
	// the game.
	if gdoc.ChallengeRule == ipc.ChallengeRule_ChallengeRule_TRIPLE {
		// Set the winner and loser before calling PlayMove, as
		// that changes who is on turn
		var winner int32
		if playLegal {
			// The challenge was wrong, they lose the game
			winner = int32(challengee)
		} else {
			// The challenger was right, they win the game
			winner = int32(gdoc.PlayerOnTurn)
			// Take the play off the board.
			gdoc.Events = append(gdoc.Events, offBoardEvent)
			err := unplayLastMove(ctx, gdoc, dist)
			if err != nil {
				return err
			}
			gdoc.Racks[challengee] = lastEvent.Rack
		}
		gdoc.Winner = winner
		gdoc.PlayState = ipc.PlayState_GAME_OVER
		gdoc.EndReason = ipc.GameEndReason_TRIPLE_CHALLENGE

	} else if !playLegal {
		log.Debug().Msg("Successful challenge")

		// the play comes off the board. Add the offBoardEvent.
		gdoc.Events = append(gdoc.Events, offBoardEvent)

		// Unplay the last move to restore everything as it was board-wise
		// (and un-end the game if it had ended)
		err := unplayLastMove(ctx, gdoc, dist)
		if err != nil {
			return err
		}

		// We must also set the last known rack of the challengee back to
		// their rack before they played the phony.
		gdoc.Racks[challengee] = lastEvent.Rack
		if gdoc.ScorelessTurns == MaxConsecutiveScorelessTurns {
			err = handleConsecutiveScorelessTurns(gdoc, dist)
			if err != nil {
				return err
			}
		}
	} else {
		log.Debug().Msg("Unsuccessful challenge")

		addPts := int32(0)
		shouldAddPts := false

		bonusScoreEvent := func(bonus int32) *ipc.GameEvent {
			return &ipc.GameEvent{
				PlayerIndex: challengee,
				Type:        ipc.GameEvent_CHALLENGE_BONUS,
				Rack:        gdoc.Racks[challengee],
				Bonus:       bonus,
				Cumulative:  cumeScoreBeforeChallenge + bonus,
				// Note: these millis remaining would be the challenger's
				MillisRemaining: int32(tr),
			}
		}

		switch gdoc.ChallengeRule {
		case ipc.ChallengeRule_ChallengeRule_DOUBLE:
			// This "draconian" American system makes it so someone always loses
			// their turn.
			// challenger was wrong. They lose their turn.
			err = playMove(ctx, gdoc, move{
				mtype: ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS,
			}, tr)

		case ipc.ChallengeRule_ChallengeRule_FIVE_POINT:
			// Append a bonus to the event.
			shouldAddPts = true
			addPts = 5

		case ipc.ChallengeRule_ChallengeRule_TEN_POINT:
			shouldAddPts = true
			addPts = 10

		case ipc.ChallengeRule_ChallengeRule_SINGLE:
			shouldAddPts = true
			addPts = 0
		}

		if shouldAddPts {
			evt := bonusScoreEvent(addPts)
			log.Debug().Interface("evt", evt).Msg("adding bonus score evt")
			gdoc.Events = append(gdoc.Events, evt)
			gdoc.CurrentScores[challengee] += addPts
		}

		if gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS {
			gdoc.PlayState = ipc.PlayState_GAME_OVER
			gdoc.EndReason = ipc.GameEndReason_STANDARD

			// Game is actually over now, after the failed challenge.
			err = endRackCalcs(gdoc, dist, int(challengee))
			if err != nil {
				return err
			}
		}

	}

	return err
}

func unplayLastMove(ctx context.Context, gdoc *ipc.GameDocument, dist *tiles.LetterDistribution) error {
	// unplay the last move. This function already assumes the off-board event
	// exists in the gdoc's History.

	// 1) Remove the tiles from the board, put them back on our hand
	// 2) Add the tiles that we picked back to the bag
	// 3) Set the scores of the players back to what they used to be (off-board event
	// has this info)
	// 4) Set the game playing state back to what it used to be (in case game ended)
	nevt := len(gdoc.Events)
	if nevt < 2 {
		return errors.New("not enough events to unplay")
	}

	offboardEvent := gdoc.Events[nevt-1]
	originalEvent := gdoc.Events[nevt-2]
	postPhonyRack := gdoc.Racks[offboardEvent.PlayerIndex]

	if offboardEvent.Type != ipc.GameEvent_PHONY_TILES_RETURNED {
		return errors.New("wrong event type for offboard event")
	}
	if originalEvent.Type != ipc.GameEvent_TILE_PLACEMENT_MOVE {
		return errors.New("wrong event type for original event")
	}
	if originalEvent.PlayerIndex != offboardEvent.PlayerIndex {
		return errors.New("player indexes don't match")
	}

	mw := runemapping.FromByteArr(originalEvent.PlayedTiles)

	err := board.UnplaceMoveTiles(gdoc.Board, mw, int(originalEvent.Row),
		int(originalEvent.Column), originalEvent.Direction == ipc.GameEvent_VERTICAL)
	if err != nil {
		return err
	}

	leaveAfterPhony := originalEvent.Leave
	drewPostPhony, err := Leave(
		runemapping.FromByteArr(postPhonyRack),
		runemapping.FromByteArr(leaveAfterPhony))
	if err != nil {
		return err
	}

	tiles.PutBack(gdoc.Bag, drewPostPhony)
	gdoc.PlayState = ipc.PlayState_PLAYING
	gdoc.CurrentScores[offboardEvent.PlayerIndex] = offboardEvent.Cumulative

	// recalculate number of scoreless turns by going back in the history
	// and applying some heuristics.
	scorelessTurns := 0
	sawReturnedPhony := false

evtCounter:
	for i := len(gdoc.Events) - 1; i >= 0; i-- {
		evt := gdoc.Events[i]
		switch evt.Type {
		case ipc.GameEvent_TILE_PLACEMENT_MOVE:
			if sawReturnedPhony {
				// This can only be associated with this tile placement move
				sawReturnedPhony = false
				scorelessTurns++
			} else {
				break evtCounter
			}
		case ipc.GameEvent_PHONY_TILES_RETURNED:
			sawReturnedPhony = true
		case ipc.GameEvent_EXCHANGE, ipc.GameEvent_PASS, ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
			scorelessTurns++
			sawReturnedPhony = false
		default:
			sawReturnedPhony = false

		}
	}

	gdoc.ScorelessTurns = uint32(scorelessTurns)

	return nil
}
