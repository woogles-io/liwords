// Package gameplay should know nothing about protocols or databases.
// It is mostly a pass-through interface to a Macondo game,
// but also implements a timer and other related logic.
// This is a use-case in the clean architecture hierarchy.
package gameplay

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/domino14/macondo/runner"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/alphabet"
	"github.com/domino14/macondo/board"
	"github.com/domino14/macondo/cross_set"
	"github.com/domino14/macondo/gaddag"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/macondo/move"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stats"
	gstore "github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/user"
	gs "github.com/domino14/liwords/rpc/api/proto/game_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

var (
	errGameNotActive   = errors.New("game is not currently active")
	errGameSyncError   = errors.New("game synchronization error")
	errNotOnTurn       = errors.New("player not on turn")
	errTimeDidntRunOut = errors.New("got time ran out, but it did not actually")
)

// GameStore is an interface for getting a full game.
type GameStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
	GetFromBacking(ctx context.Context, id string) (*entity.Game, error)
	GetMetadata(ctx context.Context, id string) (*gs.GameInfoResponse, error)
	GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error)
	GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*gs.GameInfoResponses, error)
	GetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*gs.GameInfoResponses, error)
	Set(context.Context, *entity.Game) error
	Create(context.Context, *entity.Game) error
	Exists(ctx context.Context, id string) (bool, error)
	ListActive(context.Context, string, bool) (*gs.GameInfoResponses, error)
	Count(ctx context.Context) (int64, error)
	CachedCount(ctx context.Context) int
	SetGameEventChan(c chan<- *entity.EventWrapper)
	Unload(context.Context, string)
	SetReady(ctx context.Context, gid string, pidx int) (int, error)
	SetGameEndReason(ctx context.Context, g *entity.Game) error
	GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error)
}

type ConfigCtxKey string

// InstantiateNewGame instantiates a game and returns it.
func InstantiateNewGame(ctx context.Context, gameStore GameStore, cfg *config.Config,
	users [2]*entity.User, assignedFirst int, req *pb.GameRequest, tdata *entity.TournamentData) (*entity.Game, error) {

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

	dist, err := alphabet.Get(&cfg.MacondoConfig, req.Rules.LetterDistributionName)
	if err != nil {
		return nil, err
	}

	dawg, err := gaddag.GetDawg(&cfg.MacondoConfig, req.Lexicon)
	if err != nil {
		return nil, err
	}

	rules := game.NewGameRules(
		&cfg.MacondoConfig, dist, board.MakeBoard(bd),
		&gaddag.Lexicon{GenericDawg: dawg},
		cross_set.CrossScoreOnlyGenerator{Dist: dist})

	var gameRunner *runner.GameRunner
	for {
		gameRunner, err = runner.NewGameRunnerFromRules(&runner.GameOptions{
			FirstIsAssigned: firstAssigned,
			GoesFirst:       assignedFirst,
			ChallengeRule:   req.ChallengeRule,
		}, players, rules)
		if err != nil {
			return nil, err
		}

		exists, err := gameStore.Exists(ctx, gameRunner.Game.Uid())
		if err != nil {
			return nil, err
		}
		if exists {
			continue
			// This UUID exists in the database. This is only possible because
			// we are purposely shortening the UUID in macondo for nicer URLs.
			// 57^8 should still give us 111 trillion games. (and we can add more
			// characters if we get close to that number)
		}
		break
		// There's still a chance of a race condition here if another thread
		// creates the same game ID at the same time, but the chances
		// of that are so astronomically unlikely we won't bother.
	}

	entGame := entity.NewGame(&gameRunner.Game, req)
	entGame.PlayerDBIDs = dbids
	entGame.TournamentData = tdata

	ratingKey, err := entGame.RatingKey()
	if err != nil {
		return nil, err
	}

	// Create player info in entGame.Quickdata
	playerinfos := make([]*gs.PlayerInfo, len(players))

	for idx, u := range users {
		playerinfos[idx] = &gs.PlayerInfo{
			Nickname: u.Username,
			UserId:   u.UUID,
			Rating:   u.GetRelevantRating(ratingKey),
			IsBot:    u.IsBot,
			First:    gameRunner.FirstPlayer().UserId == u.UUID,
		}
		if u.Profile != nil {
			playerinfos[idx].FullName = u.RealName()
			playerinfos[idx].CountryCode = u.Profile.CountryCode
			playerinfos[idx].Title = u.Profile.Title
			// There is no avatar URL yet.
			// playerinfos[idx].AvatarUrl = u.Profile.AvatarUrl
		}
	}

	// Create the Quickdata now with the original player info.
	entGame.Quickdata = &entity.Quickdata{
		OriginalRequestId: req.OriginalRequestId,
		PlayerInfo:        playerinfos,
	}
	// This timestamp will be very close to whatever gets saved in the DB
	// as the CreatedAt date. We need to put it here though in order to
	// keep the cached version in sync with the saved version at the beginning.
	entGame.CreatedAt = time.Now()

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

	case pb.ClientGameplayEvent_CHALLENGE_PLAY:
		m := move.NewChallengeMove(rack.TilesOn(), g.Alphabet())
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
	log.Debug().Msg("going-to-save")
	// Save the game back to the store always.
	if err := gameStore.Set(ctx, entGame); err != nil {
		log.Err(err).Msg("error-saving")
		return err
	}
	log.Debug().Msg("saved-game-to-store")
	if err := entGame.RegisterChangeHook(eventChan); err != nil {
		return err
	}
	log.Debug().Interface("history", entGame.Game.History()).Msg("game history")

	evt := entGame.HistoryRefresherEvent()
	wrapped := entity.WrapEvent(evt, pb.MessageType_GAME_HISTORY_REFRESHER)
	wrapped.AddAudience(entity.AudGameTV, entGame.GameID())
	for _, p := range players(entGame) {
		wrapped.AddAudience(entity.AudUser, p+".game."+id)
	}
	entGame.SendChange(wrapped)

	// If the previous game was a rematch, notify
	// the viewers that this game has started.
	rematchStreak, err := gameStore.GetRematchStreak(ctx, entGame.Quickdata.OriginalRequestId)
	if err != nil {
		return err
	}
	if len(rematchStreak.Streak) > 0 {
		previousGameID := rematchStreak.Streak[0].GameId
		evt := &pb.RematchStartedEvent{RematchGameId: entGame.GameID()}
		wrappedRematch := entity.WrapEvent(evt, pb.MessageType_REMATCH_STARTED)
		wrappedRematch.AddAudience(entity.AudGameTV, previousGameID)
		entGame.SendChange(wrappedRematch)
	}

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
	userStore user.Store, listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore,
	timeRemaining int, challengerID string) error {
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
	evt := entity.WrapEvent(resultEvent, pb.MessageType_SERVER_CHALLENGE_RESULT_EVENT)
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
		// Handle this special case. If an unsuccessful challenge turn loss
		// was added, we are not handling time increment properly.
		// This is because macondo adds this move automatically without
		// knowing anything about timers. So we must edit the event a little bit.
		if sge.Event.Type == macondopb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS {
			// This is a bit of a hack. Temporarily set the onTurn to the
			// player who made the challenge so we can record their time
			// of move.
			onTurn := entGame.PlayerOnTurn()
			entGame.SetPlayerOnTurn(1 - onTurn)
			entGame.RecordTimeOfMove(1 - onTurn)
			// Then set the player on turn back.
			entGame.SetPlayerOnTurn(onTurn)
			// Set the time remaining to the actual time remaining that was
			// calculated by RecordTimeOfMove for the relevant player.
			sge.TimeRemaining = int32(entGame.Timers.TimeRemaining[1-onTurn])
		}

		evt = entity.WrapEvent(sge, pb.MessageType_SERVER_GAMEPLAY_EVENT)
		evt.AddAudience(entity.AudGameTV, entGame.GameID())
		for _, uid := range players(entGame) {
			evt.AddAudience(entity.AudUser, uid+".game."+entGame.GameID())
		}
		entGame.SendChange(evt)
	}

	if entGame.Game.Playing() == macondopb.PlayState_GAME_OVER {
		if entGame.ChallengeRule() == macondopb.ChallengeRule_TRIPLE {
			entGame.SetGameEndReason(pb.GameEndReason_TRIPLE_CHALLENGE)
			// Player may have accrued overtime penalties before challenging.
			onTurn := entGame.PlayerOnTurn()
			entGame.RecordTimeOfMove(onTurn)
			winner := int(entGame.History().Winner)
			entGame.SetWinnerIdx(winner)
			entGame.SetLoserIdx(1 - winner)
		}
		err = performEndgameDuties(ctx, entGame, gameStore, userStore, listStatStore, tournamentStore)
		if err != nil {
			return err
		}
	}

	return nil
}

func PlayMove(ctx context.Context,
	entGame *entity.Game,
	gameStore GameStore,
	userStore user.Store,
	listStatStore stats.ListStatStore,
	tournamentStore tournament.TournamentStore,
	userID string, onTurn,
	timeRemaining int,
	m *move.Move) error {

	log.Debug().Msg("validating")

	_, err := entGame.Game.ValidateMove(m)
	if err != nil {
		return err
	}

	if m.Action() == move.MoveTypeChallenge {
		// Handle in another way
		return handleChallenge(ctx, entGame, gameStore, userStore, listStatStore, tournamentStore, timeRemaining, userID)
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
		sge.GameId = entGame.GameID()
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
		wrapped := entity.WrapEvent(sge, pb.MessageType_SERVER_GAMEPLAY_EVENT)
		wrapped.AddAudience(entity.AudGameTV, entGame.GameID())
		for _, p := range players {
			wrapped.AddAudience(entity.AudUser, p+".game."+entGame.GameID())
		}
		entGame.SendChange(wrapped)
	}
	if playing == macondopb.PlayState_GAME_OVER {
		err = performEndgameDuties(ctx, entGame, gameStore, userStore, listStatStore, tournamentStore)
		if err != nil {
			return err
		}
	}
	return nil
}

// HandleEvent handles a gameplay event from the socket
func HandleEvent(ctx context.Context, gameStore GameStore, userStore user.Store,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore,
	userID string, cge *pb.ClientGameplayEvent) (*entity.Game, error) {

	entGame, err := gameStore.Get(ctx, cge.GameId)
	if err != nil {
		return nil, err
	}

	return handleEventWithGame(ctx, gameStore, userStore, listStatStore,
		tournamentStore, userID, cge, entGame)
}

func handleEventWithGame(ctx context.Context, gameStore GameStore, userStore user.Store,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore, userID string, cge *pb.ClientGameplayEvent,
	entGame *entity.Game) (*entity.Game, error) {

	if entGame.Game.Playing() == macondopb.PlayState_GAME_OVER {
		return entGame, errGameNotActive
	}
	onTurn := entGame.Game.PlayerOnTurn()

	// Ensure that it is actually the correct player's turn
	if cge.Type != pb.ClientGameplayEvent_RESIGN && entGame.Game.PlayerIDOnTurn() != userID {
		log.Info().Interface("client-event", cge).Msg("not on turn")
		return entGame, errNotOnTurn
	}
	timeRemaining := entGame.TimeRemaining(onTurn)
	log.Debug().Int("time-remaining", timeRemaining).Msg("checking-time-remaining")
	// Check that we didn't run out of time.
	// Allow auto-passing.
	if !(entGame.Game.Playing() == macondopb.PlayState_WAITING_FOR_FINAL_PASS &&
		cge.Type == pb.ClientGameplayEvent_PASS) &&
		entGame.TimeRanOut(onTurn) {
		// Game is over!
		log.Debug().Msg("got-move-too-late")

		// If an ending game gets "challenge" just before "timed out",
		// ignore the challenge, pass instead.
		if entGame.Game.Playing() == macondopb.PlayState_WAITING_FOR_FINAL_PASS {
			log.Debug().Msg("timed out, so passing instead of processing the submitted move")
			cge = &pb.ClientGameplayEvent{
				Type:   pb.ClientGameplayEvent_PASS,
				GameId: cge.GameId,
			}
		} else {
			// Basically skip to the bottom and exit.
			return entGame, setTimedOut(ctx, entGame, onTurn, gameStore, userStore, listStatStore, tournamentStore)
		}
	}

	log.Debug().Msg("going to turn into a macondo gameevent")

	// Turn the event into a macondo GameEvent.
	if cge.Type == pb.ClientGameplayEvent_RESIGN {
		entGame.SetGameEndReason(pb.GameEndReason_RESIGNED)
		// Player may have accrued overtime penalties before resigning.
		entGame.RecordTimeOfMove(onTurn)
		winner := 1 - onTurn
		// If opponent is the one who resigned, current player wins.
		if entGame.Game.PlayerIDOnTurn() != userID {
			winner = onTurn
		}
		entGame.History().Winner = int32(winner)
		entGame.SetWinnerIdx(winner)
		entGame.SetLoserIdx(1 - winner)
		err := performEndgameDuties(ctx, entGame, gameStore, userStore, listStatStore, tournamentStore)
		if err != nil {
			return entGame, err
		}
	} else {
		m, err := clientEventToMove(cge, &entGame.Game)
		if err != nil {
			return entGame, err
		}

		err = PlayMove(ctx, entGame, gameStore, userStore, listStatStore, tournamentStore, userID, onTurn, timeRemaining, m)
		if err != nil {
			return entGame, err
		}
	}
	// If the game hasn't ended yet, save it to the store. If it HAS ended,
	// it was already saved to the store above in performEndgameDuties
	if entGame.GameEndReason == pb.GameEndReason_NONE {
		if err := gameStore.Set(ctx, entGame); err != nil {
			log.Err(err).Msg("error-saving")
			return entGame, err
		}
	}
	return entGame, nil
}

// TimedOut gets called when the client thinks the user's time ran out. We
// verify that that is actually the case.
func TimedOut(ctx context.Context, gameStore GameStore, userStore user.Store,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore, timedout string, gameID string) error {
	// Note: we can get this event multiple times; the opponent and the player on turn
	// both send it, as well as the adjudicator.
	log.Debug().Str("timedout", timedout).Msg("got-timed-out")
	// Bust the cache for this signal.
	entGame, err := gameStore.(*gstore.Cache).GetFromBacking(ctx, gameID)
	if err != nil {
		return err
	}
	// We need some way of locking here so that the game end stuff doesn't get run
	// multiple times across different threads / nodes.
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

	// If opponent played out, auto-pass instead of forfeiting.
	if entGame.Game.Playing() == macondopb.PlayState_WAITING_FOR_FINAL_PASS {
		log.Debug().Msg("timed out, so auto-passing instead of forfeiting")
		_, err = handleEventWithGame(ctx, gameStore, userStore, listStatStore, tournamentStore,
			entGame.Game.PlayerIDOnTurn(), &pb.ClientGameplayEvent{
				Type:   pb.ClientGameplayEvent_PASS,
				GameId: gameID,
			}, entGame)
		return err
	}

	return setTimedOut(ctx, entGame, onTurn, gameStore, userStore, listStatStore, tournamentStore)
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
