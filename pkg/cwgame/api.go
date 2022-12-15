package cwgame

import (
	"context"
	"errors"
	"regexp"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame/board"
	"github.com/domino14/liwords/pkg/cwgame/dawg"
	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/domino14/liwords/pkg/cwgame/tiles"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
)

var (
	errGameNotActive             = errors.New("game not active")
	errNotOnTurn                 = errors.New("not on turn")
	errOnlyPassOrChallenge       = errors.New("can only pass or challenge")
	errExchangeNotPermitted      = errors.New("you can only exchange with 7 or more tiles in the bag")
	errMoveTypeNotUserInputtable = errors.New("that move type is not available")
	errResignNotValid            = errors.New("you are not allowed to resign")
	errStartNotPermitted         = errors.New("game has already been started")
	errUnmatchedGameId           = errors.New("game ids do not match")
	errPlayerNotInGame           = errors.New("player not in this game")
)

// move is an intermediate type used by this package, to aid in the conversion from
// a frontend-generated ClientGameplayEvent into a GameEvent that gets saved in a
// GameHistory.
type move struct {
	row, col  int
	direction ipc.GameEvent_Direction
	mtype     ipc.GameEvent_Type
	leave     []runemapping.MachineLetter
	tilesUsed []runemapping.MachineLetter
	clientEvt *ipc.ClientGameplayEvent
}

var reVertical, reHorizontal *regexp.Regexp

func init() {
	reVertical = regexp.MustCompile(`^(?P<col>[A-Z])(?P<row>[0-9]+)$`)
	reHorizontal = regexp.MustCompile(`^(?P<row>[0-9]+)(?P<col>[A-Z])$`)
}

// NewGame creates a new GameDocument. The playerinfo array contains
// the players, which must be in order of who goes first!
func NewGame(cfg *config.Config, rules *GameRules, playerinfo []*ipc.GameDocument_MinimalPlayerInfo) (*ipc.GameDocument, error) {
	// try to instantiate all aspects of the game from the given rules.

	dist, err := tiles.GetDistribution(cfg, rules.distname)
	if err != nil {
		return nil, err
	}
	_, err = dawg.GetDawg(cfg, rules.lexicon)
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
		Events:             make([]*ipc.GameEvent, 0),
		Players:            playerinfo,
		Lexicon:            rules.lexicon,
		Version:            GameDocumentVersion,
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
		},
		PlayState: ipc.PlayState_UNSTARTED,
	}

	return gdoc, nil
}

func StartGame(ctx context.Context, gdoc *ipc.GameDocument) error {
	if gdoc.PlayState != ipc.PlayState_UNSTARTED {
		return errStartNotPermitted
	}
	for idx := range gdoc.Players {
		t := make([]runemapping.MachineLetter, RackTileLimit)
		err := tiles.Draw(gdoc.Bag, RackTileLimit, t)
		if err != nil {
			return err
		}
		gdoc.Racks[idx] = runemapping.MachineWord(t).ToByteArr()
	}
	resetTimersAndStart(gdoc, globalNower)
	// Outside of this:
	// XXX: send changes to channel(s); see StartGame in gameplay package.
	// XXX: outside of this, send rematch event
	// XXX: potentially send bot move request?
	return nil
}

// ProcessGameplayEvent processes a ClientGameplayEvent submitted by userID.
// The game document is also passed in; the caller should take care to load it
// from wherever. This function can modify the document in-place. The caller
// should be responsible for saving it back to whatever store is required if
// there is no error.
func ProcessGameplayEvent(ctx context.Context, evt *ipc.ClientGameplayEvent,
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
		m, err := clientEventToMove(ctx, evt, gdoc)
		if err != nil {
			return err
		}
		err = playMove(ctx, gdoc, m, tr)
		if err != nil {
			return err
		}
	}

	return nil
}

func clientEventToMove(ctx context.Context, evt *ipc.ClientGameplayEvent, gdoc *ipc.GameDocument) (move, error) {
	playerid := gdoc.PlayerOnTurn
	rackmw := runemapping.FromByteArr(gdoc.Racks[playerid])
	cfg, ok := ctx.Value(config.CtxKeyword).(*config.Config)
	m := move{}
	if !ok {
		return m, errors.New("config does not exist in context")
	}

	dist, err := tiles.GetDistribution(cfg, gdoc.LetterDistribution)
	if err != nil {
		return m, err
	}

	switch evt.Type {
	case ipc.ClientGameplayEvent_TILE_PLACEMENT:
		row, col, dir := fromBoardGameCoords(evt.PositionCoords)
		mw, err := runemapping.ToMachineLetters(evt.Tiles, dist.RuneMapping())
		if err != nil {
			return m, err
		}
		leave, err := Leave(rackmw, mw)
		if err != nil {
			return m, err
		}
		return move{
			row:       row,
			col:       col,
			direction: dir,
			mtype:     ipc.GameEvent_TILE_PLACEMENT_MOVE,
			tilesUsed: mw,
			leave:     leave,
			clientEvt: evt,
		}, nil

	case ipc.ClientGameplayEvent_PASS:
		return move{
			mtype:     ipc.GameEvent_PASS,
			leave:     rackmw,
			clientEvt: evt,
		}, nil
	case ipc.ClientGameplayEvent_EXCHANGE:
		mw, err := runemapping.ToMachineLetters(evt.Tiles, dist.RuneMapping())
		if err != nil {
			return m, err
		}
		leave, err := Leave(rackmw, mw)
		if err != nil {
			return m, err
		}

		return move{
			mtype:     ipc.GameEvent_EXCHANGE,
			tilesUsed: mw,
			leave:     leave,
			clientEvt: evt,
		}, nil
	case ipc.ClientGameplayEvent_CHALLENGE_PLAY:
		return move{
			mtype:     ipc.GameEvent_CHALLENGE,
			leave:     rackmw,
			clientEvt: evt}, nil

	}
	return m, errors.New("unhandled evt type: " + evt.Type.String())
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
	i := (start + uint32(1)) % uint32(len(gdoc.Players))
	for i != start {
		if !gdoc.Players[i].Quit {
			gdoc.PlayerOnTurn = i
			return nil
		}
		i = (i + uint32(1)) % uint32(len(gdoc.Players))
	}
	return errors.New("everyone quit")

}

// XXX need a TimedOut function here as well.
// XXX: no, put it in the top level.
