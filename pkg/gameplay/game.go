// Package gameplay should know nothing about protocols or databases.
// It is mostly a pass-through interface to a Macondo game,
// but also implements a timer and other related logic.
// This is a use-case in the clean architecture hierarchy.
package gameplay

import (
	"context"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/alphabet"
	"github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/cross_set"
	"github.com/domino14/macondo/gaddag"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/move"
	"github.com/domino14/macondo/runner"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
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
	Create(context.Context, *entity.Game) error
	ListActive(context.Context) ([]*pb.GameMeta, error)
	SetGameEventChan(c chan<- *entity.EventWrapper)
}

type ConfigCtxKey string

// InstantiateNewGame instantiates a game and returns it.
func InstantiateNewGame(ctx context.Context, gameStore GameStore, cfg *config.Config,
	users [2]*entity.User, assignedFirst int, req *pb.GameRequest) (*entity.Game, error) {

	var players []*macondopb.PlayerInfo
	var dbids [2]uint

	for idx, u := range users {
		players = append(players, &macondopb.PlayerInfo{
			Nickname: u.Username,
			UserId:   u.UUID,
			RealName: u.RealName(),
		})
		dbids[idx] = u.ID
	}

	if req.Rules == nil {
		return nil, errors.New("no rules")
	}

	var bd []string
	switch req.Rules.BoardLayoutName {
	case entity.CrosswordGame:
		bd = board.CrosswordGameBoard
	default:
		return nil, errors.New("unsupported board layout")
	}

	firstAssigned := false
	if assignedFirst != -1 {
		firstAssigned = true
	}

	dist, err := alphabet.LoadLetterDistribution(&cfg.MacondoConfig, req.Rules.LetterDistributionName)
	if err != nil {
		return nil, err
	}

	gd, err := gaddag.LoadFromCache(&cfg.MacondoConfig, req.Lexicon)
	if err != nil {
		return nil, err
	}

	rules := game.NewGameRules(
		&cfg.MacondoConfig, dist, board.MakeBoard(bd),
		&gaddag.Lexicon{GenericDawg: gd},
		cross_set.CrossScoreOnlyGenerator{Dist: dist})

	runner, err := runner.NewGameRunnerFromRules(&runner.GameOptions{
		FirstIsAssigned: firstAssigned,
		GoesFirst:       assignedFirst,
		ChallengeRule:   req.ChallengeRule,
	}, players, rules)
	if err != nil {
		return nil, err
	}

	entGame := entity.NewGame(&runner.Game, req)
	entGame.PlayerDBIDs = dbids
	// Save the game to the store.
	if err = gameStore.Create(ctx, entGame); err != nil {
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

	newEvts := entGame.Game.History().Events

	for eidx := numEvts; eidx < len(newEvts); eidx++ {
		sge := &pb.ServerGameplayEvent{
			Event:         newEvts[eidx],
			GameId:        entGame.GameID(),
			TimeRemaining: int32(newEvts[eidx].MillisRemaining),
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
		checkGameOverAndModifyScores(ctx, entGame, userStore)
	}

	return gameStore.Set(ctx, entGame)

}

// PlayMove handles a gameplay event from the socket
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
		log.Info().Interface("client-event", cge).Msg("not on turn")
		return errNotOnTurn
	}
	timeRemaining := entGame.TimeRemaining(onTurn)
	log.Debug().Int("time-remaining", timeRemaining).Msg("checking-time-remaining")
	// Check that we didn't run out of time.
	if entGame.TimeRanOut(onTurn) {
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
		checkGameOverAndModifyScores(ctx, entGame, userStore)
	}

	return gameStore.Set(ctx, entGame)
}

func checkGameOverAndModifyScores(ctx context.Context, entGame *entity.Game, userStore user.Store) {
	discernEndgameReason(entGame)
	performEndgameDuties(ctx, entGame, userStore)
}

// TimedOut gets called when the client thinks the user's time ran out. We
// verify that that is actually the case.
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
	if !entGame.TimeRanOut(onTurn) {
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
	return gameStore.Set(ctx, entGame)
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
	var now = time.Now().Unix()
	if g.CreationRequest().RatingMode == pb.RatingMode_RATED {
		ratings, err = Rate(ctx, scores, g, winner, userStore, now)
		if err != nil {
			log.Err(err).Msg("rating-error")
		}
	}
	evt := &pb.GameEndedEvent{
		Scores:     scores,
		NewRatings: ratings,
		EndReason:  g.GameEndReason,
		Winner:     winner,
		Loser:      loser,
		Tie:        tie,
		Time:       g.Timers.TimeOfLastUpdate,
	}

	log.Debug().Interface("game-ended-event", evt).Msg("game-ended")
	return evt
}

func performEndgameDuties(ctx context.Context, g *entity.Game, userStore user.Store) {
	evts := []*pb.ServerGameplayEvent{}

	var p0penalty, p1penalty int
	if g.CachedTimeRemaining(0) < 0 {
		p0penalty = 10 * int(math.Ceil(float64(-g.CachedTimeRemaining(0))/60000.0))
	}
	if g.CachedTimeRemaining(1) < 0 {
		p1penalty = 10 * int(math.Ceil(float64(-g.CachedTimeRemaining(1))/60000.0))
	}

	if p0penalty > 0 {
		newscore := g.PointsFor(0) - p0penalty
		// >Pakorn: ISBALI (time) -10 409
		evts = append(evts, &pb.ServerGameplayEvent{
			Event: &macondopb.GameEvent{
				Nickname:        g.History().Players[0].Nickname,
				Rack:            g.RackLettersFor(0),
				Type:            macondopb.GameEvent_TIME_PENALTY,
				LostScore:       int32(p0penalty),
				Cumulative:      int32(newscore),
				MillisRemaining: int32(g.CachedTimeRemaining(0)),
			},
			GameId:  g.GameID(),
			Playing: macondopb.PlayState_GAME_OVER,
		})
		g.SetPointsFor(0, newscore)
	}
	if p1penalty > 0 {
		newscore := g.PointsFor(1) - p1penalty
		evts = append(evts, &pb.ServerGameplayEvent{
			Event: &macondopb.GameEvent{
				Nickname:        g.History().Players[1].Nickname,
				Rack:            g.RackLettersFor(1),
				Type:            macondopb.GameEvent_TIME_PENALTY,
				LostScore:       int32(p1penalty),
				Cumulative:      int32(newscore),
				MillisRemaining: int32(g.CachedTimeRemaining(1)),
			},
			GameId:  g.GameID(),
			Playing: macondopb.PlayState_GAME_OVER,
		})
		g.SetPointsFor(1, newscore)
	}

	for _, sge := range evts {
		wrapped := entity.WrapEvent(sge, pb.MessageType_SERVER_GAMEPLAY_EVENT,
			g.GameID())
		wrapped.AddAudience(entity.AudGameTV, g.GameID())
		wrapped.AddAudience(entity.AudGame, g.GameID())
		g.SendChange(wrapped)
		g.History().Events = append(g.History().Events, sge.Event)
	}

	if !g.WinnerWasSet() {
		// Compute the winner. The winner is already set if someone timed out
		// or resigned, so we only do this if we need to calculate the winner.
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

	log.Debug().Int("p0penalty", p0penalty).Int("p1penalty", p1penalty).Msg("time-penalties")

	// One more thing -- if the Macondo game doesn't know the game is over, which
	// can happen if the game didn't end normally (for example, a timeout or a resign)
	// Then we need to set the final scores here.
	if len(g.History().FinalScores) == 0 || len(evts) > 0 {
		g.AddFinalScoresToHistory()
	}
	g.History().PlayState = macondopb.PlayState_GAME_OVER

	// We need to edit the history's winner to match the reality of the situation.
	// The history's winner is set in macondo based on just the score of the game
	// However we are possibly editing it above.
	g.History().Winner = int32(g.WinnerIdx)

	// Send a gameEndedEvent, which rates the game.
	evt := gameEndedEvent(ctx, g, userStore)
	wrapped := entity.WrapEvent(evt,
		pb.MessageType_GAME_ENDED_EVENT, g.GameID())
	// Once the game ends, we do not need to "sanitize" the packets
	// going to the users anymore. So just send the data to the right
	// audiences.
	wrapped.AddAudience(entity.AudGame, g.GameID())
	wrapped.AddAudience(entity.AudGameTV, g.GameID())
	g.SendChange(wrapped)

	// Compute stats for the player and for the game.
	variantKey, err := g.RatingKey()
	if err != nil {
		log.Err(err).Msg("getting variant key")
	} else {
		gameStats, err := computeGameStats(ctx, g.History(), g.GameReq, variantKey, evt, userStore)
		if err != nil {
			log.Err(err).Msg("computing stats")
		} else {
			g.Stats = gameStats
		}
	}
	// And finally, send a notification to the lobby that this
	// game ended. This will remove it from the list of live games.
	wrapped = entity.WrapEvent(&pb.GameDeletion{Id: g.GameID()},
		pb.MessageType_GAME_DELETION, "")
	wrapped.AddAudience(entity.AudLobby, "gameEnded")
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
}

func computeGameStats(ctx context.Context, history *macondopb.GameHistory, req *pb.GameRequest,
	variantKey entity.VariantKey, evt *pb.GameEndedEvent, userStore user.Store) (*entity.Stats, error) {
	// stats := stats.InstantiateNewStats(1, 2)
	p0id, p1id := history.Players[0].UserId, history.Players[1].UserId
	if history.SecondWentFirst {
		p0id, p1id = p1id, p0id
		history.Players[0], history.Players[1] = history.Players[1], history.Players[0]
		history.FinalScores[0], history.FinalScores[1] = history.FinalScores[1], history.FinalScores[0]
		if history.Winner != -1 {
			history.Winner = 1 - history.Winner
		}
	}

	// Fetch the Macondo config
	config := ctx.Value(ConfigCtxKey("config")).(*macondoconfig.Config)

	// Here, p0 went first and p1 went second, no matter what.
	gameStats := stats.InstantiateNewStats(p0id, p1id)

	err := stats.AddGame(gameStats, history, req, config, evt, history.Uid)
	if err != nil {
		return nil, err
	}

	if history.SecondWentFirst {
		// Flip it back
		history.Players[0], history.Players[1] = history.Players[1], history.Players[0]
		history.FinalScores[0], history.FinalScores[1] = history.FinalScores[1], history.FinalScores[0]
		if history.Winner != -1 {
			history.Winner = 1 - history.Winner
		}
	}

	p0NewProfileStats := stats.InstantiateNewStats(p0id, "")
	p1NewProfileStats := stats.InstantiateNewStats(p1id, "")

	p0ProfileStats, err := statsForUser(ctx, p0id, userStore, variantKey)
	if err != nil {
		return nil, err
	}

	p1ProfileStats, err := statsForUser(ctx, p1id, userStore, variantKey)
	if err != nil {
		return nil, err
	}

	err = stats.AddStats(p0NewProfileStats, p0ProfileStats)
	if err != nil {
		return nil, err
	}
	err = stats.AddStats(p1NewProfileStats, p1ProfileStats)
	if err != nil {
		return nil, err
	}
	err = stats.AddStats(p0NewProfileStats, gameStats)
	if err != nil {
		return nil, err
	}
	err = stats.AddStats(p1NewProfileStats, gameStats)
	if err != nil {
		return nil, err
	}
	stats.Finalize(p0NewProfileStats, []string{}, p0id, p1id)
	stats.Finalize(p1NewProfileStats, []string{}, p1id, p0id)
	// Save all stats back to the database.
	err = userStore.SetStats(ctx, p0id, variantKey, p0NewProfileStats)
	if err != nil {
		return nil, err
	}
	err = userStore.SetStats(ctx, p1id, variantKey, p1NewProfileStats)
	if err != nil {
		return nil, err
	}
	return gameStats, nil
}

func statsForUser(ctx context.Context, id string, userStore user.Store,
	variantKey entity.VariantKey) (*entity.Stats, error) {

	u, err := userStore.GetByUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	userStats, ok := u.Profile.Stats.Data[variantKey]
	if !ok {
		log.Debug().Str("variantKey", string(variantKey)).Str("pid", id).Msg("instantiating new; no data for variant")
		// The second user ID does not matter; this is the per user stat.
		userStats = stats.InstantiateNewStats(id, "")
	}

	return userStats, nil
}
