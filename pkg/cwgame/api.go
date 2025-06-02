package cwgame

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	wglconfig "github.com/domino14/word-golib/config"
	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/woogles-io/liwords/pkg/cwgame/board"
	"github.com/woogles-io/liwords/pkg/cwgame/tiles"
	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var (
	errGameNotActive             = errors.New("game not active")
	errNotOnTurn                 = errors.New("not on turn")
	errOnlyPassOrChallenge       = errors.New("can only pass or challenge")
	errExchangeNotPermitted      = errors.New("you can only exchange with 7 or more tiles in the bag")
	errMoveTypeNotUserInputtable = errors.New("that move type is not available")
	errStartNotPermitted         = errors.New("game has already been started")
	errUnmatchedGameId           = errors.New("game ids do not match")
	errPlayerNotInGame           = errors.New("player not in this game")
)

var reVertical, reHorizontal *regexp.Regexp

type RackAssignBehavior int

const (
	// NeverAssignEmpty does not assign empty racks
	NeverAssignEmpty RackAssignBehavior = iota
	// AlwaysAssignEmpty always assigns empty racks
	AlwaysAssignEmpty
	// AssignEmptyIfUnambiguous assigns empty racks if they're the only thing
	// they can be. For example, if we assign a rack of 5 letters to player 1,
	// and there are only 6 unassigned letters left, all 6 of these letters
	// will be assigned to player 2. If there were 8 letters left, none of them
	// would be assigned to player 2.
	AssignEmptyIfUnambiguous
)

func init() {
	reVertical = regexp.MustCompile(`^(?P<col>[A-Z])(?P<row>[0-9]+)$`)
	reHorizontal = regexp.MustCompile(`^(?P<row>[0-9]+)(?P<col>[A-Z])$`)
}

// NewGame creates a new GameDocument. The playerinfo array contains
// the players, which must be in order of who goes first!
func NewGame(cfg *wglconfig.Config, rules *GameRules, playerinfo []*ipc.GameDocument_MinimalPlayerInfo) (*ipc.GameDocument, error) {
	// try to instantiate all aspects of the game from the given rules.

	dist, err := tilemapping.GetDistribution(cfg, rules.distname)
	if err != nil {
		return nil, err
	}
	_, err = kwg.GetKWG(cfg, rules.lexicon)
	if err != nil {
		return nil, err
	}
	layout, err := board.GetBoardLayout(rules.boardLayout)
	if err != nil {
		return nil, err
	}
	uniqueUserIds := make(map[string]bool)
	for _, u := range playerinfo {
		uniqueUserIds[u.UserId] = true
	}
	if len(uniqueUserIds) != len(playerinfo) {
		return nil, errors.New("user IDs must be unique")
	}
	if len(rules.secondsPerPlayer) != len(playerinfo) {
		return nil, errors.New("must have a time remaining per player")
	}

	timeRemaining := make([]int64, len(playerinfo))
	for i, t := range rules.secondsPerPlayer {
		timeRemaining[i] = int64(t * 1000)
	}

	gdoc := &ipc.GameDocument{
		Uid:                shortuuid.New(),
		Events:             make([]*ipc.GameEvent, 0),
		Players:            playerinfo,
		Lexicon:            rules.lexicon,
		Version:            stores.CurrentGameDocumentVersion,
		Variant:            string(rules.variant),
		BoardLayout:        rules.boardLayout,
		LetterDistribution: rules.distname,
		Racks:              make([][]byte, len(playerinfo)),
		Type:               ipc.GameType_NATIVE,
		CreatedAt:          timestamppb.Now(),
		Board:              board.NewBoard(layout),
		Bag:                tiles.TileBag(dist),
		PlayerOnTurn:       0, // player-on-turn always start as 0
		CurrentScores:      make([]int32, len(playerinfo)),
		Timers: &ipc.Timers{
			TimeRemaining:    timeRemaining,
			MaxOvertime:      int32(rules.maxOvertimeMins),
			IncrementSeconds: int32(rules.incrementSeconds),
			Untimed:          rules.untimed,
		},
		PlayState:     ipc.PlayState_UNSTARTED,
		ChallengeRule: rules.challengeRule,
	}

	return gdoc, nil
}

func StartGame(ctx context.Context, gdoc *ipc.GameDocument) error {
	if gdoc.PlayState != ipc.PlayState_UNSTARTED {
		return errStartNotPermitted
	}
	for idx := range gdoc.Players {
		t := make([]tilemapping.MachineLetter, RackTileLimit)
		err := tiles.Draw(gdoc.Bag, RackTileLimit, t)
		if err != nil {
			return err
		}
		gdoc.Racks[idx] = tilemapping.MachineWord(t).ToByteArr()
	}
	resetTimersAndStart(gdoc, globalNower)
	// Outside of this:
	// XXX: send changes to channel(s); see StartGame in gameplay package.
	// XXX: outside of this, send rematch event
	// XXX: potentially send bot move request?
	return nil
}

// AssignRacks assigns racks to the players. If assignEmpty is true, it will
// assign a random rack to any players with empty racks in the racks array.
func AssignRacks(gdoc *ipc.GameDocument, racks [][]byte, assignEmpty RackAssignBehavior) error {
	if len(racks) != len(gdoc.Players) {
		return errors.New("racks length must match players length")
	}
	// Throw in existing racks.
	for i := range gdoc.Players {
		if len(gdoc.Racks[i]) > 0 {
			log.Debug().Interface("rack", gdoc.Racks[i]).Int("player", i).Msg("throwing in rack for player")
		}
		mls := tilemapping.FromByteArr(gdoc.Racks[i])
		tiles.PutBack(gdoc.Bag, mls)
		gdoc.Racks[i] = nil
	}
	empties := []int{}
	for i, r := range racks {
		if len(r) == 0 {
			// empty
			empties = append(empties, i)
		} else {
			rackml := tilemapping.FromByteArr(r)
			err := tiles.RemoveTiles(gdoc.Bag, rackml)
			if err != nil {
				return err
			}
			gdoc.Racks[i] = r
		}
	}
	bagWillBeEmpty := false
	if tiles.InBag(gdoc.Bag) <= len(empties)*RackTileLimit {
		// The bag is empty after assigning racks
		bagWillBeEmpty = true
	}

	// Conditionally draw new tiles for empty racks
	if assignEmpty == AlwaysAssignEmpty ||
		(assignEmpty == AssignEmptyIfUnambiguous && bagWillBeEmpty) {

		for _, i := range empties {
			placeholder := make([]tilemapping.MachineLetter, RackTileLimit)
			drew, err := tiles.DrawAtMost(gdoc.Bag, RackTileLimit, placeholder)
			if err != nil {
				return err
			}
			drawn := placeholder[:drew]
			gdoc.Racks[i] = tilemapping.MachineWord(drawn).ToByteArr()
		}
	}
	return nil
}

func EditOldRack(ctx context.Context, cfg *wglconfig.Config, gdoc *ipc.GameDocument, evtNumber uint32, rack []byte) error {

	// Determine whether it is possible to edit the rack to the passed-in rack at this point in the game.
	// First clone and truncate the document.
	gc := proto.Clone(gdoc).(*ipc.GameDocument)
	evt := gdoc.Events[evtNumber]

	// replay until the event before evt.
	err := ReplayEvents(ctx, cfg, gc, gc.Events[:evtNumber], false)
	if err != nil {
		return err
	}
	evtTurn := evt.PlayerIndex
	racks := make([][]byte, len(gdoc.Players))
	racks[evtTurn] = rack
	err = AssignRacks(gc, racks, AssignEmptyIfUnambiguous)
	if err != nil {
		return err
	}
	// If it is possible to assign racks without issue, then do it on the
	// real document.
	evt.Rack = rack

	return nil
}

// ReplayEvents plays the events on the game document. For simplicity,
// assume these events replace every event in the game document; i.e.,
// initialize from scratch.
func ReplayEvents(ctx context.Context, cfg *wglconfig.Config, gdoc *ipc.GameDocument, evts []*ipc.GameEvent, rememberRacks bool) error {
	dist, err := tilemapping.GetDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return err
	}

	layout, err := board.GetBoardLayout(gdoc.BoardLayout)
	if err != nil {
		return err
	}

	gdoc.PlayState = ipc.PlayState_PLAYING
	gdoc.CurrentScores = make([]int32, len(gdoc.Players))
	gdoc.Events = []*ipc.GameEvent{}
	gdoc.Board = board.NewBoard(layout)
	gdoc.Bag = tiles.TileBag(dist)
	gdoc.ScorelessTurns = 0
	gdoc.PlayerOnTurn = 0
	var savedRacks [][]byte
	if rememberRacks {
		savedRacks = gdoc.Racks
	}
	savedTimers := proto.Clone(gdoc.Timers)
	gdoc.Racks = make([][]byte, len(gdoc.Players))
	// Replaying events is not as simple as just calling playMove with the event.
	// Because of the randomness factor, the drawn tiles after each play/exchange
	// etc won't be the same. We have to set the racks manually before each play.
	for idx, evt := range evts {
		if evt.Type == ipc.GameEvent_END_RACK_PTS {
			// don't append this. This event should be automatically generated
			// and appended by the regular gameplay events.
			continue
		}
		toAssign := make([][]byte, len(gdoc.Players))
		toAssign[evt.PlayerIndex] = evt.Rack

		err = AssignRacks(gdoc, toAssign, AssignEmptyIfUnambiguous)
		if err != nil {
			return err
		}

		gdoc.PlayerOnTurn = evt.PlayerIndex

		switch evt.Type {
		case ipc.GameEvent_TILE_PLACEMENT_MOVE,
			ipc.GameEvent_EXCHANGE,
			ipc.GameEvent_PASS, ipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS:

			if idx+1 <= len(evts)-1 {
				if evt.Type == ipc.GameEvent_TILE_PLACEMENT_MOVE &&
					evts[idx+1].Type == ipc.GameEvent_PHONY_TILES_RETURNED {

					// In this case, do not play the move since it will be
					// taken back. No need to calculate the bag/board for this event.
					// We still want to append the event, however.
					gdoc.Events = append(gdoc.Events, evt)
					break
					// Go on to the next event.

				}
			}

			tr := evt.MillisRemaining
			// Use playMove to just play the event. This should apply all relevant
			// changes to the doc (scores, keeping track of scoreless turns, etc)
			err = playMove(ctx, gdoc, evt, int64(tr))
			if err != nil {
				return err
			}

		default:
			// If it's another type of game event, all we care about is the cumulative
			// score.
			gdoc.CurrentScores[evt.PlayerIndex] = evt.Cumulative
			gdoc.Events = append(gdoc.Events, evt)

			// XXX not handling 6-consecutive zeroes case
		}
	}
	// At the end, make sure to set the racks to whatever they are in the doc.
	// log.Debug().Interface("savedRacks", savedRacks).Msg("call-assign-racks")
	if rememberRacks {
		err = AssignRacks(gdoc, savedRacks, AssignEmptyIfUnambiguous)
		if err != nil {
			return err
		}
	}

	gdoc.Timers = savedTimers.(*ipc.Timers)
	// Based on the very last game event, we may potentially have to change
	// the "on-turn" player.
	if len(evts) > 0 {
		switch evts[len(evts)-1].Type {
		// If it's one of the four types handled above, we already changed the turn.

		case ipc.GameEvent_PHONY_TILES_RETURNED,
			ipc.GameEvent_CHALLENGE_BONUS:
			// Switch the turn.
			err = assignTurnToNextNonquitter(gdoc, gdoc.PlayerOnTurn)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ProcessGameplayEvent processes a ClientGameplayEvent submitted by userID.
// The game document is also passed in; the caller should take care to load it
// from wherever. This function can modify the document in-place. The caller
// should be responsible for saving it back to whatever store is required if
// there is no error.
func ProcessGameplayEvent(ctx context.Context, cfg *wglconfig.Config, evt *ipc.ClientGameplayEvent,
	userID string, gdoc *ipc.GameDocument) error {

	log := zerolog.Ctx(ctx)

	if gdoc.PlayState == ipc.PlayState_GAME_OVER {
		return errGameNotActive
	}
	if evt.GameId != gdoc.GetUid() {
		return errUnmatchedGameId
	}
	onTurn := gdoc.PlayerOnTurn

	if evt.Type != ipc.ClientGameplayEvent_RESIGN && gdoc.Players[onTurn].UserId != userID {
		return errNotOnTurn
	}
	tr := getTimeRemaining(gdoc, globalNower, onTurn)
	log.Debug().Interface("cge", evt).Int64("now", globalNower.Now()).Int64("time-remaining", tr).Msg("process-gameplay-event")

	if !(gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS &&
		evt.Type == ipc.ClientGameplayEvent_PASS) && timeRanOut(gdoc, globalNower, onTurn) {

		log.Debug().Msg("got-move-too-late")

		// If an ending game gets "challenge" just before "timed out",
		// ignore the challenge, pass instead.
		if gdoc.PlayState == ipc.PlayState_WAITING_FOR_FINAL_PASS {
			log.Debug().Msg("timed out, so passing instead of processing the submitted move")
			evt = &ipc.ClientGameplayEvent{
				Type:   ipc.ClientGameplayEvent_PASS,
				GameId: evt.GameId,
			}
		} else {
			return setTimedOut(ctx, gdoc, onTurn)
		}
	}

	if evt.Type == ipc.ClientGameplayEvent_RESIGN {
		_, resigneridx, found := lo.FindIndexOf(gdoc.Players, func(p *ipc.GameDocument_MinimalPlayerInfo) bool {
			return p.UserId == userID
		})
		if !found {
			return errPlayerNotInGame
		}

		recordTimeOfMove(gdoc, globalNower, onTurn, false)
		gdoc.Events = append(gdoc.Events, &ipc.GameEvent{
			Type:            ipc.GameEvent_RESIGNED,
			PlayerIndex:     uint32(resigneridx),
			MillisRemaining: int32(gdoc.Timers.TimeRemaining[resigneridx]),
		})
		gdoc.Players[resigneridx].Quit = true
		winner, found := findOnlyNonquitter(gdoc)
		if found {
			gdoc.Winner = int32(winner)
			gdoc.EndReason = ipc.GameEndReason_RESIGNED
			gdoc.PlayState = ipc.PlayState_GAME_OVER

			// XXX perform endgame duties -- this is definitely outside the scope
			// of this package.
		} else {
			// assign next turn
			err := assignTurnToNextNonquitter(gdoc, onTurn)
			if err != nil {
				return err
			}
		}

	} else {
		// convt to internal move
		gevt, err := clientEventToGameEvent(cfg, evt, gdoc)
		if err != nil {
			return err
		}
		// At this point, we have validated the play can be made from
		// the player's rack, but we haven't validated the play itself
		// (adherence to rules, valid words if applicable, etc)

		err = playMove(ctx, gdoc, gevt, tr)
		if err != nil {
			return err
		}
	}

	return nil
}

// ToCGP converts the game to a CGP string.
func ToCGP(cfg *wglconfig.Config, gdoc *ipc.GameDocument) (string, error) {
	dist, err := tilemapping.GetDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return "", err
	}
	rm := dist.TileMapping()
	fen := board.ToFEN(gdoc.Board, dist)
	var ss strings.Builder
	ss.WriteString(fen)
	ss.WriteString(" ")
	for i := gdoc.PlayerOnTurn; i < gdoc.PlayerOnTurn+uint32(len(gdoc.Players)); i++ {
		rack := gdoc.Racks[int(i)%len(gdoc.Players)]
		mls := tilemapping.FromByteArr(rack)
		for _, ml := range mls {
			uv := ml.UserVisible(rm, false)
			rc := utf8.RuneCountInString(uv)
			if rc > 1 {
				ss.WriteString("[")
			}
			ss.WriteString(uv)
			if rc > 1 {
				ss.WriteString("]")
			}
		}
		if i != gdoc.PlayerOnTurn+uint32(len(gdoc.Players))-1 {
			ss.WriteString("/")
		}
	}
	ss.WriteString(" ")
	for i := gdoc.PlayerOnTurn; i < gdoc.PlayerOnTurn+uint32(len(gdoc.Players)); i++ {
		score := gdoc.CurrentScores[int(i)%len(gdoc.Players)]
		ss.WriteString(strconv.Itoa(int(score)))
		if i != gdoc.PlayerOnTurn+uint32(len(gdoc.Players))-1 {
			ss.WriteString("/")
		}
	}
	ss.WriteString(" ")
	ss.WriteString(strconv.Itoa(int(gdoc.ScorelessTurns)))
	ss.WriteString(" ")
	ss.WriteString("lex ")
	ss.WriteString(gdoc.Lexicon)
	ss.WriteString("; ld ")
	ss.WriteString(gdoc.LetterDistribution)
	ss.WriteString(";")
	return ss.String(), nil
}

func clientEventToGameEvent(cfg *wglconfig.Config, evt *ipc.ClientGameplayEvent, gdoc *ipc.GameDocument) (*ipc.GameEvent, error) {
	playerid := gdoc.PlayerOnTurn
	rackmw := tilemapping.FromByteArr(gdoc.Racks[playerid])

	dist, err := tilemapping.GetDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return nil, err
	}
	if len(evt.Tiles) > 0 && len(evt.MachineLetters) > 0 {
		return nil, errors.New("cannot specify both tiles and machineletters")
	}
	switch evt.Type {
	case ipc.ClientGameplayEvent_TILE_PLACEMENT:
		row, col, dir := fromBoardGameCoords(evt.PositionCoords)
		var mw []tilemapping.MachineLetter
		var err error
		if len(evt.Tiles) > 0 {
			mw, err = tilemapping.ToMachineLetters(evt.Tiles, dist.TileMapping())
			if err != nil {
				return nil, err
			}
		} else {
			mw = tilemapping.FromByteArr(evt.MachineLetters)
		}
		_, err = tilemapping.Leave(rackmw, mw, false)
		if err != nil {
			return nil, err
		}
		return &ipc.GameEvent{
			Row:         int32(row),
			Column:      int32(col),
			Direction:   dir,
			Type:        ipc.GameEvent_TILE_PLACEMENT_MOVE,
			Rack:        gdoc.Racks[playerid],
			PlayedTiles: tilemapping.MachineWord(mw).ToByteArr(),
			Position:    evt.PositionCoords,
			PlayerIndex: gdoc.PlayerOnTurn,
		}, nil

	case ipc.ClientGameplayEvent_PASS:
		return &ipc.GameEvent{
			Type:        ipc.GameEvent_PASS,
			Rack:        gdoc.Racks[playerid],
			PlayerIndex: gdoc.PlayerOnTurn,
		}, nil
	case ipc.ClientGameplayEvent_EXCHANGE:
		var mw []tilemapping.MachineLetter
		var err error
		if len(evt.Tiles) > 0 {
			mw, err = tilemapping.ToMachineLetters(evt.Tiles, dist.TileMapping())
			if err != nil {
				return nil, err
			}
		} else {
			mw = tilemapping.FromByteArr(evt.MachineLetters)
		}
		_, err = tilemapping.Leave(rackmw, mw, true)
		if err != nil {
			return nil, err
		}
		return &ipc.GameEvent{
			Type:        ipc.GameEvent_EXCHANGE,
			Rack:        gdoc.Racks[playerid],
			Exchanged:   tilemapping.MachineWord(mw).ToByteArr(),
			PlayerIndex: gdoc.PlayerOnTurn,
		}, nil
	case ipc.ClientGameplayEvent_CHALLENGE_PLAY:
		return &ipc.GameEvent{
			Type:        ipc.GameEvent_CHALLENGE,
			PlayerIndex: gdoc.PlayerOnTurn,
			Rack:        gdoc.Racks[playerid],
		}, nil

	}
	return nil, errors.New("unhandled evt type: " + evt.Type.String())
}

func fromBoardGameCoords(c string) (int, int, ipc.GameEvent_Direction) {
	vMatches := reVertical.FindStringSubmatch(c)
	var row, col int
	if len(vMatches) == 3 {
		// It's vertical
		row, _ = strconv.Atoi(vMatches[2])
		col = int(vMatches[1][0] - 'A')
		return row - 1, col, ipc.GameEvent_VERTICAL
	}
	hMatches := reHorizontal.FindStringSubmatch(c)
	if len(hMatches) == 3 {
		row, _ = strconv.Atoi(hMatches[1])
		col = int(hMatches[2][0] - 'A')
		return row - 1, col, ipc.GameEvent_HORIZONTAL
	}
	// It's inconvenient that this is actually a valid set of coordinates.
	// Maybe this function should return an error.
	return 0, 0, ipc.GameEvent_HORIZONTAL
}

func setTimedOut(ctx context.Context, gdoc *ipc.GameDocument, onturn uint32) error {
	log := zerolog.Ctx(ctx)
	log.Debug().Interface("playstate", gdoc.PlayState).Msg("timed out!")
	// The losing player always overtimes by the maximum amount.
	// Not less, even if no moves in the final minute.
	// Not more, even if game is abandoned and resumed/adjudicated much later.
	gdoc.Timers.TimeRemaining[onturn] = int64(gdoc.Timers.MaxOvertime * -60000)
	gdoc.Events = append(gdoc.Events, &ipc.GameEvent{
		Type:            ipc.GameEvent_TIMED_OUT,
		PlayerIndex:     onturn,
		MillisRemaining: int32(gdoc.Timers.TimeRemaining[onturn]),
	})
	gdoc.Players[onturn].Quit = true
	winner, found := findOnlyNonquitter(gdoc)
	if found {
		gdoc.Winner = int32(winner)
		gdoc.EndReason = ipc.GameEndReason_TIME
		gdoc.PlayState = ipc.PlayState_GAME_OVER
	} else {
		err := assignTurnToNextNonquitter(gdoc, onturn)
		if err != nil {
			return err
		}
	}
	// perform endgame duties outside of the scope of this
	return nil
}

func findOnlyNonquitter(gdoc *ipc.GameDocument) (int, bool) {
	numQuit := 0
	nPlayers := len(gdoc.Players)
	winner := 0
	for i := range gdoc.Players {
		if gdoc.Players[i].Quit {
			numQuit++
		} else {
			winner = i
		}
	}
	if nPlayers == numQuit+1 {
		return winner, true
	}
	return -1, false
}

func assignTurnToNextNonquitter(gdoc *ipc.GameDocument, start uint32) error {
	if gdoc.PlayState == ipc.PlayState_GAME_OVER {
		// Game is already over, don't bother changing on-turn
		return nil
	}
	i := (start + uint32(1)) % uint32(len(gdoc.Players))
	for i != start {
		if !gdoc.Players[i].Quit {
			gdoc.PlayerOnTurn = i
			log.Debug().Uint32("on-turn", i).Msg("assign-turn")
			return nil
		}
		i = (i + uint32(1)) % uint32(len(gdoc.Players))
	}
	return errors.New("everyone quit")

}

// XXX need a TimedOut function here as well.
// XXX: no, put it in the top level.
