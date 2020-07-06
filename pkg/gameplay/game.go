// Package gameplay should know nothing about protocols or databases.
// It is mostly a pass-through interface to a Macondo game,
// but also implements a timer and other related logic.
// This is a use-case in the clean architecture hierarchy.
package gameplay

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/domino14/macondo/alphabet"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/move"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/game"
)

const (
	CrosswordGame string = "CrosswordGame"
)

var (
	errGameNotActive   = errors.New("game is not currently active")
	errNotOnTurn       = errors.New("player not on turn")
	errTimeDidntRunOut = errors.New("got time ran out, but it did not actually")
)

// GameStore is an interface for getting a full game.
type GameStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
	Set(context.Context, *entity.Game) error
}

// InstantiateNewGame instantiates a game and returns it.
func InstantiateNewGame(ctx context.Context, gameStore GameStore, cfg *config.Config,
	users [2]*entity.User, req *pb.GameRequest) (*entity.Game, error) {

	var players []*macondopb.PlayerInfo

	for _, u := range users {
		players = append(players, &macondopb.PlayerInfo{
			Nickname: u.Username,
			UserId:   u.UUID,
			RealName: u.RealName(),
		})
	}

	var bd []string
	if req.Rules == nil {
		return nil, errors.New("no rules")
	}
	switch req.Rules.BoardLayoutName {
	case CrosswordGame:
		bd = board.CrosswordGameBoard
	default:
		return nil, errors.New("unsupported board layout")
	}

	rules, err := game.NewGameRules(&cfg.MacondoConfig, bd,
		req.Lexicon, req.Rules.LetterDistributionName)

	if err != nil {
		return nil, err
	}
	g, err := game.NewGame(rules, players)
	if err != nil {
		return nil, err
	}
	// StartGame creates a new history Uid and deals tiles, etc.
	g.StartGame()
	g.SetBackupMode(game.InteractiveGameplayMode)
	g.SetChallengeRule(req.ChallengeRule)

	entGame := entity.NewGame(g, req)
	// Save the game to the store.
	if err = gameStore.Set(ctx, entGame); err != nil {
		return nil, err
	}
	return entGame, nil
	// We return the instantiated game. Although the tiles have technically been
	// dealt out, we need to call StartGame to actually start the timer
	// and forward game events to the right channels.
}

func clientEventToMove(cge *pb.ClientGameplayEvent, g *game.Game) (*move.Move, error) {
	playerid := g.PlayerOnTurn()
	rack := g.RackFor(playerid)

	switch cge.Type {
	case pb.ClientGameplayEvent_TILE_PLACEMENT:
		m, err := g.CreateAndScorePlacementMove(cge.PositionCoords, cge.Tiles, rack.String())
		if err != nil {
			return nil, err
		}
		log.Debug().Msg("got a client gameplay event tile placement")
		// Note that we don't validate the move here, but we do so later.
		return m, nil

	case pb.ClientGameplayEvent_PASS:
		m := move.NewPassMove(rack.TilesOn(), g.Alphabet())
		return m, nil
	case pb.ClientGameplayEvent_EXCHANGE:
		tiles, err := alphabet.ToMachineWord(cge.Tiles, g.Alphabet())
		if err != nil {
			return nil, err
		}
		leaveMW, err := game.Leave(rack.TilesOn(), tiles)
		if err != nil {
			return nil, err
		}
		m := move.NewExchangeMove(tiles, leaveMW, g.Alphabet())
		return m, nil
	}
	return nil, errors.New("client gameplay event not handled")
}

func StartGame(ctx context.Context, gameStore GameStore, eventChan chan<- *entity.EventWrapper, id string) error {
	// Note that StartGame does _not_ start the Macondo game, which
	// has already started, but we don't "know" that. It is _this_
	// function that will actually start the game in the user's eyes.
	// It needs to reset the timer to now.
	entGame, err := gameStore.Get(ctx, id)
	if err != nil {
		return err
	}
	// This should be True, see comment above.
	if entGame.Game.Playing() != macondopb.PlayState_PLAYING {
		return errGameNotActive
	}
	log.Debug().Str("gameid", id).Msg("reset timers (and start)")
	entGame.ResetTimersAndStart()

	// Save the game back to the store always.
	if err := gameStore.Set(ctx, entGame); err != nil {
		return err
	}
	if err := entGame.RegisterChangeHook(eventChan); err != nil {
		return err
	}
	log.Debug().Interface("history", entGame.Game.History()).Msg("game history")

	evt := entGame.HistoryRefresherEvent()
	wrapped := entity.WrapEvent(evt, pb.MessageType_GAME_HISTORY_REFRESHER,
		entGame.GameID())
	wrapped.AddAudience(entity.AudGameTV, entGame.GameID())
	for _, p := range players(entGame) {
		wrapped.AddAudience(entity.AudUser, p)
	}
	entGame.SendChange(wrapped)

	return nil
}

func players(entGame *entity.Game) []string {
	// Return user IDs of players.
	ps := []string{}
	for _, p := range entGame.History().Players {
		ps = append(ps, p.UserId)
	}
	return ps
}

func handleChallenge(ctx context.Context, entGame *entity.Game, gameStore GameStore,
	userStore user.Store, timeRemaining int, challengerID string) error {
	if entGame.ChallengeRule() == macondopb.ChallengeRule_VOID {
		// The front-end shouldn't even show the button.
		return errors.New("challenges not acceptable in void")
	}
	numEvts := len(entGame.Game.History().Events)
	// curTurn := entGame.Game.Turn()
	valid, err := entGame.Game.ChallengeEvent(0, timeRemaining)
	if err != nil {
		return err
	}
	resultEvent := &pb.ServerChallengeResultEvent{
		Valid:         valid,
		ChallengeRule: entGame.ChallengeRule(),
		Challenger:    challengerID,
	}
	evt := entity.WrapEvent(resultEvent, pb.MessageType_SERVER_CHALLENGE_RESULT_EVENT,
		entGame.GameID())
	evt.AddAudience(entity.AudGame, entGame.GameID())
	evt.AddAudience(entity.AudGameTV, entGame.GameID())
	entGame.SendChange(evt)

	// We need to send the turn history from curTurn onwards.
	// turns := entGame.History().Turns[curTurn-1:]
	// refresher := &pb.GameTurnsRefresher{
	// 	Turns:        turns,
	// 	PlayState:    entGame.Game.Playing(),
	// 	StartingTurn: int32(curTurn - 1),
	// }
	// evt = entity.WrapEvent(refresher, pb.MessageType_GAME_TURNS_REFRESHER,
	// 	entGame.GameID())

	// Send just the whole history for now. Sorry, this game turns refresher
	// thing is just too complicated to handle on the front end; we can
	// try again later if we need to reduce bandwidth.

	// refresher := entGame.HistoryRefresherEvent()
	// evt = entity.WrapEvent(refresher, pb.MessageType_GAME_HISTORY_REFRESHER,
	// 	entGame.GameID())

	newEvts := entGame.Game.History().Events
	if len(newEvts) > numEvts {
		if len(newEvts)-numEvts > 1 {
			return fmt.Errorf("unexpected number of new evts: %v %v",
				newEvts, numEvts)
		}
		// This event is either a bonus addition, a loss of turn from an
		// incorrect challenge, or a "phony tiles removed" event.
		relevantEvent := newEvts[len(newEvts)-1]
		sge := &pb.ServerGameplayEvent{
			Event:         relevantEvent,
			GameId:        entGame.GameID(),
			TimeRemaining: int32(relevantEvent.MillisRemaining),
			Playing:       entGame.Game.Playing(),
			// Does the user id matter?
		}
		evt = entity.WrapEvent(sge, pb.MessageType_SERVER_GAMEPLAY_EVENT,
			entGame.GameID())
		evt.AddAudience(entity.AudGameTV, entGame.GameID())
		for _, uid := range players(entGame) {
			evt.AddAudience(entity.AudUser, uid)
		}
		entGame.SendChange(evt)
	}

	if entGame.Game.Playing() == macondopb.PlayState_GAME_OVER {
		discernEndgameReason(entGame)
		performEndgameDuties(ctx, entGame, userStore)
	}

	err = gameStore.Set(ctx, entGame)
	if err != nil {
		return err
	}

	return nil
}

func PlayMove(ctx context.Context, gameStore GameStore, userStore user.Store, userID string,
	cge *pb.ClientGameplayEvent) error {

	// XXX: VERIFY THAT THE CLIENT GAME ID CORRESPONDS TO THE GAME
	// THE PLAYER IS PLAYING!
	entGame, err := gameStore.Get(ctx, cge.GameId)
	if err != nil {
		return err
	}
	entGame.Lock()
	defer entGame.Unlock()
	if entGame.Game.Playing() == macondopb.PlayState_GAME_OVER {
		return errGameNotActive
	}
	onTurn := entGame.Game.PlayerOnTurn()

	// Ensure that it is actually the correct player's turn
	if entGame.Game.PlayerIDOnTurn() != userID {
		return errNotOnTurn
	}
	timeRemaining := entGame.TimeRemaining(onTurn)
	log.Debug().Int("time-remaining", timeRemaining).Msg("checking-time-remaining")
	// Check that we didn't run out of time.
	if timeRemaining < 0 {
		// Game is over!
		log.Debug().Msg("got-move-too-late")
		entGame.Game.SetPlaying(macondopb.PlayState_GAME_OVER)
		// Basically skip to the bottom and exit.
		return setTimedOut(ctx, entGame, onTurn, gameStore, userStore)
	}

	log.Debug().Msg("going to turn into a macondo gameevent")

	if cge.Type == pb.ClientGameplayEvent_CHALLENGE_PLAY {
		// Handle in another way
		return handleChallenge(ctx, entGame, gameStore, userStore, timeRemaining, userID)
	}

	// Turn the event into a macondo GameEvent.
	m, err := clientEventToMove(cge, &entGame.Game)
	if err != nil {
		return err
	}
	log.Debug().Msg("validating")

	_, err = entGame.Game.ValidateMove(m)
	if err != nil {
		return err
	}
	oldTurnLength := len(entGame.Game.History().Events)

	// Don't back up the move, but add to history
	log.Debug().Msg("playing the move")
	// Register time BEFORE playing the move, so the turn doesn't switch.
	entGame.RecordTimeOfMove(onTurn)
	err = entGame.Game.PlayMove(m, true, timeRemaining)
	if err != nil {
		return err
	}
	// Get the turn(s) that we _just_ appended to the history
	turns := entGame.Game.History().Events[oldTurnLength:]
	if len(turns) > 1 {
		// This happens with six zeroes for example.
		log.Debug().Msg("more than one turn appended")
	}
	// Create a set of ServerGameplayEvents to send back.
	log.Debug().Interface("turns", turns).Msg("sending turns back")
	evts := []*pb.ServerGameplayEvent{}

	for _, evt := range turns {
		sge := &pb.ServerGameplayEvent{}
		sge.Event = evt
		sge.GameId = cge.GameId
		// note that `onTurn` is correct as it was saved up there before
		// we played the turn.
		sge.TimeRemaining = int32(entGame.TimeRemaining(onTurn))
		sge.NewRack = entGame.Game.RackLettersFor(onTurn)
		sge.Playing = entGame.Game.Playing()
		sge.UserId = userID
		evts = append(evts, sge)
	}

	// Since the move was successful, we assume the user gameplay event is valid.
	// Send the server change event.
	playing := entGame.Game.Playing()
	players := players(entGame)
	for _, sge := range evts {
		wrapped := entity.WrapEvent(sge, pb.MessageType_SERVER_GAMEPLAY_EVENT,
			entGame.GameID())
		wrapped.AddAudience(entity.AudGameTV, entGame.GameID())
		for _, p := range players {
			wrapped.AddAudience(entity.AudUser, p)
		}
		entGame.SendChange(wrapped)
	}
	if playing == macondopb.PlayState_GAME_OVER {
		discernEndgameReason(entGame)
		performEndgameDuties(ctx, entGame, userStore)
	}

	err = gameStore.Set(ctx, entGame)
	if err != nil {
		return err
	}

	return nil
}

func TimedOut(ctx context.Context, gameStore GameStore, userStore user.Store,
	timedout string, gameID string) error {
	// XXX: VERIFY THAT THE GAME ID is the client's current game!!
	// Note: we can get this event multiple times; the opponent and the player on turn
	// both send it.
	log.Debug().Str("timedout", timedout).Msg("got-timed-out")
	entGame, err := gameStore.Get(ctx, gameID)
	if err != nil {
		return err
	}
	entGame.Lock()
	defer entGame.Unlock()
	if entGame.Game.Playing() == macondopb.PlayState_GAME_OVER {
		log.Debug().Msg("game not active anymore.")
		return nil
	}
	onTurn := entGame.Game.PlayerOnTurn()

	// Ensure that it is actually the correct player's turn
	if entGame.Game.PlayerIDOnTurn() != timedout {
		return errNotOnTurn
	}
	if entGame.TimeRemaining(onTurn) > 0 {
		log.Error().Int("TimeRemaining", entGame.TimeRemaining(onTurn)).
			Int("onturn", onTurn).Msg("time-didnt-run-out")
		return errTimeDidntRunOut
	}
	// Ok, the time did run out after all.

	return setTimedOut(ctx, entGame, onTurn, gameStore, userStore)
}

// sanitizeEvent removes rack information from the event; it is meant to be
// sent to someone currently in a game.
func sanitizeEvent(sge *pb.ServerGameplayEvent) *pb.ServerGameplayEvent {
	cloned := proto.Clone(sge).(*pb.ServerGameplayEvent)
	cloned.NewRack = ""
	cloned.Event.Rack = ""
	if len(cloned.Event.Exchanged) > 0 {
		cloned.Event.Exchanged = strconv.Itoa(len(cloned.Event.Exchanged))
	}
	return cloned
}

func setTimedOut(ctx context.Context, entGame *entity.Game, pidx int, gameStore GameStore,
	userStore user.Store) error {
	log.Debug().Msg("timed out!")
	entGame.Game.SetPlaying(macondopb.PlayState_GAME_OVER)

	// And send a game end event.
	entGame.SetGameEndReason(pb.GameEndReason_TIME)
	entGame.SetWinnerIdx(1 - pidx)
	entGame.SetLoserIdx(pidx)
	performEndgameDuties(ctx, entGame, userStore)

	// Store the game back into the store
	err := gameStore.Set(ctx, entGame)
	if err != nil {
		return err
	}

	return nil
}

func gameEndedEvent(ctx context.Context, g *entity.Game, userStore user.Store) *pb.GameEndedEvent {
	var winner, loser string
	var tie bool
	winnerIdx := g.GetWinnerIdx()
	if winnerIdx == 0 || winnerIdx == -1 {
		winner = g.History().Players[0].Nickname
		loser = g.History().Players[1].Nickname
	} else if winnerIdx == 1 {
		winner = g.History().Players[1].Nickname
		loser = g.History().Players[0].Nickname
	}
	if winnerIdx == -1 {
		tie = true
	}

	scores := map[string]int32{
		g.History().Players[0].Nickname: int32(g.PointsFor(0)),
		g.History().Players[1].Nickname: int32(g.PointsFor(1))}

	ratings := map[string]int32{}
	var err error
	if g.CreationRequest().RatingMode == pb.RatingMode_RATED {
		ratings, err = rate(ctx, scores, g, winner, userStore)
		if err != nil {
			log.Err(err).Msg("rating-error")
		}
	}

	// Otherwise the winner will be blank, because it was a tie.
	evt := &pb.GameEndedEvent{
		Scores:     scores,
		NewRatings: ratings,
		EndReason:  g.GameEndReason(),
		Winner:     winner,
		Loser:      loser,
		Tie:        tie,
	}
	log.Debug().Interface("game-ended-event", evt).Msg("game-ended")
	return evt
}

func performEndgameDuties(ctx context.Context, g *entity.Game, userStore user.Store) {
	wrapped := entity.WrapEvent(gameEndedEvent(ctx, g, userStore),
		pb.MessageType_GAME_ENDED_EVENT, g.GameID())
	// Once the game ends, we do not need to "sanitize" the packets
	// going to the users anymore. So just send the data to the right
	// audiences.
	wrapped.AddAudience(entity.AudGame, g.GameID())
	wrapped.AddAudience(entity.AudGameTV, g.GameID())
	g.SendChange(wrapped)

}

func discernEndgameReason(g *entity.Game) {
	// Figure out why the game ended. Here there are only two options,
	// standard or six-zero. The game ending on a timeout is handled in
	// another branch (see setTimedOut above) and resignation/etc will
	// also be handled elsewhere.
	if g.RackLettersFor(0) == "" || g.RackLettersFor(1) == "" {
		g.SetGameEndReason(pb.GameEndReason_STANDARD)
	} else {
		g.SetGameEndReason(pb.GameEndReason_CONSECUTIVE_ZEROES)
	}
	if g.PointsFor(0) > g.PointsFor(1) {
		g.SetWinnerIdx(0)
		g.SetLoserIdx(1)
	} else if g.PointsFor(1) > g.PointsFor(0) {
		g.SetWinnerIdx(1)
		g.SetLoserIdx(0)
	} else {
		// They're the same.
		g.SetWinnerIdx(-1)
		g.SetLoserIdx(-1)
	}
}
