// Package cwgame implements the rules for playing a crossword board game.
// It is heavily dependent on the GameDocument object in protobuf.
package cwgame

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/cwgame/board"
	"github.com/woogles-io/liwords/pkg/cwgame/tiles"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const (
	RackTileLimit                = 7
	ExchangePermittedTilesInBag  = 7
	MaxConsecutiveScorelessTurns = 6
)

var globalNower Nower = GameTimer{}

type InvalidWordsError struct {
	rm    *tilemapping.TileMapping
	words []tilemapping.MachineWord
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

func playMove(ctx context.Context, gdoc *ipc.GameDocument, gevt *ipc.GameEvent, tr int64) error {
	log.Debug().Interface("gevt", gevt).Msg("play-move")
	cfg, err := config.Ctx(ctx)
	if err != nil {
		return err
	}

	if gevt.Type == ipc.GameEvent_CHALLENGE {
		return challengeEvent(ctx, cfg, gdoc, tr)
	}

	err = validateMove(cfg, gevt, gdoc)
	if err != nil {
		return err
	}

	// register time before playing the move
	recordTimeOfMove(gdoc, globalNower, gdoc.PlayerOnTurn, true)

	// Note: in case of error, anything that modifies gdoc should not save
	// gdoc back to the store; this must be enforced.
	switch gevt.Type {
	case ipc.GameEvent_TILE_PLACEMENT_MOVE:
		err := playTilePlacementMove(cfg, gevt, gdoc, tr)
		if err != nil {
			return err
		}

	case ipc.GameEvent_PASS, ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS:
		gevt.MillisRemaining = int32(tr)
		gevt.Cumulative = gdoc.CurrentScores[gdoc.PlayerOnTurn]
		gevt.Rack = gdoc.Racks[gdoc.PlayerOnTurn]
		gdoc.Events = append(gdoc.Events, gevt)

		if gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS {
			gdoc.PlayState = ipc.PlayState_GAME_OVER
			gdoc.EndReason = ipc.GameEndReason_STANDARD
			dist, err := tilemapping.GetDistribution(cfg.WGLConfig(), gdoc.LetterDistribution)
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
		// Use TileInventory to handle the exchange
		inv := NewTileInventory(gdoc, cfg.WGLConfig())

		// Set the rack to what's in the event (in case it differs from current)
		if err := inv.SetRack(int(gdoc.PlayerOnTurn), gevt.Rack); err != nil {
			return err
		}

		// Exchange the tiles
		exchangedTiles := tilemapping.FromByteArr(gevt.Exchanged)
		if err := inv.ExchangeTiles(int(gdoc.PlayerOnTurn), exchangedTiles); err != nil {
			return err
		}

		// In annotated games, auto-assign or top off the next player's rack
		// This ensures the opponent always has a full rack after an exchange
		if gdoc.Type == ipc.GameType_ANNOTATED {
			nextPlayer := 1 - gdoc.PlayerOnTurn
			_, err := inv.DrawToFillRack(int(nextPlayer))
			if err != nil {
				return err
			}
		}

		gdoc.ScorelessTurns += 1
		gevt.MillisRemaining = int32(tr)
		gevt.Cumulative = gdoc.CurrentScores[gdoc.PlayerOnTurn]
		gdoc.Events = append(gdoc.Events, gevt)
		log.Debug().Interface("new-rack", gdoc.Racks[gdoc.PlayerOnTurn]).
			Uint32("onturn", gdoc.PlayerOnTurn).
			Msg("exchanged")

	}
	if gdoc.ScorelessTurns == MaxConsecutiveScorelessTurns {
		dist, err := tilemapping.GetDistribution(cfg.WGLConfig(), gdoc.LetterDistribution)
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

func playTilePlacementMove(cfg *config.Config, gevt *ipc.GameEvent, gdoc *ipc.GameDocument, tr int64) error {
	dist, err := tilemapping.GetDistribution(cfg.WGLConfig(), gdoc.LetterDistribution)
	if err != nil {
		return err
	}

	// validate the tile play move
	gd, err := kwg.GetKWG(cfg.WGLConfig(), gdoc.Lexicon)
	if err != nil {
		return err
	}

	wordsFormed, err := validateTilePlayMove(gd, dist.TileMapping(), gevt, gdoc)
	if err != nil {
		return err
	}
	tilesUsed := tilemapping.FromByteArr(gevt.PlayedTiles)

	// CRITICAL: Validate that the rack contains the needed tiles BEFORE placing them on the board
	// This prevents tile corruption where board.PlayMove creates tiles on the board even if
	// they're not in the rack. Without this check, tiles get placed on the board (creating extras),
	// then the Leave validation fails, leaving us with a corrupted game state.
	rackTiles := tilemapping.FromByteArr(gevt.Rack)
	_, err = tilemapping.Leave(rackTiles, tilesUsed, true)
	if err != nil {
		return fmt.Errorf("rack doesn't contain tiles needed for move: %w", err)
	}

	score, err := board.PlayMove(gdoc.Board, gdoc.BoardLayout, dist,
		tilesUsed, int(gevt.Row), int(gevt.Column), gevt.Direction == ipc.GameEvent_VERTICAL)
	if err != nil {
		return err
	}
	tilesPlayed := 0
	for _, t := range tilesUsed {
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

	// Calculate leave (tiles remaining in rack after playing)
	// zeroIsPlaythrough=true: playing on board, tile 0 in tilesUsed represents play-through markers
	leave, err := tilemapping.Leave(tilemapping.FromByteArr(gevt.Rack), tilesUsed, true)
	if err != nil {
		return err
	}

	// NOTE: board.PlayMove (above) has already placed tiles on gdoc.Board, so those
	// tiles are counted on the board. We now update the rack to reflect what remains.
	// This briefly creates an invalid state (tiles double-counted), but TileInventory's
	// validation (below) will catch any issues. We keep board.PlayMove separate from
	// TileInventory because they have different responsibilities:
	//   - board.PlayMove = game rules (word validation, scoring, board placement)
	//   - TileInventory = tile accounting (conservation, validation)
	gdoc.Racks[gdoc.PlayerOnTurn] = tilemapping.MachineWord(leave).ToByteArr()

	// Use TileInventory to draw replacement tiles and validate tile conservation
	inv := NewTileInventory(gdoc, cfg.WGLConfig())
	_, err = inv.DrawToFillRack(int(gdoc.PlayerOnTurn))
	if err != nil {
		return err
	}

	// Get the new rack after drawing for end-game detection
	newRack := gdoc.Racks[gdoc.PlayerOnTurn]

	gevt.Score = score
	gevt.IsBingo = tilesPlayed == RackTileLimit
	gevt.MillisRemaining = int32(tr)

	// In annotated games, auto-assign or top off the next player's rack
	// This ensures the opponent always has a full rack after a play
	if gdoc.Type == ipc.GameType_ANNOTATED {
		nextPlayer := 1 - gdoc.PlayerOnTurn
		_, err := inv.DrawToFillRack(int(nextPlayer))
		if err != nil {
			return err
		}
	}

	gevt.WordsFormed = make([][]byte, len(wordsFormed))
	gevt.WordsFormedFriendly = make([]string, len(wordsFormed))
	gevt.Cumulative = gdoc.CurrentScores[gdoc.PlayerOnTurn]
	for i, w := range wordsFormed {
		gevt.WordsFormed[i] = w.ToByteArr()
		gevt.WordsFormedFriendly[i] = w.UserVisiblePlayedTiles(dist.TileMapping())
	}
	gdoc.Events = append(gdoc.Events, gevt)

	if len(newRack) == 0 {
		// if the challenge rule is not void we should wait for a final pass
		// however, if we are in replay mode, there's no pass.
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

func validateMove(cfg *config.Config, gevt *ipc.GameEvent, gdoc *ipc.GameDocument) error {
	if gdoc.PlayState == ipc.PlayState_GAME_OVER {
		return errGameNotActive
	}
	if gevt.Type == ipc.GameEvent_EXCHANGE {
		if gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS {
			return errOnlyPassOrChallenge
		}
		if len(gdoc.Bag.Tiles) < ExchangePermittedTilesInBag {
			return errExchangeNotPermitted
		}
		return nil
	} else if gevt.Type == ipc.GameEvent_TILE_PLACEMENT_MOVE {
		if gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS {
			return errOnlyPassOrChallenge
		}
		return nil
	}
	return nil

}

func validateTilePlayMove(gd *kwg.KWG, rm *tilemapping.TileMapping, gevt *ipc.GameEvent, gdoc *ipc.GameDocument) (
	[]tilemapping.MachineWord, error) {

	// convert play to machine letters
	playedTiles := tilemapping.FromByteArr(gevt.PlayedTiles)
	err := board.ErrorIfIllegalPlay(gdoc.Board, int(gevt.Row), int(gevt.Column),
		gevt.Direction == ipc.GameEvent_VERTICAL, playedTiles)
	if err != nil {
		return nil, err
	}

	for _, t := range playedTiles {
		if t.IsBlanked() {
			unblanked := t.Unblank()
			if unblanked < 1 || uint8(unblanked) > rm.NumLetters()-1 {
				return nil, fmt.Errorf("invalid blank tile: %v", unblanked)
			}
		} else {
			if uint8(t) > rm.NumLetters()-1 {
				return nil, fmt.Errorf("tile not in lexicon: %v", t)
			}
		}
	}

	// The play is legal. What words does it form?
	formedWords, err := board.FormedWords(gdoc.Board, int(gevt.Row), int(gevt.Column),
		gevt.Direction == ipc.GameEvent_VERTICAL, playedTiles)
	if err != nil {
		return nil, err
	}
	if gdoc.ChallengeRule == ipc.ChallengeRule_ChallengeRule_VOID {
		// Actually check the validity of the words.
		illegalWords := validateWords(gd, rm, formedWords, gdoc.Variant)

		if len(illegalWords) > 0 {
			return nil, &InvalidWordsError{rm: rm, words: illegalWords}
		}
	}
	return formedWords, nil
}

func validateWords(gd *kwg.KWG, rm *tilemapping.TileMapping, words []tilemapping.MachineWord,
	variant string) []tilemapping.MachineWord {
	var illegalWords []tilemapping.MachineWord
	lex := kwg.Lexicon{KWG: *gd}
	for _, word := range words {
		var valid bool
		if variant == VarWordSmog || variant == VarWordSmogSuper {
			valid = lex.HasAnagram(word)
		} else {
			valid = lex.HasWord(word)
		}
		if !valid {
			illegalWords = append(illegalWords, word)
		}
	}
	return illegalWords
}

func handleConsecutiveScorelessTurns(gdoc *ipc.GameDocument, dist *tilemapping.LetterDistribution) error {

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
		ptsOnRack := dist.WordScore(tilemapping.FromByteArr(gdoc.Racks[p]))
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

func endRackCalcs(gdoc *ipc.GameDocument, dist *tilemapping.LetterDistribution, wentout int) error {
	unplayedPts := 0
	var otherRack bytes.Buffer

	for _, r := range gdoc.Racks {
		_, err := otherRack.Write(r)
		if err != nil {
			return err
		}
		unplayedPts += dist.WordScore(tilemapping.FromByteArr(r))
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

	dist, err := tilemapping.GetDistribution(cfg.WGLConfig(), gdoc.LetterDistribution)
	if err != nil {
		return err
	}
	gd, err := kwg.GetKWG(cfg.WGLConfig(), gdoc.Lexicon)
	if err != nil {
		return err
	}

	// Note that the player on turn right now needs to be the player
	// who is making the challenge.
	lastMWs := make([]tilemapping.MachineWord, len(lastWordsFormed))
	for i, w := range lastWordsFormed {
		lastMWs[i] = tilemapping.FromByteArr(w)
	}

	illegalWords := validateWords(gd, dist.TileMapping(), lastMWs, gdoc.Variant)
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
			err := unplayLastMove(ctx, cfg, gdoc, dist)
			if err != nil {
				return err
			}
			// Rack is already restored by unplayLastMove
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
		err := unplayLastMove(ctx, cfg, gdoc, dist)
		if err != nil {
			return err
		}

		// Rack is already restored by unplayLastMove

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
			err = playMove(ctx, gdoc,
				&ipc.GameEvent{
					Type:        ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS,
					PlayerIndex: gdoc.PlayerOnTurn,
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

func unplayLastMove(ctx context.Context, cfg *config.Config, gdoc *ipc.GameDocument, dist *tilemapping.LetterDistribution) error {
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

	if offboardEvent.Type != ipc.GameEvent_PHONY_TILES_RETURNED {
		return errors.New("wrong event type for offboard event")
	}
	if originalEvent.Type != ipc.GameEvent_TILE_PLACEMENT_MOVE {
		return errors.New("wrong event type for original event")
	}
	if originalEvent.PlayerIndex != offboardEvent.PlayerIndex {
		return errors.New("player indexes don't match")
	}

	mw := tilemapping.FromByteArr(originalEvent.PlayedTiles)

	err := board.UnplaceMoveTiles(gdoc.Board, mw, int(originalEvent.Row),
		int(originalEvent.Column), originalEvent.Direction == ipc.GameEvent_VERTICAL)
	if err != nil {
		return err
	}

	// Calculate what the rack was after playing the phony (before drawing new tiles)
	// Use zeroIsPlaythrough=true because originalEvent.PlayedTiles may contain play-through markers
	leaveAfterPhony, err := tilemapping.Leave(
		tilemapping.FromByteArr(originalEvent.Rack), mw, true)
	if err != nil {
		return err
	}

	// Calculate what was drawn after the phony
	postPhonyRack := gdoc.Racks[offboardEvent.PlayerIndex]
	drewPostPhony, err := tilemapping.Leave(tilemapping.FromByteArr(postPhonyRack),
		leaveAfterPhony, false)
	if err != nil {
		return err
	}

	// Put back only the tiles that were drawn after the phony
	// The tiles from the board (mw) go directly back to the rack via originalEvent.Rack
	tiles.PutBack(gdoc.Bag, drewPostPhony)
	gdoc.Racks[offboardEvent.PlayerIndex] = originalEvent.Rack
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

// Return a user-friendly event description. Used for debugging.
func EventDescription(evt *ipc.GameEvent, rm *tilemapping.TileMapping) string {
	switch evt.Type {

	case ipc.GameEvent_PASS:
		return fmt.Sprintf("(Pass) +0 %d", evt.Cumulative)

	case ipc.GameEvent_TILE_PLACEMENT_MOVE:
		return fmt.Sprintf("%s %s %s +%d %d",
			evt.Position,
			tilemapping.FromByteArr(evt.Rack).UserVisible(rm),
			tilemapping.FromByteArr(evt.PlayedTiles).UserVisiblePlayedTiles(rm),
			evt.Score,
			evt.Cumulative,
		)

	case ipc.GameEvent_EXCHANGE:
		return fmt.Sprintf("%s [exch %s]  +0 %d",
			tilemapping.FromByteArr(evt.Rack).UserVisible(rm),
			tilemapping.FromByteArr(evt.Exchanged).UserVisiblePlayedTiles(rm),
			evt.Cumulative,
		)

	case ipc.GameEvent_CHALLENGE_BONUS:
		return fmt.Sprintf("+%d %d", evt.Bonus, evt.Cumulative)

	case ipc.GameEvent_PHONY_TILES_RETURNED:
		return fmt.Sprintf("[phony tiles returned %s] -%d %d",
			tilemapping.FromByteArr(evt.Rack).UserVisible(rm),
			evt.LostScore,
			evt.Cumulative)
	case ipc.GameEvent_END_RACK_PTS:
		return fmt.Sprintf("[end rack pts %s] +%d %d",
			tilemapping.FromByteArr(evt.Rack).UserVisible(rm),
			evt.EndRackPoints,
			evt.Cumulative)
	default:
		return fmt.Sprintf("Unknown event %s", evt.Type.String())
	}
}
