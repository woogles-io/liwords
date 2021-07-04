package bus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/tournament"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var (
	errGamesDisabled = errors.New("new games are temporarily disabled; please try again in a few minutes")
)

const (
	TournamentReadyExpire = 3600
)

func (b *Bus) instantiateAndStartGame(ctx context.Context, accUser *entity.User, requester string,
	gameReq *pb.GameRequest, sg *entity.SoughtGame, reqID, acceptingConnID string) error {

	reqUser, err := b.userStore.GetByUUID(ctx, requester)
	if err != nil {
		return err
	}

	enabled, err := b.configStore.GamesEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return errGamesDisabled
	}

	// disallow anon game acceptance for now.
	if accUser.Anonymous || reqUser.Anonymous {
		return errors.New("you must log in to play games")
	}

	if (accUser.Anonymous || reqUser.Anonymous) && gameReq.RatingMode == pb.RatingMode_RATED {
		return errors.New("anonymous-players-cant-play-rated")
	}

	log.Debug().Interface("req", sg).
		Str("seek-conn", sg.ConnID()).
		Str("accepting-conn", acceptingConnID).Msg("game-request-accepted")
	assignedFirst := -1
	var tournamentID string
	if sg.Type == entity.TypeMatch {
		if sg.MatchRequest.RematchFor != "" {
			// Assign firsts to be the other player.
			gameID := sg.MatchRequest.RematchFor
			g, err := b.gameStore.Get(ctx, gameID)
			if err != nil {
				return err
			}
			wentFirst := 0
			players := g.History().Players
			if g.History().SecondWentFirst {
				wentFirst = 1
			}
			log.Debug().Str("went-first", players[wentFirst].Nickname).Msg("determining-first")

			// These are indices in the array passed to InstantiateNewGame
			if accUser.UUID == players[wentFirst].UserId {
				assignedFirst = 1 // reqUser should go first
			} else if reqUser.UUID == players[wentFirst].UserId {
				assignedFirst = 0 // accUser should go first
			}
		}
		if sg.MatchRequest.TournamentId != "" {
			t, err := b.tournamentStore.Get(ctx, sg.MatchRequest.TournamentId)
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

	g, err := gameplay.InstantiateNewGame(ctx, b.gameStore, b.config,
		[2]*entity.User{accUser, reqUser}, assignedFirst, gameReq, trdata)
	if err != nil {
		return err
	}
	// Broadcast a seek delete event, and send both parties a game redirect.
	if reqID != BotRequestID {
		b.soughtGameStore.Delete(ctx, reqID)
		err = b.sendSoughtGameDeletion(ctx, sg)
		if err != nil {
			log.Err(err).Msg("broadcasting-sg-deletion")
		}
	}

	err = b.broadcastGameCreation(g, accUser, reqUser)
	if err != nil {
		log.Err(err).Msg("broadcasting-game-creation")
	}
	// This event will result in a redirect.
	ngevt := entity.WrapEvent(&pb.NewGameEvent{
		GameId:       g.GameID(),
		AccepterCid:  acceptingConnID,
		RequesterCid: sg.ConnID(),
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

func (b *Bus) goHandleBotMove(ctx context.Context, g *entity.Game) {
	// This function should only be called if it's the bot's turn.
	// Call it while holding at least a read lock!
	onTurn := g.Game.PlayerOnTurn()
	userID := g.Game.PlayerIDOnTurn()

	go b.handleBotMoveInternally(ctx, g, onTurn, userID)
}

func (b *Bus) handleBotMoveInternally(ctx context.Context, g *entity.Game, onTurn int, userID string) {
	// This function should only be called by goHandleBotMove.
	// Caller should pass the bot's onTurn and userID.
	g.Lock()
	defer g.Unlock()
	// We check if that game is not over because a triple challenge
	// could have ended it
	for g.PlayerOnTurn() == onTurn && g.Game.Playing() != macondopb.PlayState_GAME_OVER {
		hist := g.History()
		req := macondopb.BotRequest{GameHistory: hist}
		data, err := proto.Marshal(&req)
		if err != nil {
			log.Err(err).Msg("bot-cant-move")
			return
		}
		res, err := b.natsconn.Request("macondo.bot", data, 10*time.Second)

		if err != nil {
			if b.natsconn.LastError() != nil {
				log.Error().Msgf("bot-cant-move %v for request", b.natsconn.LastError())
			}
			log.Error().Msgf("bot-cant-move %v for request", err)
			return
		}
		log.Debug().Msgf("res: %v", string(res.Data))

		resp := macondopb.BotResponse{}
		err = proto.Unmarshal(res.Data, &resp)
		if err != nil {
			log.Err(err).Msg("bot-cant-move-unmarshal-error")
			return
		}
		switch r := resp.Response.(type) {
		case *macondopb.BotResponse_Move:
			timeRemaining := g.TimeRemaining(onTurn)

			m := game.MoveFromEvent(r.Move, g.Alphabet(), g.Board())
			err = gameplay.PlayMove(ctx, g, b.gameStore, b.userStore, b.notorietyStore, b.listStatStore, b.tournamentStore, userID, onTurn, timeRemaining, m)
			if err != nil {
				log.Err(err).Msg("bot-cant-move-play-error")
				return
			}
		case *macondopb.BotResponse_Error:
			log.Error().Str("error", r.Error).Msg("bot-error")
			return
		default:
			log.Err(errors.New("should never happen")).Msg("bot-cant-move")
		}
	}

	err := b.gameStore.Set(ctx, g)
	if err != nil {
		log.Err(err).Msg("setting-game-after-bot-move")
	}

}

func (b *Bus) readyForGame(ctx context.Context, evt *pb.ReadyForGame, userID string) error {
	g, err := b.gameStore.Get(ctx, evt.GameId)
	if err != nil {
		return err
	}
	g.Lock()
	defer g.Unlock()
	log.Debug().Str("userID", userID).Interface("playing", g.Playing()).Msg("ready-for-game")
	if g.Playing() != macondopb.PlayState_PLAYING {
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
	rf, err := b.gameStore.SetReady(ctx, evt.GameId, readyID)
	if err != nil {
		log.Err(err).Msg("in-set-ready")
		return err
	}
	log.Debug().Interface("rf", rf).Msg("ready-flag")
	// Start the game if both players are ready (or if it's a bot game).
	// readyflag will be (01 | 10) = 3 for two players.
	if rf == (1<<len(g.History().Players))-1 || g.GameReq.PlayerVsBot {
		err = gameplay.StartGame(ctx, b.gameStore, b.userStore, b.gameEventChan, g.GameID())
		if err != nil {
			log.Err(err).Msg("starting-game")
		}
		// Note: for PlayerVsBot, readyForGame is called twice when player is ready and every time player refreshes, why? :-(
		g.SendChange(g.NewActiveGameEntry(true))

		if g.GameReq.PlayerVsBot && g.PlayerIDOnTurn() != userID {
			// Make a bot move if it's the bot's turn at the beginning.
			b.goHandleBotMove(ctx, g)
		}
	}
	return nil
}

func (b *Bus) readyForTournamentGame(ctx context.Context, evt *pb.ReadyForTournamentGame, userID, connID string) error {
	if !evt.Unready {
		enabled, err := b.configStore.GamesEnabled(ctx)
		if err != nil {
			return err
		}
		if !enabled {
			return errGamesDisabled
		}
	}

	reqUser, err := b.userStore.GetByUUID(ctx, userID)
	if err != nil {
		return err
	}

	fullUserID := reqUser.TournamentID()

	t, err := b.tournamentStore.Get(ctx, evt.TournamentId)
	if err != nil {
		return err
	}

	playerIDs, bothReady, err := tournament.SetReadyForGame(ctx, b.tournamentStore, t, fullUserID, connID,
		evt.Division, int(evt.Round), int(evt.GameIndex), evt.Unready)

	if err != nil {
		return err
	}

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
				return errGamesDisabled
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
	users[otherUserIdx], err = b.userStore.GetByUUID(ctx, otherID)
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

	g, err := gameplay.InstantiateNewGame(ctx, b.gameStore, b.config,
		users, 0, gameReq, tdata)
	if err != nil {
		return err
	}

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

func (b *Bus) gameRefresher(ctx context.Context, gameID string) (*entity.EventWrapper, error) {
	// Get a game refresher event.
	entGame, err := b.gameStore.Get(ctx, string(gameID))
	if err != nil {
		return nil, err
	}
	entGame.RLock()
	defer entGame.RUnlock()
	if !entGame.Started && entGame.GameEndReason == pb.GameEndReason_NONE {
		return entity.WrapEvent(&pb.ServerMessage{Message: "Game is starting soon!"},
			pb.MessageType_SERVER_MESSAGE), nil
	}
	hre := entGame.HistoryRefresherEvent()
	hre.History = mod.CensorHistory(ctx, b.userStore, hre.History)
	evt := entity.WrapEvent(hre,
		pb.MessageType_GAME_HISTORY_REFRESHER)
	return evt, nil
}

func (b *Bus) adjudicateGames(ctx context.Context) error {
	// Always bust the cache when we're adjudicating games.
	gs, err := b.gameStore.ListActive(ctx, "", true)

	if err != nil {
		return err
	}
	now := time.Now()
	log.Debug().Interface("active-games", gs).Msg("maybe-adjudicating...")
	for _, g := range gs.GameInfo {
		// These will likely be in the cache.
		entGame, err := b.gameStore.Get(ctx, g.GameId)
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
			err = gameplay.TimedOut(ctx, b.gameStore, b.userStore, b.notorietyStore,
				b.listStatStore, b.tournamentStore, entGame.Game.PlayerIDOnTurn(), g.GameId)
			log.Err(err).Msg("adjudicating-after-gameplay-timed-out")
		} else if !started && now.Sub(entGame.CreatedAt) > CancelAfter {
			log.Debug().Str("gid", g.GameId).
				Str("tid", g.TournamentId).
				Interface("now", now).
				Interface("created", entGame.CreatedAt).
				Msg("canceling-never-started")

				// need to lock game to abort? maybe lock inside AbortGame?
			entGame.Lock()
			err = gameplay.AbortGame(ctx, b.gameStore, b.tournamentStore,
				entGame, pb.GameEndReason_CANCELLED)
			log.Err(err).Msg("adjudicating-after-abort-game")
			entGame.Unlock()
			// Delete the game from the lobby. We do this here instead
			// of inside the gameplay package because the game event channel
			// was never registered with an unstarted game.
			wrapped := entity.WrapEvent(&pb.GameDeletion{Id: g.GameId},
				pb.MessageType_GAME_DELETION)
			// XXX: Fix for tourneys ?
			wrapped.AddAudience(entity.AudLobby, "gameEnded")
			b.gameEventChan <- wrapped
		}
	}
	return nil
}

func (b *Bus) gameMetaEvent(ctx context.Context, evt *pb.GameMetaEvent, userID string) error {
	// Make sure we are not sending more abort/etc requests than allowed.

	// Overwrite whatever was passed in with the userID we know made this request.
	evt.PlayerId = userID
	if evt.OrigEventId == "" {
		evt.OrigEventId = shortuuid.New()
	}

	return gameplay.HandleMetaEvent(ctx, evt, b.gameEventChan, b.gameStore, b.userStore, b.notorietyStore, b.listStatStore, b.tournamentStore)
}
