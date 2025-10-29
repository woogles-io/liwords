package bus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"
	"lukechampine.com/frand"

	"github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gen/api/proto/macondo"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/tournament"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var (
	errGamesDisabled = errors.New("new games are temporarily disabled while we update Woogles; please try again in a few minutes")
)

const (
	TournamentReadyExpire = 3600
)

func (b *Bus) errIfGamesDisabled(ctx context.Context) error {
	enabled, err := b.stores.ConfigStore.GamesEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return errGamesDisabled
	}
	return nil
}

func (b *Bus) instantiateAndStartGame(ctx context.Context, accUser *entity.User, requester string,
	gameReq *pb.GameRequest, sg *entity.SoughtGame, reqID, acceptingConnID string) error {

	reqUser, err := b.stores.UserStore.GetByUUID(ctx, requester)
	if err != nil {
		return err
	}

	err = b.errIfGamesDisabled(ctx)
	if err != nil {
		return err
	}

	// disallow anon game acceptance for now.
	if accUser.Anonymous || reqUser.Anonymous {
		return errors.New("you must log in to play games")
	}

	if (accUser.Anonymous || reqUser.Anonymous) && gameReq.RatingMode == pb.RatingMode_RATED {
		return errors.New("anonymous-players-cant-play-rated")
	}

	log.Debug().Interface("req", sg).
		Str("accepting-conn", acceptingConnID).Msg("game-request-accepted")
	assignedFirst := -1
	var tournamentID string
	if sg.SeekRequest.ReceiverIsPermanent {
		if sg.SeekRequest.RematchFor != "" {
			// Assign firsts to be the other player.
			gameID := sg.SeekRequest.RematchFor
			gh, err := b.stores.GameStore.GetHistory(ctx, gameID)
			if err != nil {
				return err
			}
			players := gh.Players
			log.Debug().Str("went-first", players[0].Nickname).Msg("determining-first")

			// These are indices in the array passed to InstantiateNewGame
			if accUser.UUID == players[0].UserId {
				assignedFirst = 1 // reqUser should go first
			} else if reqUser.UUID == players[0].UserId {
				assignedFirst = 0 // accUser should go first
			}
		}
		if sg.SeekRequest.TournamentId != "" {
			t, err := b.stores.TournamentStore.Get(ctx, sg.SeekRequest.TournamentId)
			if err != nil {
				return errors.New("tournament not found")
			}
			tournamentID = t.UUID
		}
	}
	// If tournamentID is defined, this is a clubhouse game, so there's no
	// round/division/etc, just a simple "tournament ID"
	trdata := &entity.TournamentData{
		Id: tournamentID,
	}
	if assignedFirst == -1 {
		assignedFirst = frand.Intn(2)
		log.Debug().Int("first", assignedFirst).Msg("assigned-first-randomly")
	}
	var users [2]*entity.User
	if assignedFirst == 0 {
		users = [2]*entity.User{accUser, reqUser}
	} else {
		users = [2]*entity.User{reqUser, accUser}
	}

	g, err := gameplay.InstantiateNewGame(ctx, b.stores.GameStore, b.config,
		users, gameReq, trdata)
	if err != nil {
		return err
	}
	// Broadcast a seek delete event, and send both parties a game redirect.
	if reqID != BotRequestID {
		b.stores.SoughtGameStore.Delete(ctx, reqID)
		err = b.sendSoughtGameDeletion(ctx, sg)
		if err != nil {
			log.Err(err).Msg("broadcasting-sg-deletion")
		}
	}

	err = b.broadcastGameCreation(g, accUser, reqUser)
	if err != nil {
		log.Err(err).Msg("broadcasting-game-creation")
	}

	// Auto-start correspondence games immediately (skip ready flag)
	if gameReq.GameMode == pb.GameMode_CORRESPONDENCE {
		log.Debug().Str("gameID", g.GameID()).Msg("auto-starting-correspondence-game")
		err = gameplay.StartGame(ctx, b.stores, b.gameEventChan, g)
		if err != nil {
			log.Err(err).Msg("auto-starting-correspondence-game")
			return err
		}
	}

	// This event will result in a redirect.
	seekerConnID, err := sg.SeekerConnID()
	if err != nil {
		return err
	}
	ngevt := entity.WrapEvent(&pb.NewGameEvent{
		GameId:       g.GameID(),
		AccepterCid:  acceptingConnID,
		RequesterCid: seekerConnID,
	}, pb.MessageType_NEW_GAME_EVENT)
	// The front end keeps track of which tabs seek/accept games etc
	// so we don't attach any extra channel info here.
	b.pubToUser(accUser.UUID, ngevt, "")
	b.pubToUser(reqUser.UUID, ngevt, "")

	tcname, variant, err := entity.VariantFromGameReq(gameReq)
	if err != nil {
		return err
	}

	log.Info().Str("newgameid", g.History().Uid).
		Str("sender", accUser.UUID).
		Str("requester", requester).
		Str("reqID", reqID).
		Str("lexicon", gameReq.Lexicon).
		Str("timectrl", string(tcname)).
		Str("variant", string(variant)).
		Str("onturn", g.NickOnTurn()).Msg("game-accepted")

	return nil
}

func (b *Bus) goHandleBotMove(ctx context.Context, resp *macondo.BotResponse,
	gid, replyChan string) {

	go func() {
		defer func() {
			// acknowledge the NATS message - empty response is fine.
			err := b.natsconn.Publish(replyChan, []byte{})
			if err != nil {
				log.Err(err).Str("gid", gid).Msg("error-acknowledging")
			}
		}()
		g, err := b.stores.GameStore.Get(ctx, gid)
		if err != nil {
			log.Err(err).Str("gid", gid).Msg("cant-handle-bot-move")
			return
		}
		g.Lock()
		defer g.Unlock()
		// These should both be the bot:
		onTurn := g.Game.PlayerOnTurn()
		userID := g.Game.PlayerIDOnTurn()
		switch r := resp.Response.(type) {
		case *macondo.BotResponse_Move:
			timeRemaining := g.TimeRemaining(onTurn)

			m, err := game.MoveFromEvent(r.Move, g.Alphabet(), g.Board())
			if err != nil {
				log.Err(err).Msg("move-from-event-error")
				return
			}
			err = gameplay.PlayMove(ctx, g, b.stores, userID, onTurn, timeRemaining, m)
			if err != nil {
				log.Err(err).Msg("bot-cant-move-play-error")
				return
			}
		case *macondopb.BotResponse_Error:
			log.Error().Str("error", r.Error).Msg("bot-error")
			return
		}
		// And save the game after playing the move. Note that PlayMove doesn't do
		// this.
		err = b.stores.GameStore.Set(ctx, g)
		if err != nil {
			log.Err(err).Msg("setting-game-after-bot-move")
		}
	}()
}

func (b *Bus) readyForGame(ctx context.Context, evt *pb.ReadyForGame, userID string) error {
	g, err := b.stores.GameStore.Get(ctx, evt.GameId)
	if err != nil {
		return err
	}
	g.Lock()
	defer g.Unlock()

	log.Debug().Str("userID", userID).Interface("playing", g.Playing()).Msg("ready-for-game")
	if g.Playing() == macondopb.PlayState_GAME_OVER {
		return errors.New("game is over")
	}

	var readyID int

	if g.History().Players[0].UserId == userID {
		readyID = 0
	} else if g.History().Players[1].UserId == userID {
		readyID = 1
	} else {
		log.Error().Str("userID", userID).Str("gameID", evt.GameId).Msg("not-in-game")
		return errors.New("ready for game but not in game")
	}

	log.Debug().Str("gameID", evt.GameId).Int("readyID", readyID).Msg("setReady")
	rf, err := b.stores.GameStore.SetReady(ctx, evt.GameId, readyID)
	if err != nil {
		log.Err(err).Msg("in-set-ready")
		return err
	}
	log.Debug().Interface("rf", rf).Msg("ready-flag")
	// Start the game if both players are ready (or if it's a bot game).
	// readyflag will be (01 | 10) = 3 for two players.
	if rf == (1<<len(g.History().Players))-1 || g.GameReq.PlayerVsBot {
		err = gameplay.StartGame(ctx, b.stores, b.gameEventChan, g)
		if err != nil {
			log.Err(err).Msg("starting-game")
		} else {
			// Note: for PlayerVsBot, readyForGame is called twice when player is ready and every time player refreshes, why? :-(
			g.SendChange(g.NewActiveGameEntry(true))
		}

	}
	return nil
}

func (b *Bus) readyForTournamentGame(ctx context.Context, evt *pb.ReadyForTournamentGame, userID, connID string) error {
	if !evt.Unready {
		err := b.errIfGamesDisabled(ctx)
		if err != nil {
			return err
		}
	}

	reqUser, err := b.stores.UserStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}

	fullUserID := reqUser.TournamentID()

	t, err := b.stores.TournamentStore.Get(ctx, evt.TournamentId)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().
		Str("userID", userID).
		Str("username", reqUser.Username).
		Str("fullUserID", fullUserID).
		Str("tournamentID", evt.TournamentId).
		Str("tournamentName", t.Name).
		Str("division", evt.Division).
		Int32("round", evt.Round).
		Int32("gameIndex", evt.GameIndex).
		Bool("unready", evt.Unready).
		Str("connID", connID).
		Msg("user-clicked-ready")

	playerIDs, bothReady, err := tournament.SetReadyForGame(ctx, b.stores.TournamentStore, t, fullUserID, connID,
		evt.Division, int(evt.Round), int(evt.GameIndex), evt.Unready)

	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("userID", userID).
			Str("username", reqUser.Username).
			Str("tournamentID", evt.TournamentId).
			Str("division", evt.Division).
			Int32("round", evt.Round).
			Msg("error-setting-ready")
		return err
	}

	log.Ctx(ctx).Info().
		Str("userID", userID).
		Str("username", reqUser.Username).
		Bool("bothReady", bothReady).
		Interface("playerIDs", playerIDs).
		Msg("ready-state-updated")

	// Let's send the ready message to both players.
	evt.PlayerId = fullUserID

	ngevt := entity.WrapEvent(evt, pb.MessageType_READY_FOR_TOURNAMENT_GAME)
	// We'll publish it to both users (across any connections they might be on)
	// so that the widget updates properly in every context.
	for _, p := range playerIDs {
		s := strings.Split(p, ":")
		if len(s) != 3 {
			return fmt.Errorf("unexpected player readystate: %v", p)
		}
		b.pubToUser(s[0], ngevt, "")
	}

	if !bothReady {
		if !evt.Unready {
			// Store temporary ready state in Redis.
			conn := b.redisPool.Get()
			defer conn.Close()
			bts, err := json.Marshal(evt)
			if err != nil {
				return err
			}
			_, err = conn.Do("SET", "tready:"+connID, bts, "EX", TournamentReadyExpire)
			if err != nil {
				return err
			}
		}
		return nil
	}
	redisConn := b.redisPool.Get()
	defer redisConn.Close()

	// Both players are ready! Instantiate and start a new game.
	log.Ctx(ctx).Info().
		Str("userID", userID).
		Str("username", reqUser.Username).
		Str("tournamentID", evt.TournamentId).
		Str("division", evt.Division).
		Int32("round", evt.Round).
		Interface("playerIDs", playerIDs).
		Msg("both-players-ready-instantiating-game")

	foundUs := false
	otherID := ""
	users := [2]*entity.User{nil, nil}
	connIDs := [2]string{"", ""}
	otherUserIdx := -1
	// playerIDs are in order of first/second
	for idx, pid := range playerIDs {
		// userid:username:conn_id
		splitid := strings.Split(pid, ":")
		if len(splitid) != 3 {
			return errors.New("unexpected playerID: " + pid)
		}
		if userID == splitid[0] {
			foundUs = true
			users[idx] = reqUser
		} else {
			otherID = splitid[0]
			otherUserIdx = idx
		}
		connIDs[idx] = splitid[2]
		// Delete the ready state if it existed in Redis.
		_, err = redisConn.Do("DEL", "tready:"+connIDs[idx])
		if err != nil {
			return err
		}
	}
	if !foundUs {
		return errors.New("unexpected behavior; did not find us")
	}
	if otherID == userID {
		return errors.New("both users have same ID?")
	}
	if otherUserIdx == -1 {
		return errors.New("unexpected behavior; did not find other player")
	}
	users[otherUserIdx], err = b.stores.UserStore.GetByUUID(ctx, otherID)
	if err != nil {
		return err
	}
	if t.Divisions[evt.Division].DivisionManager == nil {
		return fmt.Errorf("division manager for division %s is nil", evt.Division)
	}
	gameReq := t.Divisions[evt.Division].DivisionManager.GetDivisionControls().GameRequest
	tdata := &entity.TournamentData{
		Id:        evt.TournamentId,
		Division:  evt.Division,
		Round:     int(evt.Round),
		GameIndex: int(evt.GameIndex),
	}

	g, err := gameplay.InstantiateNewGame(ctx, b.stores.GameStore, b.config,
		users, gameReq, tdata)
	if err != nil {
		log.Ctx(ctx).Error().
			Err(err).
			Str("userID", userID).
			Str("username", reqUser.Username).
			Str("otherUserID", otherID).
			Str("otherUsername", users[otherUserIdx].Username).
			Str("tournamentID", evt.TournamentId).
			Msg("error-instantiating-tournament-game")
		return err
	}

	log.Ctx(ctx).Info().
		Str("userID", userID).
		Str("username", reqUser.Username).
		Str("otherUserID", otherID).
		Str("otherUsername", users[otherUserIdx].Username).
		Str("gameID", g.GameID()).
		Str("tournamentID", evt.TournamentId).
		Str("division", evt.Division).
		Int32("round", evt.Round).
		Msg("tournament-game-created")

	err = b.broadcastGameCreation(g, reqUser, users[otherUserIdx])
	if err != nil {
		log.Err(err).Msg("broadcasting-game-creation")
	}

	// redirect users to the right game
	ngevt = entity.WrapEvent(&pb.NewGameEvent{
		GameId: g.GameID(),
		// doesn't matter who's the accepter or requester here.
		AccepterCid:  connIDs[0],
		RequesterCid: connIDs[1],
	}, pb.MessageType_NEW_GAME_EVENT)

	b.pubToConnectionID(connIDs[0], users[0].UUID, ngevt)
	b.pubToConnectionID(connIDs[1], users[1].UUID, ngevt)

	tcname, variant, err := entity.VariantFromGameReq(gameReq)
	if err != nil {
		return err
	}

	log.Info().Str("newgameid", g.History().Uid).
		Str("p0", userID).
		Str("p1", otherID).
		Str("lexicon", gameReq.Lexicon).
		Str("timectrl", string(tcname)).
		Str("variant", string(variant)).
		Str("tournamentID", string(evt.TournamentId)).
		Str("onturn", g.NickOnTurn()).Msg("tournament-game-started")

	// This is untested.
	g.SendChange(g.NewActiveGameEntry(true))
	return nil
}

func (b *Bus) sendGameRefresher(ctx context.Context, gameID, connID, userID string) error {
	// Get a game refresher event.
	entGame, err := b.stores.GameStore.Get(ctx, string(gameID))
	if err != nil {
		return err
	}
	entGame.RLock()
	defer entGame.RUnlock()

	if entGame.Type == pb.GameType_ANNOTATED {
		// Temporary solution for using a different game store to fetch these.
		// In the future, we will use the same game store for all games,
		// as all games will be GameDocuments.
		// For now we will not send a game refresher for annotated games, but
		// instead the front-end should request the GameDocument in this case.
		return nil
	}

	var evt *entity.EventWrapper
	log.Debug().Str("gameid", entGame.History().Uid).Msg("sent-refresher")

	if !entGame.Started && entGame.GameEndReason == pb.GameEndReason_NONE {
		log.Debug().Str("gameid", entGame.History().Uid).Msg("sent-refresher-unstarted-game")
		evt = entity.WrapEvent(&pb.ServerMessage{Message: "Game is starting soon!"},
			pb.MessageType_SERVER_MESSAGE)
	} else {
		log.Debug().Str("gameid", entGame.History().Uid).Msg("sent-refresher-for-started-game")
		hre := entGame.HistoryRefresherEvent()
		hre.History = mod.CensorHistory(ctx, b.stores.UserStore, hre.History)
		evt = entity.WrapEvent(hre,
			pb.MessageType_GAME_HISTORY_REFRESHER)
	}
	err = b.pubToConnectionID(connID, userID, evt)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bus) adjudicateGames(ctx context.Context, correspondenceOnly bool) error {
	// Always bust the cache when we're adjudicating games.

	if correspondenceOnly {
		// Optimized path for correspondence games - check timeouts without loading full games
		return b.adjudicateCorrespondenceGames(ctx)
	}

	// Original path for realtime games
	var gs *pb.GameInfoResponses
	var err error

	gs, err = b.stores.GameStore.ListActive(ctx, "", true)

	if err != nil {
		return err
	}
	now := time.Now()
	log.Debug().Bool("correspondence", correspondenceOnly).Interface("active-games", gs).Msg("maybe-adjudicating...")
	for _, g := range gs.GameInfo {
		// These will likely be in the cache.
		entGame, err := b.stores.GameStore.Get(ctx, g.GameId)
		if err != nil {
			return err
		}
		entGame.RLock()
		onTurn := entGame.Game.PlayerOnTurn()
		started := entGame.Started
		timeRanOut := entGame.TimeRanOut(onTurn)
		entGame.RUnlock()

		if started && timeRanOut {
			log.Debug().Str("gid", g.GameId).Msg("adjudicating-time-ran-out")
			err = gameplay.TimedOut(ctx, b.stores, entGame.Game.PlayerIDOnTurn(), g.GameId)
			log.Err(err).Msg("adjudicating-after-gameplay-timed-out")
		} else if !started && !entGame.IsCorrespondence() {
			// Only real-time games can be "not started" - correspondence games
			// are auto-started immediately when accepted, so they never enter this state.
			cancelThreshold := CancelAfter // 60 seconds for real-time

			if now.Sub(entGame.CreatedAt) > cancelThreshold {
				log.Debug().Str("gid", g.GameId).
					Str("tid", g.TournamentId).
					Str("threshold", cancelThreshold.String()).
					Bool("correspondence", entGame.IsCorrespondence()).
					Interface("now", now).
					Interface("created", entGame.CreatedAt).
					Msg("canceling-never-started")

					// need to lock game to abort? maybe lock inside AbortGame?
				log.Debug().Str("gid", g.GameId).Msg("locking")
				entGame.Lock()
				err = gameplay.AbortGame(ctx, b.stores, entGame, pb.GameEndReason_CANCELLED)
				log.Err(err).Msg("adjudicating-after-abort-game")
				entGame.Unlock()
				log.Debug().Str("gid", g.GameId).Msg("unlocking")

				// Delete the game from the lobby. We do this here instead
				// of inside the gameplay package because the game event channel
				// was never registered with an unstarted game.
				wrapped := entity.WrapEvent(&pb.GameDeletion{Id: g.GameId},
					pb.MessageType_GAME_DELETION)
				wrapped.AddAudience(entity.AudLobby, "gameEnded")
				// send it to the tournament channel too if it's in one
				if g.TournamentId != "" {
					wrapped.AddAudience(entity.AudTournament, g.TournamentId)
				}
				b.gameEventChan <- wrapped
			}
		}
	}
	log.Debug().Interface("active-games", gs).Msg("exiting-adjudication...")

	return nil
}

// adjudicateCorrespondenceGames is an optimized version for correspondence games
// that checks for timeouts without loading the full game from the database.
func (b *Bus) adjudicateCorrespondenceGames(ctx context.Context) error {
	// Get active correspondence games with timers
	games, err := b.stores.GameStore.ListActiveCorrespondenceRaw(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	log.Debug().Int("num-games", len(games)).Msg("adjudicating-correspondence-games")

	for _, g := range games {
		// Skip if not started
		if !g.Started.Valid || !g.Started.Bool {
			continue
		}

		// Skip if no player on turn (shouldn't happen but be defensive)
		if !g.PlayerOnTurn.Valid {
			continue
		}

		playerOnTurn := int(g.PlayerOnTurn.Int32)

		// Check if time ran out using only DB columns
		// This mirrors the logic in entity.Game.TimeRanOut()

		// Skip annotated games
		if g.Type.Valid && g.Type.Int32 == int32(pb.GameType_ANNOTATED) {
			continue
		}

		// For correspondence games, check time bank before declaring timeout
		// Correspondence games use ResetToIncrementAfterTurn behavior
		turnTime := now - g.Timers.TimeOfLastUpdate
		allowedTime := int64(g.GameRequest.IncrementSeconds) * 1000

		// If within allowed time, no timeout
		if turnTime <= allowedTime {
			continue
		}

		// Player took too long, check if time bank can cover the deficit
		deficit := turnTime - allowedTime
		if len(g.Timers.TimeBank) > playerOnTurn && g.Timers.TimeBank[playerOnTurn] >= deficit {
			// Time bank covers it, no timeout
			continue
		}

		// Time bank exhausted, player timed out!
		gameID := g.Uuid.String
		log.Debug().Str("gid", gameID).Int64("deficit", deficit).
			Int64("timeBank", g.Timers.TimeBank[playerOnTurn]).
			Msg("adjudicating-time-ran-out")

		// Get the player ID from quickdata
		if playerOnTurn < 0 || playerOnTurn >= len(g.Quickdata.PlayerInfo) {
			log.Error().Str("gid", gameID).Int("playerOnTurn", playerOnTurn).Msg("invalid-player-on-turn")
			continue
		}
		playerID := g.Quickdata.PlayerInfo[playerOnTurn].UserId

		err = gameplay.TimedOut(ctx, b.stores, playerID, gameID)
		log.Err(err).Msg("adjudicating-after-gameplay-timed-out")
	}

	log.Debug().Msg("exiting-correspondence-adjudication")
	return nil
}

func (b *Bus) gameMetaEvent(ctx context.Context, evt *pb.GameMetaEvent, userID string) error {
	// Make sure we are not sending more abort/etc requests than allowed.

	// Overwrite whatever was passed in with the userID we know made this request.
	evt.PlayerId = userID
	if evt.OrigEventId == "" {
		evt.OrigEventId = shortuuid.New()
	}

	return gameplay.HandleMetaEvent(ctx, evt, b.gameEventChan, b.stores)
}
