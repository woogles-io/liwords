package gameplay

import (
	"context"
	"math"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
)

func checkGameOverAndModifyScores(ctx context.Context, entGame *entity.Game,
	userStore user.Store, listStatStore stats.ListStatStore) {
	// Figure out why the game ended. Here there are only two options,
	// standard or six-zero. The game ending on a timeout is handled in
	// another branch (see setTimedOut above) and resignation/etc will
	// also be handled elsewhere.

	if entGame.RackLettersFor(0) == "" || entGame.RackLettersFor(1) == "" {
		entGame.SetGameEndReason(pb.GameEndReason_STANDARD)
	} else {
		entGame.SetGameEndReason(pb.GameEndReason_CONSECUTIVE_ZEROES)
	}
	performEndgameDuties(ctx, entGame, userStore, listStatStore)
}

func performEndgameDuties(ctx context.Context, g *entity.Game,
	userStore user.Store, listStatStore stats.ListStatStore) {
	evts := []*pb.ServerGameplayEvent{}

	var p0penalty, p1penalty int
	if g.CachedTimeRemaining(0) < 0 {
		p0penalty = 10 * int(math.Ceil(float64(-g.CachedTimeRemaining(0))/60000.0))
	}
	if g.CachedTimeRemaining(1) < 0 {
		p1penalty = 10 * int(math.Ceil(float64(-g.CachedTimeRemaining(1))/60000.0))
	}
	// Limit time penalties to the max OT. This is so that we don't get situations
	// where a game was adjudicated days after the fact and a user ends up with
	// a score of -37500
	if p0penalty > 50 {
		p0penalty = 50
	}
	if p1penalty > 50 {
		p1penalty = 50
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
		gameStats, err := computeGameStats(ctx, g.History(), g.GameReq, variantKey,
			evt, userStore, listStatStore)
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

func computeGameStats(ctx context.Context, history *macondopb.GameHistory, req *pb.GameRequest,
	variantKey entity.VariantKey, evt *pb.GameEndedEvent, userStore user.Store,
	listStatStore stats.ListStatStore) (*entity.Stats, error) {

	// stats := entity.InstantiateNewStats(1, 2)
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

	err := stats.AddGame(gameStats, listStatStore, history, req, config, evt, history.Uid)
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

func setTimedOut(ctx context.Context, entGame *entity.Game, pidx int, gameStore GameStore,
	userStore user.Store, listStatStore stats.ListStatStore) error {
	log.Debug().Interface("playing", entGame.Game.Playing()).Msg("timed out!")
	entGame.Game.SetPlaying(macondopb.PlayState_GAME_OVER)

	// And send a game end event.
	entGame.SetGameEndReason(pb.GameEndReason_TIME)
	entGame.SetWinnerIdx(1 - pidx)
	entGame.SetLoserIdx(pidx)
	performEndgameDuties(ctx, entGame, userStore, listStatStore)

	// Store the game back into the store
	err := gameStore.Set(ctx, entGame)
	if err != nil {
		return err
	}
	// Unload the game

	gameStore.Unload(ctx, entGame.GameID())

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
