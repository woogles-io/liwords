package gameplay

import (
	"context"
	"math"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

func performEndgameDuties(ctx context.Context, g *entity.Game, gameStore GameStore,
	userStore user.Store, notorietyStore mod.NotorietyStore, listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore) error {
	log := zerolog.Ctx(ctx)

	log.Debug().Interface("game-end-reason", g.GameEndReason).Msg("checking-game-over")
	// The game is over already. Set an end game reason if there hasn't been
	// one set.
	if g.GameEndReason == pb.GameEndReason_NONE {
		if g.RackLettersFor(0) == "" || g.RackLettersFor(1) == "" {
			g.SetGameEndReason(pb.GameEndReason_STANDARD)
		} else {
			g.SetGameEndReason(pb.GameEndReason_CONSECUTIVE_ZEROES)
		}
	}
	g.Game.SetPlaying(macondopb.PlayState_GAME_OVER)

	// evts := []*pb.ServerGameplayEvent{}

	var p0penalty, p1penalty int
	// Limit time penalties to the max OT. This is so that we don't get situations
	// where a game was adjudicated days after the fact and a user ends up with
	// a score of -37500
	maxOvertimeMs := g.Timers.MaxOvertime * 60000
	p0overtimeMs := -g.CachedTimeRemaining(0)
	if p0overtimeMs > maxOvertimeMs {
		p0overtimeMs = maxOvertimeMs
	}
	p1overtimeMs := -g.CachedTimeRemaining(1)
	if p1overtimeMs > maxOvertimeMs {
		p1overtimeMs = maxOvertimeMs
	}
	if p0overtimeMs > 0 {
		p0penalty = 10 * int(math.Ceil(float64(p0overtimeMs)/60000.0))
	}
	if p1overtimeMs > 0 {
		p1penalty = 10 * int(math.Ceil(float64(p1overtimeMs)/60000.0))
	}
	penaltyApplied := false

	if p0penalty > 0 {
		penaltyApplied = true
		newscore := g.PointsFor(0) - p0penalty
		// >Pakorn: ISBALI (time) -10 409
		g.History().Events = append(g.History().Events, &macondopb.GameEvent{
			Nickname:        g.History().Players[0].Nickname,
			Rack:            g.RackLettersFor(0),
			Type:            macondopb.GameEvent_TIME_PENALTY,
			LostScore:       int32(p0penalty),
			Cumulative:      int32(newscore),
			MillisRemaining: int32(g.CachedTimeRemaining(0)),
		})
		g.SetPointsFor(0, newscore)
	}
	if p1penalty > 0 {
		penaltyApplied = true
		newscore := g.PointsFor(1) - p1penalty
		g.History().Events = append(g.History().Events, &macondopb.GameEvent{
			Nickname:        g.History().Players[1].Nickname,
			Rack:            g.RackLettersFor(1),
			Type:            macondopb.GameEvent_TIME_PENALTY,
			LostScore:       int32(p1penalty),
			Cumulative:      int32(newscore),
			MillisRemaining: int32(g.CachedTimeRemaining(1)),
		})
		g.SetPointsFor(1, newscore)
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
	if len(g.History().FinalScores) == 0 || penaltyApplied {
		g.AddFinalScoresToHistory()
	}

	g.History().PlayState = macondopb.PlayState_GAME_OVER

	// We need to edit the history's winner to match the reality of the situation.
	// The history's winner is set in macondo based on just the score of the game
	// However we are possibly editing it above.
	g.History().Winner = int32(g.WinnerIdx)

	// Set the game quickdata

	g.Quickdata.FinalScores = g.History().FinalScores

	// Grab a lock on both users to ensure that the following
	// ratings calculations are not interleaved across threads.
	// Enforce an arbitrary locking order to ensure no deadlocks
	// occur between threads.

	u0, err := userStore.GetByUUID(ctx, g.History().Players[0].UserId)
	if err != nil {
		return err
	}
	u1, err := userStore.GetByUUID(ctx, g.History().Players[1].UserId)
	if err != nil {
		return err
	}
	users := []*entity.User{u0, u1}

	firstLockingIndex := 0
	if users[0].UUID > users[1].UUID {
		firstLockingIndex = 1
	}

	users[firstLockingIndex].Lock()
	defer users[firstLockingIndex].Unlock()
	users[1-firstLockingIndex].Lock()
	defer users[1-firstLockingIndex].Unlock()

	// Send a gameEndedEvent, which rates the game.
	evt := gameEndedEvent(ctx, g, userStore)
	wrapped := entity.WrapEvent(evt, pb.MessageType_GAME_ENDED_EVENT)
	for _, p := range players(g) {
		// why not AudGame?
		wrapped.AddAudience(entity.AudUser, p+".game."+g.GameID())
	}
	wrapped.AddAudience(entity.AudGameTV, g.GameID())

	// Compute stats for the player and for the game.
	variantKey, err := g.RatingKey()
	if err != nil {
		log.Err(err).Msg("getting variant key")
	} else {
		gameStats, err := ComputeGameStats(ctx, g.History(), g.GameReq, variantKey,
			evt, userStore, listStatStore)
		if err != nil {
			log.Err(err).Msg("computing stats")
		} else {
			g.Stats = gameStats
		}
	}

	// Send the event here instead of above the computation of Game Stats
	// to avoid race conditions (ComputeGameStats modifies the history).
	g.SendChange(wrapped)

	// Send a notification to the lobby that this
	// game ended. This will remove it from the list of live games.
	wrapped = entity.WrapEvent(&pb.GameDeletion{Id: g.GameID()},
		pb.MessageType_GAME_DELETION)
	wrapped.AddAudience(entity.AudLobby, "gameEnded")
	g.SendChange(wrapped)

	if g.TournamentData != nil && g.TournamentData.Id != "" {
		err := tournament.HandleTournamentGameEnded(ctx, tournamentStore, userStore, g)
		if err != nil {
			log.Err(err).Msg("error-tourney-game-ended")
		}
	} else if g.GameReq.RatingMode == pb.RatingMode_RATED {
		// Applies penalties to players who have misbehaved during the game
		// Does not apply for tournament games
		err = mod.Automod(ctx, userStore, notorietyStore, u0, u1, g)
		if err != nil {
			log.Err(err).Msg("automod-error")
		}
	}

	// Save and unload the game from the cache.

	err = gameStore.Set(ctx, g)
	if err != nil {
		return err
	}

	log.Info().Str("gameID", g.GameID()).Msg("game-ended-unload-cache")
	gameStore.Unload(ctx, g.GameID())
	g.SendChange(g.NewActiveGameEntry(false))

	// send each player their new profile with updated ratings.
	sendProfileUpdate(ctx, g, users)
	return nil
}

func sendProfileUpdate(ctx context.Context, g *entity.Game, users []*entity.User) {
	for _, u := range users {
		ratingProto, err := u.GetProtoRatings()
		if err != nil {
			continue
		}
		wrapped := entity.WrapEvent(&pb.ProfileUpdate{UserId: u.UUID, Ratings: ratingProto},
			pb.MessageType_PROFILE_UPDATE_EVENT)
		wrapped.AddAudience(entity.AudUser, u.UUID)
		g.SendChange(wrapped)
	}
}

// Dangerous function that should not have existed.
func flipPlayersInHistoryIfNecessary(history *macondopb.GameHistory) {
	if history.SecondWentFirst {
		history.Players[0], history.Players[1] = history.Players[1], history.Players[0]
		history.FinalScores[0], history.FinalScores[1] = history.FinalScores[1], history.FinalScores[0]
		if history.Winner != -1 {
			history.Winner = 1 - history.Winner
		}
	}
}

func ComputeGameStats(ctx context.Context, history *macondopb.GameHistory, req *pb.GameRequest,
	variantKey entity.VariantKey, evt *pb.GameEndedEvent, userStore user.Store,
	listStatStore stats.ListStatStore) (*entity.Stats, error) {

	// stats := entity.InstantiateNewStats(1, 2)
	flipPlayersInHistoryIfNecessary(history)
	defer flipPlayersInHistoryIfNecessary(history)

	// Fetch the Macondo config
	macondoConfig, err := config.GetMacondoConfig(ctx)
	if err != nil {
		return nil, err
	}
	// Here, p0 went first and p1 went second, no matter what.
	p0id, p1id := history.Players[0].UserId, history.Players[1].UserId
	gameStats := stats.InstantiateNewStats(p0id, p1id)

	err = stats.AddGame(gameStats, listStatStore, history, req, macondoConfig, evt, history.Uid)
	if err != nil {
		return nil, err
	}

	// Only add the game to profile stats if the game was rated
	// and was not triple challenge.
	if req.RatingMode == pb.RatingMode_RATED &&
		history.ChallengeRule != macondopb.ChallengeRule_TRIPLE {
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

		// Save all stats back to the database.
		err = userStore.SetStats(ctx, p0id, p1id, variantKey, p0NewProfileStats, p1NewProfileStats)
		if err != nil {
			return nil, err
		}
	}

	return gameStats, nil
}

func setTimedOut(ctx context.Context, entGame *entity.Game, pidx int, gameStore GameStore,
	userStore user.Store, notorietyStore mod.NotorietyStore, listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore) error {
	log.Debug().Interface("playing", entGame.Game.Playing()).Msg("timed out!")

	// The losing player always overtimes by the maximum amount.
	// Not less, even if no moves in the final minute.
	// Not more, even if game is abandoned and resumed/adjudicated much later.
	entGame.Timers.TimeRemaining[pidx] = entGame.Timers.MaxOvertime * -60000

	// And send a game end event.
	entGame.SetGameEndReason(pb.GameEndReason_TIME)
	entGame.SetWinnerIdx(1 - pidx)
	entGame.SetLoserIdx(pidx)
	return performEndgameDuties(ctx, entGame, gameStore, userStore, notorietyStore, listStatStore, tournamentStore)
}

func redoCancelledGamePairings(ctx context.Context, tstore tournament.TournamentStore,
	g *entity.Game) error {

	tid := g.TournamentData.Id
	t, err := tstore.Get(ctx, tid)
	if err != nil {
		return err
	}

	if t.Type == entity.TypeClub || t.Type == entity.TypeLegacy {
		log.Info().Str("tid", tid).Msg("no pairings to redo for club or legacy type")
		return nil
	}
	div := g.TournamentData.Division
	round := g.TournamentData.Round
	gidx := g.TournamentData.GameIndex

	// this should match entity.User.TournamentID()
	userID := g.Quickdata.PlayerInfo[0].UserId + ":" + g.Quickdata.PlayerInfo[0].Nickname

	return tournament.ClearReadyStates(ctx, tstore, t, div, userID, round, gidx)

}

// AbortGame aborts a game. This should be done for games that never started,
// or games that were aborted by mutual consent.
// It will send events to the correct places, and takes in a locked game.
func AbortGame(ctx context.Context, gameStore GameStore, tournamentStore tournament.TournamentStore,
	g *entity.Game, gameEndReason pb.GameEndReason) error {

	g.SetGameEndReason(gameEndReason)
	g.History().PlayState = macondopb.PlayState_GAME_OVER
	g.Game.SetPlaying(macondopb.PlayState_GAME_OVER)

	// save the game back into the store
	err := gameStore.Set(ctx, g)
	if err != nil {
		return err
	}
	// Unload the game
	log.Info().Str("gameID", g.GameID()).Msg("game-aborted-unload-cache")
	gameStore.Unload(ctx, g.GameID())

	// We use this instead of the game's event channel directly because there's
	// a possibility that a game that never got started never got its channel
	// registered.
	evtChan := gameStore.GameEventChan()

	wrapped := entity.WrapEvent(&pb.GameDeletion{Id: g.GameID()},
		pb.MessageType_GAME_DELETION)
	wrapped.AddAudience(entity.AudLobby, "gameEnded")
	// send event to the tournament channel if it is a tournament game
	if g.TournamentData != nil && g.TournamentData.Id != "" {
		wrapped.AddAudience(entity.AudTournament, g.TournamentData.Id)
	}
	evtChan <- wrapped

	evt := &pb.GameEndedEvent{
		EndReason: gameEndReason,
		Time:      g.Timers.TimeOfLastUpdate,
		History:   g.History(),
	}

	wrapped = entity.WrapEvent(evt, pb.MessageType_GAME_ENDED_EVENT)
	for _, p := range players(g) {
		// why not AudGame?
		wrapped.AddAudience(entity.AudUser, p+".game."+g.GameID())
	}
	wrapped.AddAudience(entity.AudGameTV, g.GameID())
	evtChan <- wrapped
	evtChan <- g.NewActiveGameEntry(false)

	// If this game is part of a tournament that is not in clubhouse
	// mode, we must allow the players to try to play again.
	if g.TournamentData != nil && g.TournamentData.Id != "" {
		err = redoCancelledGamePairings(ctx, tournamentStore, g)
		log.Err(err).Msg("redo-cancelled-game-pairings")
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

	ratings := map[string][2]int32{}
	deltas := map[string]int32{}
	newRatings := map[string]int32{}
	var err error
	var now = time.Now().Unix()
	if g.CreationRequest().RatingMode == pb.RatingMode_RATED {
		ratings, err = Rate(ctx, scores, g, winner, userStore, now)
		if err != nil {
			log.Err(err).Msg("rating-error")
		}
	}
	for u, rat := range ratings {
		newRatings[u] = rat[1]
		deltas[u] = rat[1] - rat[0]
	}

	evt := &pb.GameEndedEvent{
		Scores:       scores,
		NewRatings:   newRatings,
		EndReason:    g.GameEndReason,
		Winner:       winner,
		Loser:        loser,
		Tie:          tie,
		Time:         g.Timers.TimeOfLastUpdate,
		RatingDeltas: deltas,
		History:      g.History(),
	}

	log.Debug().Interface("game-ended-event", evt).Msg("game-ended")
	return evt
}
